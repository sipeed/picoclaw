package utils

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateURL checks that a URL is safe to fetch, blocking private/internal IPs,
// localhost, link-local addresses, and cloud metadata endpoints.
func ValidateURL(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow http/https schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("only http/https URLs are allowed, got: %s", parsedURL.Scheme)
	}

	host := parsedURL.Hostname()
	if host == "" {
		return fmt.Errorf("missing host in URL")
	}

	// Block localhost variants
	lowerHost := strings.ToLower(host)
	if lowerHost == "localhost" || lowerHost == "ip6-localhost" || lowerHost == "ip6-loopback" {
		return fmt.Errorf("access to localhost is blocked")
	}

	// Resolve host to IP addresses
	ips, err := net.LookupHost(host)
	if err != nil {
		// If DNS resolution fails, try parsing as IP directly
		ip := net.ParseIP(host)
		if ip == nil {
			return fmt.Errorf("failed to resolve host: %w", err)
		}
		ips = []string{ip.String()}
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}

		if err := validateIP(ip); err != nil {
			return err
		}
	}

	return nil
}

// validateIP checks whether an IP address is safe to access.
func validateIP(ip net.IP) error {
	// Block loopback (127.0.0.0/8, ::1)
	if ip.IsLoopback() {
		return fmt.Errorf("access to loopback address %s is blocked", ip)
	}

	// Block private networks (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)
	if ip.IsPrivate() {
		return fmt.Errorf("access to private network address %s is blocked", ip)
	}

	// Block link-local (169.254.0.0/16, fe80::/10)
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("access to link-local address %s is blocked", ip)
	}

	// Block unspecified (0.0.0.0, ::)
	if ip.IsUnspecified() {
		return fmt.Errorf("access to unspecified address %s is blocked", ip)
	}

	// Block cloud metadata endpoints (169.254.169.254)
	metadataIP := net.ParseIP("169.254.169.254")
	if ip.Equal(metadataIP) {
		return fmt.Errorf("access to cloud metadata endpoint %s is blocked", ip)
	}

	return nil
}
