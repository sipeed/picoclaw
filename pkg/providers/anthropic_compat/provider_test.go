package anthropic_compat

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestProviderChat_BasicChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		// Verify headers
		if r.Header.Get("x-api-key") != "test-key" {
			t.Fatalf("expected x-api-key header, got %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Fatalf("expected anthropic-version header, got %q", r.Header.Get("anthropic-version"))
		}

		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "Hello!"},
			},
			"stop_reason": "end_turn",
			"usage": map[string]any{
				"input_tokens":  10,
				"output_tokens": 5,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	out, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "Hi"}},
		nil,
		"claude-3-5-sonnet-20241022",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if out.Content != "Hello!" {
		t.Fatalf("Content = %q, want %q", out.Content, "Hello!")
	}
}

func TestProviderChat_ParsesToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content": []map[string]any{
				{
					"type": "text",
					"text": "I'll get the weather for you.",
				},
				{
					"type": "tool_use",
					"id":   "toolu_123",
					"name": "get_weather",
					"input": map[string]any{
						"city": "Beijing",
					},
				},
			},
			"stop_reason": "tool_use",
			"usage": map[string]any{
				"input_tokens":  10,
				"output_tokens": 20,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	out, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "What's the weather?"}}, nil, "claude-3-5-sonnet", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if out.Content != "I'll get the weather for you." {
		t.Fatalf("Content = %q, want %q", out.Content, "I'll get the weather for you.")
	}
	if len(out.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(out.ToolCalls))
	}
	if out.ToolCalls[0].Name != "get_weather" {
		t.Fatalf("ToolCalls[0].Name = %q, want %q", out.ToolCalls[0].Name, "get_weather")
	}
	if out.ToolCalls[0].Arguments["city"] != "Beijing" {
		t.Fatalf("ToolCalls[0].Arguments[city] = %v, want Beijing", out.ToolCalls[0].Arguments["city"])
	}
	if out.ToolCalls[0].ID != "toolu_123" {
		t.Fatalf("ToolCalls[0].ID = %q, want toolu_123", out.ToolCalls[0].ID)
	}
	if out.FinishReason != "tool_calls" {
		t.Fatalf("FinishReason = %q, want tool_calls", out.FinishReason)
	}
}

func TestProviderChat_HandlesSystemMessages(t *testing.T) {
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "ok"},
			},
			"stop_reason": "end_turn",
			"usage":       map[string]any{"input_tokens": 10, "output_tokens": 5},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	_, err := p.Chat(
		t.Context(),
		[]Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello"},
		},
		nil,
		"claude-3-5-sonnet",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	system, ok := requestBody["system"].([]any)
	if !ok {
		t.Fatalf("expected system in request body")
	}
	if len(system) != 1 || system[0] != "You are a helpful assistant." {
		t.Fatalf("system = %v, want [You are a helpful assistant.]", system)
	}
}

func TestProviderChat_PassesToolsToRequest(t *testing.T) {
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "ok"},
			},
			"stop_reason": "end_turn",
			"usage":       map[string]any{"input_tokens": 10, "output_tokens": 5},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "get_weather",
				Description: "Get weather for a city",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"city": map[string]any{"type": "string"},
					},
					"required": []string{"city"},
				},
			},
		},
	}
	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "Weather in Beijing?"}},
		tools,
		"claude-3-5-sonnet",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	toolsInReq, ok := requestBody["tools"].([]any)
	if !ok {
		t.Fatalf("expected tools in request body")
	}
	if len(toolsInReq) != 1 {
		t.Fatalf("len(tools) = %d, want 1", len(toolsInReq))
	}
	tool := toolsInReq[0].(map[string]any)
	if tool["name"] != "get_weather" {
		t.Fatalf("tool name = %v, want get_weather", tool["name"])
	}
}

func TestProviderChat_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "claude-3-5-sonnet", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestProviderChat_EmptyAPIBase(t *testing.T) {
	p := NewProvider("key", "", "")
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "claude-3-5-sonnet", nil)
	if err == nil {
		t.Fatal("expected error for empty API base, got nil")
	}
	if err.Error() != "API base not configured" {
		t.Fatalf("error = %q, want %q", err.Error(), "API base not configured")
	}
}

func TestProvider_ProxyConfigured(t *testing.T) {
	proxyURL := "http://127.0.0.1:8080"
	p := NewProvider("key", "https://example.com", proxyURL)

	transport, ok := p.httpClient.Transport.(*http.Transport)
	if !ok || transport == nil {
		t.Fatalf("expected http transport with proxy, got %T", p.httpClient.Transport)
	}

	req := &http.Request{URL: &url.URL{Scheme: "https", Host: "api.example.com"}}
	gotProxy, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("proxy function returned error: %v", err)
	}
	if gotProxy == nil || gotProxy.String() != proxyURL {
		t.Fatalf("proxy = %v, want %s", gotProxy, proxyURL)
	}
}

func TestProviderChat_AcceptsOptions(t *testing.T) {
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "ok"},
			},
			"stop_reason": "end_turn",
			"usage":       map[string]any{"input_tokens": 10, "output_tokens": 5},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	// Note: current implementation only accepts int type for max_tokens
	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"claude-3-5-sonnet",
		map[string]any{"max_tokens": 2048, "temperature": 0.5},
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if requestBody["temperature"] != 0.5 {
		t.Fatalf("temperature = %v, want 0.5", requestBody["temperature"])
	}
}

func TestProviderChat_DefaultMaxTokens(t *testing.T) {
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "ok"},
			},
			"stop_reason": "end_turn",
			"usage":       map[string]any{"input_tokens": 10, "output_tokens": 5},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "claude-3-5-sonnet", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	// JSON unmarshals numbers as float64 by default
	maxTokens, ok := requestBody["max_tokens"].(float64)
	if !ok || maxTokens != 4096 {
		t.Fatalf("max_tokens = %v (type %T), want 4096", requestBody["max_tokens"], requestBody["max_tokens"])
	}
}

func TestProviderChat_HandleToolResultMessages(t *testing.T) {
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "The weather is sunny."},
			},
			"stop_reason": "end_turn",
			"usage":       map[string]any{"input_tokens": 10, "output_tokens": 5},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	_, err := p.Chat(
		t.Context(),
		[]Message{
			{Role: "assistant", Content: "", ToolCalls: []ToolCall{
				{ID: "toolu_1", Name: "get_weather", Arguments: map[string]any{"city": "Beijing"}},
			}},
			{Role: "tool", Content: "Sunny, 25°C", ToolCallID: "toolu_1"},
		},
		nil,
		"claude-3-5-sonnet",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	messages, ok := requestBody["messages"].([]any)
	if !ok {
		t.Fatalf("expected messages in request body")
	}
	// Should have 2 messages: assistant (with tool_use) and user (with tool_result)
	if len(messages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(messages))
	}
}

func TestProviderChat_StopReasonMaxTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "partial..."},
			},
			"stop_reason": "max_tokens",
			"usage":       map[string]any{"input_tokens": 10, "output_tokens": 4096},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	out, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "claude-3-5-sonnet", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if out.FinishReason != "length" {
		t.Fatalf("FinishReason = %q, want length", out.FinishReason)
	}
}

func TestProviderChat_UsageInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "ok"},
			},
			"stop_reason": "end_turn",
			"usage": map[string]any{
				"input_tokens":  100,
				"output_tokens": 50,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	out, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "claude-3-5-sonnet", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if out.Usage.PromptTokens != 100 {
		t.Fatalf("PromptTokens = %d, want 100", out.Usage.PromptTokens)
	}
	if out.Usage.CompletionTokens != 50 {
		t.Fatalf("CompletionTokens = %d, want 50", out.Usage.CompletionTokens)
	}
	if out.Usage.TotalTokens != 150 {
		t.Fatalf("TotalTokens = %d, want 150", out.Usage.TotalTokens)
	}
}

func TestBuildRequestBody_PassesModelDirectly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestBody map[string]any
		json.NewDecoder(r.Body).Decode(&requestBody)

		resp := map[string]any{
			"content":     []map[string]any{{"type": "text", "text": "ok"}},
			"stop_reason": "end_turn",
			"usage":       map[string]any{"input_tokens": 10, "output_tokens": 5},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)

		if requestBody["model"] != "claude-3-5-sonnet" {
			t.Errorf("model = %q, want %q", requestBody["model"], "claude-3-5-sonnet")
		}
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "claude-3-5-sonnet", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
}

func TestNewProvider_TrimsTrailingSlash(t *testing.T) {
	p := NewProvider("key", "https://api.example.com/", "")
	if p.apiBase != "https://api.example.com" {
		t.Fatalf("apiBase = %q, want %q", p.apiBase, "https://api.example.com")
	}
}
