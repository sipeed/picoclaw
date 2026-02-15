package gateway

import (
	"net/http"
	"strings"
)

// AuthMiddleware validates the Authorization: Bearer <token> header
// against the configured API key. Public paths bypass auth.
func AuthMiddleware(apiKey string, publicPaths []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for public paths
		for _, p := range publicPaths {
			if r.URL.Path == p {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Skip auth if no API key configured (open mode)
		if apiKey == "" {
			next.ServeHTTP(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		if auth == "" {
			writeError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) {
			writeError(w, http.StatusUnauthorized, "invalid authorization format")
			return
		}

		token := auth[len(prefix):]
		if token != apiKey {
			writeError(w, http.StatusUnauthorized, "invalid API key")
			return
		}

		next.ServeHTTP(w, r)
	})
}
