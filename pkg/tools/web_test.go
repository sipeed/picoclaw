package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

const testFetchLimit = int64(10 * 1024 * 1024)

// TestWebTool_WebFetch_Success verifies successful URL fetching
func TestWebTool_WebFetch_Success(t *testing.T) {
	withPrivateWebFetchHostsAllowed(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><h1>Test Page</h1><p>Content here</p></body></html>"))
	}))
	defer server.Close()

	tool, err := NewWebFetchTool(50000, testFetchLimit)
	if err != nil {
		t.Fatalf("Failed to create web fetch tool: %v", err)
	}

	ctx := context.Background()
	args := map[string]any{
		"url": server.URL,
	}

	result := tool.Execute(ctx, args)

	// Success should not be an error
	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	// ForLLM should contain the fetched content (full JSON result)
	if !strings.Contains(result.ForLLM, "Test Page") {
		t.Errorf("Expected ForLLM to contain 'Test Page', got: %s", result.ForLLM)
	}

	// ForUser should contain summary
	if !strings.Contains(result.ForUser, "bytes") && !strings.Contains(result.ForUser, "extractor") {
		t.Errorf("Expected ForUser to contain summary, got: %s", result.ForUser)
	}
}

// TestWebTool_WebFetch_JSON verifies JSON content handling
func TestWebTool_WebFetch_JSON(t *testing.T) {
	withPrivateWebFetchHostsAllowed(t)

	testData := map[string]string{"key": "value", "number": "123"}
	expectedJSON, _ := json.MarshalIndent(testData, "", "  ")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(expectedJSON)
	}))
	defer server.Close()

	tool, err := NewWebFetchTool(50000, testFetchLimit)
	if err != nil {
		logger.ErrorCF("agent", "Failed to create web fetch tool", map[string]any{"error": err.Error()})
	}

	ctx := context.Background()
	args := map[string]any{
		"url": server.URL,
	}

	result := tool.Execute(ctx, args)

	// Success should not be an error
	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	// ForLLM should contain formatted JSON
	if !strings.Contains(result.ForLLM, "key") && !strings.Contains(result.ForLLM, "value") {
		t.Errorf("Expected ForLLM to contain JSON data, got: %s", result.ForLLM)
	}
}

// TestWebTool_WebFetch_InvalidURL verifies error handling for invalid URL
func TestWebTool_WebFetch_InvalidURL(t *testing.T) {
	tool, err := NewWebFetchTool(50000, testFetchLimit)
	if err != nil {
		logger.ErrorCF("agent", "Failed to create web fetch tool", map[string]any{"error": err.Error()})
	}

	ctx := context.Background()
	args := map[string]any{
		"url": "not-a-valid-url",
	}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error for invalid URL")
	}

	// Should contain error message (either "invalid URL" or scheme error)
	if !strings.Contains(result.ForLLM, "URL") && !strings.Contains(result.ForUser, "URL") {
		t.Errorf("Expected error message for invalid URL, got ForLLM: %s", result.ForLLM)
	}
}

// TestWebTool_WebFetch_UnsupportedScheme verifies error handling for non-http URLs
func TestWebTool_WebFetch_UnsupportedScheme(t *testing.T) {
	tool, err := NewWebFetchTool(50000, testFetchLimit)
	if err != nil {
		logger.ErrorCF("agent", "Failed to create web fetch tool", map[string]any{"error": err.Error()})
	}

	ctx := context.Background()
	args := map[string]any{
		"url": "ftp://example.com/file.txt",
	}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error for unsupported URL scheme")
	}

	// Should mention only http/https allowed
	if !strings.Contains(result.ForLLM, "http/https") && !strings.Contains(result.ForUser, "http/https") {
		t.Errorf("Expected scheme error message, got ForLLM: %s", result.ForLLM)
	}
}

// TestWebTool_WebFetch_MissingURL verifies error handling for missing URL
func TestWebTool_WebFetch_MissingURL(t *testing.T) {
	tool, err := NewWebFetchTool(50000, testFetchLimit)
	if err != nil {
		logger.ErrorCF("agent", "Failed to create web fetch tool", map[string]any{"error": err.Error()})
	}

	ctx := context.Background()
	args := map[string]any{}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error when URL is missing")
	}

	// Should mention URL is required
	if !strings.Contains(result.ForLLM, "url is required") && !strings.Contains(result.ForUser, "url is required") {
		t.Errorf("Expected 'url is required' message, got ForLLM: %s", result.ForLLM)
	}
}

// TestWebTool_WebFetch_Truncation verifies content truncation
func TestWebTool_WebFetch_Truncation(t *testing.T) {
	withPrivateWebFetchHostsAllowed(t)

	longContent := strings.Repeat("x", 20000)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(longContent))
	}))
	defer server.Close()

	tool, err := NewWebFetchTool(1000, testFetchLimit) // Limit to 1000 chars
	if err != nil {
		logger.ErrorCF("agent", "Failed to create web fetch tool", map[string]any{"error": err.Error()})
	}

	ctx := context.Background()
	args := map[string]any{
		"url": server.URL,
	}

	result := tool.Execute(ctx, args)

	// Success should not be an error
	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	// ForLLM should contain truncated content (not the full 20000 chars)
	resultMap := make(map[string]any)
	json.Unmarshal([]byte(result.ForLLM), &resultMap)
	if text, ok := resultMap["text"].(string); ok {
		if len(text) > 1100 { // Allow some margin
			t.Errorf("Expected content to be truncated to ~1000 chars, got: %d", len(text))
		}
	}

	// Should be marked as truncated
	if truncated, ok := resultMap["truncated"].(bool); !ok || !truncated {
		t.Errorf("Expected 'truncated' to be true in result")
	}
}

func TestWebFetchTool_PayloadTooLarge(t *testing.T) {
	withPrivateWebFetchHostsAllowed(t)

	// Create a mock HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		// Generate a payload intentionally larger than our limit.
		// Limit: 10 * 1024 * 1024 (10MB). We generate 10MB + 100 bytes of the letter 'A'.
		largeData := bytes.Repeat([]byte("A"), int(testFetchLimit)+100)

		w.Write(largeData)
	}))
	// Ensure the server is shut down at the end of the test
	defer ts.Close()

	// Initialize the tool
	tool, err := NewWebFetchTool(50000, testFetchLimit)
	if err != nil {
		logger.ErrorCF("agent", "Failed to create web fetch tool", map[string]any{"error": err.Error()})
	}

	// Prepare the arguments pointing to the URL of our local mock server
	args := map[string]any{
		"url": ts.URL,
	}

	// Execute the tool
	ctx := context.Background()
	result := tool.Execute(ctx, args)

	// Assuming ErrorResult sets the ForLLM field with the error text.
	if result == nil {
		t.Fatal("expected a ToolResult, got nil")
	}

	// Search for the exact error string we set earlier in the Execute method
	expectedErrorMsg := fmt.Sprintf("size exceeded %d bytes limit", testFetchLimit)

	if !strings.Contains(result.ForLLM, expectedErrorMsg) && !strings.Contains(result.ForUser, expectedErrorMsg) {
		t.Errorf("test failed: expected error %q, but got: %+v", expectedErrorMsg, result)
	}
}

// TestWebTool_WebSearch_NoApiKey verifies that no tool is created when API key is missing
func TestWebTool_WebSearch_NoApiKey(t *testing.T) {
	tool, err := NewWebSearchTool(WebSearchToolOptions{BraveEnabled: true, BraveAPIKeys: nil})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if tool != nil {
		t.Errorf("Expected nil tool when Brave API key is empty")
	}

	// Also nil when nothing is enabled
	tool, err = NewWebSearchTool(WebSearchToolOptions{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if tool != nil {
		t.Errorf("Expected nil tool when no provider is enabled")
	}
}

// TestWebTool_WebSearch_MissingQuery verifies error handling for missing query
func TestWebTool_WebSearch_MissingQuery(t *testing.T) {
	tool, err := NewWebSearchTool(WebSearchToolOptions{
		BraveEnabled:    true,
		BraveAPIKeys:    []string{"test-key"},
		BraveMaxResults: 5,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	ctx := context.Background()
	args := map[string]any{}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error when query is missing")
	}
}

// TestWebTool_WebFetch_HTMLExtraction verifies HTML text extraction
func TestWebTool_WebFetch_HTMLExtraction(t *testing.T) {
	withPrivateWebFetchHostsAllowed(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(
			[]byte(
				`<html><body><script>alert('test');</script><style>body{color:red;}</style><h1>Title</h1><p>Content</p></body></html>`,
			),
		)
	}))
	defer server.Close()

	tool, err := NewWebFetchTool(50000, testFetchLimit)
	if err != nil {
		logger.ErrorCF("agent", "Failed to create web fetch tool", map[string]any{"error": err.Error()})
	}

	ctx := context.Background()
	args := map[string]any{
		"url": server.URL,
	}

	result := tool.Execute(ctx, args)

	// Success should not be an error
	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	// ForLLM should contain extracted text (without script/style tags)
	if !strings.Contains(result.ForLLM, "Title") && !strings.Contains(result.ForLLM, "Content") {
		t.Errorf("Expected ForLLM to contain extracted text, got: %s", result.ForLLM)
	}

	// Should NOT contain script or style tags in ForLLM
	if strings.Contains(result.ForLLM, "<script>") || strings.Contains(result.ForLLM, "<style>") {
		t.Errorf("Expected script/style tags to be removed, got: %s", result.ForLLM)
	}
}

// TestWebFetchTool_extractText verifies text extraction preserves newlines
func TestWebFetchTool_extractText(t *testing.T) {
	tool := &WebFetchTool{}

	tests := []struct {
		name     string
		input    string
		wantFunc func(t *testing.T, got string)
	}{
		{
			name:  "preserves newlines between block elements",
			input: "<html><body><h1>Title</h1>\n<p>Paragraph 1</p>\n<p>Paragraph 2</p></body></html>",
			wantFunc: func(t *testing.T, got string) {
				lines := strings.Split(got, "\n")
				if len(lines) < 2 {
					t.Errorf("Expected multiple lines, got %d: %q", len(lines), got)
				}
				if !strings.Contains(got, "Title") || !strings.Contains(got, "Paragraph 1") ||
					!strings.Contains(got, "Paragraph 2") {
					t.Errorf("Missing expected text: %q", got)
				}
			},
		},
		{
			name:  "removes script and style tags",
			input: "<script>alert('x');</script><style>body{}</style><p>Keep this</p>",
			wantFunc: func(t *testing.T, got string) {
				if strings.Contains(got, "alert") || strings.Contains(got, "body{}") {
					t.Errorf("Expected script/style content removed, got: %q", got)
				}
				if !strings.Contains(got, "Keep this") {
					t.Errorf("Expected 'Keep this' to remain, got: %q", got)
				}
			},
		},
		{
			name:  "collapses excessive blank lines",
			input: "<p>A</p>\n\n\n\n\n<p>B</p>",
			wantFunc: func(t *testing.T, got string) {
				if strings.Contains(got, "\n\n\n") {
					t.Errorf("Expected excessive blank lines collapsed, got: %q", got)
				}
			},
		},
		{
			name:  "collapses horizontal whitespace",
			input: "<p>hello     world</p>",
			wantFunc: func(t *testing.T, got string) {
				if strings.Contains(got, "     ") {
					t.Errorf("Expected spaces collapsed, got: %q", got)
				}
				if !strings.Contains(got, "hello world") {
					t.Errorf("Expected 'hello world', got: %q", got)
				}
			},
		},
		{
			name:  "empty input",
			input: "",
			wantFunc: func(t *testing.T, got string) {
				if got != "" {
					t.Errorf("Expected empty string, got: %q", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tool.extractText(tt.input)
			tt.wantFunc(t, got)
		})
	}
}

func withPrivateWebFetchHostsAllowed(t *testing.T) {
	t.Helper()
	previous := allowPrivateWebFetchHosts.Load()
	allowPrivateWebFetchHosts.Store(true)
	t.Cleanup(func() {
		allowPrivateWebFetchHosts.Store(previous)
	})
}

func serverHostAndPort(t *testing.T, rawURL string) (string, string) {
	t.Helper()
	hostPort := strings.TrimPrefix(rawURL, "http://")
	hostPort = strings.TrimPrefix(hostPort, "https://")
	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		t.Fatalf("failed to split host/port from %q: %v", rawURL, err)
	}
	return host, port
}

func singleHostCIDR(t *testing.T, host string) string {
	t.Helper()
	ip := net.ParseIP(host)
	if ip == nil {
		t.Fatalf("failed to parse IP %q", host)
	}
	if ip.To4() != nil {
		return ip.String() + "/32"
	}
	return ip.String() + "/128"
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

func TestWebTool_WebFetch_PrivateHostAllowedByExactWhitelist(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("exact whitelist ok"))
	}))
	defer server.Close()

	host, _ := serverHostAndPort(t, server.URL)
	tool, err := NewWebFetchToolWithConfig(50000, "", testFetchLimit, []string{host})
	if err != nil {
		t.Fatalf("Failed to create web fetch tool: %v", err)
	}

	result := tool.Execute(context.Background(), map[string]any{
		"url": server.URL,
	})
	if result.IsError {
		t.Fatalf("expected success for exact whitelisted private IP, got %q", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "exact whitelist ok") {
		t.Fatalf("expected fetched content, got %q", result.ForLLM)
	}
}

func TestWebTool_WebFetch_PrivateHostAllowedByCIDRWhitelist(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("cidr whitelist ok"))
	}))
	defer server.Close()

	host, _ := serverHostAndPort(t, server.URL)
	tool, err := NewWebFetchToolWithConfig(50000, "", testFetchLimit, []string{singleHostCIDR(t, host)})
	if err != nil {
		t.Fatalf("Failed to create web fetch tool: %v", err)
	}

	result := tool.Execute(context.Background(), map[string]any{
		"url": server.URL,
	})
	if result.IsError {
		t.Fatalf("expected success for CIDR-whitelisted private IP, got %q", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "cidr whitelist ok") {
		t.Fatalf("expected fetched content, got %q", result.ForLLM)
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

// TestWebFetch_BlocksIPv4MappedIPv6Loopback verifies ::ffff:127.0.0.1 is blocked
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

// TestWebFetch_BlocksMetadataIP verifies 169.254.169.254 is blocked
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

// TestWebFetch_BlocksIPv6UniqueLocal verifies fc00::/7 addresses are blocked
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

// TestWebFetch_Blocks6to4WithPrivateEmbed verifies 6to4 with private embedded IPv4 is blocked
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

// TestWebFetch_Allows6to4WithPublicEmbed verifies 6to4 with public embedded IPv4 is NOT blocked
func TestWebFetch_Allows6to4WithPublicEmbed(t *testing.T) {
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
		t.Error("6to4 with public embedded IPv4 should not be blocked as private")
	}
}

// TestWebFetch_RedirectToPrivateBlocked verifies redirects to private IPs are blocked
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

func TestNewSafeDialContext_BlocksPrivateDNSResolutionWithoutWhitelist(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on loopback: %v", err)
	}
	defer listener.Close()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to split listener address: %v", err)
	}

	dialContext := newSafeDialContext(&net.Dialer{Timeout: time.Second}, nil)
	_, err = dialContext(context.Background(), "tcp", net.JoinHostPort("localhost", port))
	if err == nil {
		t.Fatal("expected localhost DNS resolution to be blocked without whitelist")
	}
	if !strings.Contains(err.Error(), "private") && !strings.Contains(err.Error(), "whitelisted") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewSafeDialContext_AllowsWhitelistedPrivateDNSResolution(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on loopback: %v", err)
	}
	defer listener.Close()

	accepted := make(chan struct{}, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		conn.Close()
		accepted <- struct{}{}
	}()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to split listener address: %v", err)
	}

	whitelist, err := newPrivateHostWhitelist([]string{"127.0.0.0/8"})
	if err != nil {
		t.Fatalf("failed to parse whitelist: %v", err)
	}

	dialContext := newSafeDialContext(&net.Dialer{Timeout: time.Second}, whitelist)
	conn, err := dialContext(context.Background(), "tcp", net.JoinHostPort("localhost", port))
	if err != nil {
		t.Fatalf("expected localhost DNS resolution to succeed with whitelist, got %v", err)
	}
	conn.Close()

	select {
	case <-accepted:
	case <-time.After(time.Second):
		t.Fatal("expected localhost listener to accept a connection")
	}
}

// TestIsPrivateOrRestrictedIP_Table tests IP classification logic
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

// TestWebTool_WebFetch_MissingDomain verifies error handling for URL without domain
func TestWebTool_WebFetch_MissingDomain(t *testing.T) {
	tool, err := NewWebFetchTool(50000, testFetchLimit)
	if err != nil {
		logger.ErrorCF("agent", "Failed to create web fetch tool", map[string]any{"error": err.Error()})
	}

	ctx := context.Background()
	args := map[string]any{
		"url": "https://",
	}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error for URL without domain")
	}

	// Should mention missing domain
	if !strings.Contains(result.ForLLM, "domain") && !strings.Contains(result.ForUser, "domain") {
		t.Errorf("Expected domain error message, got ForLLM: %s", result.ForLLM)
	}
}

func TestNewWebFetchToolWithProxy(t *testing.T) {
	tool, err := NewWebFetchToolWithProxy(1024, "http://127.0.0.1:7890", testFetchLimit)
	if err != nil {
		logger.ErrorCF("agent", "Failed to create web fetch tool", map[string]any{"error": err.Error()})
	} else if tool.maxChars != 1024 {
		t.Fatalf("maxChars = %d, want %d", tool.maxChars, 1024)
	}

	if tool.proxy != "http://127.0.0.1:7890" {
		t.Fatalf("proxy = %q, want %q", tool.proxy, "http://127.0.0.1:7890")
	}

	tool, err = NewWebFetchToolWithProxy(0, "http://127.0.0.1:7890", testFetchLimit)
	if err != nil {
		logger.ErrorCF("agent", "Failed to create web fetch tool", map[string]any{"error": err.Error()})
	}

	if tool.maxChars != 50000 {
		t.Fatalf("default maxChars = %d, want %d", tool.maxChars, 50000)
	}
}

func TestNewWebFetchToolWithConfig_InvalidPrivateHostWhitelist(t *testing.T) {
	_, err := NewWebFetchToolWithConfig(1024, "", testFetchLimit, []string{"not-an-ip-or-cidr"})
	if err == nil {
		t.Fatal("expected invalid whitelist entry to fail")
	}
	if !strings.Contains(err.Error(), "invalid entry") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewWebSearchTool_PropagatesProxy(t *testing.T) {
	t.Run("perplexity", func(t *testing.T) {
		tool, err := NewWebSearchTool(WebSearchToolOptions{
			PerplexityEnabled:    true,
			PerplexityAPIKeys:    []string{"k"},
			PerplexityMaxResults: 3,
			Proxy:                "http://127.0.0.1:7890",
		})
		if err != nil {
			t.Fatalf("NewWebSearchTool() error: %v", err)
		}
		p, ok := tool.provider.(*PerplexitySearchProvider)
		if !ok {
			t.Fatalf("provider type = %T, want *PerplexitySearchProvider", tool.provider)
		}
		if p.proxy != "http://127.0.0.1:7890" {
			t.Fatalf("provider proxy = %q, want %q", p.proxy, "http://127.0.0.1:7890")
		}
	})

	t.Run("brave", func(t *testing.T) {
		tool, err := NewWebSearchTool(WebSearchToolOptions{
			BraveEnabled:    true,
			BraveAPIKeys:    []string{"k"},
			BraveMaxResults: 3,
			Proxy:           "http://127.0.0.1:7890",
		})
		if err != nil {
			t.Fatalf("NewWebSearchTool() error: %v", err)
		}
		p, ok := tool.provider.(*BraveSearchProvider)
		if !ok {
			t.Fatalf("provider type = %T, want *BraveSearchProvider", tool.provider)
		}
		if p.proxy != "http://127.0.0.1:7890" {
			t.Fatalf("provider proxy = %q, want %q", p.proxy, "http://127.0.0.1:7890")
		}
	})

	t.Run("duckduckgo", func(t *testing.T) {
		tool, err := NewWebSearchTool(WebSearchToolOptions{
			DuckDuckGoEnabled:    true,
			DuckDuckGoMaxResults: 3,
			Proxy:                "http://127.0.0.1:7890",
		})
		if err != nil {
			t.Fatalf("NewWebSearchTool() error: %v", err)
		}
		p, ok := tool.provider.(*DuckDuckGoSearchProvider)
		if !ok {
			t.Fatalf("provider type = %T, want *DuckDuckGoSearchProvider", tool.provider)
		}
		if p.proxy != "http://127.0.0.1:7890" {
			t.Fatalf("provider proxy = %q, want %q", p.proxy, "http://127.0.0.1:7890")
		}
	})
}

// TestWebTool_TavilySearch_Success verifies successful Tavily search
func TestWebTool_TavilySearch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify payload
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)
		if payload["api_key"] != "test-key" {
			t.Errorf("Expected api_key test-key, got %v", payload["api_key"])
		}
		if payload["query"] != "test query" {
			t.Errorf("Expected query 'test query', got %v", payload["query"])
		}

		// Return mock response
		response := map[string]any{
			"results": []map[string]any{
				{
					"title":   "Test Result 1",
					"url":     "https://example.com/1",
					"content": "Content for result 1",
				},
				{
					"title":   "Test Result 2",
					"url":     "https://example.com/2",
					"content": "Content for result 2",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	tool, err := NewWebSearchTool(WebSearchToolOptions{
		TavilyEnabled:    true,
		TavilyAPIKeys:    []string{"test-key"},
		TavilyBaseURL:    server.URL,
		TavilyMaxResults: 5,
	})
	if err != nil {
		t.Fatalf("NewWebSearchTool() error: %v", err)
	}

	ctx := context.Background()
	args := map[string]any{
		"query": "test query",
	}

	result := tool.Execute(ctx, args)

	// Success should not be an error
	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	// ForUser should contain result titles and URLs
	if !strings.Contains(result.ForUser, "Test Result 1") ||
		!strings.Contains(result.ForUser, "https://example.com/1") {
		t.Errorf("Expected results in output, got: %s", result.ForUser)
	}

	// Should mention via Tavily
	if !strings.Contains(result.ForUser, "via Tavily") {
		t.Errorf("Expected 'via Tavily' in output, got: %s", result.ForUser)
	}
}

func TestAPIKeyPool(t *testing.T) {
	pool := NewAPIKeyPool([]string{"key1", "key2", "key3"})
	if len(pool.keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(pool.keys))
	}
	if pool.keys[0] != "key1" || pool.keys[1] != "key2" || pool.keys[2] != "key3" {
		t.Fatalf("unexpected keys: %v", pool.keys)
	}

	// Test Iterator: each iterator should cover all keys exactly once
	iter := pool.NewIterator()
	expected := []string{"key1", "key2", "key3"}
	for i, want := range expected {
		k, ok := iter.Next()
		if !ok {
			t.Fatalf("iter.Next() returned false at step %d", i)
		}
		if k != want {
			t.Errorf("step %d: expected %s, got %s", i, want, k)
		}
	}
	// Should be exhausted
	if _, ok := iter.Next(); ok {
		t.Errorf("expected iterator exhausted after all keys")
	}

	// Second iterator starts at next position (load balancing)
	iter2 := pool.NewIterator()
	k, ok := iter2.Next()
	if !ok {
		t.Fatal("iter2.Next() returned false")
	}
	if k != "key2" {
		t.Errorf("expected key2 (round-robin), got %s", k)
	}

	// Empty pool
	emptyPool := NewAPIKeyPool([]string{})
	emptyIter := emptyPool.NewIterator()
	if _, ok := emptyIter.Next(); ok {
		t.Errorf("expected false for empty pool")
	}

	// Single key pool
	singlePool := NewAPIKeyPool([]string{"single"})
	singleIter := singlePool.NewIterator()
	if k, ok := singleIter.Next(); !ok || k != "single" {
		t.Errorf("expected single, got %s (ok=%v)", k, ok)
	}
	if _, ok := singleIter.Next(); ok {
		t.Errorf("expected exhausted after single key")
	}
}

func TestWebTool_TavilySearch_Failover(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode payload: %v", err)
		}

		apiKey := payload["api_key"].(string)

		if apiKey == "key1" {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Rate limited"))
			return
		}

		if apiKey == "key2" {
			// Success
			response := map[string]any{
				"results": []map[string]any{
					{
						"title":   "Success Result",
						"url":     "https://example.com/success",
						"content": "Success content",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	tool, err := NewWebSearchTool(WebSearchToolOptions{
		TavilyEnabled:    true,
		TavilyAPIKeys:    []string{"key1", "key2"},
		TavilyBaseURL:    server.URL,
		TavilyMaxResults: 5,
	})
	if err != nil {
		t.Fatalf("NewWebSearchTool() error: %v", err)
	}

	ctx := context.Background()
	args := map[string]any{
		"query": "test query",
	}

	result := tool.Execute(ctx, args)

	if result.IsError {
		t.Errorf("Expected success, got Error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForUser, "Success Result") {
		t.Errorf("Expected failover to second key and success result, got: %s", result.ForUser)
	}
}

func TestWebTool_GLMSearch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Authorization") != "Bearer test-glm-key" {
			t.Errorf("Expected Authorization Bearer test-glm-key, got %s", r.Header.Get("Authorization"))
		}

		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)
		if payload["search_query"] != "test query" {
			t.Errorf("Expected search_query 'test query', got %v", payload["search_query"])
		}
		if payload["search_engine"] != "search_std" {
			t.Errorf("Expected search_engine 'search_std', got %v", payload["search_engine"])
		}

		response := map[string]any{
			"id":      "web-search-test",
			"created": 1709568000,
			"search_result": []map[string]any{
				{
					"title":        "Test GLM Result",
					"content":      "GLM search snippet",
					"link":         "https://example.com/glm",
					"media":        "Example",
					"publish_date": "2026-03-04",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	tool, err := NewWebSearchTool(WebSearchToolOptions{
		GLMSearchEnabled: true,
		GLMSearchAPIKey:  "test-glm-key",
		GLMSearchBaseURL: server.URL,
		GLMSearchEngine:  "search_std",
	})
	if err != nil {
		t.Fatalf("NewWebSearchTool() error: %v", err)
	}

	result := tool.Execute(context.Background(), map[string]any{
		"query": "test query",
	})

	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForUser, "Test GLM Result") {
		t.Errorf("Expected 'Test GLM Result' in output, got: %s", result.ForUser)
	}
	if !strings.Contains(result.ForUser, "https://example.com/glm") {
		t.Errorf("Expected URL in output, got: %s", result.ForUser)
	}
	if !strings.Contains(result.ForUser, "via GLM Search") {
		t.Errorf("Expected 'via GLM Search' in output, got: %s", result.ForUser)
	}
}

func TestWebTool_GLMSearch_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid api key"}`))
	}))
	defer server.Close()

	tool, err := NewWebSearchTool(WebSearchToolOptions{
		GLMSearchEnabled: true,
		GLMSearchAPIKey:  "bad-key",
		GLMSearchBaseURL: server.URL,
		GLMSearchEngine:  "search_std",
	})
	if err != nil {
		t.Fatalf("NewWebSearchTool() error: %v", err)
	}

	result := tool.Execute(context.Background(), map[string]any{
		"query": "test query",
	})

	if !result.IsError {
		t.Errorf("Expected IsError=true for 401 response")
	}
	if !strings.Contains(result.ForLLM, "status 401") {
		t.Errorf("Expected status 401 in error, got: %s", result.ForLLM)
	}
}

func TestWebTool_GLMSearch_Priority(t *testing.T) {
	// GLM Search should only be selected when all other providers are disabled
	tool, err := NewWebSearchTool(WebSearchToolOptions{
		DuckDuckGoEnabled:    true,
		DuckDuckGoMaxResults: 5,
		GLMSearchEnabled:     true,
		GLMSearchAPIKey:      "test-key",
		GLMSearchBaseURL:     "https://example.com",
		GLMSearchEngine:      "search_std",
	})
	if err != nil {
		t.Fatalf("NewWebSearchTool() error: %v", err)
	}

	// DuckDuckGo should win over GLM Search
	if _, ok := tool.provider.(*DuckDuckGoSearchProvider); !ok {
		t.Errorf("Expected DuckDuckGoSearchProvider when both enabled, got %T", tool.provider)
	}

	// With DuckDuckGo disabled, GLM Search should be selected
	tool2, err := NewWebSearchTool(WebSearchToolOptions{
		DuckDuckGoEnabled: false,
		GLMSearchEnabled:  true,
		GLMSearchAPIKey:   "test-key",
		GLMSearchBaseURL:  "https://example.com",
		GLMSearchEngine:   "search_std",
	})
	if err != nil {
		t.Fatalf("NewWebSearchTool() error: %v", err)
	}
	if _, ok := tool2.provider.(*GLMSearchProvider); !ok {
		t.Errorf("Expected GLMSearchProvider when only GLM enabled, got %T", tool2.provider)
	}
}
