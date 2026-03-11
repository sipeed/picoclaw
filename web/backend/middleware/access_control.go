package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// IPAllowlist restricts access to requests from configured CIDR ranges.
// Loopback addresses are always allowed for local administration.
// Empty CIDR list means no restriction.
func IPAllowlist(allowedCIDRs []string, next http.Handler) (http.Handler, error) {
	if len(allowedCIDRs) == 0 {
		return next, nil
	}

	nets := make([]*net.IPNet, 0, len(allowedCIDRs))
	for _, cidr := range allowedCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
		}
		nets = append(nets, ipNet)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIPFromRemoteAddr(r.RemoteAddr)
		if ip == nil {
			rejectByPolicy(w, r)
			return
		}
		if ip.IsLoopback() {
			next.ServeHTTP(w, r)
			return
		}
		for _, ipNet := range nets {
			if ipNet.Contains(ip) {
				next.ServeHTTP(w, r)
				return
			}
		}

		rejectByPolicy(w, r)
	}), nil
}

func clientIPFromRemoteAddr(remoteAddr string) net.IP {
	host := remoteAddr
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = h
	}
	return net.ParseIP(host)
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
