package openai_compat

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestProviderChat_UsesMaxCompletionTokensForGLM(t *testing.T) {
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
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

	p := NewProvider("key", server.URL, "")
	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"glm-4.7",
		map[string]any{"max_tokens": 1234},
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if _, ok := requestBody["max_completion_tokens"]; !ok {
		t.Fatalf("expected max_completion_tokens in request body")
	}
	if _, ok := requestBody["max_tokens"]; ok {
		t.Fatalf("did not expect max_tokens key for glm model")
	}
}

func TestProviderChat_ParsesToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
									"arguments": "{\"city\":\"SF\"}",
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

	p := NewProvider("key", server.URL, "")
	out, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "gpt-4o", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if len(out.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(out.ToolCalls))
	}
	if out.ToolCalls[0].Name != "get_weather" {
		t.Fatalf("ToolCalls[0].Name = %q, want %q", out.ToolCalls[0].Name, "get_weather")
	}
	if out.ToolCalls[0].Arguments["city"] != "SF" {
		t.Fatalf("ToolCalls[0].Arguments[city] = %v, want SF", out.ToolCalls[0].Arguments["city"])
	}
}

func TestProviderChat_ParsesReasoningContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content":           "The answer is 2",
						"reasoning_content": "Let me think step by step... 1+1=2",
						"tool_calls": []map[string]any{
							{
								"id":   "call_1",
								"type": "function",
								"function": map[string]any{
									"name":      "calculator",
									"arguments": "{\"expr\":\"1+1\"}",
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

	p := NewProvider("key", server.URL, "")
	out, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "1+1=?"}}, nil, "kimi-k2.5", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if out.ReasoningContent != "Let me think step by step... 1+1=2" {
		t.Fatalf("ReasoningContent = %q, want %q", out.ReasoningContent, "Let me think step by step... 1+1=2")
	}
	if out.Content != "The answer is 2" {
		t.Fatalf("Content = %q, want %q", out.Content, "The answer is 2")
	}
	if len(out.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(out.ToolCalls))
	}
}

func TestProviderChat_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "gpt-4o", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestProviderChat_StripsMoonshotPrefixAndNormalizesKimiTemperature(t *testing.T) {
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

	p := NewProvider("key", server.URL, "")
	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"moonshot/kimi-k2.5",
		map[string]any{"temperature": 0.3},
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if requestBody["model"] != "kimi-k2.5" {
		t.Fatalf("model = %v, want kimi-k2.5", requestBody["model"])
	}
	if requestBody["temperature"] != 1.0 {
		t.Fatalf("temperature = %v, want 1.0", requestBody["temperature"])
	}
}

func TestProviderChat_StripsGroqAndOllamaPrefixes(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantModel string
	}{
		{
			name:      "strips groq prefix and keeps nested model",
			input:     "groq/openai/gpt-oss-120b",
			wantModel: "openai/gpt-oss-120b",
		},
		{
			name:      "strips ollama prefix",
			input:     "ollama/qwen2.5:14b",
			wantModel: "qwen2.5:14b",
		},
		{
			name:      "strips deepseek prefix",
			input:     "deepseek/deepseek-chat",
			wantModel: "deepseek-chat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			p := NewProvider("key", server.URL, "")
			_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, tt.input, nil)
			if err != nil {
				t.Fatalf("Chat() error = %v", err)
			}

			if requestBody["model"] != tt.wantModel {
				t.Fatalf("model = %v, want %s", requestBody["model"], tt.wantModel)
			}
		})
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

func TestProviderChat_AcceptsNumericOptionTypes(t *testing.T) {
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

	p := NewProvider("key", server.URL, "")
	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"gpt-4o",
		map[string]any{"max_tokens": float64(512), "temperature": 1},
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if requestBody["max_tokens"] != float64(512) {
		t.Fatalf("max_tokens = %v, want 512", requestBody["max_tokens"])
	}
	if requestBody["temperature"] != float64(1) {
		t.Fatalf("temperature = %v, want 1", requestBody["temperature"])
	}
}

func TestNormalizeModel_UsesAPIBase(t *testing.T) {
	if got := normalizeModel("deepseek/deepseek-chat", "https://api.deepseek.com/v1"); got != "deepseek-chat" {
		t.Fatalf("normalizeModel(deepseek) = %q, want %q", got, "deepseek-chat")
	}
	if got := normalizeModel("openrouter/auto", "https://openrouter.ai/api/v1"); got != "openrouter/auto" {
		t.Fatalf("normalizeModel(openrouter) = %q, want %q", got, "openrouter/auto")
	}
}

func TestProviderChat_ParsesMiniMaxToolCallsAndThinking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": "<think>\nThinking about the news...\n</think>\n\nI will search for the news.\n\n[TOOL_CALL]<invoke name=\"web_search\"><parameter name=\"count\">5</parameter><parameter name=\"query\">AI news</parameter></invoke></minimax:tool_call>",
					},
					"finish_reason": "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	out, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "minimax-m2.5", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if !strings.Contains(out.ReasoningContent, "Thinking about the news") {
		t.Errorf("ReasoningContent = %q, want it to contain reasoning", out.ReasoningContent)
	}
	if out.Content != "I will search for the news." {
		t.Errorf("Content = %q, want %q", out.Content, "I will search for the news.")
	}
	if len(out.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(out.ToolCalls))
	}
	if out.ToolCalls[0].Name != "web_search" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", out.ToolCalls[0].Name, "web_search")
	}
	if out.ToolCalls[0].Arguments["query"] != "AI news" {
		t.Errorf("ToolCalls[0].Arguments[query] = %v, want AI news", out.ToolCalls[0].Arguments["query"])
	}
	if out.ToolCalls[0].Arguments["count"] != "5" {
		t.Errorf("ToolCalls[0].Arguments[count] = %v, want 5", out.ToolCalls[0].Arguments["count"])
	}
	if out.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want %q", out.FinishReason, "tool_calls")
	}

	// Verify tags are removed from content/reasoning
	if strings.Contains(out.Content, "[TOOL_CALL]") {
		t.Errorf("Content contains [TOOL_CALL]")
	}
	if strings.Contains(out.ReasoningContent, "<think>") {
		t.Errorf("ReasoningContent contains <think>")
	}
}

func TestProviderChat_ParsesMiniMaxMultipleToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": "[TOOL_CALL]<invoke name=\"web_search\"><parameter name=\"query\">AI news</parameter></invoke></minimax:tool_call>\n\n[TOOL_CALL]<invoke name=\"translate\"><parameter name=\"text\">hello</parameter><parameter name=\"target_lang\">es</parameter></invoke></minimax:tool_call>",
					},
					"finish_reason": "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	out, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "minimax-m2.5", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if len(out.ToolCalls) != 2 {
		t.Fatalf("len(ToolCalls) = %d, want 2", len(out.ToolCalls))
	}
	if out.ToolCalls[0].Name != "web_search" || out.ToolCalls[1].Name != "translate" {
		t.Errorf("ToolCall names mismatch")
	}
}

func TestProviderChat_ParsesMiniMaxToolCallsWithSpecialCharsAndEmptyValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": "[TOOL_CALL]<invoke name=\"web_search\"><parameter name=\"query\">AI \"news\" &amp; &lt;trends&gt;</parameter><parameter name=\"optional\"></parameter></invoke></minimax:tool_call>",
					},
					"finish_reason": "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	out, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "minimax-m2.5", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if len(out.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(out.ToolCalls))
	}
	query := out.ToolCalls[0].Arguments["query"].(string)
	if query != `AI "news" & <trends>` {
		t.Errorf("query = %q, want decoded XML special chars", query)
	}

	optionalVal, ok := out.ToolCalls[0].Arguments["optional"]
	if !ok {
		t.Fatalf(`optional argument missing; want present with value ""`)
	} else if optionalStr, ok := optionalVal.(string); !ok || optionalStr != "" {
		t.Errorf("optional = %#v, want empty string", optionalVal)
	}
}

func TestProviderChat_MergesJSONAndMiniMaxToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": "[TOOL_CALL]<invoke name=\"web_search\"><parameter name=\"query\">AI news</parameter></invoke></minimax:tool_call>",
						"tool_calls": []map[string]any{
							{
								"id":   "call_1",
								"type": "function",
								"function": map[string]any{
									"name":      "get_weather",
									"arguments": `{"location":"San Francisco"}`,
								},
							},
						},
					},
					"finish_reason": "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	out, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "minimax-m2.5", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if len(out.ToolCalls) != 2 {
		t.Fatalf("len(ToolCalls) = %d, want 2 (1 JSON tool call + 1 MiniMax tool call)", len(out.ToolCalls))
	}

	// Verify both names exist
	names := map[string]bool{out.ToolCalls[0].Name: true, out.ToolCalls[1].Name: true}
	if !names["web_search"] || !names["get_weather"] {
		t.Errorf("merged tool calls mismatch: %v", names)
	}

	if out.FinishReason != "tool_calls" {
		t.Fatalf("FinishReason = %q, want %q", out.FinishReason, "tool_calls")
	}
}

func TestProviderChat_ParsesMiniMaxToolCallsWithAlternativeClosingTag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": "[TOOL_CALL]<invoke name=\"web_search\"><parameter name=\"query\">AI news</parameter></invoke>[/minimax:tool_call]",
					},
					"finish_reason": "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	out, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "minimax-m2.5", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if len(out.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(out.ToolCalls))
	}
}

func TestProviderChat_ParsesMiniMaxToolCallsWithMalformedXML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": "[TOOL_CALL]<invoke name=\"web_search\"><parameter name=\"query\">AI news</minimax:tool_call>",
					},
					"finish_reason": "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	out, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "minimax-m2.5", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if len(out.ToolCalls) != 0 {
		t.Errorf("len(ToolCalls) = %d, want 0 for malformed XML", len(out.ToolCalls))
	}
}


func TestProvider_RequestTimeoutDefault(t *testing.T) {
	p := NewProviderWithMaxTokensFieldAndTimeout("key", "https://example.com/v1", "", "", 0)
	if p.httpClient.Timeout != defaultRequestTimeout {
		t.Fatalf("http timeout = %v, want %v", p.httpClient.Timeout, defaultRequestTimeout)
	}
}

func TestProvider_RequestTimeoutOverride(t *testing.T) {
	p := NewProviderWithMaxTokensFieldAndTimeout("key", "https://example.com/v1", "", "", 300)
	if p.httpClient.Timeout != 300*time.Second {
		t.Fatalf("http timeout = %v, want %v", p.httpClient.Timeout, 300*time.Second)
	}
}

func TestProvider_FunctionalOptionMaxTokensField(t *testing.T) {
	p := NewProvider("key", "https://example.com/v1", "", WithMaxTokensField("max_completion_tokens"))
	if p.maxTokensField != "max_completion_tokens" {
		t.Fatalf("maxTokensField = %q, want %q", p.maxTokensField, "max_completion_tokens")
	}
}

func TestProvider_FunctionalOptionRequestTimeout(t *testing.T) {
	p := NewProvider("key", "https://example.com/v1", "", WithRequestTimeout(45*time.Second))
	if p.httpClient.Timeout != 45*time.Second {
		t.Fatalf("http timeout = %v, want %v", p.httpClient.Timeout, 45*time.Second)
	}
}

func TestProvider_FunctionalOptionRequestTimeoutNonPositive(t *testing.T) {
	p := NewProvider("key", "https://example.com/v1", "", WithRequestTimeout(-1*time.Second))
	if p.httpClient.Timeout != defaultRequestTimeout {
		t.Fatalf("http timeout = %v, want %v", p.httpClient.Timeout, defaultRequestTimeout)
	}
}

func TestProviderChat_ParsesMiniMaxToolCallsGeneralization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": "[TOOL_CALL]<invoke name=\"shell_execute\"><parameter name=\"command\">ls -la</parameter><parameter name=\"timeout\">30</parameter></invoke></minimax:tool_call>\n" +
							"[TOOL_CALL]<invoke name=\"read_file\"><parameter name=\"path\">/etc/passwd</parameter></invoke>[/minimax:tool_call]",
					},
					"finish_reason": "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	out, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "minimax-m2.5", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if len(out.ToolCalls) != 2 {
		t.Fatalf("len(ToolCalls) = %d, want 2", len(out.ToolCalls))
	}

	// Verify tool 1: shell_execute
	tc1 := out.ToolCalls[0]
	if tc1.Name != "shell_execute" {
		t.Errorf("tc1.Name = %q, want shell_execute", tc1.Name)
	}
	if tc1.Arguments["command"] != "ls -la" || tc1.Arguments["timeout"] != "30" {
		t.Errorf("tc1 arguments mismatch: %v", tc1.Arguments)
	}

	// Verify tool 2: read_file
	tc2 := out.ToolCalls[1]
	if tc2.Name != "read_file" {
		t.Errorf("tc2.Name = %q, want read_file", tc2.Name)
	}
	if tc2.Arguments["path"] != "/etc/passwd" {
		t.Errorf("tc2 arguments mismatch: %v", tc2.Arguments)
	}
}
