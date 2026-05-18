package itab

import (
	"net/http"
	"strconv"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

const sessionKeyUserID = "itab_uid"

// builtinAuthAdapter is a session-based auth adapter using GoFrame sessions
// and the kernel's own users/roles tables. It is installed by WithBuiltinAuth().
type builtinAuthAdapter struct {
	k *Kernel
}

func (a *builtinAuthAdapter) CurrentUser(r *ghttp.Request) (User, error) {
	uid := r.Session.MustGet(sessionKeyUserID).Int64()
	if uid == 0 {
		return User{}, ErrUnauthenticated
	}
	row, err := a.k.db.Model(BuiltinUsers).Ctx(r.Context()).
		Where("id", uid).
		Where("disabled", false).
		Where("status", UserStatusActive).
		One()
	if err != nil || row.IsEmpty() {
		_ = r.Session.Remove(sessionKeyUserID)
		return User{}, ErrUnauthenticated
	}
	return User{
		ID:      strconv.FormatInt(uid, 10),
		LocalID: uid,
		Name:    row["display_name"].String(),
	}, nil
}

func (a *builtinAuthAdapter) RolesOf(u User) []string {
	rows, err := a.k.db.Model(BuiltinUserRoles+" ur").
		LeftJoin(BuiltinRoles+" r", "r.id = ur.role_id").
		Where("ur.user_id", u.LocalID).
		Fields("r.name").
		All()
	if err != nil || rows.IsEmpty() {
		return nil
	}
	roles := make([]string, 0, len(rows))
	for _, row := range rows {
		if name := row["name"].String(); name != "" {
			roles = append(roles, name)
		}
	}
	return roles
}

// WithBuiltinAuth installs a session-based local password auth adapter.
// This registers /auth/local/login and /auth/logout routes in Mount.
func WithBuiltinAuth() Option {
	return func(k *Kernel) {
		adapter := &builtinAuthAdapter{k: k}
		k.auth = adapter
		k.aclDisabled = false
		k.builtinAuth = true
	}
}

// handleLocalLogin handles POST /auth/local/login
func (k *Kernel) handleLocalLogin(r *ghttp.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := r.Parse(&body); err != nil {
		writeErr(r, http.StatusBadRequest, "invalid request body", err)
		return
	}
	u, err := VerifyPassword(r.Context(), k.db, body.Username, body.Password)
	if err != nil {
		writeErr(r, http.StatusUnauthorized, "用户名或密码错误", nil)
		return
	}
	if err := r.Session.Set(sessionKeyUserID, u.LocalID); err != nil {
		writeErr(r, http.StatusInternalServerError, "session error", err)
		return
	}
	r.Response.WriteJsonExit(g.Map{
		"data": g.Map{"id": u.ID, "name": u.Name},
	})
}

// handleLogout handles POST /auth/logout
func (k *Kernel) handleLogout(r *ghttp.Request) {
	_ = r.Session.RemoveAll()
	r.Response.WriteJsonExit(g.Map{"data": "ok"})
}
