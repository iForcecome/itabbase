package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"

	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/model"
)

// ACLWrap returns a handler that enforces collection ACL before delegating.
func (e *Env) ACLWrap(c model.Collection, action string, h ghttp.HandlerFunc) ghttp.HandlerFunc {
	if e.ACLDisabled {
		return h
	}
	return func(r *ghttp.Request) {
		var roles []string
		authed := false
		u, err := e.Auth.CurrentUser(r)
		if err == nil {
			authed = true
			roles = e.Auth.RolesOf(u)
			ctx := model.CtxWithUser(r.Context(), u)
			ctx = model.CtxWithRoles(ctx, roles, true)
			r.SetCtx(ctx)
		} else if !errors.Is(err, model.ErrUnauthenticated) {
			writeErr(r, http.StatusInternalServerError, "auth lookup failed", err)
			return
		} else {
			roles = []string{model.RoleAnonymous}
			r.SetCtx(model.CtxWithRoles(r.Context(), roles, false))
		}
		if !model.DecideAccess(c, action, roles, authed) {
			if !authed {
				writeErr(r, http.StatusUnauthorized, "unauthenticated", nil)
			} else {
				writeErr(r, http.StatusForbidden, "forbidden", nil)
			}
			return
		}
		h(r)
	}
}

// RouteACLWrap returns a handler that enforces RouteACL before delegating.
func (e *Env) RouteACLWrap(acl model.RouteACL, h ghttp.HandlerFunc) ghttp.HandlerFunc {
	if e.ACLDisabled {
		return h
	}
	return func(r *ghttp.Request) {
		u, err := e.Auth.CurrentUser(r)
		authed := err == nil
		if authed {
			userRoles := e.Auth.RolesOf(u)
			ctx := model.CtxWithUser(r.Context(), u)
			ctx = model.CtxWithRoles(ctx, userRoles, true)
			r.SetCtx(ctx)
			if len(acl.Roles) > 0 && !anyMatch(acl.Roles, userRoles) {
				writeErr(r, http.StatusForbidden, "forbidden", nil)
				return
			}
			h(r)
			return
		}
		if !errors.Is(err, model.ErrUnauthenticated) {
			writeErr(r, http.StatusInternalServerError, "auth lookup failed", err)
			return
		}
		r.SetCtx(model.CtxWithRoles(r.Context(), []string{model.RoleAnonymous}, false))
		if acl.Anonymous {
			h(r)
			return
		}
		writeErr(r, http.StatusUnauthorized, "unauthenticated", nil)
	}
}

// MountCustomRoute registers a custom route on the given router group.
func MountCustomRoute(group *ghttp.RouterGroup, route model.Route, h ghttp.HandlerFunc) error {
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
