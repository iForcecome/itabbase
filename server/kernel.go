// Package itab is the v0 minimal kernel: collection registry + auto CRUD + schema sync.
//
// Usage:
//
//	k := itab.New(itab.WithDB(g.DB()))
//	k.RegisterCollection(itab.Collection{Name: "todos", Fields: []itab.Field{...}})
//	if err := k.Mount(group); err != nil { ... }
package itab

import (
	"context"
	"errors"
	"fmt"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/net/ghttp"
)

type Kernel struct {
	db            gdb.DB
	collections   []Collection
	apiPrefix     string
	auth          AuthAdapter
	aclDisabled   bool
	builtinAuth   bool
	customRoutes  []Route
	reservedPaths []string
}

type Option func(*Kernel)

func WithDB(db gdb.DB) Option {
	return func(k *Kernel) { k.db = db }
}

func WithAPIPrefix(p string) Option {
	return func(k *Kernel) { k.apiPrefix = p }
}

// WithReservedPaths adds collection names that must NOT be registered, on top
// of kernel's defaults ("auth", "admin", "meta"). Use this to declare scaffold
// routes that share the apiPrefix namespace and would collide with a
// same-named collection (e.g. "health", "notifications").
func WithReservedPaths(names ...string) Option {
	return func(k *Kernel) {
		k.reservedPaths = append(k.reservedPaths, names...)
	}
}

// WithAuth installs the auth adapter that resolves the current user and roles.
// Either WithAuth or WithoutAuth must be set before Mount.
func WithAuth(a AuthAdapter) Option {
	return func(k *Kernel) {
		k.auth = a
		k.aclDisabled = false
	}
}

// WithoutAuth disables the ACL layer entirely. All requests pass through.
// Intended for dev/test scaffolding where SSO isn't wired up yet; not for prod.
func WithoutAuth() Option {
	return func(k *Kernel) {
		k.auth = nil
		k.aclDisabled = true
	}
}

func New(opts ...Option) *Kernel {
	k := &Kernel{
		apiPrefix:     "/api",
		reservedPaths: []string{"auth", "admin", "meta"},
	}
	for _, o := range opts {
		o(k)
	}
	k.registerBuiltins()
	return k
}

// collectionByName returns the registered collection with the given name.
func (k *Kernel) collectionByName(name string) (Collection, bool) {
	for _, c := range k.collections {
		if c.Name == name {
			return c, true
		}
	}
	return Collection{}, false
}

func (k *Kernel) RegisterCollection(c Collection) {
	if err := c.Validate(); err != nil {
		panic("itab: " + err.Error())
	}
	for _, existing := range k.collections {
		if existing.Name == c.Name {
			panic(fmt.Sprintf("itab: collection %q already registered", c.Name))
		}
	}
	k.collections = append(k.collections, c)
}

// Mount syncs schema for all registered collections and registers CRUD routes
// onto the given router group, prefixed by apiPrefix (default "/api").
// Meta endpoints (whoami, collections) are mounted under <apiPrefix>/meta/*.
// The admin SPA static path "/admin/*" is mounted at the group root.
func (k *Kernel) Mount(group *ghttp.RouterGroup) error {
	if k.db == nil {
		return errors.New("itab: WithDB is required")
	}
	if group == nil {
		return errors.New("itab: Mount requires a non-nil router group")
	}
	if k.auth == nil && !k.aclDisabled {
		return errors.New("itab: WithAuth or WithoutAuth must be called before Mount")
	}
	if err := k.checkReservedPaths(); err != nil {
		return err
	}

	ctx := context.Background()

	// Phase 1: sync builtin tables (including _collections / _fields).
	if err := k.syncCollections(ctx); err != nil {
		return fmt.Errorf("itab: schema sync (builtins): %w", err)
	}

	// Phase 2: load dynamic collections from _collections + _fields DB tables.
	if err := k.loadDynamicCollections(ctx); err != nil {
		return fmt.Errorf("itab: load dynamic collections: %w", err)
	}

	// Phase 3: sync dynamic collection tables (CREATE TABLE / ADD COLUMN).
	if err := k.syncCollections(ctx); err != nil {
		return fmt.Errorf("itab: schema sync (dynamic): %w", err)
	}

	// Phase 4: bootstrap seed data (roles, admin user, default settings).
	if err := k.ensureBootstrap(ctx); err != nil {
		return fmt.Errorf("itab: bootstrap: %w", err)
	}

	var registerErr error
	group.Group(k.apiPrefix, func(sub *ghttp.RouterGroup) {
		// 1. Custom routes first (most specific paths).
		for _, route := range k.customRoutes {
			wrapped := k.routeACLWrap(route.ACL, route.Handler)
			if err := mountCustomRoute(sub, route, wrapped); err != nil {
				registerErr = err
				return
			}
		}

		// 2. Auth endpoints (only when using built-in auth).
		if k.builtinAuth {
			sub.POST("/auth/local/login", k.handleLocalLogin)
			sub.POST("/auth/logout", k.handleLogout)
		}

		// 3. Meta endpoints (explicit paths, highest priority).
		sub.GET("/meta/whoami", k.routeACLWrap(RequireAuthed(), k.handleWhoami))
		sub.GET("/meta/collections", k.routeACLWrap(RequireAuthed(), k.handleMetaCollections))

		// 4. Dynamic collection management API (admin-only).
		sub.POST("/meta/collections", k.routeACLWrap(RequireRole("admin"), k.handleCreateCollection))
		sub.PATCH("/meta/collections/:name", k.routeACLWrap(RequireRole("admin"), k.handleUpdateCollection))
		sub.DELETE("/meta/collections/:name", k.routeACLWrap(RequireRole("admin"), k.handleDeleteCollection))
		sub.POST("/meta/collections/:name/fields", k.routeACLWrap(RequireRole("admin"), k.handleAddField))
		sub.PATCH("/meta/collections/:name/fields/:fieldName", k.routeACLWrap(RequireRole("admin"), k.handleUpdateField))
		sub.DELETE("/meta/collections/:name/fields/:fieldName", k.routeACLWrap(RequireRole("admin"), k.handleDeleteField))

		// 5. Universal dynamic CRUD — resolves collection by name at request time.
		// This handles ALL collections (builtin + code + dynamic), including ones
		// created at runtime after Mount(). GoFrame routes exact paths (meta/*)
		// before parameter paths (/:_col), so there are no conflicts.
		sub.GET("/:_col", k.dynamicCRUD(ActionList))
		sub.GET("/:_col/:id", k.dynamicCRUD(ActionGet))
		sub.POST("/:_col", k.dynamicCRUD(ActionCreate))
		sub.PATCH("/:_col/:id", k.dynamicCRUD(ActionUpdate))
		sub.DELETE("/:_col/:id", k.dynamicCRUD(ActionDelete))
	})
	if registerErr != nil {
		return registerErr
	}
	// Admin SPA static files at <group>/admin/*. The SPA handles its own
	// login UX so these routes are public.
	//
	// gf v2's `/admin/*any` wildcard also matches `/admin` (no trailing slash),
	// so the trailing-slash redirect lives inside serveAdminSPA itself — a
	// separate `group.GET("/admin", ...)` is shadowed by the wildcard.
	group.GET("/admin", k.serveAdminSPA)
	group.GET("/admin/*any", k.serveAdminSPA)
	return nil
}

// checkReservedPaths fails Mount if any registered collection name collides
// with a reserved path under apiPrefix.
func (k *Kernel) checkReservedPaths() error {
	for _, c := range k.collections {
		for _, rp := range k.reservedPaths {
			if c.Name == rp {
				return fmt.Errorf("itab: collection %q collides with reserved path %q under apiPrefix %q (use WithReservedPaths to inspect, or rename the collection)", c.Name, rp, k.apiPrefix)
			}
		}
	}
	return nil
}
