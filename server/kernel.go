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
	if err := k.syncCollections(context.Background()); err != nil {
		return fmt.Errorf("itab: schema sync: %w", err)
	}
	if err := k.ensureBootstrap(context.Background()); err != nil {
		return fmt.Errorf("itab: bootstrap: %w", err)
	}
	var registerErr error
	group.Group(k.apiPrefix, func(sub *ghttp.RouterGroup) {
		for i := range k.collections {
			c := k.collections[i]
			sub.GET(c.Name, k.aclWrap(c, ActionList, k.handleList(c)))
			sub.GET(c.Name+"/:id", k.aclWrap(c, ActionGet, k.handleGet(c)))
			sub.POST(c.Name, k.aclWrap(c, ActionCreate, k.handleCreate(c)))
			sub.PATCH(c.Name+"/:id", k.aclWrap(c, ActionUpdate, k.handleUpdate(c)))
			sub.DELETE(c.Name+"/:id", k.aclWrap(c, ActionDelete, k.handleDelete(c)))
		}
		for _, route := range k.customRoutes {
			wrapped := k.routeACLWrap(route.ACL, route.Handler)
			if err := mountCustomRoute(sub, route, wrapped); err != nil {
				registerErr = err
				return
			}
		}
		// Meta endpoints under <apiPrefix>/meta/*, auth required.
		sub.GET("/meta/whoami", k.routeACLWrap(RequireAuthed(), k.handleWhoami))
		sub.GET("/meta/collections", k.routeACLWrap(RequireAuthed(), k.handleMetaCollections))
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
