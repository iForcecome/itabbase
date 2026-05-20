package auth

import (
	"net/http"
	"strconv"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/model"
)

// BuiltinAdapter is a session-based auth adapter using the kernel's own
// sessions/users/roles tables.
type BuiltinAdapter struct {
	DB     gdb.DB
	Config model.SSOConfig
}

func (a *BuiltinAdapter) CurrentUser(r *ghttp.Request) (model.User, error) {
	uid, err := currentSessionUserID(r, a.DB, a.Config)
	if err != nil {
		uid = r.Session.MustGet("itab_uid").Int64()
		if uid == 0 {
			return model.User{}, err
		}
	}
	row, err := a.DB.Model(model.BuiltinUsers).Ctx(r.Context()).
		Where("id", uid).
		Where("disabled", false).
		Where("status", model.UserStatusActive).
		One()
	if err != nil || row.IsEmpty() {
		deleteCurrentSession(r, a.DB, a.Config)
		_ = r.Session.Remove("itab_uid")
		return model.User{}, model.ErrUnauthenticated
	}
	if r.Cookie.Get(normalizeSessionConfig(a.Config).CookieName).String() == "" {
		_, _ = createSession(r, a.DB, a.Config, uid)
	}
	return model.User{
		ID:      strconv.FormatInt(uid, 10),
		LocalID: uid,
		Name:    row["display_name"].String(),
	}, nil
}

func (a *BuiltinAdapter) RolesOf(u model.User) []string {
	rows, err := a.DB.Model(model.BuiltinUserRoles+" ur").
		LeftJoin(model.BuiltinRoles+" r", "r.id = ur.role_id").
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

// HandleLocalLogin handles POST /auth/local/login.
func HandleLocalLogin(db gdb.DB, cfg model.SSOConfig) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := r.Parse(&body); err != nil {
			writeErr(r, http.StatusBadRequest, "invalid request body", err)
			return
		}
		u, err := VerifyPassword(r.Context(), db, body.Username, body.Password)
		if err != nil {
			writeErr(r, http.StatusUnauthorized, "用户名或密码错误", nil)
			return
		}
		if _, err := createSession(r, db, cfg, u.LocalID); err != nil {
			writeErr(r, http.StatusInternalServerError, "session error", err)
			return
		}
		_ = r.Session.Set("itab_uid", u.LocalID)
		r.Response.WriteJsonExit(g.Map{
			"data": g.Map{"id": u.ID, "display_name": u.Name},
		})
	}
}

// HandleLogout handles POST /auth/logout.
func HandleLogout(db gdb.DB, cfg model.SSOConfig) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		deleteCurrentSession(r, db, cfg)
		_ = r.Session.Remove("itab_uid")
		r.Response.WriteJsonExit(g.Map{"data": "ok"})
	}
}

func writeErr(r *ghttp.Request, status int, msg string, cause error) {
	r.Response.Status = status
	payload := g.Map{"error": msg}
	if cause != nil {
		payload["detail"] = cause.Error()
	}
	r.Response.WriteJsonExit(payload)
}
