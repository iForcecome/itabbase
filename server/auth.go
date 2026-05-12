package itab

import (
	"context"
	"errors"

	"github.com/gogf/gf/v2/net/ghttp"
)

// ErrUnauthenticated should be returned by AuthAdapter.CurrentUser when the
// request carries no valid session. The kernel maps it to HTTP 401.
var ErrUnauthenticated = errors.New("itab: unauthenticated")

// RoleAnonymous is the role assigned to requests without a valid session.
// Collections that want public access must list this role in their ACL.
const RoleAnonymous = "anonymous"

type User struct {
	ID      string
	LocalID int64
	Name    string
}

// AuthAdapter is the narrow contract between kernel and the host application's
// authentication system. The kernel never reads cookies or session stores
// directly; everything goes through this interface.
type AuthAdapter interface {
	CurrentUser(r *ghttp.Request) (User, error)
	RolesOf(u User) []string
}

type ctxKey int

const userCtxKey ctxKey = 1

func ctxWithUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, userCtxKey, u)
}

// UserFromCtx returns the authenticated user attached by the kernel's
// auth middleware. ok is false for anonymous requests.
func UserFromCtx(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userCtxKey).(User)
	return u, ok
}
