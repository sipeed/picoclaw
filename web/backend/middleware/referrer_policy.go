package middleware

import "net/http"

// ReferrerPolicyNoReferrer sets Referrer-Policy: no-referrer on every response so sensitive
// query parameters (e.g. ?token= for dashboard bootstrap) are not leaked via the Referer header.
func ReferrerPolicyNoReferrer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
