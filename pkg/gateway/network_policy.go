package gateway

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/netpolicy"
)

const (
	gatewayDefaultLoopbackHost = "127.0.0.1"
	gatewayFallbackBindHostV4  = "0.0.0.0"
	gatewayFallbackBindHostV6  = "::"
)

type gatewayListenDecision struct {
	BindHost       string
	AllowedCIDRs   []string
	AutoFallback   bool
	FallbackReason string
}

var (
	probeGatewayBind     = probeTCPBind
	discoverGatewayCIDRs = discoverLocalInterfaceCIDRs

	fallbackPrivateCIDRs = mustParseCIDRs(
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"100.64.0.0/10",
		"fc00::/7",
	)
)

func resolveGatewayListenDecision(
	configuredHost string,
	port int,
	configuredCIDRs []string,
) (*gatewayListenDecision, error) {
	host := strings.TrimSpace(configuredHost)
	if host == "" {
		host = gatewayDefaultLoopbackHost
	}

	normalizedCIDRs, err := normalizeAndValidateCIDRs(configuredCIDRs)
	if err != nil {
		return nil, fmt.Errorf("invalid gateway allowed_cidrs: %w", err)
	}

	bindErr := probeGatewayBind(host, port)
	if bindErr == nil {
		return &gatewayListenDecision{
			BindHost:     host,
			AllowedCIDRs: normalizedCIDRs,
		}, nil
	}

	if !isLoopbackHost(host) {
		return nil, fmt.Errorf("bind %s:%d failed: %w", host, port, bindErr)
	}

	// Before widening exposure via wildcard bind, try other common loopback hosts.
	for _, altHost := range alternativeLoopbackHosts(host) {
		if err := probeGatewayBind(altHost, port); err == nil {
			return &gatewayListenDecision{
				BindHost:     altHost,
				AllowedCIDRs: normalizedCIDRs,
				AutoFallback: true,
				FallbackReason: fmt.Sprintf(
					"loopback bind %s:%d failed, fallback to loopback host %s:%d",
					host,
					port,
					altHost,
					port,
				),
			}, nil
		}
	}

	var discoveredCIDRs []string
	if len(normalizedCIDRs) == 0 {
		var discoverErr error
		discoveredCIDRs, discoverErr = discoverGatewayCIDRs()
		if discoverErr != nil {
			return nil, fmt.Errorf(
				"loopback bind %s:%d failed: %w; interface discovery failed: %v",
				host,
				port,
				bindErr,
				discoverErr,
			)
		}
		if len(discoveredCIDRs) == 0 {
			return nil, fmt.Errorf(
				"loopback bind %s:%d failed: %w; no private non-loopback interface CIDRs discovered",
				host,
				port,
				bindErr,
			)
		}
	}

	fallbackCIDRs := normalizedCIDRs
	if len(fallbackCIDRs) == 0 {
		fallbackCIDRs = discoveredCIDRs
	}
	if len(fallbackCIDRs) == 0 {
		return nil, fmt.Errorf("loopback bind %s:%d failed: %w; fallback allowlist is empty", host, port, bindErr)
	}

	fallbackCandidates := fallbackBindCandidates(host)
	fallbackHost, fallbackBindErr := probeFallbackBindCandidates(fallbackCandidates, port)
	if fallbackBindErr != nil {
		return nil, fmt.Errorf(
			"loopback bind %s:%d failed: %w; fallback bind candidates %v failed: %v",
			host,
			port,
			bindErr,
			fallbackCandidates,
			fallbackBindErr,
		)
	}

	return &gatewayListenDecision{
		BindHost:     fallbackHost,
		AllowedCIDRs: fallbackCIDRs,
		AutoFallback: true,
		FallbackReason: fmt.Sprintf(
			"loopback bind %s:%d failed, fallback to %s:%d with CIDR allowlist",
			host,
			port,
			fallbackHost,
			port,
		),
	}, nil
}

func alternativeLoopbackHosts(configuredHost string) []string {
	configured := strings.TrimSpace(strings.ToLower(configuredHost))
	candidates := []string{"127.0.0.1", "::1", "localhost"}
	out := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))

	for _, candidate := range candidates {
		normalized := strings.TrimSpace(strings.ToLower(candidate))
		if normalized == configured {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, candidate)
	}

	return out
}

func fallbackBindCandidates(configuredLoopbackHost string) []string {
	lower := strings.TrimSpace(strings.ToLower(configuredLoopbackHost))
	if lower == "localhost" {
		return []string{gatewayFallbackBindHostV6, gatewayFallbackBindHostV4}
	}

	ip := net.ParseIP(lower)
	if ip != nil && ip.To4() == nil {
		return []string{gatewayFallbackBindHostV6, gatewayFallbackBindHostV4}
	}

	return []string{gatewayFallbackBindHostV4, gatewayFallbackBindHostV6}
}

func probeFallbackBindCandidates(candidates []string, port int) (string, error) {
	errList := make([]error, 0, len(candidates))
	for _, host := range candidates {
		err := probeGatewayBind(host, port)
		if err == nil {
			return host, nil
		}
		errList = append(errList, fmt.Errorf("%s:%d: %w", host, port, err))
	}

	if len(errList) == 0 {
		return "", fmt.Errorf("no fallback bind candidates")
	}

	return "", errors.Join(errList...)
}

func isWildcardBindHost(host string) bool {
	switch strings.TrimSpace(host) {
	case gatewayFallbackBindHostV4, gatewayFallbackBindHostV6:
		return true
	default:
		return false
	}
}

func normalizeAndValidateCIDRs(cidrs []string) ([]string, error) {
	if len(cidrs) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(cidrs))
	out := make([]string, 0, len(cidrs))
	for _, raw := range cidrs {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		_, ipNet, err := net.ParseCIDR(trimmed)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %q: %w", trimmed, err)
		}
		canonical := ipNet.String()
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}
		out = append(out, canonical)
	}
	if len(out) == 0 {
		return nil, nil
	}
	sort.Strings(out)
	return out, nil
}

func discoverLocalInterfaceCIDRs() ([]string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(ifaces))
	seen := make(map[string]struct{})
	ifaceErrs := make([]error, 0)

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			ifaceErrs = append(ifaceErrs, fmt.Errorf("%s: %w", iface.Name, err))
			continue
		}

		for _, addr := range addrs {
			ipNet := toIPNet(addr)
			if ipNet == nil || ipNet.IP == nil {
				continue
			}
			ip := ipNet.IP
			if ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() ||
				ip.IsInterfaceLocalMulticast() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() {
				continue
			}

			mask := ipNet.Mask
			if len(mask) == 0 {
				continue
			}
			ip = normalizeIPForMask(ip, mask)
			if !isSafeFallbackIP(ip) {
				continue
			}

			masked := ip.Mask(mask)
			if masked == nil {
				continue
			}
			canonical := (&net.IPNet{IP: masked, Mask: mask}).String()
			if _, ok := seen[canonical]; ok {
				continue
			}
			seen[canonical] = struct{}{}
			out = append(out, canonical)
		}
	}

	sort.Strings(out)
	if len(out) == 0 && len(ifaceErrs) > 0 {
		return nil, errors.Join(ifaceErrs...)
	}
	return out, nil
}

func mustParseCIDRs(cidrs ...string) []*net.IPNet {
	parsed := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("invalid built-in fallback CIDR %q: %v", cidr, err))
		}
		parsed = append(parsed, ipNet)
	}
	return parsed
}

func isSafeFallbackIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, ipNet := range fallbackPrivateCIDRs {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

func normalizeIPForMask(ip net.IP, mask net.IPMask) net.IP {
	switch len(mask) {
	case net.IPv4len:
		if ip4 := ip.To4(); ip4 != nil {
			return ip4
		}
	case net.IPv6len:
		if ip16 := ip.To16(); ip16 != nil {
			return ip16
		}
	}
	return ip
}

func toIPNet(addr net.Addr) *net.IPNet {
	switch v := addr.(type) {
	case *net.IPNet:
		return v
	case *net.IPAddr:
		if v.IP == nil {
			return nil
		}
		bits := 128
		if v.IP.To4() != nil {
			bits = 32
		}
		return &net.IPNet{IP: v.IP, Mask: net.CIDRMask(bits, bits)}
	default:
		return nil
	}
}

func probeTCPBind(host string, port int) error {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	_ = ln.Close()
	return nil
}

func isLoopbackHost(host string) bool {
	normalized := strings.TrimSpace(strings.ToLower(host))
	if normalized == "localhost" {
		return true
	}
	ip := net.ParseIP(normalized)
	return ip != nil && ip.IsLoopback()
}

func newCIDRAllowlistMiddleware(allowedCIDRs []string) channels.HTTPMiddleware {
	effectiveCIDRs := append([]string(nil), allowedCIDRs...)
	return func(next http.Handler) (http.Handler, error) {
		allowlist, err := netpolicy.NewIPAllowlist(effectiveCIDRs)
		if err != nil {
			return nil, err
		}
		if allowlist.IsOpen() {
			return next, nil
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Loopback is always allowed for local administration.
			// When deployed behind a local reverse proxy/tunnel, forwarded
			// external traffic may still appear as loopback at this layer.
			if allowlist.AllowsRemoteAddr(r.RemoteAddr) {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "Forbidden", http.StatusForbidden)
		}), nil
	}
}
