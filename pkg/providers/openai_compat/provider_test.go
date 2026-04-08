package openai_compat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers/common"
	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
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

func TestProviderChat_ParsesToolCallsWithObjectArguments(t *testing.T) {
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
									"name": "get_weather",
									"arguments": map[string]any{
										"city":   "SF",
										"metric": true,
									},
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
	if out.ToolCalls[0].Arguments["metric"] != true {
		t.Fatalf("ToolCalls[0].Arguments[metric] = %v, want true", out.ToolCalls[0].Arguments["metric"])
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

func TestProviderChat_PreservesReasoningContentInHistory(t *testing.T) {
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

	// Simulate a multi-turn conversation where the assistant's previous
	// reply included reasoning_content (e.g. from kimi-k2.5).
	messages := []Message{
		{Role: "user", Content: "What is 1+1?"},
		{Role: "assistant", Content: "2", ReasoningContent: "Let me think... 1+1=2"},
		{Role: "user", Content: "What about 2+2?"},
	}

	_, err := p.Chat(t.Context(), messages, nil, "kimi-k2.5", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	// Verify reasoning_content is preserved in the serialized request.
	reqMessages, ok := requestBody["messages"].([]any)
	if !ok {
		t.Fatalf("messages is not []any: %T", requestBody["messages"])
	}
	assistantMsg, ok := reqMessages[1].(map[string]any)
	if !ok {
		t.Fatalf("assistant message is not map[string]any: %T", reqMessages[1])
	}
	if assistantMsg["reasoning_content"] != "Let me think... 1+1=2" {
		t.Errorf("reasoning_content not preserved in request, got %v", assistantMsg["reasoning_content"])
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

func TestProviderChat_JSONHTTPErrorDoesNotReportHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "gpt-4o", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Status: 400") {
		t.Fatalf("expected status code in error, got %v", err)
	}
	if strings.Contains(err.Error(), "returned HTML instead of JSON") {
		t.Fatalf("expected non-HTML http error, got %v", err)
	}
}

func TestProviderChat_HTMLResponsesReturnHelpfulError(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		statusCode  int
		body        string
	}{
		{
			name:        "html success response",
			contentType: "text/html; charset=utf-8",
			statusCode:  http.StatusOK,
			body:        "<!DOCTYPE html><html><body>gateway login</body></html>",
		},
		{
			name:        "html error response",
			contentType: "text/html; charset=utf-8",
			statusCode:  http.StatusBadGateway,
			body:        "<!DOCTYPE html><html><body>bad gateway</body></html>",
		},
		{
			name:        "mislabeled html success response",
			contentType: "application/json",
			statusCode:  http.StatusOK,
			body:        "   \r\n\t<!DOCTYPE html><html><body>gateway login</body></html>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			p := NewProvider("key", server.URL, "")
			_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "gpt-4o", nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), fmt.Sprintf("Status: %d", tt.statusCode)) {
				t.Fatalf("expected status code in error, got %v", err)
			}
			if !strings.Contains(err.Error(), "returned HTML instead of JSON") {
				t.Fatalf("expected helpful HTML error, got %v", err)
			}
			if !strings.Contains(err.Error(), "check api_base or proxy configuration") {
				t.Fatalf("expected configuration hint, got %v", err)
			}
		})
	}
}

func TestProviderChat_SuccessResponseUsesStreamingDecoder(t *testing.T) {
	content := strings.Repeat("a", 1024)
	body := `{"choices":[{"message":{"content":"` + content + `"},"finish_reason":"stop"}]}`

	p := NewProvider("key", "https://example.com/v1", "")
	p.httpClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body: &errAfterDataReadCloser{
					data:      []byte(body),
					chunkSize: 64,
				},
			}, nil
		}),
	}

	out, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "gpt-4o", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if out.Content != content {
		t.Fatalf("Content = %q, want %q", out.Content, content)
	}
}

func TestProviderChat_LargeHTMLResponsePreviewIsTruncated(t *testing.T) {
	body := append([]byte("<!DOCTYPE html><html><body>"), bytes.Repeat([]byte("A"), 2048)...)
	body = append(body, []byte("</body></html>")...)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write(body)
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "gpt-4o", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Body:   <!DOCTYPE html><html><body>") {
		t.Fatalf("expected html preview in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "...") {
		t.Fatalf("expected truncated preview, got %v", err)
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

func TestProviderChat_StripsKnownProviderPrefixes(t *testing.T) {
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
	tests := []struct {
		name      string
		input     string
		wantModel string
	}{
		{
			name:      "strips litellm prefix and preserves proxy model name",
			input:     "litellm/my-proxy-alias",
			wantModel: "my-proxy-alias",
		},
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
			name:      "strips lmstudio prefix and keeps nested model",
			input:     "lmstudio/openai/gpt-oss-20b",
			wantModel: "openai/gpt-oss-20b",
		},
		{
			name:      "strips venice prefix",
			input:     "venice/venice-uncensored",
			wantModel: "venice-uncensored",
		},
		{
			name:      "strips deepseek prefix",
			input:     "deepseek/deepseek-chat",
			wantModel: "deepseek-chat",
		},
		{
			name:      "strips vivgrid prefix",
			input:     "vivgrid/auto",
			wantModel: "auto",
		},
		{
			name:      "strips novita prefix deepseek model",
			input:     "novita/deepseek/deepseek-v3.2",
			wantModel: "deepseek/deepseek-v3.2",
		},
		{
			name:      "strips novita prefix zai model",
			input:     "novita/zai-org/glm-5",
			wantModel: "zai-org/glm-5",
		},
		{
			name:      "strips novita prefix minimax model",
			input:     "novita/minimax/minimax-m2.5",
			wantModel: "minimax/minimax-m2.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	if got := normalizeModel("lmstudio/openai/gpt-oss-20b", "http://localhost:1234/v1"); got != "openai/gpt-oss-20b" {
		t.Fatalf("normalizeModel(lmstudio) = %q, want %q", got, "openai/gpt-oss-20b")
	}
	if got := normalizeModel("venice/venice-uncensored", "https://api.venice.ai/api/v1"); got != "venice-uncensored" {
		t.Fatalf("normalizeModel(venice) = %q, want %q", got, "venice-uncensored")
	}
	if got := normalizeModel("openrouter/auto", "https://openrouter.ai/api/v1"); got != "openrouter/auto" {
		t.Fatalf("normalizeModel(openrouter) = %q, want %q", got, "openrouter/auto")
	}
	if got := normalizeModel("vivgrid/managed", "https://api.vivgrid.com/v1"); got != "managed" {
		t.Fatalf("normalizeModel(vivgrid) = %q, want %q", got, "managed")
	}
	if got := normalizeModel("vivgrid/auto", "https://api.vivgrid.com/v1"); got != "auto" {
		t.Fatalf("normalizeModel(vivgrid auto) = %q, want %q", got, "auto")
	}
	if got := normalizeModel(
		"novita/deepseek/deepseek-v3.2",
		"https://api.novita.ai/openai",
	); got != "deepseek/deepseek-v3.2" {
		t.Fatalf("normalizeModel(novita) = %q, want %q", got, "deepseek/deepseek-v3.2")
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

func TestProviderChat_ExtraBodyInjected(t *testing.T) {
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

	extraBody := map[string]any{"reasoning_split": true, "custom_field": "test"}
	p := NewProvider("key", server.URL, "", WithExtraBody(extraBody))

	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"minimax/abab7",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if got, ok := requestBody["reasoning_split"]; !ok || got != true {
		t.Fatalf("reasoning_split = %v, want true", got)
	}
	if got, ok := requestBody["custom_field"]; !ok || got != "test" {
		t.Fatalf("custom_field = %v, want test", got)
	}
}

func TestProviderChat_ExtraBodyOverridesOptions(t *testing.T) {
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

	extraBody := map[string]any{"temperature": 0.9}
	p := NewProvider("key", server.URL, "", WithExtraBody(extraBody))

	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"gpt-4o",
		map[string]any{"temperature": 0.5},
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	// ExtraBody takes precedence over options since it is merged last.
	if got := requestBody["temperature"]; got != float64(0.9) {
		t.Fatalf("temperature = %v, want 0.9 (from extraBody, overriding options)", got)
	}
}

func TestProviderChat_CustomHeadersInjected(t *testing.T) {
	var gotSource, gotAuth, gotUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSource = r.Header.Get("X-Source")
		gotAuth = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
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

	p := NewProvider(
		"key",
		server.URL,
		"",
		WithUserAgent("PicoClaw/Test"),
		WithCustomHeaders(map[string]string{
			"X-Source":      "coding-plan",
			"Authorization": "Token custom-auth",
			"User-Agent":    "Custom-UA/1.0",
		}),
	)

	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"gpt-4o",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if gotSource != "coding-plan" {
		t.Fatalf("X-Source = %q, want %q", gotSource, "coding-plan")
	}
	if gotAuth != "Token custom-auth" {
		t.Fatalf("Authorization = %q, want %q", gotAuth, "Token custom-auth")
	}
	if gotUserAgent != "Custom-UA/1.0" {
		t.Fatalf("User-Agent = %q, want %q", gotUserAgent, "Custom-UA/1.0")
	}
}

func TestProviderChatStream_CustomHeadersInjected(t *testing.T) {
	var gotSource, gotAuth, gotUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSource = r.Header.Get("X-Source")
		gotAuth = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	p := NewProvider(
		"key",
		server.URL,
		"",
		WithUserAgent("PicoClaw/Test"),
		WithCustomHeaders(map[string]string{
			"X-Source":      "coding-plan",
			"Authorization": "Token stream-auth",
			"User-Agent":    "Custom-UA/Stream",
		}),
	)

	out, err := p.ChatStream(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"gpt-4o",
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}
	if out.Content != "ok" {
		t.Fatalf("Content = %q, want %q", out.Content, "ok")
	}
	if gotSource != "coding-plan" {
		t.Fatalf("X-Source = %q, want %q", gotSource, "coding-plan")
	}
	if gotAuth != "Token stream-auth" {
		t.Fatalf("Authorization = %q, want %q", gotAuth, "Token stream-auth")
	}
	if gotUserAgent != "Custom-UA/Stream" {
		t.Fatalf("User-Agent = %q, want %q", gotUserAgent, "Custom-UA/Stream")
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type errAfterDataReadCloser struct {
	data      []byte
	chunkSize int
	offset    int
}

func (r *errAfterDataReadCloser) Read(p []byte) (int, error) {
	if r.offset >= len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}

	n := r.chunkSize
	if n <= 0 || n > len(p) {
		n = len(p)
	}
	remaining := len(r.data) - r.offset
	if n > remaining {
		n = remaining
	}
	copy(p, r.data[r.offset:r.offset+n])
	r.offset += n
	return n, nil
}

func (r *errAfterDataReadCloser) Close() error {
	return nil
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

func TestSerializeMessages_PlainText(t *testing.T) {
	messages := []protocoltypes.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi", ReasoningContent: "thinking..."},
	}
	result := common.SerializeMessages(messages)

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}

	var msgs []map[string]any
	json.Unmarshal(data, &msgs)

	if msgs[0]["content"] != "hello" {
		t.Fatalf("expected plain string content, got %v", msgs[0]["content"])
	}
	if msgs[1]["reasoning_content"] != "thinking..." {
		t.Fatalf("reasoning_content not preserved, got %v", msgs[1]["reasoning_content"])
	}
}

func TestSerializeMessages_WithMedia(t *testing.T) {
	messages := []protocoltypes.Message{
		{Role: "user", Content: "describe this", Media: []string{"data:image/png;base64,abc123"}},
	}
	result := common.SerializeMessages(messages)

	data, _ := json.Marshal(result)
	var msgs []map[string]any
	json.Unmarshal(data, &msgs)

	content, ok := msgs[0]["content"].([]any)
	if !ok {
		t.Fatalf("expected array content for media message, got %T", msgs[0]["content"])
	}
	if len(content) != 2 {
		t.Fatalf("expected 2 content parts, got %d", len(content))
	}

	textPart := content[0].(map[string]any)
	if textPart["type"] != "text" || textPart["text"] != "describe this" {
		t.Fatalf("text part mismatch: %v", textPart)
	}

	imgPart := content[1].(map[string]any)
	if imgPart["type"] != "image_url" {
		t.Fatalf("expected image_url type, got %v", imgPart["type"])
	}
	imgURL := imgPart["image_url"].(map[string]any)
	if imgURL["url"] != "data:image/png;base64,abc123" {
		t.Fatalf("image url mismatch: %v", imgURL["url"])
	}
}

func TestSerializeMessages_MediaWithToolCallID(t *testing.T) {
	messages := []protocoltypes.Message{
		{Role: "tool", Content: "image result", Media: []string{"data:image/png;base64,xyz"}, ToolCallID: "call_1"},
	}
	result := common.SerializeMessages(messages)

	data, _ := json.Marshal(result)
	var msgs []map[string]any
	json.Unmarshal(data, &msgs)

	if msgs[0]["tool_call_id"] != "call_1" {
		t.Fatalf("tool_call_id not preserved with media, got %v", msgs[0]["tool_call_id"])
	}
	// Content should be multipart array
	if _, ok := msgs[0]["content"].([]any); !ok {
		t.Fatalf("expected array content, got %T", msgs[0]["content"])
	}
}

// chatWithCacheKey sets up a test server, sends a Chat request with prompt_cache_key,
// and returns the decoded request body for assertion.
func chatWithCacheKey(t *testing.T, apiBase string) map[string]any {
	t.Helper()
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
	p.apiBase = apiBase
	p.httpClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			r.URL, _ = url.Parse(server.URL + r.URL.Path)
			return http.DefaultTransport.RoundTrip(r)
		}),
	}

	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"test-model",
		map[string]any{"prompt_cache_key": "agent-main"},
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	return requestBody
}

func TestProviderChat_PromptCacheKeySentToOpenAI(t *testing.T) {
	body := chatWithCacheKey(t, "https://api.openai.com/v1")
	if body["prompt_cache_key"] != "agent-main" {
		t.Fatalf("prompt_cache_key = %v, want %q", body["prompt_cache_key"], "agent-main")
	}
}

func TestProviderChat_PromptCacheKeyOmittedForNonOpenAI(t *testing.T) {
	tests := []struct {
		name    string
		apiBase string
	}{
		{"mistral", "https://api.mistral.ai/v1"},
		{"gemini", "https://generativelanguage.googleapis.com/v1beta"},
		{"deepseek", "https://api.deepseek.com/v1"},
		{"groq", "https://api.groq.com/openai/v1"},
		{"minimax", "https://api.minimaxi.com/v1"},
		{"ollama_local", "http://localhost:11434/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := chatWithCacheKey(t, tt.apiBase)
			if _, exists := body["prompt_cache_key"]; exists {
				t.Fatalf("prompt_cache_key should NOT be sent to %s, but was included in request", tt.name)
			}
		})
	}
}

func TestSupportsPromptCacheKey(t *testing.T) {
	tests := []struct {
		apiBase string
		want    bool
	}{
		{"https://api.openai.com/v1", true},
		{"https://api.openai.com/v1/", true},
		{"https://myresource.openai.azure.com/openai/deployments/gpt-4", true},
		{"https://eastus.openai.azure.com/v1", true},
		{"https://api.mistral.ai/v1", false},
		{"https://generativelanguage.googleapis.com/v1beta", false},
		{"https://api.deepseek.com/v1", false},
		{"https://api.groq.com/openai/v1", false},
		{"http://localhost:11434/v1", false},
		{"https://openrouter.ai/api/v1", false},
		// Edge cases: proxy URLs with openai.com in path should NOT match
		{"https://my-proxy.com/api.openai.com/v1", false},
		{"https://proxy.example.com/openai.azure.com/v1", false},
		// Malformed or empty
		{"", false},
		{"not-a-url", false},
	}
	for _, tt := range tests {
		if got := supportsPromptCacheKey(tt.apiBase); got != tt.want {
			t.Errorf("supportsPromptCacheKey(%q) = %v, want %v", tt.apiBase, got, tt.want)
		}
	}
}

func TestBuildToolsList_NativeSearchAddsWebSearchPreview(t *testing.T) {
	tools := []ToolDefinition{
		{Type: "function", Function: ToolFunctionDefinition{Name: "read_file", Description: "read"}},
	}
	result := buildToolsList(tools, true)
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
	wsEntry, ok := result[1].(map[string]any)
	if !ok {
		t.Fatalf("web search entry is %T, want map[string]any", result[1])
	}
	if wsEntry["type"] != "web_search_preview" {
		t.Fatalf("type = %v, want web_search_preview", wsEntry["type"])
	}
}

func TestBuildToolsList_NativeSearchFiltersClientWebSearch(t *testing.T) {
	tools := []ToolDefinition{
		{Type: "function", Function: ToolFunctionDefinition{Name: "web_search", Description: "search"}},
		{Type: "function", Function: ToolFunctionDefinition{Name: "read_file", Description: "read"}},
	}
	result := buildToolsList(tools, true)
	for _, entry := range result {
		if td, ok := entry.(ToolDefinition); ok && strings.EqualFold(td.Function.Name, "web_search") {
			t.Fatal("client-side web_search should be filtered out when native search is enabled")
		}
	}
	if len(result) != 2 { // read_file + web_search_preview
		t.Fatalf("len(result) = %d, want 2 (read_file + web_search_preview)", len(result))
	}
}

func TestBuildToolsList_NoNativeSearchPassesThrough(t *testing.T) {
	tools := []ToolDefinition{
		{Type: "function", Function: ToolFunctionDefinition{Name: "web_search", Description: "search"}},
		{Type: "function", Function: ToolFunctionDefinition{Name: "read_file", Description: "read"}},
	}
	result := buildToolsList(tools, false)
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
}

func TestIsNativeSearchHost(t *testing.T) {
	tests := []struct {
		apiBase string
		want    bool
	}{
		{"https://api.openai.com/v1", true},
		{"https://myresource.openai.azure.com/openai/deployments/gpt-4", true},
		{"https://api.mistral.ai/v1", false},
		{"https://api.deepseek.com/v1", false},
		{"https://api.groq.com/openai/v1", false},
		{"http://localhost:11434/v1", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isNativeSearchHost(tt.apiBase); got != tt.want {
			t.Errorf("isNativeSearchHost(%q) = %v, want %v", tt.apiBase, got, tt.want)
		}
	}
}

func TestSupportsNativeSearch_OpenAI(t *testing.T) {
	p := NewProvider("key", "https://api.openai.com/v1", "")
	if !p.SupportsNativeSearch() {
		t.Fatal("OpenAI provider should support native search")
	}
}

func TestSupportsNativeSearch_NonOpenAI(t *testing.T) {
	p := NewProvider("key", "https://api.deepseek.com/v1", "")
	if p.SupportsNativeSearch() {
		t.Fatal("DeepSeek provider should not support native search")
	}
}

func TestProviderChat_NativeSearchToolInjected(t *testing.T) {
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
	p.apiBase = "https://api.openai.com/v1"
	p.httpClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			r.URL, _ = url.Parse(server.URL + r.URL.Path)
			return http.DefaultTransport.RoundTrip(r)
		}),
	}
	tools := []ToolDefinition{
		{Type: "function", Function: ToolFunctionDefinition{Name: "read_file", Description: "read"}},
	}
	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		tools,
		"gpt-5.4",
		map[string]any{"native_search": true},
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	toolsRaw, ok := requestBody["tools"].([]any)
	if !ok {
		t.Fatalf("tools is %T, want []any", requestBody["tools"])
	}
	if len(toolsRaw) != 2 {
		t.Fatalf("len(tools) = %d, want 2 (read_file + web_search_preview)", len(toolsRaw))
	}

	lastTool, ok := toolsRaw[1].(map[string]any)
	if !ok {
		t.Fatalf("last tool is %T, want map[string]any", toolsRaw[1])
	}
	if lastTool["type"] != "web_search_preview" {
		t.Fatalf("last tool type = %v, want web_search_preview", lastTool["type"])
	}
}

func TestProviderChat_NativeSearchNotInjectedWithoutOption(t *testing.T) {
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
	tools := []ToolDefinition{
		{Type: "function", Function: ToolFunctionDefinition{Name: "web_search", Description: "search"}},
	}
	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		tools,
		"gpt-5.4",
		map[string]any{},
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	toolsRaw, ok := requestBody["tools"].([]any)
	if !ok {
		t.Fatalf("tools is %T, want []any", requestBody["tools"])
	}
	if len(toolsRaw) != 1 {
		t.Fatalf("len(tools) = %d, want 1 (web_search only)", len(toolsRaw))
	}
}

// TestProviderChat_NativeSearchIgnoredOnNonOpenAI verifies that when native_search
// is true in options but the provider's apiBase is not OpenAI (e.g. fallback to DeepSeek),
// we do not inject web_search_preview to avoid API errors.
func TestProviderChat_NativeSearchIgnoredOnNonOpenAI(t *testing.T) {
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

	// Use server.URL so host is not api.openai.com — simulates DeepSeek/other provider
	p := NewProvider("key", server.URL, "")
	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"deepseek-chat",
		map[string]any{"native_search": true},
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	// Should not have tools at all (no tools passed, and we must not add web_search_preview)
	if toolsRaw, ok := requestBody["tools"]; ok {
		t.Fatalf("tools should be omitted for non-OpenAI when only native_search was requested, got %v", toolsRaw)
	}
}

func TestSerializeMessages_StripsSystemParts(t *testing.T) {
	messages := []protocoltypes.Message{
		{
			Role:    "system",
			Content: "you are helpful",
			SystemParts: []protocoltypes.ContentBlock{
				{Type: "text", Text: "you are helpful"},
			},
		},
	}
	result := common.SerializeMessages(messages)

	data, _ := json.Marshal(result)
	raw := string(data)
	if strings.Contains(raw, "system_parts") {
		t.Fatal("system_parts should not appear in serialized output")
	}
}
