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
)

const (
	gatewayDefaultLoopbackHost = "127.0.0.1"
	gatewayFallbackBindHost    = "0.0.0.0"
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
				"loopback bind %s:%d failed: %w; no non-loopback interface CIDRs discovered",
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

	if fallbackBindErr := probeGatewayBind(gatewayFallbackBindHost, port); fallbackBindErr != nil {
		return nil, fmt.Errorf(
			"loopback bind %s:%d failed: %w; fallback bind %s:%d failed: %v",
			host,
			port,
			bindErr,
			gatewayFallbackBindHost,
			port,
			fallbackBindErr,
		)
	}

	return &gatewayListenDecision{
		BindHost:     gatewayFallbackBindHost,
		AllowedCIDRs: fallbackCIDRs,
		AutoFallback: true,
		FallbackReason: fmt.Sprintf(
			"loopback bind %s:%d failed, fallback to %s:%d with CIDR allowlist",
			host,
			port,
			gatewayFallbackBindHost,
			port,
		),
	}, nil
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
		if len(effectiveCIDRs) == 0 {
			return next, nil
		}

		nets := make([]*net.IPNet, 0, len(effectiveCIDRs))
		for _, cidr := range effectiveCIDRs {
			_, ipNet, err := net.ParseCIDR(cidr)
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
			}
			nets = append(nets, ipNet)
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIPFromRemoteAddr(r.RemoteAddr)
			if ip == nil {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			// Loopback is always allowed for local administration.
			// When deployed behind a local reverse proxy/tunnel, forwarded
			// external traffic may still appear as loopback at this layer.
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
			http.Error(w, "Forbidden", http.StatusForbidden)
		}), nil
	}
}

func clientIPFromRemoteAddr(remoteAddr string) net.IP {
	host := remoteAddr
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = h
	}
	return net.ParseIP(strings.TrimSpace(host))
}
