package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"
)

// LauncherDashboardCookieName is the HttpOnly cookie set after a successful password login.
const LauncherDashboardCookieName = "picoclaw_launcher_auth"

// launcherDashboardSessionMaxAgeSec is the dashboard session cookie lifetime (31 days).
const launcherDashboardSessionMaxAgeSec = 31 * 24 * 3600

const (
	launcherSessionCookieBytes = 32
	launcherGrantNonceBytes    = 32
	// LauncherDashboardLocalAutoLoginPath is the one-shot local browser
	// bootstrap endpoint used by the launcher-managed auto-open flow.
	LauncherDashboardLocalAutoLoginPath = "/launcher-auto-login"
	// LauncherDashboardSetupPath is the setup page used before the dashboard
	// password is initialized.
	LauncherDashboardSetupPath = "/launcher-setup"
)

// NewLauncherDashboardSessionCookie creates the per-process session cookie value.
func NewLauncherDashboardSessionCookie() (string, error) {
	return randomURLToken(launcherSessionCookieBytes)
}

func randomURLToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// LauncherDashboardAuthConfig holds runtime material for dashboard access checks.
type LauncherDashboardAuthConfig struct {
	ExpectedCookie string
	// LocalAutoLogin enables one-shot startup auto-login.
	LocalAutoLogin *LauncherDashboardLocalAutoLogin
	// SecureCookie sets the session cookie's Secure flag. If nil, DefaultLauncherDashboardSecureCookie is used.
	SecureCookie func(*http.Request) bool
}

// LauncherDashboardLocalAutoLogin is an in-memory, one-shot startup grant.
// It is not a reusable credential; it only lets the launcher-opened browser
// receive the current process session cookie.
type LauncherDashboardLocalAutoLogin struct {
	grant *launcherDashboardOneTimeGrant
}

type launcherDashboardOneTimeGrant struct {
	mu       sync.Mutex
	expires  time.Time
	consumed bool
	nonce    string
	now      func() time.Time
}

// NewLauncherDashboardLocalAutoLogin creates a one-shot local auto-login grant.
func NewLauncherDashboardLocalAutoLogin(ttl time.Duration) (*LauncherDashboardLocalAutoLogin, error) {
	grant, err := newLauncherDashboardOneTimeGrant(ttl)
	if err != nil {
		return nil, err
	}
	return &LauncherDashboardLocalAutoLogin{
		grant: grant,
	}, nil
}

// URLPath returns the one-shot local auto-login URL path including its nonce.
func (a *LauncherDashboardLocalAutoLogin) URLPath() string {
	return launcherGrantQueryPath(LauncherDashboardLocalAutoLoginPath, a.grant)
}

// DefaultLauncherDashboardSecureCookie mirrors typical production HTTPS detection (TLS or X-Forwarded-Proto).
func DefaultLauncherDashboardSecureCookie(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// SetLauncherDashboardSessionCookie writes the HttpOnly session cookie after successful dashboard password login.
func SetLauncherDashboardSessionCookie(
	w http.ResponseWriter,
	r *http.Request,
	sessionValue string,
	secure func(*http.Request) bool,
) {
	if secure == nil {
		secure = DefaultLauncherDashboardSecureCookie
	}
	http.SetCookie(w, &http.Cookie{
		Name:     LauncherDashboardCookieName,
		Value:    sessionValue,
		Path:     "/",
		MaxAge:   launcherDashboardSessionMaxAgeSec,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure(r),
	})
}

// ClearLauncherDashboardSessionCookie clears the dashboard session (e.g. logout).
func ClearLauncherDashboardSessionCookie(w http.ResponseWriter, r *http.Request, secure func(*http.Request) bool) {
	if secure == nil {
		secure = DefaultLauncherDashboardSecureCookie
	}
	http.SetCookie(w, &http.Cookie{
		Name:     LauncherDashboardCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure(r),
		Expires:  time.Unix(0, 0),
	})
}

// LauncherDashboardAuth requires a valid session cookie before calling next.
// Public paths are login/setup pages and /api/auth/* handlers.
func LauncherDashboardAuth(cfg LauncherDashboardAuthConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := canonicalAuthPath(r.URL.Path)
		if p == LauncherDashboardLocalAutoLoginPath {
			handleLauncherLocalAutoLogin(w, r, cfg)
			return
		}
		if isPublicLauncherDashboardPath(r.Method, p) {
			next.ServeHTTP(w, r)
			return
		}
		if validLauncherDashboardAuth(r, cfg) {
			next.ServeHTTP(w, r)
			return
		}
		rejectLauncherDashboardAuth(w, r, p)
	})
}

// canonicalAuthPath matches path cleaning used for routing decisions so
// prefixes like /assets/../ cannot bypass auth (CVE-class traversal).

func handleLauncherLocalAutoLogin(w http.ResponseWriter, r *http.Request, cfg LauncherDashboardAuthConfig) {
	if validLauncherDashboardAuth(r, cfg) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte("method not allowed"))
		return
	}
	if r.Method == http.MethodHead {
		rejectLauncherDashboardAuth(w, r, LauncherDashboardLocalAutoLoginPath)
		return
	}
	if cfg.LocalAutoLogin != nil && cfg.LocalAutoLogin.consume(r.URL.Query().Get("nonce")) {
		SetLauncherDashboardSessionCookie(w, r, cfg.ExpectedCookie, cfg.SecureCookie)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	rejectLauncherDashboardAuth(w, r, LauncherDashboardLocalAutoLoginPath)
}

func (a *LauncherDashboardLocalAutoLogin) consume(nonce string) bool {
	if a == nil || a.grant == nil {
		return false
	}
	return a.grant.use(nonce, nil) == nil
}

func newLauncherDashboardOneTimeGrant(ttl time.Duration) (*launcherDashboardOneTimeGrant, error) {
	nonce, err := randomURLToken(launcherGrantNonceBytes)
	if err != nil {
		return nil, err
	}
	return &launcherDashboardOneTimeGrant{
		expires: time.Now().Add(ttl),
		nonce:   nonce,
		now:     time.Now,
	}, nil
}

func launcherGrantQueryPath(basePath string, grant *launcherDashboardOneTimeGrant) string {
	if grant == nil {
		return basePath
	}
	return basePath + "?nonce=" + url.QueryEscape(grant.nonce)
}

// ErrInvalidLauncherDashboardGrant reports that an auto-login grant is missing,
// expired, already consumed, or otherwise invalid.
var ErrInvalidLauncherDashboardGrant = errors.New("invalid launcher dashboard grant")

func (g *launcherDashboardOneTimeGrant) use(nonce string, fn func() error) error {
	if g == nil {
		return ErrInvalidLauncherDashboardGrant
	}
	if len(nonce) != len(g.nonce) ||
		subtle.ConstantTimeCompare([]byte(nonce), []byte(g.nonce)) != 1 {
		return ErrInvalidLauncherDashboardGrant
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now
	if g.now != nil {
		now = g.now
	}
	if g.consumed || !now().Before(g.expires) {
		return ErrInvalidLauncherDashboardGrant
	}
	if fn != nil {
		if err := fn(); err != nil {
			return err
		}
	}
	g.consumed = true
	return nil
}

func canonicalAuthPath(raw string) string {
	if raw == "" {
		return "/"
	}
	c := path.Clean(raw)
	switch c {
	case ".", "":
		return "/"
	default:
		if c[0] != '/' {
			return "/" + c
		}
		return c
	}
}

func isPublicLauncherDashboardPath(method, p string) bool {
	if isPublicLauncherDashboardStatic(method, p) {
		return true
	}
	switch p {
	case "/api/auth/login":
		return method == http.MethodPost
	case "/api/auth/logout":
		return method == http.MethodPost
	case "/api/auth/status":
		return method == http.MethodGet
	case "/api/auth/setup":
		return method == http.MethodPost
	}
	return false
}

// isPublicLauncherDashboardStatic allows the SPA login route and embedded
// frontend assets without a session (GET/HEAD only).
func isPublicLauncherDashboardStatic(method, p string) bool {
	if method != http.MethodGet && method != http.MethodHead {
		return false
	}
	if p == "/launcher-login" || p == "/launcher-setup" {
		return true
	}
	if strings.HasPrefix(p, "/assets/") {
		return true
	}
	switch p {
	case "/favicon.ico", "/favicon.svg", "/favicon-96x96.png",
		"/apple-touch-icon.png", "/site.webmanifest", "/robots.txt":
		return true
	default:
		return false
	}
}

func validLauncherDashboardAuth(r *http.Request, cfg LauncherDashboardAuthConfig) bool {
	if c, err := r.Cookie(LauncherDashboardCookieName); err == nil {
		if subtle.ConstantTimeCompare([]byte(c.Value), []byte(cfg.ExpectedCookie)) == 1 {
			return true
		}
	}
	return false
}

func rejectLauncherDashboardAuth(w http.ResponseWriter, r *http.Request, canonicalPath string) {
	if canonicalPath == "/pico/ws" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if strings.HasPrefix(canonicalPath, "/api/") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
		return
	}
	http.Redirect(w, r, "/launcher-login", http.StatusFound)
}
