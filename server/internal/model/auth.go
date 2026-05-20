package model

import (
	"context"
	"errors"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"
)

// ErrUnauthenticated should be returned by AuthAdapter.CurrentUser when the
// request carries no valid session. The kernel maps it to HTTP 401.
var ErrUnauthenticated = errors.New("itab: unauthenticated")

// ErrBadCredentials is returned by VerifyPassword for any auth failure.
// Callers must not expose which (avoid user enumeration).
var ErrBadCredentials = errors.New("itab: bad credentials")

// RoleAnonymous is the role assigned to requests without a valid session.
const RoleAnonymous = "anonymous"

type User struct {
	ID      string
	LocalID int64
	Name    string
}

// SSOConfig holds provider-specific configuration.
type SSOConfig struct {
	AppID        string
	AppSecret    string
	RedirectURI  string
	BaseURL      string
	Scopes       []string
	EnableSign   bool
	CookieName   string
	CookieSecure bool
	CookieDomain string
	SessionTTL   time.Duration
}

// OAuthProvider abstracts an SSO identity provider.
type OAuthProvider interface {
	Name() string
	AuthorizeURL(cfg SSOConfig, state string) string
	ExchangeToken(ctx context.Context, cfg SSOConfig, code string) (OAuthToken, error)
	FetchUser(ctx context.Context, cfg SSOConfig, token OAuthToken) (OAuthUserInfo, error)
}

// OAuthToken is the token set returned by the provider's token endpoint.
type OAuthToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	Scope        string
}

// OAuthUserInfo is the normalised user profile returned by an OAuthProvider.
type OAuthUserInfo struct {
	ExternalID     string
	Name           string
	Avatar         string
	Department     string
	DepartmentPath string
	CompanyID      string
}

// AuthAdapter is the narrow contract between kernel and the host application's
// authentication system.
type AuthAdapter interface {
	CurrentUser(r *ghttp.Request) (User, error)
	RolesOf(u User) []string
}
