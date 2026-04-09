package middleware

import (
	"net/http"
	"strings"

	"github.com/sipeed/picoclaw/pkg/netpolicy"
)

// IPAllowlist restricts access to requests from configured CIDR ranges.
// Loopback addresses are always allowed for local administration.
// Empty CIDR list means no restriction.
func IPAllowlist(allowedCIDRs []string, next http.Handler) (http.Handler, error) {
	allowlist, err := netpolicy.NewIPAllowlist(allowedCIDRs)
	if err != nil {
		return nil, err
	}

	if allowlist.IsOpen() {
		return next, nil
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if allowlist.AllowsRemoteAddr(r.RemoteAddr) {
			next.ServeHTTP(w, r)
			return
		}

		rejectByPolicy(w, r)
	}), nil
}

func rejectByPolicy(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"access denied by network policy"}`))
		return
	}
	http.Error(w, "Forbidden", http.StatusForbidden)
}
