package itab

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"
)

// Route declares a non-CRUD endpoint registered under the kernel's apiPrefix.
type Route struct {
	Method  string
	Path    string
	Handler ghttp.HandlerFunc
	ACL     RouteACL
}

// RouteACL controls access to a custom route.
//
// Default zero-value (RequireAuthed): any authenticated user, no role check.
// Use RequireRole(...) to restrict to specific roles.
// Use AllowAnonymous() to permit unauthenticated access.
type RouteACL struct {
	Roles     []string
	Anonymous bool
}

// RequireAuthed permits any authenticated user (any role).
func RequireAuthed() RouteACL { return RouteACL{} }

// RequireRole permits only authenticated users that hold any of the listed roles.
func RequireRole(roles ...string) RouteACL { return RouteACL{Roles: roles} }

// AllowAnonymous permits unauthenticated requests.
func AllowAnonymous() RouteACL { return RouteACL{Anonymous: true} }

func (k *Kernel) RegisterCustomRoute(r Route) {
	if r.Method == "" || r.Path == "" || r.Handler == nil {
		panic("itab: Route requires Method, Path and Handler")
	}
	k.customRoutes = append(k.customRoutes, r)
}

func (k *Kernel) routeACLWrap(acl RouteACL, h ghttp.HandlerFunc) ghttp.HandlerFunc {
	if k.aclDisabled {
		return h
	}
	return func(r *ghttp.Request) {
		u, err := k.auth.CurrentUser(r)
		authed := err == nil
		if authed {
			userRoles := k.auth.RolesOf(u)
			ctx := ctxWithUser(r.Context(), u)
			ctx = ctxWithRoles(ctx, userRoles, true)
			r.SetCtx(ctx)
			if len(acl.Roles) > 0 && !anyMatch(acl.Roles, userRoles) {
				writeErr(r, http.StatusForbidden, "forbidden", nil)
				return
			}
			h(r)
			return
		}
		if !errors.Is(err, ErrUnauthenticated) {
			writeErr(r, http.StatusInternalServerError, "auth lookup failed", err)
			return
		}
		r.SetCtx(ctxWithRoles(r.Context(), []string{RoleAnonymous}, false))
		if acl.Anonymous {
			h(r)
			return
		}
		writeErr(r, http.StatusUnauthorized, "unauthenticated", nil)
	}
}

func anyMatch(want, have []string) bool {
	for _, w := range want {
		for _, h := range have {
			if w == h {
				return true
			}
		}
	}
	return false
}

func mountCustomRoute(group *ghttp.RouterGroup, route Route, h ghttp.HandlerFunc) error {
	switch strings.ToUpper(route.Method) {
	case "GET":
		group.GET(route.Path, h)
	case "POST":
		group.POST(route.Path, h)
	case "PUT":
		group.PUT(route.Path, h)
	case "PATCH":
		group.PATCH(route.Path, h)
	case "DELETE":
		group.DELETE(route.Path, h)
	default:
		return fmt.Errorf("itab: unsupported HTTP method %q for route %s", route.Method, route.Path)
	}
	return nil
}
