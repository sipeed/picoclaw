package anthropicprovider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	anthropicoption "github.com/anthropics/anthropic-sdk-go/option"
)

func TestBuildParams_BasicMessage(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}
	params, err := buildParams(messages, nil, "claude-sonnet-4.6", map[string]any{
		"max_tokens": 1024,
	})
	if err != nil {
		t.Fatalf("buildParams() error: %v", err)
	}
	if string(params.Model) != "claude-sonnet-4.6" {
		t.Errorf("Model = %q, want %q", params.Model, "claude-sonnet-4.6")
	}
	if params.MaxTokens != 1024 {
		t.Errorf("MaxTokens = %d, want 1024", params.MaxTokens)
	}
	if len(params.Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1", len(params.Messages))
	}
}

func TestBuildParams_SystemMessage(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "You are helpful"},
		{Role: "user", Content: "Hi"},
	}
	params, err := buildParams(messages, nil, "claude-sonnet-4.6", map[string]any{})
	if err != nil {
		t.Fatalf("buildParams() error: %v", err)
	}
	if len(params.System) != 1 {
		t.Fatalf("len(System) = %d, want 1", len(params.System))
	}
	if params.System[0].Text != "You are helpful" {
		t.Errorf("System[0].Text = %q, want %q", params.System[0].Text, "You are helpful")
	}
	if len(params.Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1", len(params.Messages))
	}
}

func TestBuildParams_ToolCallMessage(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "What's the weather?"},
		{
			Role:    "assistant",
			Content: "",
			ToolCalls: []ToolCall{
				{
					ID:        "call_1",
					Name:      "get_weather",
					Arguments: map[string]any{"city": "SF"},
				},
			},
		},
		{Role: "tool", Content: `{"temp": 72}`, ToolCallID: "call_1"},
	}
	params, err := buildParams(messages, nil, "claude-sonnet-4.6", map[string]any{})
	if err != nil {
		t.Fatalf("buildParams() error: %v", err)
	}
	if len(params.Messages) != 3 {
		t.Fatalf("len(Messages) = %d, want 3", len(params.Messages))
	}
}

func TestBuildParams_WithTools(t *testing.T) {
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
					"required": []any{"city"},
				},
			},
		},
	}
	params, err := buildParams([]Message{{Role: "user", Content: "Hi"}}, tools, "claude-sonnet-4.6", map[string]any{})
	if err != nil {
		t.Fatalf("buildParams() error: %v", err)
	}
	if len(params.Tools) != 1 {
		t.Fatalf("len(Tools) = %d, want 1", len(params.Tools))
	}
}

func TestParseResponse_TextOnly(t *testing.T) {
	resp := &anthropic.Message{
		Content: []anthropic.ContentBlockUnion{},
		Usage: anthropic.Usage{
			InputTokens:  10,
			OutputTokens: 20,
		},
	}
	result := parseResponse(resp)
	if result.Usage.PromptTokens != 10 {
		t.Errorf("PromptTokens = %d, want 10", result.Usage.PromptTokens)
	}
	if result.Usage.CompletionTokens != 20 {
		t.Errorf("CompletionTokens = %d, want 20", result.Usage.CompletionTokens)
	}
	if result.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", result.FinishReason, "stop")
	}
}

func TestParseResponse_StopReasons(t *testing.T) {
	tests := []struct {
		stopReason anthropic.StopReason
		want       string
	}{
		{anthropic.StopReasonEndTurn, "stop"},
		{anthropic.StopReasonMaxTokens, "length"},
		{anthropic.StopReasonToolUse, "tool_calls"},
	}
	for _, tt := range tests {
		resp := &anthropic.Message{
			StopReason: tt.stopReason,
		}
		result := parseResponse(resp)
		if result.FinishReason != tt.want {
			t.Errorf("StopReason %q: FinishReason = %q, want %q", tt.stopReason, result.FinishReason, tt.want)
		}
	}
}

func TestProvider_ChatRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var reqBody map[string]any
		json.NewDecoder(r.Body).Decode(&reqBody)

		resp := map[string]any{
			"id":          "msg_test",
			"type":        "message",
			"role":        "assistant",
			"model":       reqBody["model"],
			"stop_reason": "end_turn",
			"content": []map[string]any{
				{"type": "text", "text": "Hello! How can I help you?"},
			},
			"usage": map[string]any{
				"input_tokens":  15,
				"output_tokens": 8,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewProviderWithClient(createAnthropicTestClient(server.URL, "test-token"))
	messages := []Message{{Role: "user", Content: "Hello"}}
	resp, err := provider.Chat(t.Context(), messages, nil, "claude-sonnet-4.6", map[string]any{"max_tokens": 1024})
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if resp.Content != "Hello! How can I help you?" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello! How can I help you?")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
	if resp.Usage.PromptTokens != 15 {
		t.Errorf("PromptTokens = %d, want 15", resp.Usage.PromptTokens)
	}
}

func TestProvider_GetDefaultModel(t *testing.T) {
	p := NewProvider("test-token")
	if got := p.GetDefaultModel(); got != "claude-sonnet-4.6" {
		t.Errorf("GetDefaultModel() = %q, want %q", got, "claude-sonnet-4.6")
	}
}

func TestProvider_NewProviderWithBaseURL_NormalizesV1Suffix(t *testing.T) {
	p := NewProviderWithBaseURL("token", "https://api.anthropic.com/v1/")
	if got := p.BaseURL(); got != "https://api.anthropic.com" {
		t.Fatalf("BaseURL() = %q, want %q", got, "https://api.anthropic.com")
	}
}

func TestProvider_ChatUsesTokenSource(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		atomic.AddInt32(&requests, 1)

		if got := r.Header.Get("Authorization"); got != "Bearer refreshed-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var reqBody map[string]any
		json.NewDecoder(r.Body).Decode(&reqBody)

		resp := map[string]any{
			"id":          "msg_test",
			"type":        "message",
			"role":        "assistant",
			"model":       reqBody["model"],
			"stop_reason": "end_turn",
			"content": []map[string]any{
				{"type": "text", "text": "ok"},
			},
			"usage": map[string]any{
				"input_tokens":  1,
				"output_tokens": 1,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProviderWithTokenSourceAndBaseURL("stale-token", func() (string, error) {
		return "refreshed-token", nil
	}, server.URL)

	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hello"}},
		nil,
		"claude-sonnet-4.6",
		map[string]any{},
	)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

func createAnthropicTestClient(baseURL, token string) *anthropic.Client {
	c := anthropic.NewClient(
		anthropicoption.WithAuthToken(token),
		anthropicoption.WithBaseURL(baseURL),
	)
	return &c
}
