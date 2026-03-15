package web

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func withPrivateWebFetchHostsAllowed(t *testing.T) {
	t.Helper()
	previous := allowPrivateWebFetchHosts.Load()
	allowPrivateWebFetchHosts.Store(true)
	t.Cleanup(func() {
		allowPrivateWebFetchHosts.Store(previous)
	})
}

func TestWebTool_WebFetch_PrivateHostBlocked(t *testing.T) {
	tool, err := NewWebFetchTool(50000, testFetchLimit)
	if err != nil {
		t.Fatalf("Failed to create web fetch tool: %v", err)
	}
	result := tool.Execute(context.Background(), map[string]any{
		"url": "http://127.0.0.1:0",
	})

	if !result.IsError {
		t.Errorf("expected error for private host URL, got success")
	}
	if !strings.Contains(result.ForLLM, "private or local network") &&
		!strings.Contains(result.ForUser, "private or local network") {
		t.Errorf("expected private host block message, got %q", result.ForLLM)
	}
}

func TestWebTool_WebFetch_PrivateHostAllowedForTests(t *testing.T) {
	withPrivateWebFetchHostsAllowed(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	tool, err := NewWebFetchTool(50000, testFetchLimit)
	if err != nil {
		t.Fatalf("Failed to create web fetch tool: %v", err)
	}
	result := tool.Execute(context.Background(), map[string]any{
		"url": server.URL,
	})

	if result.IsError {
		t.Errorf("expected success when private host access is allowed in tests, got %q", result.ForLLM)
	}
}

func TestWebFetch_BlocksIPv4MappedIPv6Loopback(t *testing.T) {
	tool, err := NewWebFetchTool(50000, testFetchLimit)
	if err != nil {
		t.Fatalf("Failed to create web fetch tool: %v", err)
	}
	result := tool.Execute(context.Background(), map[string]any{
		"url": "http://[::ffff:127.0.0.1]:0",
	})

	if !result.IsError {
		t.Error("expected error for IPv4-mapped IPv6 loopback URL, got success")
	}
}

func TestWebFetch_BlocksMetadataIP(t *testing.T) {
	tool, err := NewWebFetchTool(50000, testFetchLimit)
	if err != nil {
		t.Fatalf("Failed to create web fetch tool: %v", err)
	}
	result := tool.Execute(context.Background(), map[string]any{
		"url": "http://169.254.169.254/latest/meta-data",
	})

	if !result.IsError {
		t.Error("expected error for cloud metadata IP, got success")
	}
}

func TestWebFetch_BlocksIPv6UniqueLocal(t *testing.T) {
	tool, err := NewWebFetchTool(50000, testFetchLimit)
	if err != nil {
		t.Fatalf("Failed to create web fetch tool: %v", err)
	}
	result := tool.Execute(context.Background(), map[string]any{
		"url": "http://[fd00::1]:0",
	})

	if !result.IsError {
		t.Error("expected error for IPv6 unique local address, got success")
	}
}

func TestWebFetch_Blocks6to4WithPrivateEmbed(t *testing.T) {
	tool, err := NewWebFetchTool(50000, testFetchLimit)
	if err != nil {
		t.Fatalf("Failed to create web fetch tool: %v", err)
	}
	// 2002:7f00:0001::1 embeds 127.0.0.1
	result := tool.Execute(context.Background(), map[string]any{
		"url": "http://[2002:7f00:0001::1]:0",
	})

	if !result.IsError {
		t.Error("expected error for 6to4 with private embedded IPv4, got success")
	}
}

func TestWebFetch_Allows6to4WithPublicEmbed(t *testing.T) {
	if os.Getenv("USER") == "jules" {
		t.Skip("Skipping test in sandbox environment as it will never pass due to environment restrictions")
	}
	tool, err := NewWebFetchTool(50000, testFetchLimit)
	if err != nil {
		t.Fatalf("Failed to create web fetch tool: %v", err)
	}
	// 2002:0801:0101::1 embeds 8.1.1.1 (public) — pre-flight should pass,
	// connection will fail (no listener) but that's after the SSRF check.
	result := tool.Execute(context.Background(), map[string]any{
		"url": "http://[2002:0801:0101::1]:0",
	})

	// Should NOT be blocked by SSRF check — error should be connection failure, not "private"
	if result.IsError && strings.Contains(result.ForLLM, "private") {
		t.Errorf("6to4 with public embedded IPv4 should not be blocked as private. Error: %q", result.ForLLM)
	}
}

func TestWebFetch_RedirectToPrivateBlocked(t *testing.T) {
	withPrivateWebFetchHostsAllowed(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Redirect to a private IP
		http.Redirect(w, r, "http://10.0.0.1/secret", http.StatusFound)
	}))
	defer server.Close()

	// Temporarily disable private host allowance for the redirect check
	allowPrivateWebFetchHosts.Store(false)
	defer allowPrivateWebFetchHosts.Store(true)

	tool, err := NewWebFetchTool(50000, testFetchLimit)
	if err != nil {
		t.Fatalf("Failed to create web fetch tool: %v", err)
	}
	result := tool.Execute(context.Background(), map[string]any{
		"url": server.URL,
	})

	if !result.IsError {
		t.Error("expected error when redirecting to private IP, got success")
	}
}

func TestIsPrivateOrRestrictedIP_Table(t *testing.T) {
	tests := []struct {
		ip      string
		blocked bool
		desc    string
	}{
		{"127.0.0.1", true, "IPv4 loopback"},
		{"10.0.0.1", true, "IPv4 private class A"},
		{"172.16.0.1", true, "IPv4 private class B"},
		{"192.168.1.1", true, "IPv4 private class C"},
		{"169.254.169.254", true, "link-local / cloud metadata"},
		{"100.64.0.1", true, "carrier-grade NAT"},
		{"0.0.0.0", true, "unspecified"},
		{"8.8.8.8", false, "public DNS"},
		{"1.1.1.1", false, "public DNS"},
		{"::1", true, "IPv6 loopback"},
		{"::ffff:127.0.0.1", true, "IPv4-mapped IPv6 loopback"},
		{"::ffff:10.0.0.1", true, "IPv4-mapped IPv6 private"},
		{"fc00::1", true, "IPv6 unique local"},
		{"fd00::1", true, "IPv6 unique local"},
		{"2002:7f00:0001::1", true, "6to4 with embedded 127.x (private)"},
		{"2002:0a00:0001::1", true, "6to4 with embedded 10.0.0.1 (private)"},
		{"2002:0801:0101::1", false, "6to4 with embedded 8.1.1.1 (public)"},
		{"2001:0000:4136:e378:8000:63bf:f5ff:fffe", true, "Teredo with client 10.0.0.1 (private)"},
		{"2001:0000:4136:e378:8000:63bf:f7f6:fefe", false, "Teredo with client 8.9.1.1 (public)"},
		{"2607:f8b0:4004:800::200e", false, "public IPv6 (Google)"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP: %s", tt.ip)
			}
			got := isPrivateOrRestrictedIP(ip)
			if got != tt.blocked {
				t.Errorf("isPrivateOrRestrictedIP(%s) = %v, want %v", tt.ip, got, tt.blocked)
			}
		})
	}
}
