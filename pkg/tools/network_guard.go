package tools

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
)

var (
	cgnatPrefix     = netip.MustParsePrefix("100.64.0.0/10")
	benchmarkPrefix = netip.MustParsePrefix("198.18.0.0/15")
	reservedPrefix  = netip.MustParsePrefix("240.0.0.0/4")
)

type fetchTargetValidator struct {
	resolver     *net.Resolver
	allowedHosts map[string]struct{}
}

func newFetchTargetValidator(allowHosts []string, resolver *net.Resolver) *fetchTargetValidator {
	allowed := make(map[string]struct{}, len(allowHosts))
	for _, host := range allowHosts {
		normalized := normalizeHostToken(host)
		if normalized != "" {
			allowed[normalized] = struct{}{}
		}
	}

	if resolver == nil {
		resolver = net.DefaultResolver
	}

	return &fetchTargetValidator{
		resolver:     resolver,
		allowedHosts: allowed,
	}
}

func (v *fetchTargetValidator) validateURL(ctx context.Context, target *url.URL) error {
	host := normalizeHostToken(target.Hostname())
	if host == "" {
		return fmt.Errorf("missing domain in URL")
	}

	port := target.Port()
	if v.isAllowed(host, port) {
		return nil
	}

	if isBlockedHostname(host) {
		return fmt.Errorf("blocked destination: host %q is internal-only", target.Hostname())
	}

	if ip, ok := parseIPLiteral(host); ok {
		if IsBlockedIP(ip) {
			return fmt.Errorf("blocked destination: IP %s is not publicly routable", ip)
		}
		return nil
	}

	addrs, err := v.resolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return fmt.Errorf("failed to resolve host %q: %w", host, err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("failed to resolve host %q: no records", host)
	}

	for _, addr := range addrs {
		if IsBlockedIP(addr) {
			return fmt.Errorf("blocked destination: host %q resolves to non-public IP %s", host, addr)
		}
	}

	return nil
}

func (v *fetchTargetValidator) isAllowed(host, port string) bool {
	if len(v.allowedHosts) == 0 {
		return false
	}

	if _, ok := v.allowedHosts[host]; ok {
		return true
	}
	if port != "" {
		if _, ok := v.allowedHosts[host+":"+port]; ok {
			return true
		}
	}
	return false
}

// ValidateFetchTarget applies the default web fetch target policy.
func ValidateFetchTarget(target *url.URL) error {
	return newFetchTargetValidator(nil, net.DefaultResolver).validateURL(context.Background(), target)
}

// IsBlockedIP returns true for IPs that should not be reachable from web_fetch.
func IsBlockedIP(addr netip.Addr) bool {
	if !addr.IsValid() {
		return true
	}

	if addr.IsLoopback() ||
		addr.IsPrivate() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsMulticast() ||
		addr.IsUnspecified() ||
		addr.IsInterfaceLocalMulticast() {
		return true
	}

	if addr.Is4() {
		if cgnatPrefix.Contains(addr) || benchmarkPrefix.Contains(addr) || reservedPrefix.Contains(addr) {
			return true
		}
	}

	return false
}

func isBlockedHostname(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	if strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return true
	}
	if host == "metadata.google.internal" || host == "metadata" {
		return true
	}
	return false
}

func parseIPLiteral(host string) (netip.Addr, bool) {
	if i := strings.Index(host, "%"); i >= 0 {
		host = host[:i]
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, false
	}
	return addr.Unmap(), true
}

func normalizeHostToken(raw string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(raw)), ".")
}

func guardedDialContext(base *net.Dialer, validator *fetchTargetValidator) func(ctx context.Context, network, address string) (net.Conn, error) {
	if base == nil {
		base = &net.Dialer{}
	}

	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			host = address
			port = ""
		}

		target := &url.URL{Host: host}
		if port != "" {
			target.Host = net.JoinHostPort(host, port)
		}

		if err := validator.validateURL(ctx, target); err != nil {
			return nil, err
		}

		return base.DialContext(ctx, network, address)
	}
}
