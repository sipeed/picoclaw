package auth

import (
	"net/http"
	"strings"
)

const (
	SessionCookieName = "picoclaw_session"
	AuthorizationHeader = "Authorization"
)

type AuthMiddleware struct {
	configStore  *AuthConfigStore
	sessionStore SessionStore
}

func NewAuthMiddleware(configStore *AuthConfigStore, sessionStore SessionStore) *AuthMiddleware {
	return &AuthMiddleware{
		configStore:  configStore,
		sessionStore: sessionStore,
	}
}

func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.configStore.IsEnabled() {
			next.ServeHTTP(w, r)
			return
		}

		if IsPublicRoute(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		sessionID := m.extractSessionID(r)
		if sessionID == "" {
			m.unauthorized(w, r)
			return
		}

		if !m.sessionStore.Validate(sessionID) {
			m.unauthorized(w, r)
			return
		}

		m.sessionStore.Refresh(sessionID, m.configStore.Get().SessionTTL)
		next.ServeHTTP(w, r)
	})
}

func (m *AuthMiddleware) extractSessionID(r *http.Request) string {
	if cookie, err := r.Cookie(SessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	authHeader := r.Header.Get(AuthorizationHeader)
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
		return authHeader
	}

	return ""
}

func (m *AuthMiddleware) unauthorized(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized","code":401}`))
		return
	}
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

func IsAuthRoute(path string) bool {
	return path == "/api/auth/login" ||
		path == "/api/auth/status" ||
		path == "/api/auth/setup" ||
		strings.HasPrefix(path, "/login")
}

func IsPublicRoute(path string) bool {
	publicPrefixes := []string{
		"/api/auth/",
		"/login",
		"/assets/",
		"/favicon",
		"/web-app-manifest",
		"/site.webmanifest",
	}

	for _, prefix := range publicPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return path == "/" || path == "/index.html"
}
