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

func TestProviderChat_StripsGroqOllamaDeepseekVivgridPrefixes(t *testing.T) {
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
			name:      "strips deepseek prefix",
			input:     "deepseek/deepseek-chat",
			wantModel: "deepseek-chat",
		},
		{
			name:      "strips vivgrid prefix",
			input:     "vivgrid/auto",
			wantModel: "auto",
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
	if got := normalizeModel("vivgrid/managed", "https://api.vivgrid.com/v1"); got != "managed" {
		t.Fatalf("normalizeModel(vivgrid) = %q, want %q", got, "managed")
	}
	if got := normalizeModel("vivgrid/auto", "https://api.vivgrid.com/v1"); got != "auto" {
		t.Fatalf("normalizeModel(vivgrid auto) = %q, want %q", got, "auto")
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
	result := serializeMessages(messages)

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
	result := serializeMessages(messages)

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
	result := serializeMessages(messages)

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
	result := serializeMessages(messages)

	data, _ := json.Marshal(result)
	raw := string(data)
	if strings.Contains(raw, "system_parts") {
		t.Fatal("system_parts should not appear in serialized output")
	}
}
