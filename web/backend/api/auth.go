package api

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/sipeed/picoclaw/web/backend/middleware"
)

// LauncherAuthRouteOpts configures dashboard token login handlers.
type LauncherAuthRouteOpts struct {
	DashboardToken string
	SessionCookie  string
	SecureCookie   func(*http.Request) bool
	// TokenHelp is returned on unauthenticated /api/auth/status responses (no secrets).
	TokenHelp LauncherAuthTokenHelp
}

// LauncherAuthTokenHelp tells the login UI where users can find the dashboard token.
type LauncherAuthTokenHelp struct {
	EnvVarName    string `json:"env_var_name"`
	LogFileAbs    string `json:"log_file,omitempty"`
	TrayCopyMenu  bool   `json:"tray_copy_menu"`
	ConsoleStdout bool   `json:"console_stdout"`
}

type launcherAuthLoginBody struct {
	Token string `json:"token"`
}

type launcherAuthStatusResponse struct {
	Authenticated bool                   `json:"authenticated"`
	TokenHelp     *LauncherAuthTokenHelp `json:"token_help,omitempty"`
}

// RegisterLauncherAuthRoutes registers /api/auth/login|logout|status.
func RegisterLauncherAuthRoutes(mux *http.ServeMux, opts LauncherAuthRouteOpts) {
	secure := opts.SecureCookie
	if secure == nil {
		secure = middleware.DefaultLauncherDashboardSecureCookie
	}
	h := &launcherAuthHandlers{
		token:         opts.DashboardToken,
		sessionCookie: opts.SessionCookie,
		secureCookie:  secure,
		tokenHelp:     opts.TokenHelp,
		loginLimit:    newLoginRateLimiter(),
	}
	mux.HandleFunc("POST /api/auth/login", h.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", h.handleLogout)
	mux.HandleFunc("GET /api/auth/status", h.handleStatus)
}

type launcherAuthHandlers struct {
	token         string
	sessionCookie string
	secureCookie  func(*http.Request) bool
	tokenHelp     LauncherAuthTokenHelp
	loginLimit    *loginRateLimiter
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
	in := strings.TrimSpace(body.Token)
	if len(in) != len(h.token) || subtle.ConstantTimeCompare([]byte(in), []byte(h.token)) != 1 {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid token"}`))
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
	ok := false
	if c, err := r.Cookie(middleware.LauncherDashboardCookieName); err == nil {
		ok = subtle.ConstantTimeCompare([]byte(c.Value), []byte(h.sessionCookie)) == 1
	}
	if ok {
		_, _ = w.Write([]byte(`{"authenticated":true}`))
		return
	}
	resp := launcherAuthStatusResponse{
		Authenticated: false,
		TokenHelp:     &h.tokenHelp,
	}
	enc, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal error"}`))
		return
	}
	_, _ = w.Write(enc)
}
