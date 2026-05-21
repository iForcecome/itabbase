package auth

import (
	"net/http"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/model"
)

const defaultSessionTTL = 24 * time.Hour
const timeLayout = "2006-01-02 15:04:05"

func normalizeSessionConfig(cfg model.SSOConfig) model.SSOConfig {
	if cfg.CookieName == "" {
		cfg.CookieName = model.DefaultSessionCookieName
	}
	if cfg.CookiePath == "" {
		cfg.CookiePath = "/"
	}
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = defaultSessionTTL
	}
	return cfg
}

func SSOConfigDefaults(cfg model.SSOConfig) model.SSOConfig {
	return normalizeSessionConfig(cfg)
}

func createSession(r *ghttp.Request, db gdb.DB, cfg model.SSOConfig, userID int64) (string, error) {
	cfg = normalizeSessionConfig(cfg)
	now := time.Now()
	nowStr := now.Format(timeLayout)
	expiresStr := now.Add(cfg.SessionTTL).Format(timeLayout)
	sid := randomHex(32)
	_, err := db.Model(model.BuiltinSessions).Ctx(r.Context()).Insert(g.Map{
		"sid":        sid,
		"user_id":    userID,
		"expires_at": expiresStr,
		"created_at": nowStr,
		"updated_at": nowStr,
	})
	if err != nil {
		return "", err
	}
	setSessionCookie(r, cfg, sid)
	return sid, nil
}

func currentSessionUserID(r *ghttp.Request, db gdb.DB, cfg model.SSOConfig) (int64, error) {
	cfg = normalizeSessionConfig(cfg)
	sid := r.Cookie.Get(cfg.CookieName).String()
	if sid == "" {
		return 0, model.ErrUnauthenticated
	}
	row, err := db.Model(model.BuiltinSessions).Ctx(r.Context()).
		Where("sid", sid).
		Where("expires_at > ?", time.Now().Format(timeLayout)).
		One()
	if err != nil || row.IsEmpty() {
		clearSessionCookie(r, cfg)
		return 0, model.ErrUnauthenticated
	}
	_, _ = db.Model(model.BuiltinSessions).Ctx(r.Context()).
		Where("sid", sid).
		Update(g.Map{"updated_at": time.Now().Format(timeLayout)})
	return row["user_id"].Int64(), nil
}

func deleteCurrentSession(r *ghttp.Request, db gdb.DB, cfg model.SSOConfig) {
	cfg = normalizeSessionConfig(cfg)
	sid := r.Cookie.Get(cfg.CookieName).String()
	if sid != "" {
		_, _ = db.Model(model.BuiltinSessions).Ctx(r.Context()).Where("sid", sid).Delete()
	}
	clearSessionCookie(r, cfg)
}

func setSessionCookie(r *ghttp.Request, cfg model.SSOConfig, sid string) {
	cfg = normalizeSessionConfig(cfg)
	http.SetCookie(r.Response.Writer, &http.Cookie{
		Name:     cfg.CookieName,
		Value:    sid,
		Path:     cfg.CookiePath,
		Domain:   cfg.CookieDomain,
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(cfg.SessionTTL.Seconds()),
	})
}

func clearSessionCookie(r *ghttp.Request, cfg model.SSOConfig) {
	cfg = normalizeSessionConfig(cfg)
	http.SetCookie(r.Response.Writer, &http.Cookie{
		Name:     cfg.CookieName,
		Value:    "",
		Path:     cfg.CookiePath,
		Domain:   cfg.CookieDomain,
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
