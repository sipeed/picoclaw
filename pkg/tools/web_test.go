package tools

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestWebTool_WebFetch_Success verifies successful URL fetching
func TestWebTool_WebFetch_Success(t *testing.T) {
	withPrivateWebFetchHostsAllowed(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><h1>Test Page</h1><p>Content here</p></body></html>"))
	}))
	defer server.Close()

	tool := NewWebFetchTool(50000)
	ctx := context.Background()
	args := map[string]any{
		"url": server.URL,
	}

	result := tool.Execute(ctx, args)

	// Success should not be an error
	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	// ForUser should contain the fetched content
	if !strings.Contains(result.ForUser, "Test Page") {
		t.Errorf("Expected ForUser to contain 'Test Page', got: %s", result.ForUser)
	}

	// ForLLM should contain summary
	if !strings.Contains(result.ForLLM, "bytes") && !strings.Contains(result.ForLLM, "extractor") {
		t.Errorf("Expected ForLLM to contain summary, got: %s", result.ForLLM)
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

	tool := NewWebFetchTool(50000)
	ctx := context.Background()
	args := map[string]any{
		"url": server.URL,
	}

	result := tool.Execute(ctx, args)

	// Success should not be an error
	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	// ForUser should contain formatted JSON
	if !strings.Contains(result.ForUser, "key") && !strings.Contains(result.ForUser, "value") {
		t.Errorf("Expected ForUser to contain JSON data, got: %s", result.ForUser)
	}
}

// TestWebTool_WebFetch_InvalidURL verifies error handling for invalid URL
func TestWebTool_WebFetch_InvalidURL(t *testing.T) {
	tool := NewWebFetchTool(50000)
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
	tool := NewWebFetchTool(50000)
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
	tool := NewWebFetchTool(50000)
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

	tool := NewWebFetchTool(1000) // Limit to 1000 chars
	ctx := context.Background()
	args := map[string]any{
		"url": server.URL,
	}

	result := tool.Execute(ctx, args)

	// Success should not be an error
	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	// ForUser should contain truncated content (not the full 20000 chars)
	resultMap := make(map[string]any)
	json.Unmarshal([]byte(result.ForUser), &resultMap)
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

// TestWebTool_WebSearch_NoApiKey verifies that no tool is created when API key is missing
func TestWebTool_WebSearch_NoApiKey(t *testing.T) {
	tool := NewWebSearchTool(WebSearchToolOptions{BraveEnabled: true, BraveAPIKey: ""})
	if tool != nil {
		t.Errorf("Expected nil tool when Brave API key is empty")
	}

	// Also nil when nothing is enabled
	tool = NewWebSearchTool(WebSearchToolOptions{})
	if tool != nil {
		t.Errorf("Expected nil tool when no provider is enabled")
	}
}

// TestWebTool_WebSearch_MissingQuery verifies error handling for missing query
func TestWebTool_WebSearch_MissingQuery(t *testing.T) {
	tool := NewWebSearchTool(WebSearchToolOptions{BraveEnabled: true, BraveAPIKey: "test-key", BraveMaxResults: 5})
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

	tool := NewWebFetchTool(50000)
	ctx := context.Background()
	args := map[string]any{
		"url": server.URL,
	}

	result := tool.Execute(ctx, args)

	// Success should not be an error
	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	// ForUser should contain extracted text (without script/style tags)
	if !strings.Contains(result.ForUser, "Title") && !strings.Contains(result.ForUser, "Content") {
		t.Errorf("Expected ForUser to contain extracted text, got: %s", result.ForUser)
	}

	// Should NOT contain script or style tags
	if strings.Contains(result.ForUser, "<script>") || strings.Contains(result.ForUser, "<style>") {
		t.Errorf("Expected script/style tags to be removed, got: %s", result.ForUser)
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

func TestWebTool_WebFetch_PrivateHostBlocked(t *testing.T) {
	tool := NewWebFetchTool(50000)
	result := tool.Execute(context.Background(), map[string]any{
		"url": "http://127.0.0.1:0",
	})

	if !result.IsError {
		t.Errorf("expected error for private host URL, got success")
	}
	if !strings.Contains(result.ForLLM, "private or local network") && !strings.Contains(result.ForUser, "private or local network") {
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

	tool := NewWebFetchTool(50000)
	result := tool.Execute(context.Background(), map[string]any{
		"url": server.URL,
	})

	if result.IsError {
		t.Errorf("expected success when private host access is allowed in tests, got %q", result.ForLLM)
	}
}

// TestWebFetch_BlocksIPv4MappedIPv6Loopback verifies ::ffff:127.0.0.1 is blocked
func TestWebFetch_BlocksIPv4MappedIPv6Loopback(t *testing.T) {
	tool := NewWebFetchTool(50000)
	result := tool.Execute(context.Background(), map[string]any{
		"url": "http://[::ffff:127.0.0.1]:0",
	})

	if !result.IsError {
		t.Error("expected error for IPv4-mapped IPv6 loopback URL, got success")
	}
}

// TestWebFetch_BlocksMetadataIP verifies 169.254.169.254 is blocked
func TestWebFetch_BlocksMetadataIP(t *testing.T) {
	tool := NewWebFetchTool(50000)
	result := tool.Execute(context.Background(), map[string]any{
		"url": "http://169.254.169.254/latest/meta-data",
	})

	if !result.IsError {
		t.Error("expected error for cloud metadata IP, got success")
	}
}

// TestWebFetch_BlocksIPv6UniqueLocal verifies fc00::/7 addresses are blocked
func TestWebFetch_BlocksIPv6UniqueLocal(t *testing.T) {
	tool := NewWebFetchTool(50000)
	result := tool.Execute(context.Background(), map[string]any{
		"url": "http://[fd00::1]:0",
	})

	if !result.IsError {
		t.Error("expected error for IPv6 unique local address, got success")
	}
}

// TestWebFetch_Blocks6to4 verifies 2002::/16 addresses are blocked (may embed private IPv4)
func TestWebFetch_Blocks6to4(t *testing.T) {
	tool := NewWebFetchTool(50000)
	result := tool.Execute(context.Background(), map[string]any{
		"url": "http://[2002:7f00:0001::1]:0",
	})

	if !result.IsError {
		t.Error("expected error for 6to4 address, got success")
	}
}

// TestWebFetch_BlocksTeredo verifies 2001:0000::/32 addresses are blocked
func TestWebFetch_BlocksTeredo(t *testing.T) {
	tool := NewWebFetchTool(50000)
	result := tool.Execute(context.Background(), map[string]any{
		"url": "http://[2001:0000::1]:0",
	})

	if !result.IsError {
		t.Error("expected error for Teredo tunnel address, got success")
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

	tool := NewWebFetchTool(50000)
	result := tool.Execute(context.Background(), map[string]any{
		"url": server.URL,
	})

	if !result.IsError {
		t.Error("expected error when redirecting to private IP, got success")
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
		{"2002:7f00:0001::1", true, "6to4 with embedded 127.x"},
		{"2001:0000::1", true, "Teredo"},
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
	tool := NewWebFetchTool(50000)
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

func TestCreateHTTPClient_ProxyConfigured(t *testing.T) {
	client, err := createHTTPClient("http://127.0.0.1:7890", 12*time.Second)
	if err != nil {
		t.Fatalf("createHTTPClient() error: %v", err)
	}
	if client.Timeout != 12*time.Second {
		t.Fatalf("client.Timeout = %v, want %v", client.Timeout, 12*time.Second)
	}

	tr, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport type = %T, want *http.Transport", client.Transport)
	}
	if tr.Proxy == nil {
		t.Fatal("transport.Proxy is nil, want non-nil")
	}

	req, err := http.NewRequest("GET", "https://example.com", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error: %v", err)
	}
	proxyURL, err := tr.Proxy(req)
	if err != nil {
		t.Fatalf("transport.Proxy(req) error: %v", err)
	}
	if proxyURL == nil || proxyURL.String() != "http://127.0.0.1:7890" {
		t.Fatalf("proxy URL = %v, want %q", proxyURL, "http://127.0.0.1:7890")
	}
}

func TestCreateHTTPClient_InvalidProxy(t *testing.T) {
	_, err := createHTTPClient("://bad-proxy", 10*time.Second)
	if err == nil {
		t.Fatal("createHTTPClient() expected error for invalid proxy URL, got nil")
	}
}

func TestCreateHTTPClient_Socks5ProxyConfigured(t *testing.T) {
	client, err := createHTTPClient("socks5://127.0.0.1:1080", 8*time.Second)
	if err != nil {
		t.Fatalf("createHTTPClient() error: %v", err)
	}

	tr, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport type = %T, want *http.Transport", client.Transport)
	}
	req, err := http.NewRequest("GET", "https://example.com", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error: %v", err)
	}
	proxyURL, err := tr.Proxy(req)
	if err != nil {
		t.Fatalf("transport.Proxy(req) error: %v", err)
	}
	if proxyURL == nil || proxyURL.String() != "socks5://127.0.0.1:1080" {
		t.Fatalf("proxy URL = %v, want %q", proxyURL, "socks5://127.0.0.1:1080")
	}
}

func TestCreateHTTPClient_UnsupportedProxyScheme(t *testing.T) {
	_, err := createHTTPClient("ftp://127.0.0.1:21", 10*time.Second)
	if err == nil {
		t.Fatal("createHTTPClient() expected error for unsupported scheme, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported proxy scheme") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "unsupported proxy scheme")
	}
}

func TestCreateHTTPClient_ProxyFromEnvironmentWhenConfigEmpty(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:8888")
	t.Setenv("http_proxy", "http://127.0.0.1:8888")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:8888")
	t.Setenv("https_proxy", "http://127.0.0.1:8888")
	t.Setenv("ALL_PROXY", "")
	t.Setenv("all_proxy", "")
	t.Setenv("NO_PROXY", "")
	t.Setenv("no_proxy", "")

	client, err := createHTTPClient("", 10*time.Second)
	if err != nil {
		t.Fatalf("createHTTPClient() error: %v", err)
	}

	tr, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport type = %T, want *http.Transport", client.Transport)
	}
	if tr.Proxy == nil {
		t.Fatal("transport.Proxy is nil, want proxy function from environment")
	}

	req, err := http.NewRequest("GET", "https://example.com", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error: %v", err)
	}
	if _, err := tr.Proxy(req); err != nil {
		t.Fatalf("transport.Proxy(req) error: %v", err)
	}
}

func TestNewWebFetchToolWithProxy(t *testing.T) {
	tool := NewWebFetchToolWithProxy(1024, "http://127.0.0.1:7890")
	if tool.maxChars != 1024 {
		t.Fatalf("maxChars = %d, want %d", tool.maxChars, 1024)
	}
	if tool.proxy != "http://127.0.0.1:7890" {
		t.Fatalf("proxy = %q, want %q", tool.proxy, "http://127.0.0.1:7890")
	}

	tool = NewWebFetchToolWithProxy(0, "http://127.0.0.1:7890")
	if tool.maxChars != 50000 {
		t.Fatalf("default maxChars = %d, want %d", tool.maxChars, 50000)
	}
}

func TestNewWebSearchTool_PropagatesProxy(t *testing.T) {
	t.Run("perplexity", func(t *testing.T) {
		tool := NewWebSearchTool(WebSearchToolOptions{
			PerplexityEnabled:    true,
			PerplexityAPIKey:     "k",
			PerplexityMaxResults: 3,
			Proxy:                "http://127.0.0.1:7890",
		})
		p, ok := tool.provider.(*PerplexitySearchProvider)
		if !ok {
			t.Fatalf("provider type = %T, want *PerplexitySearchProvider", tool.provider)
		}
		if p.proxy != "http://127.0.0.1:7890" {
			t.Fatalf("provider proxy = %q, want %q", p.proxy, "http://127.0.0.1:7890")
		}
	})

	t.Run("brave", func(t *testing.T) {
		tool := NewWebSearchTool(WebSearchToolOptions{
			BraveEnabled:    true,
			BraveAPIKey:     "k",
			BraveMaxResults: 3,
			Proxy:           "http://127.0.0.1:7890",
		})
		p, ok := tool.provider.(*BraveSearchProvider)
		if !ok {
			t.Fatalf("provider type = %T, want *BraveSearchProvider", tool.provider)
		}
		if p.proxy != "http://127.0.0.1:7890" {
			t.Fatalf("provider proxy = %q, want %q", p.proxy, "http://127.0.0.1:7890")
		}
	})

	t.Run("duckduckgo", func(t *testing.T) {
		tool := NewWebSearchTool(WebSearchToolOptions{
			DuckDuckGoEnabled:    true,
			DuckDuckGoMaxResults: 3,
			Proxy:                "http://127.0.0.1:7890",
		})
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

	tool := NewWebSearchTool(WebSearchToolOptions{
		TavilyEnabled:    true,
		TavilyAPIKey:     "test-key",
		TavilyBaseURL:    server.URL,
		TavilyMaxResults: 5,
	})

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
