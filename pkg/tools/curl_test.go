package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestCurlTool_Name(t *testing.T) {
	tool, err := NewCurlTool(CurlToolOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.Name() != "curl" {
		t.Errorf("expected name 'curl', got %q", tool.Name())
	}
}

func TestCurlTool_Description(t *testing.T) {
	tool, err := NewCurlTool(CurlToolOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.Description() == "" {
		t.Error("description should not be empty")
	}
}

func TestCurlTool_Parameters(t *testing.T) {
	tool, err := NewCurlTool(CurlToolOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	params := tool.Parameters()
	if params["type"] != "object" {
		t.Errorf("expected type 'object', got %v", params["type"])
	}
	props := params["properties"].(map[string]any)
	if _, ok := props["url"]; !ok {
		t.Error("parameters should include 'url'")
	}
	if _, ok := props["method"]; !ok {
		t.Error("parameters should include 'method'")
	}
	required := params["required"].([]string)
	found := false
	for _, r := range required {
		if r == "url" {
			found = true
			break
		}
	}
	if !found {
		t.Error("'url' should be required")
	}
}

func TestCurlTool_Execute_MissingURL(t *testing.T) {
	tool, err := NewCurlTool(CurlToolOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result := tool.Execute(context.Background(), map[string]any{})
	if !result.IsError {
		t.Error("expected error for missing url")
	}
}

func TestCurlTool_Execute_InvalidURL(t *testing.T) {
	tool, err := NewCurlTool(CurlToolOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result := tool.Execute(context.Background(), map[string]any{"url": "://invalid"})
	if !result.IsError {
		t.Error("expected error for invalid URL")
	}
}

func TestCurlTool_Execute_NonHTTPScheme(t *testing.T) {
	tool, err := NewCurlTool(CurlToolOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result := tool.Execute(context.Background(), map[string]any{"url": "ftp://example.com/file"})
	if !result.IsError {
		t.Error("expected error for non-http scheme")
	}
}

func TestCurlTool_Execute_DomainNotAllowed(t *testing.T) {
	tool, err := NewCurlTool(CurlToolOptions{
		AllowedDomains: []string{"api.example.com"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result := tool.Execute(context.Background(), map[string]any{"url": "https://evil.com/data"})
	if !result.IsError {
		t.Error("expected error for disallowed domain")
	}
}

func TestCurlTool_Execute_GET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	tool, err := NewCurlTool(CurlToolOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result := tool.Execute(context.Background(), map[string]any{"url": server.URL})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if !containsStr(result.ForLLM, `"status": "ok"`) && !containsStr(result.ForLLM, `status`) {
		t.Errorf("expected response body, got: %s", result.ForLLM)
	}
}

func TestCurlTool_Execute_POST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("created"))
	}))
	defer server.Close()

	tool, err := NewCurlTool(CurlToolOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result := tool.Execute(context.Background(), map[string]any{
		"url":    server.URL,
		"method": "POST",
		"body":   `{"key":"value"}`,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if !containsStr(result.ForLLM, "201") {
		t.Errorf("expected status 201, got: %s", result.ForLLM)
	}
}

func TestCurlTool_Execute_CustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("expected Authorization header 'Bearer test-token', got %q", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tool, err := NewCurlTool(CurlToolOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result := tool.Execute(context.Background(), map[string]any{
		"url":     server.URL,
		"method":  "GET",
		"headers": map[string]any{"Authorization": "Bearer test-token"},
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
}

func TestCurlTool_Execute_DomainWhitelistSubdomain(t *testing.T) {
	tool, err := NewCurlTool(CurlToolOptions{
		AllowedDomains: []string{"example.com"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	parsed := mustParseURL("https://api.example.com/v1")
	if !isDomainAllowed(parsed.Hostname(), tool.allowedDomains) {
		t.Error("subdomain should be allowed when parent domain is in whitelist")
	}
}

func TestCurlTool_NormalizeDomains(t *testing.T) {
	domains := normalizeDomains([]string{
		"HTTPS://Example.COM/",
		"http://test.com",
		"  api.io  ",
		"",
		"Example.COM",
	})
	if len(domains) != 3 {
		t.Errorf("expected 3 domains, got %d: %v", len(domains), domains)
	}
	expected := []string{"example.com", "test.com", "api.io"}
	for i, d := range expected {
		if domains[i] != d {
			t.Errorf("expected domain[%d] = %q, got %q", i, d, domains[i])
		}
	}
}

func TestCurlTool_EmptyAllowedDomains(t *testing.T) {
	if !isDomainAllowed("any.com", []string{}) {
		t.Error("should allow any domain when whitelist is empty")
	}
	if !isDomainAllowed("any.com", nil) {
		t.Error("should allow any domain when whitelist is nil")
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}
