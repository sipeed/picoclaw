package gateway

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// authMiddleware wraps a handler with Bearer token authentication.
// If cfg.Gateway.APIKey is empty, authentication is skipped.
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := s.cfg.Gateway.APIKey
		if apiKey == "" {
			next(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeJSONError(w, http.StatusUnauthorized, "missing Authorization header")
			return
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			writeJSONError(w, http.StatusUnauthorized, "invalid Authorization format")
			return
		}

		token := authHeader[len(prefix):]
		if subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) != 1 {
			writeJSONError(w, http.StatusForbidden, "invalid token")
			return
		}

		next(w, r)
	}
}

func writeJSONError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, map[string]string{"error": message})
}
