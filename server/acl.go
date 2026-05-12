package itab

import (
	"errors"
	"net/http"

	"github.com/gogf/gf/v2/net/ghttp"
)

const (
	ActionList   = "list"
	ActionGet    = "get"
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionAll    = "*"
)

// ACL maps a role name to the list of actions it is permitted to perform on a collection.
// Action "*" grants every action.
type ACL map[string][]string

func (a ACL) allows(roles []string, action string) bool {
	for _, role := range roles {
		actions, ok := a[role]
		if !ok {
			continue
		}
		for _, ac := range actions {
			if ac == ActionAll || ac == action {
				return true
			}
		}
	}
	return false
}

// decideAccess applies the kernel's ACL policy.
// authed indicates whether the request carries a valid session.
//
// Default policy when collection.ACL is nil:
//   - authed → allow
//   - anonymous → deny
//
// Explicit ACL overrides the default; only roles listed there get access.
func decideAccess(c Collection, action string, roles []string, authed bool) bool {
	if c.ACL == nil {
		return authed
	}
	return c.ACL.allows(roles, action)
}

// aclWrap returns a handler that enforces ACL before delegating to the inner one.
// When ACL is disabled (WithoutAuth), the inner handler runs unconditionally.
func (k *Kernel) aclWrap(c Collection, action string, h ghttp.HandlerFunc) ghttp.HandlerFunc {
	if k.aclDisabled {
		return h
	}
	return func(r *ghttp.Request) {
		var roles []string
		authed := false
		u, err := k.auth.CurrentUser(r)
		if err == nil {
			authed = true
			roles = k.auth.RolesOf(u)
			ctx := ctxWithUser(r.Context(), u)
			ctx = ctxWithRoles(ctx, roles, true)
			r.SetCtx(ctx)
		} else if !errors.Is(err, ErrUnauthenticated) {
			writeErr(r, http.StatusInternalServerError, "auth lookup failed", err)
			return
		} else {
			roles = []string{RoleAnonymous}
			r.SetCtx(ctxWithRoles(r.Context(), roles, false))
		}
		if !decideAccess(c, action, roles, authed) {
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
