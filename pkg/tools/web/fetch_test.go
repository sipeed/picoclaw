package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jane/pkg/logger"
)

const testFetchLimit = int64(10 * 1024 * 1024)

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

	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForLLM, "Test Page") {
		t.Errorf("Expected ForLLM to contain 'Test Page', got: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForUser, "bytes") && !strings.Contains(result.ForUser, "extractor") {
		t.Errorf("Expected ForUser to contain summary, got: %s", result.ForUser)
	}
}

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

	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForLLM, "key") && !strings.Contains(result.ForLLM, "value") {
		t.Errorf("Expected ForLLM to contain JSON data, got: %s", result.ForLLM)
	}
}

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

	if !result.IsError {
		t.Errorf("Expected error for invalid URL")
	}

	if !strings.Contains(result.ForLLM, "URL") && !strings.Contains(result.ForUser, "URL") {
		t.Errorf("Expected error message for invalid URL, got ForLLM: %s", result.ForLLM)
	}
}

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

	if !result.IsError {
		t.Errorf("Expected error for unsupported URL scheme")
	}

	if !strings.Contains(result.ForLLM, "http/https") && !strings.Contains(result.ForUser, "http/https") {
		t.Errorf("Expected scheme error message, got ForLLM: %s", result.ForLLM)
	}
}

func TestWebTool_WebFetch_MissingURL(t *testing.T) {
	tool, err := NewWebFetchTool(50000, testFetchLimit)
	if err != nil {
		logger.ErrorCF("agent", "Failed to create web fetch tool", map[string]any{"error": err.Error()})
	}

	ctx := context.Background()
	args := map[string]any{}

	result := tool.Execute(ctx, args)

	if !result.IsError {
		t.Errorf("Expected error when URL is missing")
	}

	if !strings.Contains(result.ForLLM, "url is required") && !strings.Contains(result.ForUser, "url is required") {
		t.Errorf("Expected 'url is required' message, got ForLLM: %s", result.ForLLM)
	}
}

func TestWebTool_WebFetch_Truncation(t *testing.T) {
	withPrivateWebFetchHostsAllowed(t)

	longContent := strings.Repeat("x", 20000)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(longContent))
	}))
	defer server.Close()

	tool, err := NewWebFetchTool(1000, testFetchLimit)
	if err != nil {
		logger.ErrorCF("agent", "Failed to create web fetch tool", map[string]any{"error": err.Error()})
	}

	ctx := context.Background()
	args := map[string]any{
		"url": server.URL,
	}

	result := tool.Execute(ctx, args)

	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	resultMap := make(map[string]any)
	json.Unmarshal([]byte(result.ForLLM), &resultMap)
	if text, ok := resultMap["text"].(string); ok {
		if len(text) > 1100 {
			t.Errorf("Expected content to be truncated to ~1000 chars, got: %d", len(text))
		}
	}

	if truncated, ok := resultMap["truncated"].(bool); !ok || !truncated {
		t.Errorf("Expected 'truncated' to be true in result")
	}
}

func TestWebFetchTool_PayloadTooLarge(t *testing.T) {
	withPrivateWebFetchHostsAllowed(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		largeData := bytes.Repeat([]byte("A"), int(testFetchLimit)+100)
		w.Write(largeData)
	}))
	defer ts.Close()

	tool, err := NewWebFetchTool(50000, testFetchLimit)
	if err != nil {
		logger.ErrorCF("agent", "Failed to create web fetch tool", map[string]any{"error": err.Error()})
	}

	args := map[string]any{
		"url": ts.URL,
	}

	ctx := context.Background()
	result := tool.Execute(ctx, args)

	if result == nil {
		t.Fatal("expected a tools.ToolResult, got nil")
	}

	expectedErrorMsg := fmt.Sprintf("size exceeded %d bytes limit", testFetchLimit)

	if !strings.Contains(result.ForLLM, expectedErrorMsg) && !strings.Contains(result.ForUser, expectedErrorMsg) {
		t.Errorf("test failed: expected error %q, but got: %+v", expectedErrorMsg, result)
	}
}

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

	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForLLM, "Title") && !strings.Contains(result.ForLLM, "Content") {
		t.Errorf("Expected ForLLM to contain extracted text, got: %s", result.ForLLM)
	}

	if strings.Contains(result.ForLLM, "<script>") || strings.Contains(result.ForLLM, "<style>") {
		t.Errorf("Expected script/style tags to be removed, got: %s", result.ForLLM)
	}
}

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

	if !result.IsError {
		t.Errorf("Expected error for URL without domain")
	}

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
