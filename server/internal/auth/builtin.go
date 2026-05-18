package auth

import (
	"net/http"
	"strconv"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/model"
)

// BuiltinAdapter is a session-based auth adapter using GoFrame sessions
// and the kernel's own users/roles tables.
type BuiltinAdapter struct {
	DB gdb.DB
}

func (a *BuiltinAdapter) CurrentUser(r *ghttp.Request) (model.User, error) {
	uid := r.Session.MustGet(model.SessionKeyUserID).Int64()
	if uid == 0 {
		return model.User{}, model.ErrUnauthenticated
	}
	row, err := a.DB.Model(model.BuiltinUsers).Ctx(r.Context()).
		Where("id", uid).
		Where("disabled", false).
		Where("status", model.UserStatusActive).
		One()
	if err != nil || row.IsEmpty() {
		_ = r.Session.Remove(model.SessionKeyUserID)
		return model.User{}, model.ErrUnauthenticated
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
func HandleLocalLogin(db gdb.DB) ghttp.HandlerFunc {
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
		if err := r.Session.Set(model.SessionKeyUserID, u.LocalID); err != nil {
			writeErr(r, http.StatusInternalServerError, "session error", err)
			return
		}
		r.Response.WriteJsonExit(g.Map{
			"data": g.Map{"id": u.ID, "display_name": u.Name},
		})
	}
}

// HandleLogout handles POST /auth/logout.
func HandleLogout() ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		_ = r.Session.RemoveAll()
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
