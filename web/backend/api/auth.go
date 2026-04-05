package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/sipeed/picoclaw/web/backend/middleware"
)

// PasswordStore is the interface for bcrypt-backed dashboard password persistence.
// Implemented by dashboardauth.Store; a nil value falls back to the legacy
// static-token comparison.
type PasswordStore interface {
	IsInitialized(ctx context.Context) (bool, error)
	SetPassword(ctx context.Context, plain string) error
	VerifyPassword(ctx context.Context, plain string) (bool, error)
}

// LauncherAuthRouteOpts configures dashboard auth handlers.
type LauncherAuthRouteOpts struct {
	// DashboardToken is the fallback plaintext token used when PasswordStore is
	// nil or not yet initialized (env-var / config-file source, and ?token= auto-login).
	DashboardToken string
	SessionCookie  string
	SecureCookie   func(*http.Request) bool
	// PasswordStore enables bcrypt-backed password persistence. When non-nil and
	// initialized, web-form login verifies against the stored hash instead of
	// the plaintext DashboardToken.
	PasswordStore PasswordStore
}

type launcherAuthLoginBody struct {
	Password string `json:"password"`
}

type launcherAuthSetupBody struct {
	Password string `json:"password"`
	Confirm  string `json:"confirm"`
}

type launcherAuthStatusResponse struct {
	Authenticated bool `json:"authenticated"`
	Initialized   bool `json:"initialized"`
}

// RegisterLauncherAuthRoutes registers /api/auth/login|logout|status|setup.
func RegisterLauncherAuthRoutes(mux *http.ServeMux, opts LauncherAuthRouteOpts) {
	secure := opts.SecureCookie
	if secure == nil {
		secure = middleware.DefaultLauncherDashboardSecureCookie
	}
	h := &launcherAuthHandlers{
		token:         opts.DashboardToken,
		sessionCookie: opts.SessionCookie,
		secureCookie:  secure,
		store:         opts.PasswordStore,
		loginLimit:    newLoginRateLimiter(),
	}
	mux.HandleFunc("POST /api/auth/login", h.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", h.handleLogout)
	mux.HandleFunc("GET /api/auth/status", h.handleStatus)
	mux.HandleFunc("POST /api/auth/setup", h.handleSetup)
}

type launcherAuthHandlers struct {
	token         string
	sessionCookie string
	secureCookie  func(*http.Request) bool
	store         PasswordStore
	loginLimit    *loginRateLimiter
}

// isStoreInitialized safely queries the store.
func (h *launcherAuthHandlers) isStoreInitialized(ctx context.Context) bool {
	if h.store == nil {
		return false
	}
	ok, err := h.store.IsInitialized(ctx)
	return err == nil && ok
}

func (h *launcherAuthHandlers) handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var body launcherAuthLoginBody
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid JSON"}`))
		return
	}
	ip := clientIPForLimiter(r)
	if !h.loginLimit.allow(ip) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"too many login attempts"}`))
		return
	}
	in := strings.TrimSpace(body.Password)
	var ok bool

	if h.isStoreInitialized(r.Context()) {
		// Bcrypt path: verify against the stored hash.
		var err error
		ok, err = h.store.VerifyPassword(r.Context(), in)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"internal error"}`))
			return
		}
	} else {
		// Fallback: constant-time compare against the plaintext token.
		ok = len(in) == len(h.token) &&
			subtle.ConstantTimeCompare([]byte(in), []byte(h.token)) == 1
	}

	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid password"}`))
		return
	}

	middleware.SetLauncherDashboardSessionCookie(w, r, h.sessionCookie, h.secureCookie)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (h *launcherAuthHandlers) handleLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte(`{"error":"method not allowed"}`))
		return
	}
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if !strings.HasPrefix(ct, "application/json") {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		_, _ = w.Write([]byte(`{"error":"Content-Type must be application/json"}`))
		return
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, logoutBodyMaxBytes))
	if err := dec.Decode(&struct{}{}); err != nil && err != io.EOF {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid JSON body"}`))
		return
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid JSON body"}`))
		return
	}

	middleware.ClearLauncherDashboardSessionCookie(w, r, h.secureCookie)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (h *launcherAuthHandlers) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	authed := false
	if c, err := r.Cookie(middleware.LauncherDashboardCookieName); err == nil {
		authed = subtle.ConstantTimeCompare([]byte(c.Value), []byte(h.sessionCookie)) == 1
	}
	resp := launcherAuthStatusResponse{
		Authenticated: authed,
		Initialized:   h.isStoreInitialized(r.Context()),
	}
	enc, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal error"}`))
		return
	}
	_, _ = w.Write(enc)
}

// handleSetup sets or changes the dashboard password.
//
// Rules:
//   - If the store has no password yet, the endpoint is open (no session required).
//   - If a password is already set, the caller must hold a valid session cookie.
func (h *launcherAuthHandlers) handleSetup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.store == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_, _ = w.Write([]byte(`{"error":"password store not configured"}`))
		return
	}

	initialized := h.isStoreInitialized(r.Context())

	// If already initialized, require an active session (change-password flow).
	if initialized {
		authed := false
		if c, err := r.Cookie(middleware.LauncherDashboardCookieName); err == nil {
			authed = subtle.ConstantTimeCompare([]byte(c.Value), []byte(h.sessionCookie)) == 1
		}
		if !authed {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"must be authenticated to change password"}`))
			return
		}
	}

	var body launcherAuthSetupBody
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid JSON"}`))
		return
	}

	pw := strings.TrimSpace(body.Password)
	if pw == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"password must not be empty"}`))
		return
	}
	if pw != strings.TrimSpace(body.Confirm) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"passwords do not match"}`))
		return
	}
	if len([]rune(pw)) < 8 {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"password must be at least 8 characters"}`))
		return
	}

	if err := h.store.SetPassword(r.Context(), pw); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"failed to save password"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
