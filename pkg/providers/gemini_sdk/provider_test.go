package gemini_sdk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGeminiSDKProvider_Chat_BasicContentAndOptions(t *testing.T) {
	var (
		requestPath string
		requestBody map[string]any
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates":[
				{
					"content":{"parts":[{"text":"hello from gemini"}],"role":"model"},
					"finishReason":"STOP"
				}
			],
			"usageMetadata":{"promptTokenCount":12,"candidatesTokenCount":3,"totalTokenCount":15}
		}`))
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	resp, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"google/gemini-2.5-flash",
		map[string]any{
			"max_tokens":       123,
			"temperature":      0.2,
			"prompt_cache_key": "ignored-on-gemini",
		},
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if requestPath != "/v1beta/models/gemini-2.5-flash:generateContent" {
		t.Fatalf("path = %q, want /v1beta/models/gemini-2.5-flash:generateContent", requestPath)
	}
	if got := readNestedNumber(requestBody, "generationConfig", "maxOutputTokens"); got != 123 {
		t.Fatalf("generationConfig.maxOutputTokens = %v, want 123", got)
	}
	if got := readNestedNumber(requestBody, "generationConfig", "temperature"); got != 0.2 {
		t.Fatalf("generationConfig.temperature = %v, want 0.2", got)
	}
	if _, ok := requestBody["prompt_cache_key"]; ok {
		t.Fatalf("did not expect prompt_cache_key in Gemini request body")
	}

	if resp.Content != "hello from gemini" {
		t.Fatalf("Content = %q, want %q", resp.Content, "hello from gemini")
	}
	if resp.FinishReason != "stop" {
		t.Fatalf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 15 {
		t.Fatalf("Usage.TotalTokens = %+v, want 15", resp.Usage)
	}
}

func TestGeminiSDKProvider_Chat_ParsesToolCallsAndThoughtSignature(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates":[
				{
					"content":{
						"parts":[
							{
								"functionCall":{"name":"sum","args":{"a":1,"b":2}},
								"thoughtSignature":"YWJj"
							}
						],
						"role":"model"
					},
					"finishReason":"STOP"
				}
			]
		}`))
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	resp, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"gemini-2.5-flash",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if resp.FinishReason != "tool_calls" {
		t.Fatalf("FinishReason = %q, want %q", resp.FinishReason, "tool_calls")
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(resp.ToolCalls))
	}

	tc := resp.ToolCalls[0]
	if tc.Name != "sum" {
		t.Fatalf("ToolCalls[0].Name = %q, want sum", tc.Name)
	}
	if tc.Arguments["a"] != float64(1) {
		t.Fatalf("ToolCalls[0].Arguments = %#v", tc.Arguments)
	}
	if tc.Function == nil {
		t.Fatalf("ToolCalls[0].Function = nil, want non-nil")
	}
	if tc.Function.ThoughtSignature != "YWJj" {
		t.Fatalf("Function.ThoughtSignature = %q, want %q", tc.Function.ThoughtSignature, "YWJj")
	}
	if tc.ExtraContent == nil || tc.ExtraContent.Google == nil {
		t.Fatalf("ExtraContent.Google = %#v, want non-nil", tc.ExtraContent)
	}
	if tc.ExtraContent.Google.ThoughtSignature != "YWJj" {
		t.Fatalf("ExtraContent.Google.ThoughtSignature = %q, want %q", tc.ExtraContent.Google.ThoughtSignature, "YWJj")
	}
}

func TestGeminiSDKProvider_Chat_HistoryThoughtSignaturePreference(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates":[
				{
					"content":{"parts":[{"text":"ok"}],"role":"model"},
					"finishReason":"STOP"
				}
			]
		}`))
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	_, err := p.Chat(
		t.Context(),
		[]Message{
			{
				Role: "assistant",
				ToolCalls: []ToolCall{
					{
						ID:        "call_1",
						Name:      "sum",
						Arguments: map[string]any{"a": 1},
						Function: &FunctionCall{
							Name:             "sum",
							Arguments:        `{"a":1}`,
							ThoughtSignature: "ZnVuY3Rpb24=",
						},
						ExtraContent: &ExtraContent{
							Google: &GoogleExtra{
								ThoughtSignature: "ZXh0cmE=",
							},
						},
					},
				},
			},
			{
				Role:       "tool",
				ToolCallID: "call_1",
				Content:    `{"result": 1}`,
			},
			{
				Role:    "user",
				Content: "continue",
			},
		},
		nil,
		"gemini-2.5-flash",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	contents, ok := requestBody["contents"].([]any)
	if !ok {
		t.Fatalf("contents type = %T, want []any", requestBody["contents"])
	}
	if len(contents) < 1 {
		t.Fatalf("contents length = %d, want >= 1", len(contents))
	}

	assistant := contents[0].(map[string]any)
	parts, ok := assistant["parts"].([]any)
	if !ok || len(parts) == 0 {
		t.Fatalf("assistant parts = %#v, want non-empty", assistant["parts"])
	}
	firstPart := parts[0].(map[string]any)
	if got := stringValue(firstPart, "thoughtSignature"); got != "ZXh0cmE=" {
		t.Fatalf("thoughtSignature = %q, want extra_content value %q", got, "ZXh0cmE=")
	}
}

func TestGeminiSDKProvider_Chat_HistoryThoughtSignatureFallbackToFunction(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates":[
				{
					"content":{"parts":[{"text":"ok"}],"role":"model"},
					"finishReason":"STOP"
				}
			]
		}`))
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	_, err := p.Chat(
		t.Context(),
		[]Message{
			{
				Role: "assistant",
				ToolCalls: []ToolCall{
					{
						ID:        "call_1",
						Name:      "sum",
						Arguments: map[string]any{"a": 1},
						Function: &FunctionCall{
							Name:             "sum",
							Arguments:        `{"a":1}`,
							ThoughtSignature: "ZnVuY3Rpb24=",
						},
					},
				},
			},
			{
				Role:       "tool",
				ToolCallID: "call_1",
				Content:    `{"result": 1}`,
			},
			{
				Role:    "user",
				Content: "continue",
			},
		},
		nil,
		"gemini-2.5-flash",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	contents, ok := requestBody["contents"].([]any)
	if !ok {
		t.Fatalf("contents type = %T, want []any", requestBody["contents"])
	}
	if len(contents) < 1 {
		t.Fatalf("contents length = %d, want >= 1", len(contents))
	}

	assistant := contents[0].(map[string]any)
	parts, ok := assistant["parts"].([]any)
	if !ok || len(parts) == 0 {
		t.Fatalf("assistant parts = %#v, want non-empty", assistant["parts"])
	}
	firstPart := parts[0].(map[string]any)
	if got := stringValue(firstPart, "thoughtSignature"); got != "ZnVuY3Rpb24=" {
		t.Fatalf("thoughtSignature = %q, want function value %q", got, "ZnVuY3Rpb24=")
	}
}

func TestGeminiSDKProvider_Chat_APIBaseWithVersionPath(t *testing.T) {
	var requestPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates":[
				{
					"content":{"parts":[{"text":"ok"}],"role":"model"},
					"finishReason":"STOP"
				}
			]
		}`))
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL+"/v1beta", "")
	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"gemini-2.5-flash",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if requestPath != "/v1beta/models/gemini-2.5-flash:generateContent" {
		t.Fatalf("path = %q, want /v1beta/models/gemini-2.5-flash:generateContent", requestPath)
	}
}

func TestGeminiSDKProvider_ProxyConfig(t *testing.T) {
	p := NewProvider("test-key", "http://example.com", "http://127.0.0.1:10080")
	transport, ok := p.httpClient.Transport.(*http.Transport)
	if !ok || transport.Proxy == nil {
		t.Fatalf("expected proxy transport, got %#v", p.httpClient.Transport)
	}
}

func TestGeminiSDKProvider_Chat_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(700 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates":[
				{
					"content":{"parts":[{"text":"ok"}],"role":"model"},
					"finishReason":"STOP"
				}
			]
		}`))
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "", WithRequestTimeout(200*time.Millisecond))
	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"gemini-2.5-flash",
		nil,
	)
	if err == nil {
		t.Fatalf("Chat() error = nil, want timeout")
	}
	if !strings.Contains(err.Error(), "timeout") &&
		!strings.Contains(err.Error(), "deadline exceeded") &&
		!strings.Contains(err.Error(), "Client.Timeout exceeded") {
		t.Fatalf("timeout error = %q", err.Error())
	}
}

func TestGeminiSDKProvider_Chat_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":400,"message":"bad request","status":"INVALID_ARGUMENT"}}`))
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"gemini-2.5-flash",
		nil,
	)
	if err == nil {
		t.Fatal("Chat() expected error")
	}
	if !strings.Contains(err.Error(), "status=400") {
		t.Fatalf("error = %q, want status=400", err.Error())
	}
}

func readNestedNumber(v map[string]any, keys ...string) float64 {
	current := any(v)
	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return 0
		}
		current = m[key]
	}
	num, _ := current.(float64)
	return num
}

func stringValue(v map[string]any, key string) string {
	s, _ := v[key].(string)
	return s
}
