package netpolicy

import (
	"fmt"
	"net"
	"strings"
)

// IPAllowlist evaluates whether a remote address is allowed by CIDR policy.
// Loopback addresses are always allowed for local administration.
type IPAllowlist struct {
	nets []*net.IPNet
}

// NewIPAllowlist parses CIDR rules and constructs an allowlist checker.
// Empty CIDR list means unrestricted policy.
func NewIPAllowlist(allowedCIDRs []string) (*IPAllowlist, error) {
	if len(allowedCIDRs) == 0 {
		return &IPAllowlist{}, nil
	}

	seen := make(map[string]struct{}, len(allowedCIDRs))
	nets := make([]*net.IPNet, 0, len(allowedCIDRs))
	for _, rawCIDR := range allowedCIDRs {
		cidr := strings.TrimSpace(rawCIDR)
		if cidr == "" {
			continue
		}

		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
		}

		canonical := ipNet.String()
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}
		nets = append(nets, ipNet)
	}

	if len(nets) == 0 {
		return &IPAllowlist{}, nil
	}

	return &IPAllowlist{nets: nets}, nil
}

// IsOpen reports whether the allowlist has no restrictions.
func (a *IPAllowlist) IsOpen() bool {
	return a == nil || len(a.nets) == 0
}

// AllowsRemoteAddr checks whether RemoteAddr is permitted.
func (a *IPAllowlist) AllowsRemoteAddr(remoteAddr string) bool {
	if a.IsOpen() {
		return true
	}

	ip := ClientIPFromRemoteAddr(remoteAddr)
	return a.AllowsIP(ip)
}

// AllowsIP checks whether an IP is permitted.
func (a *IPAllowlist) AllowsIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	if a.IsOpen() {
		return true
	}

	for _, ipNet := range a.nets {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

// ClientIPFromRemoteAddr parses the IP component from net/http RemoteAddr.
func ClientIPFromRemoteAddr(remoteAddr string) net.IP {
	host := strings.TrimSpace(remoteAddr)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	// Strip IPv6 zone identifier (for example: fe80::1%eth0).
	if i := strings.LastIndex(host, "%"); i != -1 {
		host = host[:i]
	}

	return net.ParseIP(strings.TrimSpace(host))
}
