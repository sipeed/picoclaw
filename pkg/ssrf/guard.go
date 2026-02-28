// Package ssrf provides Server-Side Request Forgery protection for HTTP clients.
// It blocks requests to private IP ranges, metadata endpoints, and other sensitive destinations.
package ssrf

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Config holds SSRF protection configuration.
type Config struct {
	// Enabled controls whether SSRF protection is active.
	Enabled bool `json:"enabled"`

	// BlockPrivateIPs blocks requests to private IP ranges (RFC 1918).
	BlockPrivateIPs bool `json:"block_private_ips"`

	// BlockMetadataEndpoints blocks requests to cloud metadata endpoints.
	BlockMetadataEndpoints bool `json:"block_metadata_endpoints"`

	// BlockLocalhost blocks requests to localhost/loopback.
	BlockLocalhost bool `json:"block_localhost"`

	// AllowedHosts is a list of hosts that are explicitly allowed, bypassing SSRF checks.
	AllowedHosts []string `json:"allowed_hosts"`

	// DNSRebindingProtection enables DNS rebinding attack protection.
	DNSRebindingProtection bool `json:"dns_rebinding_protection"`

	// DNSCacheTTL is the duration to cache DNS results for rebinding protection.
	DNSCacheTTL time.Duration `json:"dns_cache_ttl"`
}

// DefaultConfig returns the default SSRF protection configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:                true,
		BlockPrivateIPs:        true,
		BlockMetadataEndpoints: true,
		BlockLocalhost:         true,
		AllowedHosts:           nil,
		DNSRebindingProtection: true,
		DNSCacheTTL:            60 * time.Second,
	}
}

// Guard provides SSRF protection for HTTP requests.
type Guard struct {
	config Config

	// dnsCache stores resolved IPs for DNS rebinding protection.
	dnsCache sync.Map // map[string]dnsCacheEntry
}

type dnsCacheEntry struct {
	ips       []net.IP
	expiresAt time.Time
}

// Error represents an SSRF protection error.
type Error struct {
	Reason string
	URL    string
}

func (e *Error) Error() string {
	return fmt.Sprintf("SSRF protection: %s (URL: %s)", e.Reason, e.URL)
}

// NewGuard creates a new SSRF guard with the given configuration.
func NewGuard(config Config) *Guard {
	return &Guard{
		config: config,
	}
}

// CheckURL validates a URL against SSRF protection rules.
// Returns an error if the URL is blocked, nil otherwise.
func (g *Guard) CheckURL(ctx context.Context, rawURL string) error {
	if !g.config.Enabled {
		return nil
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return &Error{Reason: "invalid URL", URL: rawURL}
	}

	// Only allow http and https schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return &Error{Reason: "only http/https schemes allowed", URL: rawURL}
	}

	host := parsedURL.Hostname()
	if host == "" {
		return &Error{Reason: "missing host", URL: rawURL}
	}

	// Check if host is in allowed list
	for _, allowed := range g.config.AllowedHosts {
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			return nil
		}
	}

	// Resolve host to IPs
	ips, err := g.resolveHost(ctx, host)
	if err != nil {
		return &Error{Reason: fmt.Sprintf("failed to resolve host: %v", err), URL: rawURL}
	}

	// Check each resolved IP
	for _, ip := range ips {
		if err := g.checkIP(ip, rawURL); err != nil {
			return err
		}
	}

	return nil
}

// resolveHost resolves a hostname to IP addresses with caching for DNS rebinding protection.
func (g *Guard) resolveHost(ctx context.Context, host string) ([]net.IP, error) {
	// Check if it's already an IP address
	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, nil
	}

	// Check cache for DNS rebinding protection
	if g.config.DNSRebindingProtection {
		if cached, ok := g.dnsCache.Load(host); ok {
			entry := cached.(dnsCacheEntry)
			if time.Now().Before(entry.expiresAt) {
				return entry.ips, nil
			}
		}
	}

	// Resolve the host
	resolver := &net.Resolver{}
	addrs, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}

	if len(addrs) == 0 {
		return nil, fmt.Errorf("no IP addresses found for host: %s", host)
	}

	ips := make([]net.IP, len(addrs))
	for i, addr := range addrs {
		ips[i] = addr.IP
	}

	// Cache the result for DNS rebinding protection
	if g.config.DNSRebindingProtection {
		g.dnsCache.Store(host, dnsCacheEntry{
			ips:       ips,
			expiresAt: time.Now().Add(g.config.DNSCacheTTL),
		})
	}

	return ips, nil
}

// checkIP checks if an IP address is allowed.
func (g *Guard) checkIP(ip net.IP, rawURL string) error {
	// Block localhost/loopback
	if g.config.BlockLocalhost && isLoopback(ip) {
		return &Error{Reason: "localhost/loopback address blocked", URL: rawURL}
	}

	// Block cloud metadata endpoints (169.254.169.254)
	if g.config.BlockMetadataEndpoints && isMetadataEndpoint(ip) {
		return &Error{Reason: "cloud metadata endpoint blocked", URL: rawURL}
	}

	// Block private IP ranges
	if g.config.BlockPrivateIPs && isPrivateIP(ip) {
		return &Error{Reason: "private IP address blocked", URL: rawURL}
	}

	return nil
}

// isLoopback checks if an IP is a loopback address.
func isLoopback(ip net.IP) bool {
	return ip.IsLoopback()
}

// isMetadataEndpoint checks if an IP is a cloud metadata endpoint.
func isMetadataEndpoint(ip net.IP) bool {
	// AWS/GCP/Azure metadata endpoint: 169.254.169.254
	metadataIP := net.ParseIP("169.254.169.254")
	return ip.Equal(metadataIP)
}

// isPrivateIP checks if an IP is in a private range.
func isPrivateIP(ip net.IP) bool {
	// Check if it's a private address using net's built-in method
	if ip.IsPrivate() {
		return true
	}

	// Additional checks for link-local addresses
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	return false
}

// GetResolvedIPs returns the cached IPs for a host (for DNS rebinding protection).
// This should be used when making the actual request to ensure the IP hasn't changed.
func (g *Guard) GetResolvedIPs(host string) []net.IP {
	if cached, ok := g.dnsCache.Load(host); ok {
		entry := cached.(dnsCacheEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.ips
		}
	}
	return nil
}

// ClearCache clears the DNS cache.
func (g *Guard) ClearCache() {
	g.dnsCache = sync.Map{}
}
