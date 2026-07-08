package anthropicprovider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
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
	if params.Model != "claude-sonnet-4-6" {
		t.Errorf("Model = %q, want %q", params.Model, "claude-sonnet-4-6")
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

// TestBuildParams_ToolCallFunctionFallback verifies that tool calls whose
// runtime-only fields were lost in a JSON round-trip through the session store
// (ToolCall.Name/Arguments are json:"-"; only ToolCall.Function survives) fall
// back to Function.Name / Function.Arguments, so the tool_use block is still
// emitted and its tool_result pair stays intact. Without the fallback the
// tool_use is skipped and the orphaned tool_result 400s at the API
// ("unexpected tool_use_id found in tool_result blocks").
func TestBuildParams_ToolCallFunctionFallback(t *testing.T) {
	tests := []struct {
		name         string
		toolCall     ToolCall
		wantSkipped  bool
		wantToolName string
		wantInput    map[string]any
	}{
		{
			name: "deserialized history shape falls back to Function fields",
			toolCall: ToolCall{
				ID:        "toolu-fallback-1",
				Name:      "",
				Arguments: nil,
				Function:  &FunctionCall{Name: "x", Arguments: `{"a":1}`},
			},
			wantToolName: "x",
			wantInput:    map[string]any{"a": float64(1)},
		},
		{
			name: "runtime shape with Name set and Function nil still works",
			toolCall: ToolCall{
				ID:        "toolu-runtime-1",
				Name:      "y",
				Arguments: map[string]any{"b": 2},
			},
			wantToolName: "y",
			wantInput:    map[string]any{"b": 2},
		},
		{
			name: "both Name and Function.Name empty is skipped",
			toolCall: ToolCall{
				ID:       "toolu-empty-1",
				Name:     "",
				Function: &FunctionCall{Name: "", Arguments: `{"c":3}`},
			},
			wantSkipped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := []Message{
				{Role: "user", Content: "run the tool"},
				{Role: "assistant", Content: "", ToolCalls: []ToolCall{tt.toolCall}},
				{Role: "tool", ToolCallID: tt.toolCall.ID, Content: "result"},
			}

			params, err := buildParams(messages, nil, "claude-sonnet-4.6", map[string]any{})
			if err != nil {
				t.Fatalf("buildParams() error: %v", err)
			}
			if len(params.Messages) != 3 {
				t.Fatalf("len(Messages) = %d, want 3", len(params.Messages))
			}

			assistantMsg := params.Messages[1]
			var toolUses []*anthropic.ToolUseBlockParam
			for _, block := range assistantMsg.Content {
				if block.OfToolUse != nil {
					toolUses = append(toolUses, block.OfToolUse)
				}
			}

			// The tool_result in the following user message always carries the
			// original ID; look it up once for both branches.
			toolResultMsg := params.Messages[2]
			if len(toolResultMsg.Content) != 1 || toolResultMsg.Content[0].OfToolResult == nil {
				t.Fatalf("message after assistant = %+v, want single tool_result block", toolResultMsg.Content)
			}
			toolResult := toolResultMsg.Content[0].OfToolResult

			if tt.wantSkipped {
				if len(toolUses) != 0 {
					t.Fatalf("tool_use blocks = %d, want 0 (tool call skipped)", len(toolUses))
				}
				// Note: matching current behavior, the orphaned tool_result is
				// still emitted even though its tool_use block was skipped.
				if toolResult.ToolUseID != tt.toolCall.ID {
					t.Fatalf("orphaned tool_result ToolUseID = %q, want %q",
						toolResult.ToolUseID, tt.toolCall.ID)
				}
				return
			}

			// (a) tool_use block emitted with resolved name, id, and parsed input.
			if len(toolUses) != 1 {
				t.Fatalf("tool_use blocks = %d, want 1", len(toolUses))
			}
			toolUse := toolUses[0]
			if toolUse.Name != tt.wantToolName {
				t.Errorf("tool_use Name = %q, want %q", toolUse.Name, tt.wantToolName)
			}
			if toolUse.ID != tt.toolCall.ID {
				t.Errorf("tool_use ID = %q, want %q", toolUse.ID, tt.toolCall.ID)
			}
			gotInput, ok := toolUse.Input.(map[string]any)
			if !ok || !reflect.DeepEqual(gotInput, tt.wantInput) {
				t.Errorf("tool_use Input = %#v, want %#v", toolUse.Input, tt.wantInput)
			}

			// (b) the following user message's tool_result references the same
			// id as the tool_use block — the pair is intact.
			if toolResult.ToolUseID != toolUse.ID {
				t.Errorf("tool_result ToolUseID = %q, want %q (paired with tool_use)",
					toolResult.ToolUseID, toolUse.ID)
			}
		})
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

func TestProvider_ChatStreamingRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer refreshed-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer refreshed-token")
		}
		if got := r.Header.Get("Anthropic-Beta"); got != anthropicBetaHeader {
			t.Errorf("Anthropic-Beta = %q, want %q", got, anthropicBetaHeader)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		events := []string{
			"event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_stream\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[],\"model\":\"claude-sonnet-4-6\",\"stop_reason\":null,\"usage\":{\"input_tokens\":12,\"output_tokens\":0}}}\n\n",
			"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\" world\"}}\n\n",
			"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n",
			"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":5}}\n\n",
			"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n",
		}
		for _, e := range events {
			w.Write([]byte(e))
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer server.Close()

	p := NewProviderWithTokenSourceAndBaseURL("stale-token", func() (string, error) {
		return "refreshed-token", nil
	}, server.URL)

	resp, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "Hello"}},
		nil,
		"claude-sonnet-4.6",
		map[string]any{},
	)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if resp.Content != "Hello world" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello world")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
	if resp.Usage.CompletionTokens != 5 {
		t.Errorf("CompletionTokens = %d, want 5", resp.Usage.CompletionTokens)
	}
}

func createAnthropicTestClient(baseURL, token string) *anthropic.Client {
	c := anthropic.NewClient(
		anthropicoption.WithAuthToken(token),
		anthropicoption.WithBaseURL(baseURL),
	)
	return &c
}
