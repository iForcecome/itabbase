package wps365

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gogf/gf/v2/os/glog"

	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/model"
)

// Provider is the built-in OAuthProvider for WPS 365.
var Provider model.OAuthProvider = &wps365Provider{}

const defaultBase = "https://openapi.wps.cn"

var httpClient = &http.Client{Timeout: 15 * time.Second}

type wps365Provider struct{}

func (p *wps365Provider) Name() string { return "wps365" }

func (p *wps365Provider) AuthorizeURL(cfg model.SSOConfig, state string) string {
	base := cfg.BaseURL
	if base == "" {
		base = defaultBase
	}
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"kso.user_base.read"}
	}
	params := url.Values{
		"client_id":     {cfg.AppID},
		"response_type": {"code"},
		"redirect_uri":  {cfg.RedirectURI},
		"scope":         {strings.Join(scopes, ",")},
		"state":         {state},
	}
	return fmt.Sprintf("%s/oauth2/auth?%s", base, params.Encode())
}

func (p *wps365Provider) ExchangeToken(ctx context.Context, cfg model.SSOConfig, code string) (model.OAuthToken, error) {
	base := cfg.BaseURL
	if base == "" {
		base = defaultBase
	}
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {cfg.AppID},
		"client_secret": {cfg.AppSecret},
		"code":          {code},
		"redirect_uri":  {cfg.RedirectURI},
	}
	reqURL := fmt.Sprintf("%s/oauth2/token", base)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return model.OAuthToken{}, fmt.Errorf("wps365: create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return model.OAuthToken{}, fmt.Errorf("wps365: token http: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.OAuthToken{}, fmt.Errorf("wps365: read token response: %w", err)
	}

	var tr struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		Scope        string `json:"scope"`
		Error_       string `json:"error,omitempty"`
		ErrorDesc    string `json:"error_description,omitempty"`
	}
	_ = json.Unmarshal(body, &tr)

	if resp.StatusCode != http.StatusOK || tr.Error_ != "" || tr.AccessToken == "" {
		return model.OAuthToken{}, fmt.Errorf("wps365: token exchange failed: HTTP %d, error=%s, desc=%s",
			resp.StatusCode, tr.Error_, tr.ErrorDesc)
	}

	return model.OAuthToken{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		ExpiresIn:    tr.ExpiresIn,
		Scope:        tr.Scope,
	}, nil
}

func (p *wps365Provider) FetchUser(ctx context.Context, cfg model.SSOConfig, token model.OAuthToken) (model.OAuthUserInfo, error) {
	base := cfg.BaseURL
	if base == "" {
		base = defaultBase
	}
	info, err := p.fetchBasicUser(ctx, cfg, base, token)
	if err != nil {
		return model.OAuthUserInfo{}, err
	}
	p.enrichFromDepts(ctx, cfg, base, token, &info)
	return info, nil
}

func (p *wps365Provider) fetchBasicUser(ctx context.Context, cfg model.SSOConfig, base string, token model.OAuthToken) (model.OAuthUserInfo, error) {
	fullURL := fmt.Sprintf("%s/v7/users/current", base)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return model.OAuthUserInfo{}, fmt.Errorf("wps365: create userinfo request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	if cfg.EnableSign {
		signKSO1(req, cfg.AppID, cfg.AppSecret, nil, req.URL.RequestURI())
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return model.OAuthUserInfo{}, fmt.Errorf("wps365: userinfo http: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.OAuthUserInfo{}, fmt.Errorf("wps365: read userinfo response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return model.OAuthUserInfo{}, fmt.Errorf("wps365: userinfo HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data *struct {
			ID        string `json:"id"`
			UserName  string `json:"user_name"`
			Avatar    string `json:"avatar"`
			CompanyID string `json:"company_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return model.OAuthUserInfo{}, fmt.Errorf("wps365: unmarshal userinfo: %w", err)
	}
	if result.Code != 0 || result.Data == nil {
		return model.OAuthUserInfo{}, fmt.Errorf("wps365: userinfo code=%d msg=%s", result.Code, result.Msg)
	}

	return model.OAuthUserInfo{
		ExternalID: result.Data.ID,
		Name:       result.Data.UserName,
		Avatar:     result.Data.Avatar,
		CompanyID:  result.Data.CompanyID,
	}, nil
}

func (p *wps365Provider) enrichFromDepts(ctx context.Context, cfg model.SSOConfig, base string, token model.OAuthToken, info *model.OAuthUserInfo) {
	if info.ExternalID == "" {
		return
	}

	fullURL := fmt.Sprintf("%s/v7/users/%s/depts", base, url.PathEscape(info.ExternalID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	if cfg.EnableSign {
		signPath := fmt.Sprintf("/v7/users/%s/depts", url.PathEscape(info.ExternalID))
		signKSO1(req, cfg.AppID, cfg.AppSecret, nil, signPath)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		glog.Warningf(ctx, "[wps365] enrich depts: http failed: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode >= 400 {
		glog.Warningf(ctx, "[wps365] enrich depts: HTTP %d", resp.StatusCode)
		return
	}

	var result struct {
		Code int `json:"code"`
		Data *struct {
			Items []struct {
				Name    string `json:"name"`
				AbsPath string `json:"abs_path"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Code != 0 || result.Data == nil {
		return
	}

	if len(result.Data.Items) > 0 {
		dept := result.Data.Items[0]
		info.Department = dept.Name
		info.DepartmentPath = dept.AbsPath
		glog.Infof(ctx, "[wps365] enrich: dept=%s path=%s", dept.Name, dept.AbsPath)
	}
}

func signKSO1(req *http.Request, appID, appSecret string, body []byte, signPath string) {
	contentType := req.Header.Get("Content-Type")
	ksoDate := time.Now().UTC().Format(time.RFC1123)

	bodyHash := ""
	if len(body) > 0 {
		h := sha256.New()
		h.Write(body)
		bodyHash = hex.EncodeToString(h.Sum(nil))
	}

	signString := "KSO-1" + req.Method + signPath + contentType + ksoDate + bodyHash
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write([]byte(signString))
	signature := hex.EncodeToString(mac.Sum(nil))

	req.Header.Set("X-Kso-Date", ksoDate)
	req.Header.Set("X-Kso-Authorization", fmt.Sprintf("KSO-1 %s:%s", appID, signature))
}

// DoJSON is a helper for signed JSON API calls to WPS endpoints.
func DoJSON(ctx context.Context, cfg model.SSOConfig, method, path, accessToken string, reqBody, out any) error {
	base := cfg.BaseURL
	if base == "" {
		base = defaultBase
	}

	var bodyBytes []byte
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("wps365: marshal: %w", err)
		}
		bodyBytes = b
	}

	fullURL := fmt.Sprintf("%s%s", base, path)
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("wps365: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.EnableSign {
		signKSO1(req, cfg.AppID, cfg.AppSecret, bodyBytes, req.URL.RequestURI())
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("wps365: http: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("wps365: read: %w", err)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("wps365: HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	if out != nil {
		return json.Unmarshal(respBody, out)
	}
	return nil
}
