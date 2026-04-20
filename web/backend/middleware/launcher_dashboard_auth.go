package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"path"
	"strings"
	"time"
)

// LauncherDashboardCookieName is the HttpOnly cookie set after a successful token login.
const LauncherDashboardCookieName = "picoclaw_launcher_auth"

// launcherDashboardSessionMaxAgeSec is the session cookie lifetime (7 days).
const launcherDashboardSessionMaxAgeSec = 7 * 24 * 3600

const launcherSessionMACLabel = "picoclaw-launcher-v1"

// SessionCookieValue is the expected cookie value for the given signing key and dashboard token.
func SessionCookieValue(signingKey []byte, dashboardToken string) string {
	mac := hmac.New(sha256.New, signingKey)
	_, _ = mac.Write([]byte(launcherSessionMACLabel))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(dashboardToken))
	return hex.EncodeToString(mac.Sum(nil))
}

// LauncherDashboardAuthConfig holds runtime material for dashboard access checks.
type LauncherDashboardAuthConfig struct {
	ExpectedCookie string
	Token          string
	// SecureCookie sets the session cookie's Secure flag. If nil, DefaultLauncherDashboardSecureCookie is used.
	SecureCookie func(*http.Request) bool
}

// DefaultLauncherDashboardSecureCookie mirrors typical production HTTPS detection (TLS or X-Forwarded-Proto).
func DefaultLauncherDashboardSecureCookie(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// SetLauncherDashboardSessionCookie writes the HttpOnly session cookie after successful dashboard token login.
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

// LauncherDashboardAuth requires a valid session cookie or Authorization: Bearer <token>
// before calling next. Public paths are login page and /api/auth/* handlers.
func LauncherDashboardAuth(cfg LauncherDashboardAuthConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := canonicalAuthPath(r.URL.Path)
		if handled := tryLauncherQueryTokenLogin(w, r, p, cfg); handled {
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

// tryLauncherQueryTokenLogin validates ?token= on GET only (non-/api), sets the session
// cookie when correct, and redirects with 303 so the follow-up is a plain GET without side effects.
// Invalid token is rejected like any other unauthenticated browser request.
func tryLauncherQueryTokenLogin(
	w http.ResponseWriter,
	r *http.Request,
	canonicalPath string,
	cfg LauncherDashboardAuthConfig,
) bool {
	if r.Method != http.MethodGet {
		return false
	}
	if canonicalPath == "/api" || strings.HasPrefix(canonicalPath, "/api/") {
		return false
	}
	qToken := strings.TrimSpace(r.URL.Query().Get("token"))
	if qToken == "" {
		return false
	}
	if len(qToken) != len(cfg.Token) || subtle.ConstantTimeCompare([]byte(qToken), []byte(cfg.Token)) != 1 {
		rejectLauncherDashboardAuth(w, r, canonicalPath)
		return true
	}
	SetLauncherDashboardSessionCookie(w, r, cfg.ExpectedCookie, cfg.SecureCookie)
	http.Redirect(w, r, redirectAfterQueryTokenLogin(r, canonicalPath), http.StatusSeeOther)
	return true
}

func redirectAfterQueryTokenLogin(r *http.Request, canonicalPath string) string {
	if canonicalPath == "/launcher-login" {
		return "/"
	}
	q := r.URL.Query()
	q.Del("token")
	enc := q.Encode()
	if enc != "" {
		return canonicalPath + "?" + enc
	}
	return canonicalPath
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
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if strings.HasPrefix(auth, prefix) {
		token := strings.TrimSpace(auth[len(prefix):])
		if len(token) == len(cfg.Token) && subtle.ConstantTimeCompare([]byte(token), []byte(cfg.Token)) == 1 {
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
