package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/model"
)

// SSOHandler groups SSO-related HTTP handlers and state.
type SSOHandler struct {
	DB       gdb.DB
	Provider model.OAuthProvider
	Config   model.SSOConfig
	preAuth  *preAuthStore
	mu       sync.Mutex
}

func (s *SSOHandler) getPreAuthStore() *preAuthStore {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.preAuth == nil {
		s.preAuth = newPreAuthStore()
	}
	return s.preAuth
}

const ssoPreAuthCookie = "itab_sso_state"

// HandleSSOLogin redirects the user to the provider's authorize page.
func (s *SSOHandler) HandleSSOLogin(r *ghttp.Request) {
	returnURL := r.Get("return", "").String()
	if returnURL != "" && !isValidReturnURL(r, returnURL) {
		returnURL = ""
	}

	store := s.getPreAuthStore()
	id, state := store.Create(returnURL)

	http.SetCookie(r.Response.Writer, &http.Cookie{
		Name:     ssoPreAuthCookie,
		Value:    id,
		Path:     "/",
		Domain:   s.Config.CookieDomain,
		HttpOnly: true,
		Secure:   s.Config.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})

	authURL := s.Provider.AuthorizeURL(s.Config, state)
	r.Response.RedirectTo(authURL, http.StatusFound)
}

// HandleSSOCallback handles the provider's OAuth2 callback.
func (s *SSOHandler) HandleSSOCallback(r *ghttp.Request) {
	q := r.URL.Query()

	if errParam := q.Get("error"); errParam != "" {
		s.ssoRedirectError(r, errParam)
		return
	}

	code := q.Get("code")
	state := q.Get("state")
	if code == "" {
		s.ssoRedirectError(r, "missing_code")
		return
	}

	preAuthID := r.Cookie.Get(ssoPreAuthCookie).String()
	if preAuthID == "" {
		s.ssoRedirectError(r, "invalid_state")
		return
	}
	store := s.getPreAuthStore()
	returnURL, ok := store.Validate(preAuthID, state)
	if !ok {
		s.ssoRedirectError(r, "invalid_state")
		return
	}

	http.SetCookie(r.Response.Writer, &http.Cookie{
		Name: ssoPreAuthCookie, Value: "", Path: "/",
		Domain: s.Config.CookieDomain, HttpOnly: true, MaxAge: -1,
	})

	ctx := r.Context()

	token, err := s.Provider.ExchangeToken(ctx, s.Config, code)
	if err != nil {
		g.Log().Errorf(ctx, "[sso] exchange token failed: %v", err)
		s.ssoRedirectError(r, "token_exchange_failed")
		return
	}

	info, err := s.Provider.FetchUser(ctx, s.Config, token)
	if err != nil {
		g.Log().Errorf(ctx, "[sso] fetch user info failed: %v", err)
		s.ssoRedirectError(r, "userinfo_failed")
		return
	}

	kernelUserID, status, err := s.upsertSSOUser(ctx, info)
	if err != nil {
		g.Log().Errorf(ctx, "[sso] upsert user failed: %v", err)
		s.ssoRedirectError(r, "internal")
		return
	}

	if status == model.UserStatusPending {
		target := returnURL
		if target == "" {
			target = ssoDefaultReturn(r)
		}
		if u, err := url.Parse(target); err == nil {
			q := u.Query()
			q.Set("need_access", "pending")
			u.RawQuery = q.Encode()
			target = u.String()
		}
		r.Response.RedirectTo(target, http.StatusFound)
		return
	}

	if status == model.UserStatusRejected {
		target := returnURL
		if target == "" {
			target = ssoDefaultReturn(r)
		}
		if u, err := url.Parse(target); err == nil {
			q := u.Query()
			q.Set("need_access", "apply")
			u.RawQuery = q.Encode()
			target = u.String()
		}
		r.Response.RedirectTo(target, http.StatusFound)
		return
	}

	if err := r.Session.Set(model.SessionKeyUserID, kernelUserID); err != nil {
		g.Log().Errorf(ctx, "[sso] session set failed: %v", err)
		s.ssoRedirectError(r, "internal")
		return
	}

	target := returnURL
	if target == "" {
		target = ssoDefaultReturn(r)
	}
	g.Log().Infof(ctx, "[sso] login ok user=%s name=%s", info.ExternalID, info.Name)
	r.Response.RedirectTo(target, http.StatusFound)
}

func (s *SSOHandler) upsertSSOUser(ctx context.Context, info model.OAuthUserInfo) (int64, string, error) {
	row, err := s.DB.Model(model.BuiltinUsers).Ctx(ctx).
		Where("external_id", info.ExternalID).One()
	if err != nil {
		return 0, "", err
	}

	now := time.Now()
	profileData := g.Map{
		"display_name":    info.Name,
		"login_name":      info.LoginName,
		"avatar":          info.Avatar,
		"email":           info.Email,
		"phone":           info.Phone,
		"gender":          info.Gender,
		"employee_id":     info.EmployeeID,
		"title":           info.Title,
		"department":      info.Department,
		"department_path": info.DepartmentPath,
		"company_id":      info.CompanyID,
		"last_seen_at":    now,
	}

	if row.IsEmpty() {
		requireApproval := GetSetting(ctx, s.DB, "require_approval") == "true"
		status := model.UserStatusActive
		if requireApproval {
			status = model.UserStatusPending
		}

		insertData := g.Map{
			"external_id":   info.ExternalID,
			"status":        status,
			"disabled":      false,
			"first_seen_at": now,
		}
		for k, v := range profileData {
			insertData[k] = v
		}

		result, err := s.DB.Model(model.BuiltinUsers).Ctx(ctx).Insert(insertData)
		if err != nil {
			return 0, "", err
		}
		id, _ := result.LastInsertId()

		if status == model.UserStatusActive {
			_ = s.bindDefaultRole(ctx, id)
		}
		return id, status, nil
	}

	id := row["id"].Int64()
	status := row["status"].String()

	_, _ = s.DB.Model(model.BuiltinUsers).Ctx(ctx).Where("id", id).Update(profileData)

	return id, status, nil
}

func (s *SSOHandler) bindDefaultRole(ctx context.Context, userID int64) error {
	roleRow, err := s.DB.Model(model.BuiltinRoles).Ctx(ctx).Where("name", "user").One()
	if err != nil || roleRow.IsEmpty() {
		return err
	}
	n, _ := s.DB.Model(model.BuiltinUserRoles).Ctx(ctx).
		Where("user_id", userID).Count()
	if n > 0 {
		return nil
	}
	_, err = s.DB.Model(model.BuiltinUserRoles).Ctx(ctx).Insert(g.Map{
		"user_id": userID,
		"role_id": roleRow["id"].Int64(),
	})
	return err
}

func (s *SSOHandler) ssoRedirectError(r *ghttp.Request, errCode string) {
	target := ssoDefaultReturn(r) + "?error=" + url.QueryEscape(errCode)
	r.Response.RedirectTo(target, http.StatusFound)
}

// GetSetting reads a single value from the system_settings collection.
func GetSetting(ctx context.Context, db gdb.DB, key string) string {
	row, err := db.Model(model.BuiltinSettings).Ctx(ctx).Where("key", key).One()
	if err != nil || row.IsEmpty() {
		return ""
	}
	return row["value"].String()
}

// --- Pre-auth CSRF state ---

type preAuthEntry struct {
	state     string
	returnURL string
	expiresAt time.Time
}

type preAuthStore struct {
	mu      sync.Mutex
	entries map[string]*preAuthEntry
}

func newPreAuthStore() *preAuthStore {
	s := &preAuthStore{entries: make(map[string]*preAuthEntry)}
	go s.gc()
	return s
}

func (s *preAuthStore) Create(returnURL string) (id, state string) {
	id = randomHex(32)
	state = randomHex(16)
	s.mu.Lock()
	s.entries[id] = &preAuthEntry{
		state:     state,
		returnURL: returnURL,
		expiresAt: time.Now().Add(10 * time.Minute),
	}
	s.mu.Unlock()
	return
}

func (s *preAuthStore) Validate(id, state string) (returnURL string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, found := s.entries[id]
	if !found || time.Now().After(e.expiresAt) || e.state != state {
		delete(s.entries, id)
		return "", false
	}
	delete(s.entries, id)
	return e.returnURL, true
}

func (s *preAuthStore) gc() {
	t := time.NewTicker(2 * time.Minute)
	defer t.Stop()
	for range t.C {
		now := time.Now()
		s.mu.Lock()
		for id, e := range s.entries {
			if now.After(e.expiresAt) {
				delete(s.entries, id)
			}
		}
		s.mu.Unlock()
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func isValidReturnURL(r *ghttp.Request, raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	reqHost, _, _ := strings.Cut(r.Host, ":")
	if reqHost == "" {
		reqHost = r.Host
	}
	return u.Hostname() == reqHost
}

func ssoDefaultReturn(r *ghttp.Request) string {
	scheme := "http"
	if r.Request.TLS != nil || strings.EqualFold(r.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + r.Host + "/"
}
