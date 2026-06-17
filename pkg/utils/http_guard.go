package utils

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type SafeHTTPClientOptions struct {
	ProxyURL             string
	Timeout              time.Duration
	PrivateHostWhitelist []string
	AllowPrivateHosts    func() bool
	MaxRedirects         int
}

type PrivateHostWhitelist struct {
	exact map[string]struct{}
	cidrs []*net.IPNet
}

type allowedFirstHopHostKey struct{}

func NewPrivateHostWhitelist(entries []string) (*PrivateHostWhitelist, error) {
	if len(entries) == 0 {
		return nil, nil
	}

	whitelist := &PrivateHostWhitelist{
		exact: make(map[string]struct{}),
		cidrs: make([]*net.IPNet, 0, len(entries)),
	}
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if ip := net.ParseIP(entry); ip != nil {
			whitelist.exact[normalizeWhitelistIP(ip).String()] = struct{}{}
			continue
		}
		_, network, err := net.ParseCIDR(entry)
		if err != nil {
			return nil, fmt.Errorf("invalid entry %q: expected IP or CIDR", entry)
		}
		whitelist.cidrs = append(whitelist.cidrs, network)
	}

	if len(whitelist.exact) == 0 && len(whitelist.cidrs) == 0 {
		return nil, nil
	}
	return whitelist, nil
}

func (w *PrivateHostWhitelist) Contains(ip net.IP) bool {
	if w == nil || ip == nil {
		return false
	}

	normalized := normalizeWhitelistIP(ip)
	if _, ok := w.exact[normalized.String()]; ok {
		return true
	}
	for _, network := range w.cidrs {
		if network.Contains(normalized) {
			return true
		}
	}
	return false
}

func CreateSafeHTTPClient(opts SafeHTTPClientOptions) (*http.Client, error) {
	client, err := CreateHTTPClient(opts.ProxyURL, opts.Timeout)
	if err != nil {
		return nil, err
	}

	whitelist, err := NewPrivateHostWhitelist(opts.PrivateHostWhitelist)
	if err != nil {
		return nil, err
	}

	transport, ok := client.Transport.(*http.Transport)
	if ok {
		dialer := &net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		transport.DialContext = NewSafeDialContext(dialer, whitelist, opts.AllowPrivateHosts)
	}

	maxRedirects := opts.MaxRedirects
	if maxRedirects <= 0 {
		maxRedirects = 10
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			return fmt.Errorf("stopped after %d redirects", maxRedirects)
		}
		if IsObviousPrivateHost(req.URL.Hostname(), whitelist, opts.AllowPrivateHosts) {
			return fmt.Errorf("redirect target is private or local network host")
		}
		AllowConfiguredProxyFirstHop(req, client.Transport)
		return nil
	}

	return client, nil
}

func ValidateSafeHTTPURL(urlStr string, whitelist *PrivateHostWhitelist, allowPrivateHosts func() bool) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("only http/https URLs are allowed")
	}
	if parsedURL.Host == "" {
		return fmt.Errorf("missing domain in URL")
	}
	if IsObviousPrivateHost(parsedURL.Hostname(), whitelist, allowPrivateHosts) {
		return fmt.Errorf("fetching private or local network hosts is not allowed")
	}
	return nil
}

func NewSafeDialContext(
	dialer *net.Dialer,
	whitelist *PrivateHostWhitelist,
	allowPrivateHosts func() bool,
) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		if allowPrivateHosts != nil && allowPrivateHosts() {
			return dialer.DialContext(ctx, network, address)
		}

		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("invalid target address %q: %w", address, err)
		}
		if host == "" {
			return nil, fmt.Errorf("empty target host")
		}
		if isAllowedFirstHopHost(ctx, host) {
			return dialer.DialContext(ctx, network, address)
		}

		if ip := net.ParseIP(host); ip != nil {
			if shouldBlockPrivateIP(ip, whitelist) {
				return nil, fmt.Errorf("blocked private or local target: %s", host)
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		}

		ipAddrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve %s: %w", host, err)
		}

		attempted := 0
		var lastErr error
		for _, ipAddr := range ipAddrs {
			if shouldBlockPrivateIP(ipAddr.IP, whitelist) {
				continue
			}
			attempted++
			conn, err := dialer.DialContext(
				ctx,
				network,
				net.JoinHostPort(ipAddr.IP.String(), port),
			)
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}

		if attempted == 0 {
			return nil, fmt.Errorf(
				"all resolved addresses for %s are private, restricted, or not whitelisted",
				host,
			)
		}
		if lastErr != nil {
			return nil, fmt.Errorf(
				"failed connecting to public addresses for %s: %w",
				host,
				lastErr,
			)
		}
		return nil, fmt.Errorf("failed connecting to public addresses for %s", host)
	}
}

func AllowConfiguredProxyFirstHop(req *http.Request, rt http.RoundTripper) {
	if req == nil {
		return
	}

	transport, ok := rt.(*http.Transport)
	if !ok || transport.Proxy == nil {
		return
	}

	proxyURL, err := transport.Proxy(req)
	if err != nil || proxyURL == nil {
		return
	}

	host := normalizeAllowedFirstHopHost(proxyURL.Hostname())
	if host == "" {
		return
	}

	*req = *req.WithContext(context.WithValue(
		req.Context(),
		allowedFirstHopHostKey{},
		host,
	))
}

func IsObviousPrivateHost(
	host string,
	whitelist *PrivateHostWhitelist,
	allowPrivateHosts func() bool,
) bool {
	if allowPrivateHosts != nil && allowPrivateHosts() {
		return false
	}

	h := strings.ToLower(strings.TrimSpace(host))
	h = strings.TrimSuffix(h, ".")
	if h == "" {
		return true
	}

	if h == "localhost" || strings.HasSuffix(h, ".localhost") {
		return true
	}

	if ip := net.ParseIP(h); ip != nil {
		return shouldBlockPrivateIP(ip, whitelist)
	}

	return false
}

func IsPrivateOrRestrictedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}

	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 10 ||
			ip4[0] == 127 ||
			ip4[0] == 0 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168) ||
			(ip4[0] == 169 && ip4[1] == 254) ||
			(ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127) ||
			(ip4[0] == 198 && ip4[1] >= 18 && ip4[1] <= 19) {
			return true
		}
		return false
	}

	if len(ip) == net.IPv6len {
		if (ip[0] & 0xfe) == 0xfc {
			return true
		}
		if ip[0] == 0x20 && ip[1] == 0x02 {
			embedded := net.IPv4(ip[2], ip[3], ip[4], ip[5])
			return IsPrivateOrRestrictedIP(embedded)
		}
		if ip[0] == 0x20 && ip[1] == 0x01 && ip[2] == 0x00 && ip[3] == 0x00 {
			client := net.IPv4(ip[12]^0xff, ip[13]^0xff, ip[14]^0xff, ip[15]^0xff)
			return IsPrivateOrRestrictedIP(client)
		}
	}

	return false
}

func isAllowedFirstHopHost(ctx context.Context, host string) bool {
	allowed, ok := ctx.Value(allowedFirstHopHostKey{}).(string)
	if !ok || allowed == "" {
		return false
	}
	return allowed == normalizeAllowedFirstHopHost(host)
}

func normalizeAllowedFirstHopHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	return strings.TrimSuffix(host, ".")
}

func normalizeWhitelistIP(ip net.IP) net.IP {
	if ip == nil {
		return nil
	}
	if ip4 := ip.To4(); ip4 != nil {
		return ip4
	}
	return ip
}

func shouldBlockPrivateIP(ip net.IP, whitelist *PrivateHostWhitelist) bool {
	return IsPrivateOrRestrictedIP(ip) && !whitelist.Contains(ip)
}
