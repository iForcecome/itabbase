package itab

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/net/ghttp"

	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/auth"
	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/bootstrap"
	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/handler"
	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/model"
	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/schema"
)

type Kernel struct {
	db            gdb.DB
	collections   []model.Collection
	mu            sync.RWMutex
	apiPrefix     string
	auth          model.AuthAdapter
	aclDisabled   bool
	builtinAuth   bool
	authConfig    model.SSOConfig
	customRoutes  []model.Route
	reservedPaths []string

	ssoEnabled  bool
	ssoProvider model.OAuthProvider
	ssoConfig   model.SSOConfig
	ssoHandler  *auth.SSOHandler
}

type Option func(*Kernel)

func WithDB(db gdb.DB) Option {
	return func(k *Kernel) { k.db = db }
}

func WithAPIPrefix(p string) Option {
	return func(k *Kernel) { k.apiPrefix = p }
}

// WithReservedPaths adds collection names that must NOT be registered.
func WithReservedPaths(names ...string) Option {
	return func(k *Kernel) {
		k.reservedPaths = append(k.reservedPaths, names...)
	}
}

// WithAuth installs the auth adapter that resolves the current user and roles.
func WithAuth(a AuthAdapter) Option {
	return func(k *Kernel) {
		k.auth = a
		k.aclDisabled = false
	}
}

// WithoutAuth disables the ACL layer entirely.
func WithoutAuth() Option {
	return func(k *Kernel) {
		k.auth = nil
		k.aclDisabled = true
	}
}

// WithBuiltinAuth installs a session-based local password auth adapter.
func WithBuiltinAuth() Option {
	return func(k *Kernel) {
		cfg := auth.SSOConfigDefaults(model.SSOConfig{})
		adapter := &auth.BuiltinAdapter{DB: k.db, Config: cfg}
		k.auth = adapter
		k.aclDisabled = false
		k.builtinAuth = true
		k.authConfig = cfg
	}
}

// WithSSOAuth installs an SSO-based auth adapter using the given provider.
func WithSSOAuth(provider OAuthProvider, cfg SSOConfig) Option {
	return func(k *Kernel) {
		cfg = auth.SSOConfigDefaults(cfg)
		adapter := &auth.BuiltinAdapter{DB: k.db, Config: cfg}
		k.auth = adapter
		k.aclDisabled = false
		k.builtinAuth = true
		k.ssoEnabled = true
		k.ssoProvider = provider
		k.ssoConfig = cfg
		k.authConfig = cfg
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

func (k *Kernel) RegisterCustomRoute(r Route) {
	if r.Method == "" || r.Path == "" || r.Handler == nil {
		panic("itab: Route requires Method, Path and Handler")
	}
	k.customRoutes = append(k.customRoutes, r)
}

// Mount syncs schema for all registered collections and registers CRUD routes.
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

	// Phase 1: sync builtin tables.
	if err := schema.SyncAll(ctx, k.db, k.collections); err != nil {
		return fmt.Errorf("itab: schema sync (builtins): %w", err)
	}

	// Phase 2: load dynamic collections from DB.
	dynCols, err := schema.LoadDynamic(ctx, k.db, k.collections)
	if err != nil {
		return fmt.Errorf("itab: load dynamic collections: %w", err)
	}
	k.collections = append(k.collections, dynCols...)

	// Phase 3: sync dynamic collection tables.
	if err := schema.SyncNonBuiltin(ctx, k.db, k.collections); err != nil {
		return fmt.Errorf("itab: schema sync (dynamic): %w", err)
	}

	// Phase 4: bootstrap seed data.
	if err := bootstrap.Seed(ctx, k.db); err != nil {
		return fmt.Errorf("itab: bootstrap: %w", err)
	}

	// Build handler env with shared state.
	env := &handler.Env{
		DB:            k.db,
		Auth:          k.auth,
		ACLDisabled:   k.aclDisabled,
		Mu:            &k.mu,
		Collections:   &k.collections,
		ReservedPaths: k.reservedPaths,
	}

	// internalRoute builds a Route with only ACL set (no Collection binding).
	internalRoute := func(acl RouteACL) Route {
		return Route{ACL: acl}
	}

	var registerErr error
	group.Group(k.apiPrefix, func(sub *ghttp.RouterGroup) {
		// 1. Custom routes first.
		for _, route := range k.customRoutes {
			wrapped := env.RouteACLWrap(route, route.Handler)
			if err := handler.MountCustomRoute(sub, route, wrapped); err != nil {
				registerErr = err
				return
			}
		}

		// 2. Auth endpoints.
		if k.builtinAuth {
			sub.POST("/auth/local/login", auth.HandleLocalLogin(k.db, k.authConfig))
			sub.POST("/auth/logout", auth.HandleLogout(k.db, k.authConfig))
		}
		if k.ssoEnabled {
			k.ssoHandler = &auth.SSOHandler{
				DB:       k.db,
				Provider: k.ssoProvider,
				Config:   k.ssoConfig,
			}
			sub.GET("/auth/login", k.ssoHandler.HandleSSOLogin)
			sub.GET("/auth/callback", k.ssoHandler.HandleSSOCallback)
		}

		// 3. Meta endpoints.
		sub.GET("/meta/whoami", env.RouteACLWrap(internalRoute(RequireAuthed()), env.HandleWhoami))
		sub.GET("/meta/collections", env.RouteACLWrap(internalRoute(RequireAuthed()), env.HandleMetaCollections))

		// 4. Dynamic collection management API (admin-only).
		adminRoute := internalRoute(RequireRole("admin"))
		// POST /meta/collections is upsert (apply) semantics; see handler.HandleApplyCollection.
		sub.POST("/meta/collections", env.RouteACLWrap(adminRoute, env.HandleApplyCollection))
		sub.PATCH("/meta/collections/:name", env.RouteACLWrap(adminRoute, env.HandleUpdateCollection))
		sub.DELETE("/meta/collections/:name", env.RouteACLWrap(adminRoute, env.HandleDeleteCollection))
		sub.POST("/meta/collections/:name/fields", env.RouteACLWrap(adminRoute, env.HandleAddField))
		sub.PATCH("/meta/collections/:name/fields/:fieldName", env.RouteACLWrap(adminRoute, env.HandleUpdateField))
		sub.DELETE("/meta/collections/:name/fields/:fieldName", env.RouteACLWrap(adminRoute, env.HandleDeleteField))

		// 5. OpenAPI spec (public, no auth required).
		sub.GET("/docs.json", env.HandleOpenAPISpec(handler.OpenAPIOptions{
			APIPrefix:    k.apiPrefix,
			BuiltinAuth:  k.builtinAuth,
			SSOEnabled:   k.ssoEnabled,
			CustomRoutes: k.customRoutes,
		}))

		// 6. Universal dynamic CRUD.
		sub.GET("/:_col", env.DynamicCRUD(ActionList))
		sub.GET("/:_col/:id", env.DynamicCRUD(ActionGet))
		sub.POST("/:_col", env.DynamicCRUD(ActionCreate))
		sub.PATCH("/:_col/:id", env.DynamicCRUD(ActionUpdate))
		sub.DELETE("/:_col/:id", env.DynamicCRUD(ActionDelete))
	})
	if registerErr != nil {
		return registerErr
	}

	group.GET("/admin", k.serveAdminSPA)
	group.GET("/admin/*any", k.serveAdminSPA)
	return nil
}

func (k *Kernel) registerBuiltins() {
	builtins := schema.BuiltinCollections()
	for i := range builtins {
		if builtins[i].Name == model.BuiltinUsers {
			builtins[i].Hooks.AfterUpdate = k.bindUserRoleOnActivate
		}
		if builtins[i].Source == "" {
			builtins[i].Source = model.SourceBuiltin
			builtins[i].Internal = true
		}
		if err := builtins[i].Validate(); err != nil {
			panic("itab builtin: " + err.Error())
		}
		k.collections = append(k.collections, builtins[i])
	}
}

func (k *Kernel) bindUserRoleOnActivate(ctx context.Context, rec *Record) error {
	status, _ := rec.Get("status").(string)
	if status != UserStatusActive {
		return nil
	}
	var userID int64
	switch v := rec.Get("id").(type) {
	case int64:
		userID = v
	case int:
		userID = int64(v)
	case float64:
		userID = int64(v)
	default:
		return nil
	}
	if userID == 0 {
		return nil
	}
	n, err := k.db.Model(BuiltinUserRoles).Ctx(ctx).Where("user_id", userID).Count()
	if err != nil || n > 0 {
		return err
	}
	roleRow, err := k.db.Model(BuiltinRoles).Ctx(ctx).Where("name", "user").One()
	if err != nil || roleRow.IsEmpty() {
		return err
	}
	_, err = k.db.Model(BuiltinUserRoles).Ctx(ctx).Insert(map[string]any{
		"user_id": userID,
		"role_id": roleRow["id"].Int64(),
	})
	return err
}

func (k *Kernel) checkReservedPaths() error {
	for _, c := range k.collections {
		for _, rp := range k.reservedPaths {
			if c.Name == rp {
				return fmt.Errorf("itab: collection %q collides with reserved path %q under apiPrefix %q", c.Name, rp, k.apiPrefix)
			}
		}
	}
	return nil
}
