package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

func TestQwenOAuthProvider_GetDefaultModel(t *testing.T) {
	provider := NewQwenOAuthProvider()
	if got := provider.GetDefaultModel(); got != "coder-model" {
		t.Errorf("GetDefaultModel() = %q, want %q", got, "coder-model")
	}
}

func TestQwenOAuthProvider_ChatRoundTrip(t *testing.T) {
	expectedContent := "Hello! I am Qwen. How can I help you?"
	expectedModel := "coder-model"
	expectedPromptTokens := 10
	expectedCompletionTokens := 20
	expectedTotalTokens := 30

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Verify Authorization header
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "missing authorization", http.StatusUnauthorized)
			return
		}

		// Verify content type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			http.Error(w, "invalid content type", http.StatusBadRequest)
			return
		}

		// Parse request body
		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		// Verify model
		if reqBody["model"] != expectedModel {
			http.Error(w, "unexpected model", http.StatusBadRequest)
			return
		}

		// Return mock response
		resp := map[string]any{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   expectedModel,
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": expectedContent,
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     expectedPromptTokens,
				"completion_tokens": expectedCompletionTokens,
				"total_tokens":      expectedTotalTokens,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider with mock token source
	tokenSource := func() (string, error) {
		return "test-token", nil
	}

	provider := NewQwenOAuthProviderWithTokenSource(tokenSource, server.URL+"/v1")

	messages := []protocoltypes.Message{{Role: "user", Content: "Hello"}}
	resp, err := provider.Chat(context.Background(), messages, nil, "qwen-oauth/coder-model", map[string]any{
		"temperature": 0.7,
		"max_tokens":  1024,
	})
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	if resp.Content != expectedContent {
		t.Errorf("Content = %q, want %q", resp.Content, expectedContent)
	}

	if resp.Usage.PromptTokens != expectedPromptTokens {
		t.Errorf("PromptTokens = %d, want %d", resp.Usage.PromptTokens, expectedPromptTokens)
	}

	if resp.Usage.CompletionTokens != expectedCompletionTokens {
		t.Errorf("CompletionTokens = %d, want %d", resp.Usage.CompletionTokens, expectedCompletionTokens)
	}

	if resp.Usage.TotalTokens != expectedTotalTokens {
		t.Errorf("TotalTokens = %d, want %d", resp.Usage.TotalTokens, expectedTotalTokens)
	}
}

func TestQwenOAuthProvider_ChatWithTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any
		json.NewDecoder(r.Body).Decode(&reqBody)

		// Verify tools are present
		tools, ok := reqBody["tools"].([]any)
		if !ok || len(tools) == 0 {
			http.Error(w, "missing tools", http.StatusBadRequest)
			return
		}

		// Return response with tool call
		resp := map[string]any{
			"id":    "chatcmpl-test",
			"model": "coder-model",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":      "assistant",
						"content":   "",
						"tool_calls": []map[string]any{
							{
								"id":   "call-123",
								"type": "function",
								"function": map[string]any{
									"name":      "search_web",
									"arguments": `{"query": "test"}`,
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

	tokenSource := func() (string, error) {
		return "test-token", nil
	}

	provider := NewQwenOAuthProviderWithTokenSource(tokenSource, server.URL+"/v1")

	messages := []protocoltypes.Message{{Role: "user", Content: "Search the web"}}
	tools := []protocoltypes.ToolDefinition{
		{
			Type: "function",
			Function: protocoltypes.ToolFunctionDefinition{
				Name:        "search_web",
				Description: "Search the web",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{
							"type":        "string",
							"description": "Search query",
						},
					},
					"required": []string{"query"},
				},
			},
		},
	}

	resp, err := provider.Chat(context.Background(), messages, tools, "qwen-oauth", nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(resp.ToolCalls))
	}

	if resp.ToolCalls[0].Function.Name != "search_web" {
		t.Errorf("Tool name = %q, want %q", resp.ToolCalls[0].Function.Name, "search_web")
	}
}

func TestQwenOAuthProvider_ChatUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "Invalid token"}}`))
	}))
	defer server.Close()

	tokenSource := func() (string, error) {
		return "invalid-token", nil
	}

	provider := NewQwenOAuthProviderWithTokenSource(tokenSource, server.URL+"/v1")

	messages := []protocoltypes.Message{{Role: "user", Content: "Hello"}}
	_, err := provider.Chat(context.Background(), messages, nil, "qwen-oauth", nil)
	if err == nil {
		t.Error("expected error for unauthorized request")
	}

	if !strings.Contains(err.Error(), "OAuth token rejected") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestQwenOAuthProvider_ChatRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": {"message": "Rate limit exceeded"}}`))
	}))
	defer server.Close()

	tokenSource := func() (string, error) {
		return "test-token", nil
	}

	provider := NewQwenOAuthProviderWithTokenSource(tokenSource, server.URL+"/v1")

	messages := []protocoltypes.Message{{Role: "user", Content: "Hello"}}
	_, err := provider.Chat(context.Background(), messages, nil, "qwen-oauth", nil)
	if err == nil {
		t.Error("expected error for rate limit")
	}

	if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestQwenOAuthProvider_ChatModelError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {"message": "Internal server error", "code": "500"}}`))
	}))
	defer server.Close()

	tokenSource := func() (string, error) {
		return "test-token", nil
	}

	provider := NewQwenOAuthProviderWithTokenSource(tokenSource, server.URL+"/v1")

	messages := []protocoltypes.Message{{Role: "user", Content: "Hello"}}
	_, err := provider.Chat(context.Background(), messages, nil, "qwen-oauth", nil)
	if err == nil {
		t.Error("expected error for server error")
	}

	if !strings.Contains(err.Error(), "API error") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestQwenOAuthProvider_ModelNameStripping(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"qwen-oauth/coder-model", "coder-model"},
		{"qwen-oauth/vision-model", "vision-model"},
		{"qwen/coder-model", "coder-model"},
		{"coder-model", "coder-model"},
		{"", "coder-model"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var reqBody map[string]any
				json.NewDecoder(r.Body).Decode(&reqBody)

				model := reqBody["model"].(string)
				if model != tt.expected {
					t.Errorf("model = %q, want %q", model, tt.expected)
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{
					"choices": []map[string]any{
						{"message": map[string]any{"content": "ok"}},
					},
					"usage": map[string]any{"total_tokens": 1},
				})
			}))
			defer server.Close()

			tokenSource := func() (string, error) {
				return "test-token", nil
			}

			provider := NewQwenOAuthProviderWithTokenSource(tokenSource, server.URL+"/v1")
			_, err := provider.Chat(context.Background(), []protocoltypes.Message{{Role: "user", Content: "test"}}, nil, tt.input, nil)
			if err != nil {
				t.Fatalf("Chat() error: %v", err)
			}
		})
	}
}

func TestQwenOAuthProvider_ChatContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "ok"}},
			},
		})
	}))
	defer server.Close()

	tokenSource := func() (string, error) {
		return "test-token", nil
	}

	provider := NewQwenOAuthProviderWithTokenSource(tokenSource, server.URL+"/v1")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	messages := []protocoltypes.Message{{Role: "user", Content: "Hello"}}
	_, err := provider.Chat(ctx, messages, nil, "qwen-oauth", nil)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestQwenOAuthProvider_ParseResponseInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	tokenSource := func() (string, error) {
		return "test-token", nil
	}

	provider := NewQwenOAuthProviderWithTokenSource(tokenSource, server.URL+"/v1")

	messages := []protocoltypes.Message{{Role: "user", Content: "Hello"}}
	_, err := provider.Chat(context.Background(), messages, nil, "qwen-oauth", nil)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	if !strings.Contains(err.Error(), "parsing") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestQwenOAuthProvider_ParseResponseNoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "test",
			"usage": map[string]any{"total_tokens": 0},
		})
	}))
	defer server.Close()

	tokenSource := func() (string, error) {
		return "test-token", nil
	}

	provider := NewQwenOAuthProviderWithTokenSource(tokenSource, server.URL+"/v1")

	messages := []protocoltypes.Message{{Role: "user", Content: "Hello"}}
	_, err := provider.Chat(context.Background(), messages, nil, "qwen-oauth", nil)
	if err == nil {
		t.Error("expected error for missing choices")
	}

	if !strings.Contains(err.Error(), "no choices") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestQwenOAuthProvider_OptionsForwarding(t *testing.T) {
	var receivedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "ok"}},
			},
			"usage": map[string]any{"total_tokens": 1},
		})
	}))
	defer server.Close()

	tokenSource := func() (string, error) {
		return "test-token", nil
	}

	provider := NewQwenOAuthProviderWithTokenSource(tokenSource, server.URL+"/v1")

	messages := []protocoltypes.Message{{Role: "user", Content: "Hello"}}
	options := map[string]any{
		"temperature": 0.8,
		"max_tokens":  2048,
		"top_p":       0.9,
	}

	_, err := provider.Chat(context.Background(), messages, nil, "qwen-oauth", options)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	// Verify options were forwarded
	if got, want := receivedBody["temperature"], 0.8; got != want {
		t.Errorf("temperature = %v, want %v", got, want)
	}
	if got, want := receivedBody["max_tokens"], float64(2048); got != want {
		t.Errorf("max_tokens = %v, want %v", got, want)
	}
	if got, want := receivedBody["top_p"], 0.9; got != want {
		t.Errorf("top_p = %v, want %v", got, want)
	}
}

func TestConvertMessagesForQwen(t *testing.T) {
	messages := []protocoltypes.Message{
		{Role: "system", Content: "You are helpful"},
		{Role: "user", Content: "Hello"},
		{
			Role:    "assistant",
			Content: "Hi!",
			ToolCalls: []protocoltypes.ToolCall{
				{
					ID:   "call-1",
					Type: "function",
					Function: &protocoltypes.FunctionCall{
						Name:      "search",
						Arguments: `{"q": "test"}`,
					},
				},
			},
		},
		{
			Role:       "tool",
			Content:    "Result",
			ToolCallID: "call-1",
		},
	}

	result := convertMessagesForQwen(messages)

	if len(result) != 4 {
		t.Fatalf("Expected 4 messages, got %d", len(result))
	}

	// Check tool call message
	if result[2]["tool_calls"] == nil {
		t.Error("Expected tool_calls in assistant message")
	}

	// Check tool result message
	if result[3]["tool_call_id"] != "call-1" {
		t.Errorf("tool_call_id = %q, want %q", result[3]["tool_call_id"], "call-1")
	}
}

func TestConvertToolsForQwen(t *testing.T) {
	tools := []protocoltypes.ToolDefinition{
		{
			Type: "function",
			Function: protocoltypes.ToolFunctionDefinition{
				Name:        "search",
				Description: "Search the web",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{"type": "string"},
					},
				},
			},
		},
	}

	result := convertToolsForQwen(tools)

	if len(result) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(result))
	}

	tool := result[0]
	if tool["type"] != "function" {
		t.Errorf("type = %q, want %q", tool["type"], "function")
	}

	fn, ok := tool["function"].(map[string]any)
	if !ok {
		t.Fatal("function is not a map")
	}
	if fn["name"] != "search" {
		t.Errorf("name = %q, want %q", fn["name"], "search")
	}
	if fn["description"] != "Search the web" {
		t.Errorf("description = %q, want %q", fn["description"], "Search the web")
	}
}

func TestCreateQwenOAuthProviderFromStore(t *testing.T) {
	// This test verifies the factory function exists and returns correct type
	// Full integration test would require mocking the auth store
	provider, err := createQwenOAuthProvider()

	// We expect an error since we haven't set up credentials
	if err == nil {
		// If no error, verify provider type
		if provider == nil {
			t.Error("Expected provider or error")
		}
	}
}
