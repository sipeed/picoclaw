// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewCloudflareProvider_RequiresAPIBase(t *testing.T) {
	_, err := NewCloudflareProvider("key", "", "cf-token", "", "", 0)
	if err == nil {
		t.Fatal("expected error for empty api_base")
	}
	if !strings.Contains(err.Error(), "api_base is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewCloudflareProvider_UnifiedBilling(t *testing.T) {
	// Unified Billing mode: cf_token only, no api_key
	provider, err := NewCloudflareProvider(
		"",
		"https://gateway.ai.cloudflare.com/v1/acct/gw/compat",
		"cf-test-token",
		"", "", 0,
	)
	if err != nil {
		t.Fatalf("NewCloudflareProvider() error = %v", err)
	}
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNewCloudflareProvider_BYOK(t *testing.T) {
	// BYOK mode: api_key + cf_token
	provider, err := NewCloudflareProvider(
		"sk-upstream-key",
		"https://gateway.ai.cloudflare.com/v1/acct/gw/compat",
		"cf-test-token",
		"", "", 0,
	)
	if err != nil {
		t.Fatalf("NewCloudflareProvider() error = %v", err)
	}
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestCloudflareProvider_Chat_SendsCfAigAuthHeader(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message":       map[string]any{"content": "Hello from Cloudflare!"},
					"finish_reason": "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, err := NewCloudflareProvider("", server.URL, "cf-my-token", "", "", 0)
	if err != nil {
		t.Fatalf("NewCloudflareProvider() error = %v", err)
	}

	result, err := provider.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"openai/gpt-5.2",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	// Verify cf-aig-authorization header was sent
	cfAuth := receivedHeaders.Get(CfAIGAuthHeader)
	if cfAuth != "Bearer cf-my-token" {
		t.Errorf("cf-aig-authorization header = %q, want %q", cfAuth, "Bearer cf-my-token")
	}

	// Verify no Authorization header is set when api_key is empty
	authHeader := receivedHeaders.Get("Authorization")
	if authHeader != "" {
		t.Errorf("Authorization header should be empty for Unified Billing, got %q", authHeader)
	}

	if result.Content != "Hello from Cloudflare!" {
		t.Errorf("Content = %q, want %q", result.Content, "Hello from Cloudflare!")
	}
}

func TestCloudflareProvider_Chat_BYOK_SendsBothHeaders(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message":       map[string]any{"content": "ok"},
					"finish_reason": "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, err := NewCloudflareProvider(
		"sk-upstream-key",
		server.URL,
		"cf-my-token",
		"", "", 0,
	)
	if err != nil {
		t.Fatalf("NewCloudflareProvider() error = %v", err)
	}

	_, err = provider.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"anthropic/claude-sonnet-4.6",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	// Verify both headers are sent for BYOK mode
	cfAuth := receivedHeaders.Get(CfAIGAuthHeader)
	if cfAuth != "Bearer cf-my-token" {
		t.Errorf("cf-aig-authorization = %q, want %q", cfAuth, "Bearer cf-my-token")
	}

	authHeader := receivedHeaders.Get("Authorization")
	if authHeader != "Bearer sk-upstream-key" {
		t.Errorf("Authorization = %q, want %q", authHeader, "Bearer sk-upstream-key")
	}
}

func TestCloudflareProvider_Chat_ModelPassthrough(t *testing.T) {
	// Verify the model string (e.g., "openai/gpt-5.2") is sent as-is in the request body
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message":       map[string]any{"content": "ok"},
					"finish_reason": "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, err := NewCloudflareProvider("", server.URL, "cf-token", "", "", 0)
	if err != nil {
		t.Fatalf("NewCloudflareProvider() error = %v", err)
	}

	_, err = provider.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"openai/gpt-5.2",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	// Cloudflare expects the full provider/model string
	if requestBody["model"] != "openai/gpt-5.2" {
		t.Errorf("model = %v, want %q", requestBody["model"], "openai/gpt-5.2")
	}
}

func TestCloudflareProvider_Chat_WorkersAI(t *testing.T) {
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message":       map[string]any{"content": "Workers AI response"},
					"finish_reason": "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, err := NewCloudflareProvider("", server.URL, "cf-token", "", "", 0)
	if err != nil {
		t.Fatalf("NewCloudflareProvider() error = %v", err)
	}

	_, err = provider.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"workers-ai/@cf/meta/llama-3.3-70b-instruct-fp8-fast",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	// Workers AI model should be passed through as-is
	if requestBody["model"] != "workers-ai/@cf/meta/llama-3.3-70b-instruct-fp8-fast" {
		t.Errorf("model = %v, want %q", requestBody["model"], "workers-ai/@cf/meta/llama-3.3-70b-instruct-fp8-fast")
	}
}

func TestCloudflareProvider_Chat_NoCfToken(t *testing.T) {
	// When cf_token is empty, no cf-aig-authorization header should be sent
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message":       map[string]any{"content": "ok"},
					"finish_reason": "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, err := NewCloudflareProvider("sk-my-key", server.URL, "", "", "", 0)
	if err != nil {
		t.Fatalf("NewCloudflareProvider() error = %v", err)
	}

	_, err = provider.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"openai/gpt-5.2",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	// No cf-aig-authorization header when cf_token is empty
	cfAuth := receivedHeaders.Get(CfAIGAuthHeader)
	if cfAuth != "" {
		t.Errorf("cf-aig-authorization should be empty, got %q", cfAuth)
	}

	// Authorization header should still be set with the api_key
	authHeader := receivedHeaders.Get("Authorization")
	if authHeader != "Bearer sk-my-key" {
		t.Errorf("Authorization = %q, want %q", authHeader, "Bearer sk-my-key")
	}
}

func TestCloudflareProvider_Chat_ToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": "",
						"tool_calls": []map[string]any{
							{
								"id":   "call_cf1",
								"type": "function",
								"function": map[string]any{
									"name":      "get_weather",
									"arguments": `{"city":"London"}`,
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, err := NewCloudflareProvider("", server.URL, "cf-token", "", "", 0)
	if err != nil {
		t.Fatalf("NewCloudflareProvider() error = %v", err)
	}

	result, err := provider.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "What's the weather?"}},
		nil,
		"openai/gpt-5.2",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if len(result.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(result.ToolCalls))
	}
	if result.ToolCalls[0].Name != "get_weather" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", result.ToolCalls[0].Name, "get_weather")
	}
	if result.ToolCalls[0].Arguments["city"] != "London" {
		t.Errorf("ToolCalls[0].Arguments[city] = %v, want London", result.ToolCalls[0].Arguments["city"])
	}
	if result.Usage == nil {
		t.Fatal("expected non-nil usage")
	}
	if result.Usage.TotalTokens != 15 {
		t.Errorf("Usage.TotalTokens = %d, want 15", result.Usage.TotalTokens)
	}
}

func TestCloudflareProvider_Chat_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"rate limited"}`, http.StatusTooManyRequests)
	}))
	defer server.Close()

	provider, err := NewCloudflareProvider("", server.URL, "cf-token", "", "", 0)
	if err != nil {
		t.Fatalf("NewCloudflareProvider() error = %v", err)
	}

	_, err = provider.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"openai/gpt-5.2",
		nil,
	)
	if err == nil {
		t.Fatal("expected error for HTTP 429")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error should contain status code 429, got: %v", err)
	}
}

func TestCloudflareProvider_GetDefaultModel(t *testing.T) {
	provider, err := NewCloudflareProvider("", "https://example.com", "token", "", "", 0)
	if err != nil {
		t.Fatalf("NewCloudflareProvider() error = %v", err)
	}
	if model := provider.GetDefaultModel(); model != "" {
		t.Errorf("GetDefaultModel() = %q, want empty string", model)
	}
}
