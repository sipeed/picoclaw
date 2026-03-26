package openai_compat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

func TestNormalizeModel_OpenAIPrefix(t *testing.T) {
	if got := normalizeModel("openai/gpt-5.2", "https://api.openai.com/v1"); got != "gpt-5.2" {
		t.Fatalf("normalizeModel(openai/gpt-5.2) = %q, want %q", got, "gpt-5.2")
	}
}

func TestProviderChat_StreamingTextResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/text/chatcompletion_v2" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body["stream"] != true {
			t.Error("expected stream=true in request body")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)

		chunks := []string{
			`data: {"choices":[{"delta":{"content":"Hello"},"finish_reason":""}]}`,
			`data: {"choices":[{"delta":{"content":" world"},"finish_reason":""}]}`,
			`data: {"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`,
			`data: [DONE]`,
		}
		for _, c := range chunks {
			fmt.Fprintln(w, c)
			fmt.Fprintln(w)
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "",
		WithEndpointPath("/text/chatcompletion_v2"),
		WithStream(true),
	)
	out, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "MiniMax-M1", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if out.Content != "Hello world" {
		t.Fatalf("Content = %q, want %q", out.Content, "Hello world")
	}
	if out.FinishReason != "stop" {
		t.Fatalf("FinishReason = %q, want %q", out.FinishReason, "stop")
	}
	if out.Usage == nil || out.Usage.TotalTokens != 7 {
		t.Fatalf("Usage.TotalTokens = %v, want 7", out.Usage)
	}
}

func TestProviderChat_StreamingToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)

		chunks := []string{
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"get_weather","arguments":""}}]},"finish_reason":""}]}`,
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"city\":"}}]},"finish_reason":""}]}`,
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"SF\"}"}}]},"finish_reason":""}]}`,
			`data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":10,"completion_tokens":8,"total_tokens":18}}`,
			`data: [DONE]`,
		}
		for _, c := range chunks {
			fmt.Fprintln(w, c)
			fmt.Fprintln(w)
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "", WithStream(true))
	out, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "weather?"}}, nil, "test", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if len(out.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(out.ToolCalls))
	}
	tc := out.ToolCalls[0]
	if tc.ID != "call_1" {
		t.Fatalf("ToolCalls[0].ID = %q, want %q", tc.ID, "call_1")
	}
	if tc.Name != "get_weather" {
		t.Fatalf("ToolCalls[0].Name = %q, want %q", tc.Name, "get_weather")
	}
	if tc.Arguments["city"] != "SF" {
		t.Fatalf("ToolCalls[0].Arguments[city] = %v, want SF", tc.Arguments["city"])
	}
}

func TestProviderChat_CustomEndpointPath(t *testing.T) {
	var hitPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitPath = r.URL.Path
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "ok"}, "finish_reason": "stop"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "",
		WithEndpointPath("/text/chatcompletion_v2"),
	)
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "test", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if hitPath != "/text/chatcompletion_v2" {
		t.Fatalf("endpoint path = %q, want %q", hitPath, "/text/chatcompletion_v2")
	}
}

func TestCanStream(t *testing.T) {
	p1 := NewProvider("key", "https://example.com", "")
	if p1.CanStream() {
		t.Error("CanStream() = true for non-stream provider")
	}

	p2 := NewProvider("key", "https://example.com", "", WithStream(true))
	if !p2.CanStream() {
		t.Error("CanStream() = false for stream provider")
	}
}

func TestProvider_RequestTimeoutNonPositive(t *testing.T) {
	p := NewProviderWithMaxTokensFieldAndTimeout("key", "https://example.com/v1", "", "", -1)
	if p.httpClient.Timeout != defaultRequestTimeout {
		t.Fatalf("http timeout = %v, want %v", p.httpClient.Timeout, defaultRequestTimeout)
	}
}
