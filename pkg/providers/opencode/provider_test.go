package opencode

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProvider_GetDefaultModel(t *testing.T) {
	p := NewProvider("test-key", "", "")
	if got := p.GetDefaultModel(); got != "kimi-k2.5" {
		t.Errorf("GetDefaultModel() = %q, want %q", got, "kimi-k2.5")
	}
}

func TestProviderChat_OpenAICompatibleEndpoint(t *testing.T) {
	var requestBody map[string]any
	var requestPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message":       map[string]any{"content": "Hello from kimi"},
					"finish_reason": "stop",
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

	p := NewProvider("test-key", server.URL, "")
	out, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"kimi-k2.5",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	// Should use /chat/completions endpoint for kimi models
	if requestPath != "/chat/completions" {
		t.Errorf("request path = %q, want %q", requestPath, "/chat/completions")
	}

	if out.Content != "Hello from kimi" {
		t.Errorf("Content = %q, want %q", out.Content, "Hello from kimi")
	}

	if out.Usage == nil {
		t.Fatal("Usage should not be nil")
	}
	if out.Usage.PromptTokens != 10 {
		t.Errorf("PromptTokens = %d, want 10", out.Usage.PromptTokens)
	}
}

func TestProviderChat_AnthropicMessagesEndpoint(t *testing.T) {
	var requestBody map[string]any
	var requestPath string
	var apiKey string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		apiKey = r.Header.Get("x-api-key")
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp := map[string]any{
			"content": []map[string]any{
				{
					"type": "text",
					"text": "Hello from claude",
				},
			},
			"stop_reason": "end_turn",
			"usage": map[string]any{
				"input_tokens":  20,
				"output_tokens": 10,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	out, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"claude-opus-4-6",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	// Should use /messages endpoint for claude models
	if requestPath != "/messages" {
		t.Errorf("request path = %q, want %q", requestPath, "/messages")
	}

	// Should use x-api-key header
	if apiKey != "test-key" {
		t.Errorf("API key = %q, want %q", apiKey, "test-key")
	}

	if out.Content != "Hello from claude" {
		t.Errorf("Content = %q, want %q", out.Content, "Hello from claude")
	}

	if out.Usage == nil {
		t.Fatal("Usage should not be nil")
	}
	if out.Usage.PromptTokens != 20 {
		t.Errorf("PromptTokens = %d, want 20", out.Usage.PromptTokens)
	}
}

func TestProviderChat_OpenAIResponsesEndpoint(t *testing.T) {
	var requestBody map[string]any
	var requestPath string
	var authHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		authHeader = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp := map[string]any{
			"output": []map[string]any{
				{
					"type": "message",
					"content": []map[string]any{
						{
							"type": "text",
							"text": "Hello from GPT",
						},
					},
				},
			},
			"usage": map[string]any{
				"input_tokens":  15,
				"output_tokens": 8,
				"total_tokens":  23,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	out, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"gpt-5.3-codex",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	// Should use /responses endpoint for GPT models
	if requestPath != "/responses" {
		t.Errorf("request path = %q, want %q", requestPath, "/responses")
	}

	// Should use Authorization Bearer header
	if authHeader != "Bearer test-key" {
		t.Errorf("Authorization header = %q, want %q", authHeader, "Bearer test-key")
	}

	if out.Content != "Hello from GPT" {
		t.Errorf("Content = %q, want %q", out.Content, "Hello from GPT")
	}

	if out.Usage == nil {
		t.Fatal("Usage should not be nil")
	}
	if out.Usage.PromptTokens != 15 {
		t.Errorf("PromptTokens = %d, want 15", out.Usage.PromptTokens)
	}
}

func TestProviderChat_WithTools(t *testing.T) {
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": "",
						"tool_calls": []map[string]any{
							{
								"id":   "call_1",
								"type": "function",
								"function": map[string]any{
									"name":      "get_weather",
									"arguments": `{"location":"NYC"}`,
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "get_weather",
				Description: "Get weather for a location",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]any{
							"type": "string",
						},
					},
				},
			},
		},
	}
	out, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "What's the weather?"}},
		tools,
		"kimi-k2.5",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if len(out.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(out.ToolCalls))
	}

	if out.ToolCalls[0].Name != "get_weather" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", out.ToolCalls[0].Name, "get_weather")
	}

	if out.ToolCalls[0].Arguments["location"] != "NYC" {
		t.Errorf("ToolCalls[0].Arguments[location] = %v, want NYC", out.ToolCalls[0].Arguments["location"])
	}

	if out.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want %q", out.FinishReason, "tool_calls")
	}
}

func TestProviderChat_WithTemperatureAndMaxTokens(t *testing.T) {
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

	p := NewProvider("test-key", server.URL, "")
	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"kimi-k2.5",
		map[string]any{
			"temperature": 0.7,
			"max_tokens":  100,
		},
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	// Kimi k2 models should always use temperature=1.0
	if requestBody["temperature"] != 1.0 {
		t.Errorf("temperature = %v, want 1.0", requestBody["temperature"])
	}

	if requestBody["max_tokens"] != float64(100) {
		t.Errorf("max_tokens = %v, want 100", requestBody["max_tokens"])
	}
}

func TestProviderChat_GeminiModelsEndpoint(t *testing.T) {
	var requestBody map[string]any
	var requestPath string
	var authHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		authHeader = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp := map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"parts": []map[string]any{
							{"text": "Hello from Gemini"},
						},
						"role": "model",
					},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     25,
				"candidatesTokenCount": 12,
				"totalTokenCount":      37,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	out, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"gemini-3.1-pro",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	// Should use /models/{model} endpoint for Gemini models
	if requestPath != "/models/gemini-3.1-pro" {
		t.Errorf("request path = %q, want %q", requestPath, "/models/gemini-3.1-pro")
	}

	// Should use Authorization Bearer header
	if authHeader != "Bearer test-key" {
		t.Errorf("Authorization header = %q, want %q", authHeader, "Bearer test-key")
	}

	if out.Content != "Hello from Gemini" {
		t.Errorf("Content = %q, want %q", out.Content, "Hello from Gemini")
	}

	if out.Usage == nil {
		t.Fatal("Usage should not be nil")
	}
	if out.Usage.PromptTokens != 25 {
		t.Errorf("PromptTokens = %d, want 25", out.Usage.PromptTokens)
	}
	if out.Usage.CompletionTokens != 12 {
		t.Errorf("CompletionTokens = %d, want 12", out.Usage.CompletionTokens)
	}
}

func TestProviderChat_GeminiWithTools(t *testing.T) {
	var requestBody map[string]any
	var requestPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp := map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"parts": []map[string]any{
							{
								"functionCall": map[string]any{
									"name": "get_weather",
									"args": map[string]any{
										"location": "NYC",
									},
								},
							},
						},
						"role": "model",
					},
					"finishReason": "STOP",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "get_weather",
				Description: "Get weather for a location",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]any{
							"type": "string",
						},
					},
				},
			},
		},
	}
	out, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "What's the weather?"}},
		tools,
		"gemini-3-flash",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	// Should use /models/{model} endpoint
	if requestPath != "/models/gemini-3-flash" {
		t.Errorf("request path = %q, want %q", requestPath, "/models/gemini-3-flash")
	}

	if len(out.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(out.ToolCalls))
	}

	if out.ToolCalls[0].Name != "get_weather" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", out.ToolCalls[0].Name, "get_weather")
	}

	if out.ToolCalls[0].Arguments["location"] != "NYC" {
		t.Errorf("ToolCalls[0].Arguments[location] = %v, want NYC", out.ToolCalls[0].Arguments["location"])
	}

	if out.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want %q", out.FinishReason, "tool_calls")
	}
}

func TestProviderChat_DefaultAPIBase(t *testing.T) {
	p := NewProvider("test-key", "", "")
	if p.apiBase != "https://opencode.ai/zen/v1" {
		t.Errorf("apiBase = %q, want %q", p.apiBase, "https://opencode.ai/zen/v1")
	}
}
