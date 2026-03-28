package azure

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

// writeValidResponse writes a minimal valid Responses API response.
func writeValidResponse(w http.ResponseWriter) {
	resp := map[string]any{
		"id":     "resp_test",
		"object": "response",
		"status": "completed",
		"output": []map[string]any{
			{
				"type": "message",
				"content": []map[string]any{
					{"type": "output_text", "text": "ok"},
				},
			},
		},
		"usage": map[string]any{
			"input_tokens":          5,
			"output_tokens":         2,
			"total_tokens":          7,
			"input_tokens_details":  map[string]any{"cached_tokens": 0},
			"output_tokens_details": map[string]any{"reasoning_tokens": 0},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func TestProviderChat_AzureURLConstruction(t *testing.T) {
	var capturedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		writeValidResponse(w)
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "my-gpt5-deployment", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	wantPath := "/openai/v1/responses"
	if capturedPath != wantPath {
		t.Errorf("URL path = %q, want %q", capturedPath, wantPath)
	}
}

func TestProviderChat_AzureAuthHeader(t *testing.T) {
	var capturedAuth string
	var capturedAPIKey string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedAPIKey = r.Header.Get("Api-Key")
		writeValidResponse(w)
	}))
	defer server.Close()

	p := NewProvider("test-azure-key", server.URL, "")
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "deployment", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if capturedAuth != "Bearer test-azure-key" {
		t.Errorf("Authorization header = %q, want %q", capturedAuth, "Bearer test-azure-key")
	}
	if capturedAPIKey != "" {
		t.Errorf("Api-Key header should be empty, got %q", capturedAPIKey)
	}
}

func TestProviderChat_AzureRequestBodyContainsModel(t *testing.T) {
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&requestBody)
		writeValidResponse(w)
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "my-deployment", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if requestBody["model"] != "my-deployment" {
		t.Errorf("model = %v, want %q", requestBody["model"], "my-deployment")
	}
}

func TestProviderChat_AzureUsesMaxOutputTokens(t *testing.T) {
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&requestBody)
		writeValidResponse(w)
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"deployment",
		map[string]any{"max_tokens": 2048},
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if requestBody["max_output_tokens"] == nil {
		t.Error("request body should contain 'max_output_tokens'")
	}
	if _, exists := requestBody["max_tokens"]; exists {
		t.Error("request body should not contain 'max_tokens'")
	}
	if _, exists := requestBody["max_completion_tokens"]; exists {
		t.Error("request body should not contain 'max_completion_tokens'")
	}
}

func TestProviderChat_AzureStoreIsFalse(t *testing.T) {
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&requestBody)
		writeValidResponse(w)
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "deployment", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if requestBody["store"] != false {
		t.Errorf("store = %v, want false", requestBody["store"])
	}
}

func TestProviderChat_AzureHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
	}))
	defer server.Close()

	p := NewProvider("bad-key", server.URL, "")
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "deployment", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestProviderChat_AzureParseTextOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"id":     "resp_1",
			"object": "response",
			"status": "completed",
			"output": []map[string]any{
				{
					"type": "message",
					"content": []map[string]any{
						{"type": "output_text", "text": "Hello there!"},
					},
				},
			},
			"usage": map[string]any{
				"input_tokens": 10, "output_tokens": 5, "total_tokens": 15,
				"input_tokens_details":  map[string]any{"cached_tokens": 0},
				"output_tokens_details": map[string]any{"reasoning_tokens": 0},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	out, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "deployment", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if out.Content != "Hello there!" {
		t.Errorf("Content = %q, want %q", out.Content, "Hello there!")
	}
	if out.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", out.FinishReason, "stop")
	}
	if out.Usage.TotalTokens != 15 {
		t.Errorf("TotalTokens = %d, want 15", out.Usage.TotalTokens)
	}
}

func TestProviderChat_AzureParseToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"id":     "resp_2",
			"object": "response",
			"status": "completed",
			"output": []map[string]any{
				{
					"type":      "function_call",
					"call_id":   "call_1",
					"name":      "get_weather",
					"arguments": `{"city":"Seattle"}`,
				},
			},
			"usage": map[string]any{
				"input_tokens": 10, "output_tokens": 8, "total_tokens": 18,
				"input_tokens_details":  map[string]any{"cached_tokens": 0},
				"output_tokens_details": map[string]any{"reasoning_tokens": 0},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	out, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "weather?"}}, nil, "deployment", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if len(out.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(out.ToolCalls))
	}
	if out.ToolCalls[0].Name != "get_weather" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", out.ToolCalls[0].Name, "get_weather")
	}
	if out.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want %q", out.FinishReason, "tool_calls")
	}
}

func TestProvider_AzureEmptyAPIBase(t *testing.T) {
	p := NewProvider("test-key", "", "")
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "deployment", nil)
	if err == nil {
		t.Fatal("expected error for empty API base")
	}
}

func TestProvider_AzureRequestTimeoutDefault(t *testing.T) {
	p := NewProvider("test-key", "https://example.com", "")
	if p.httpClient.Timeout != defaultRequestTimeout {
		t.Errorf("timeout = %v, want %v", p.httpClient.Timeout, defaultRequestTimeout)
	}
}

func TestProvider_AzureRequestTimeoutOverride(t *testing.T) {
	p := NewProvider("test-key", "https://example.com", "", WithRequestTimeout(300*time.Second))
	if p.httpClient.Timeout != 300*time.Second {
		t.Errorf("timeout = %v, want %v", p.httpClient.Timeout, 300*time.Second)
	}
}

func TestProvider_AzureNewProviderWithTimeout(t *testing.T) {
	p := NewProviderWithTimeout("test-key", "https://example.com", "", 180)
	if p.httpClient.Timeout != 180*time.Second {
		t.Errorf("timeout = %v, want %v", p.httpClient.Timeout, 180*time.Second)
	}
}

func TestProviderChat_AzureNativeWebSearchInjection(t *testing.T) {
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&requestBody)
		writeValidResponse(w)
	}))
	defer server.Close()

	tools := []ToolDefinition{
		{
			Type: "function",
			Function: protocoltypes.ToolFunctionDefinition{
				Name:        "web_search",
				Description: "local web search",
				Parameters:  map[string]any{"type": "object"},
			},
		},
		{
			Type: "function",
			Function: protocoltypes.ToolFunctionDefinition{
				Name:        "read_file",
				Description: "read a file",
				Parameters:  map[string]any{"type": "object"},
			},
		},
	}

	p := NewProvider("test-key", server.URL, "")

	// With native_search=true: user-defined web_search should be replaced by built-in
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, tools, "deployment",
		map[string]any{"native_search": true})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	toolsAny, ok := requestBody["tools"].([]any)
	if !ok {
		t.Fatal("request body should contain 'tools' array")
	}
	if len(toolsAny) != 2 {
		t.Fatalf("len(tools) = %d, want 2 (read_file + web_search builtin)", len(toolsAny))
	}

	// First tool should be read_file (user-defined web_search was skipped)
	firstTool, _ := toolsAny[0].(map[string]any)
	if firstTool["name"] != "read_file" {
		t.Errorf("first tool name = %v, want %q", firstTool["name"], "read_file")
	}

	// Second tool should be built-in web_search
	secondTool, _ := toolsAny[1].(map[string]any)
	if secondTool["type"] != "web_search" {
		t.Errorf("second tool type = %v, want %q", secondTool["type"], "web_search")
	}
}

func TestProviderChat_AzureNoNativeWebSearch(t *testing.T) {
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&requestBody)
		writeValidResponse(w)
	}))
	defer server.Close()

	tools := []ToolDefinition{
		{
			Type: "function",
			Function: protocoltypes.ToolFunctionDefinition{
				Name:        "web_search",
				Description: "local web search",
				Parameters:  map[string]any{"type": "object"},
			},
		},
	}

	p := NewProvider("test-key", server.URL, "")

	// Without native_search: user-defined web_search should be kept as-is
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, tools, "deployment", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	toolsAny, ok := requestBody["tools"].([]any)
	if !ok {
		t.Fatal("request body should contain 'tools' array")
	}
	if len(toolsAny) != 1 {
		t.Fatalf("len(tools) = %d, want 1", len(toolsAny))
	}

	// Should be the user-defined function tool, not built-in
	tool, _ := toolsAny[0].(map[string]any)
	if tool["type"] != "function" {
		t.Errorf("tool type = %v, want %q", tool["type"], "function")
	}
}
