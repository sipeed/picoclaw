package openai_sdk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOpenAISDKProvider_Chat_BasicContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q, want /chat/completions", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body["model"] != "gpt-4o" {
			t.Fatalf("request model = %v, want gpt-4o", body["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-123",
			"object":"chat.completion",
			"created":1,
			"model":"gpt-4o",
			"choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"hello"}}],
			"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12}
		}`))
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	resp, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"gpt-4o",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if resp.Content != "hello" {
		t.Fatalf("Content = %q, want %q", resp.Content, "hello")
	}
	if resp.FinishReason != "stop" {
		t.Fatalf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 12 {
		t.Fatalf("Usage.TotalTokens = %+v, want 12", resp.Usage)
	}
}

func TestOpenAISDKProvider_Chat_MessageAndToolMapping(t *testing.T) {
	var body map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-123",
			"object":"chat.completion",
			"created":1,
			"model":"gpt-4o",
			"choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"ok"}}]
		}`))
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	_, err := p.Chat(
		t.Context(),
		[]Message{
			{Role: "system", Content: "sys"},
			{Role: "assistant", Content: "thinking", ToolCalls: []ToolCall{
				{
					ID:   "call_1",
					Name: "sum",
					Arguments: map[string]any{
						"a": 1,
						"b": 2,
					},
				},
			}},
			{Role: "tool", ToolCallID: "call_1", Content: `{"result":3}`},
			{Role: "user", Content: "hi"},
		},
		[]ToolDefinition{
			{
				Type: "function",
				Function: ToolFunctionDefinition{
					Name:        "sum",
					Description: "sum two integers",
					Parameters: map[string]any{
						"type": "object",
					},
				},
			},
		},
		"openai/gpt-4o",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if body["model"] != "gpt-4o" {
		t.Fatalf("request model = %v, want gpt-4o", body["model"])
	}

	msgs, ok := body["messages"].([]any)
	if !ok {
		t.Fatalf("messages type = %T, want []any", body["messages"])
	}
	if len(msgs) != 4 {
		t.Fatalf("messages length = %d, want 4", len(msgs))
	}

	assistantMsg := msgs[1].(map[string]any)
	if assistantMsg["role"] != "assistant" {
		t.Fatalf("assistant role = %v, want assistant", assistantMsg["role"])
	}
	toolCalls, ok := assistantMsg["tool_calls"].([]any)
	if !ok || len(toolCalls) != 1 {
		t.Fatalf("assistant tool_calls = %#v, want len 1", assistantMsg["tool_calls"])
	}
	toolMsg := msgs[2].(map[string]any)
	if toolMsg["role"] != "tool" || toolMsg["tool_call_id"] != "call_1" {
		t.Fatalf("tool message mismatch: %#v", toolMsg)
	}

	tools, ok := body["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %#v, want len 1", body["tools"])
	}
}

func TestOpenAISDKProvider_Chat_ParsesResponseToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-123",
			"object":"chat.completion",
			"created":1,
			"model":"gpt-4o",
			"choices":[
				{
					"index":0,
					"finish_reason":"tool_calls",
					"message":{
						"role":"assistant",
						"content":"",
						"tool_calls":[
							{
								"id":"call_1",
								"type":"function",
								"function":{"name":"sum","arguments":"{\"a\":1,\"b\":2}"}
							}
						]
					}
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
		"gpt-4o",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls length = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "sum" {
		t.Fatalf("ToolCalls[0].Name = %q, want sum", resp.ToolCalls[0].Name)
	}
	if resp.ToolCalls[0].Arguments["a"] != float64(1) {
		t.Fatalf("ToolCalls[0].Arguments = %#v", resp.ToolCalls[0].Arguments)
	}
}

func TestOpenAISDKProvider_Chat_OptionsMapping(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
					"id":"chatcmpl-123",
					"object":"chat.completion",
					"created":1,
					"model":"gpt-4o",
					"choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"ok"}}]
				}`))
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"gpt-4o",
		map[string]any{
			"max_tokens":       123,
			"temperature":      0.2,
			"prompt_cache_key": "agent-1",
		},
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if body["max_completion_tokens"] != float64(123) {
		t.Fatalf("max_completion_tokens = %v, want 123", body["max_completion_tokens"])
	}
	if _, ok := body["max_tokens"]; ok {
		t.Fatalf("did not expect max_tokens in request body")
	}
	if body["temperature"] != 0.2 {
		t.Fatalf("temperature = %v, want 0.2", body["temperature"])
	}
	if body["prompt_cache_key"] != "agent-1" {
		t.Fatalf("prompt_cache_key = %v, want agent-1", body["prompt_cache_key"])
	}
}

func TestOpenAISDKProvider_Chat_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(700 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-123",
			"object":"chat.completion",
			"created":1,
			"model":"gpt-4o",
			"choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"ok"}}]
		}`))
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "", WithRequestTimeout(200*time.Millisecond))
	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"gpt-4o",
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

func TestOpenAISDKProvider_ProxyConfig(t *testing.T) {
	p := NewProvider("test-key", "http://example.com/v1", "http://127.0.0.1:10080")
	transport, ok := p.httpClient.Transport.(*http.Transport)
	if !ok || transport.Proxy == nil {
		t.Fatalf("expected proxy transport, got %#v", p.httpClient.Transport)
	}
}

func TestOpenAISDKProvider_Chat_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"bad request"}}`))
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"gpt-4o",
		nil,
	)
	if err == nil {
		t.Fatal("Chat() expected error")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "status=400") {
		t.Fatalf("error = %q, want status=400", errMsg)
	}
}
