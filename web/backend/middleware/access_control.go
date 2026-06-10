package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// IPAllowlistConfig controls launcher network access decisions.
type IPAllowlistConfig struct {
	AllowedCIDRs         []string
	AllowLocalhostBypass bool
	TrustedProxyCIDRs    []string
}

// IPAllowlist restricts access to requests from configured CIDR ranges.
// Loopback addresses can optionally bypass CIDR checks for local administration.
// X-Forwarded-For is only trusted when the immediate peer is in a trusted CIDR.
// Empty CIDR list means no restriction.
func IPAllowlist(cfg IPAllowlistConfig, next http.Handler) (http.Handler, error) {
	allowedNets, err := parseCIDRNets(cfg.AllowedCIDRs)
	if err != nil {
		return nil, err
	}
	trustedProxyNets, err := parseCIDRNets(cfg.TrustedProxyCIDRs)
	if err != nil {
		return nil, err
	}

	if len(allowedNets) == 0 {
		return next, nil
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peerIP := clientIPFromRemoteAddr(r.RemoteAddr)
		if peerIP == nil {
			rejectByPolicy(w, r)
			return
		}

		ip := peerIP
		if containsIP(trustedProxyNets, peerIP) {
			ip = clientIPFromXForwardedFor(r.Header.Get("X-Forwarded-For"), trustedProxyNets, peerIP)
		}

		if cfg.AllowLocalhostBypass && ip.IsLoopback() {
			next.ServeHTTP(w, r)
			return
		}
		if containsIP(allowedNets, ip) {
			next.ServeHTTP(w, r)
			return
		}

		rejectByPolicy(w, r)
	}), nil
}

func parseCIDRNets(cidrs []string) ([]*net.IPNet, error) {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
		}
		nets = append(nets, ipNet)
	}
	return nets, nil
}

func containsIP(nets []*net.IPNet, ip net.IP) bool {
	for _, ipNet := range nets {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

func clientIPFromRemoteAddr(remoteAddr string) net.IP {
	host := remoteAddr
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = h
	}
	return net.ParseIP(host)
}

func clientIPFromXForwardedFor(header string, trustedProxyNets []*net.IPNet, fallback net.IP) net.IP {
	parts := strings.Split(header, ",")
	ips := make([]net.IP, 0, len(parts))
	for _, part := range parts {
		if ip := parseIPToken(part); ip != nil {
			ips = append(ips, ip)
		}
	}
	if len(ips) == 0 {
		return fallback
	}
	for i := len(ips) - 1; i >= 0; i-- {
		if !containsIP(trustedProxyNets, ips[i]) {
			return ips[i]
		}
	}
	return ips[0]
}

func parseIPToken(raw string) net.IP {
	token := strings.Trim(strings.TrimSpace(raw), `"`)
	if token == "" {
		return nil
	}
	if ip := net.ParseIP(token); ip != nil {
		return ip
	}
	if host, _, err := net.SplitHostPort(token); err == nil {
		return net.ParseIP(host)
	}
	return nil
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
