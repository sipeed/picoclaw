package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openai/openai-go/v3"
	openaiopt "github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

func TestBuildCodexParams_BasicMessage(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}
	params := buildCodexParams(messages, nil, "gpt-4o", map[string]any{
		"max_tokens":  2048,
		"temperature": 0.7,
	}, true)
	if params.Model != "gpt-4o" {
		t.Errorf("Model = %q, want %q", params.Model, "gpt-4o")
	}
	if !params.Instructions.Valid() {
		t.Fatal("Instructions should be set")
	}
	if params.Instructions.Or("") != defaultCodexInstructions {
		t.Errorf("Instructions = %q, want %q", params.Instructions.Or(""), defaultCodexInstructions)
	}
	if params.MaxOutputTokens.Valid() {
		t.Fatalf("MaxOutputTokens should not be set for Codex backend")
	}
}

func TestBuildCodexParams_SystemAsInstructions(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "You are helpful"},
		{Role: "user", Content: "Hi"},
	}
	params := buildCodexParams(messages, nil, "gpt-4o", map[string]any{}, true)
	if !params.Instructions.Valid() {
		t.Fatal("Instructions should be set")
	}
	if params.Instructions.Or("") != "You are helpful" {
		t.Errorf("Instructions = %q, want %q", params.Instructions.Or(""), "You are helpful")
	}
}

func TestBuildCodexParams_ToolCallConversation(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "What's the weather?"},
		{
			Role: "assistant",
			ToolCalls: []ToolCall{
				{ID: "call_1", Name: "get_weather", Arguments: map[string]any{"city": "SF"}},
			},
		},
		{Role: "tool", Content: `{"temp": 72}`, ToolCallID: "call_1"},
	}
	params := buildCodexParams(messages, nil, "gpt-4o", map[string]any{}, false)
	if params.Input.OfInputItemList == nil {
		t.Fatal("Input.OfInputItemList should not be nil")
	}
	if len(params.Input.OfInputItemList) != 3 {
		t.Errorf("len(Input items) = %d, want 3", len(params.Input.OfInputItemList))
	}
}

func TestBuildCodexParams_ToolCallFunctionFallback(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "Read a file"},
		{
			Role: "assistant",
			ToolCalls: []ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: &FunctionCall{
						Name:      "read_file",
						Arguments: `{"path":"README.md"}`,
					},
				},
			},
		},
		{Role: "tool", Content: "ok", ToolCallID: "call_1"},
	}

	params := buildCodexParams(messages, nil, "gpt-4o", map[string]any{}, false)
	if params.Input.OfInputItemList == nil {
		t.Fatal("Input.OfInputItemList should not be nil")
	}
	if len(params.Input.OfInputItemList) != 3 {
		t.Fatalf("len(Input items) = %d, want 3", len(params.Input.OfInputItemList))
	}

	fc := params.Input.OfInputItemList[1].OfFunctionCall
	if fc == nil {
		t.Fatal("assistant tool call should be converted to function_call input item")
	}
	if fc.Name != "read_file" {
		t.Errorf("Function call name = %q, want %q", fc.Name, "read_file")
	}
	if fc.Arguments != `{"path":"README.md"}` {
		t.Errorf("Function call arguments = %q, want %q", fc.Arguments, `{"path":"README.md"}`)
	}
}

func TestBuildCodexParams_WithTools(t *testing.T) {
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "get_weather",
				Description: "Get weather",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"city": map[string]any{"type": "string"},
					},
				},
			},
		},
	}
	params := buildCodexParams([]Message{{Role: "user", Content: "Hi"}}, tools, "gpt-4o", map[string]any{}, false)
	if len(params.Tools) != 1 {
		t.Fatalf("len(Tools) = %d, want 1", len(params.Tools))
	}
	if params.Tools[0].OfFunction == nil {
		t.Fatal("Tool should be a function tool")
	}
	if params.Tools[0].OfFunction.Name != "get_weather" {
		t.Errorf("Tool name = %q, want %q", params.Tools[0].OfFunction.Name, "get_weather")
	}
}

func TestBuildCodexParams_StoreIsFalse(t *testing.T) {
	params := buildCodexParams([]Message{{Role: "user", Content: "Hi"}}, nil, "gpt-4o", map[string]any{}, false)
	if !params.Store.Valid() || params.Store.Or(true) != false {
		t.Error("Store should be explicitly set to false")
	}
}

func TestBuildCodexParams_DefaultWebSearchEnabled(t *testing.T) {
	params := buildCodexParams([]Message{{Role: "user", Content: "Hi"}}, nil, "gpt-4o", map[string]any{}, true)
	if len(params.Tools) != 1 {
		t.Fatalf("len(Tools) = %d, want 1", len(params.Tools))
	}
	if params.Tools[0].OfWebSearch == nil {
		t.Fatal("Tool should include built-in web_search")
	}
	if params.Tools[0].OfWebSearch.Type != responses.WebSearchToolTypeWebSearch {
		t.Errorf(
			"Web search tool type = %q, want %q",
			params.Tools[0].OfWebSearch.Type,
			responses.WebSearchToolTypeWebSearch,
		)
	}
}

func TestBuildCodexParams_WebSearchFunctionReplacedWithBuiltin(t *testing.T) {
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "web_search",
				Description: "local web search",
				Parameters: map[string]any{
					"type": "object",
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "read_file",
				Description: "read file",
				Parameters: map[string]any{
					"type": "object",
				},
			},
		},
	}

	params := buildCodexParams([]Message{{Role: "user", Content: "Hi"}}, tools, "gpt-4o", map[string]any{}, true)
	if len(params.Tools) != 2 {
		t.Fatalf("len(Tools) = %d, want 2", len(params.Tools))
	}
	if params.Tools[0].OfFunction == nil || params.Tools[0].OfFunction.Name != "read_file" {
		t.Fatalf("first tool should be function read_file, got %#v", params.Tools[0])
	}
	if params.Tools[1].OfWebSearch == nil {
		t.Fatalf("second tool should be built-in web_search, got %#v", params.Tools[1])
	}
}

func TestParseCodexResponse_TextOutput(t *testing.T) {
	respJSON := `{
		"id": "resp_test",
		"object": "response",
		"status": "completed",
		"output": [
			{
				"id": "msg_1",
				"type": "message",
				"role": "assistant",
				"status": "completed",
				"content": [
					{"type": "output_text", "text": "Hello there!"}
				]
			}
		],
		"usage": {
			"input_tokens": 10,
			"output_tokens": 5,
			"total_tokens": 15,
			"input_tokens_details": {"cached_tokens": 0},
			"output_tokens_details": {"reasoning_tokens": 0}
		}
	}`

	var resp responses.Response
	if err := json.Unmarshal([]byte(respJSON), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	result := parseCodexResponse(&resp)
	if result.Content != "Hello there!" {
		t.Errorf("Content = %q, want %q", result.Content, "Hello there!")
	}
	if result.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", result.FinishReason, "stop")
	}
	if result.Usage.TotalTokens != 15 {
		t.Errorf("TotalTokens = %d, want 15", result.Usage.TotalTokens)
	}
}

func TestParseCodexResponse_FunctionCall(t *testing.T) {
	respJSON := `{
		"id": "resp_test",
		"object": "response",
		"status": "completed",
		"output": [
			{
				"id": "fc_1",
				"type": "function_call",
				"call_id": "call_abc",
				"name": "get_weather",
				"arguments": "{\"city\":\"SF\"}",
				"status": "completed"
			}
		],
		"usage": {
			"input_tokens": 10,
			"output_tokens": 8,
			"total_tokens": 18,
			"input_tokens_details": {"cached_tokens": 0},
			"output_tokens_details": {"reasoning_tokens": 0}
		}
	}`

	var resp responses.Response
	if err := json.Unmarshal([]byte(respJSON), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	result := parseCodexResponse(&resp)
	if len(result.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(result.ToolCalls))
	}
	tc := result.ToolCalls[0]
	if tc.Name != "get_weather" {
		t.Errorf("ToolCall.Name = %q, want %q", tc.Name, "get_weather")
	}
	if tc.ID != "call_abc" {
		t.Errorf("ToolCall.ID = %q, want %q", tc.ID, "call_abc")
	}
	if tc.Arguments["city"] != "SF" {
		t.Errorf("ToolCall.Arguments[city] = %v, want SF", tc.Arguments["city"])
	}
	if result.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want %q", result.FinishReason, "tool_calls")
	}
}

func TestCodexProvider_ChatRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			http.Error(w, "not found: "+r.URL.Path, http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("Chatgpt-Account-Id") != "acc-123" {
			http.Error(w, "missing account id", http.StatusBadRequest)
			return
		}

		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if reqBody["stream"] != true {
			http.Error(w, "stream must be true", http.StatusBadRequest)
			return
		}
		if _, ok := reqBody["max_output_tokens"]; ok {
			http.Error(w, "max_output_tokens is not supported", http.StatusBadRequest)
			return
		}
		toolsAny, ok := reqBody["tools"].([]any)
		if !ok || len(toolsAny) != 1 {
			http.Error(w, "missing default web search tool", http.StatusBadRequest)
			return
		}
		toolObj, ok := toolsAny[0].(map[string]any)
		if !ok || toolObj["type"] != "web_search" {
			http.Error(w, "expected web_search tool", http.StatusBadRequest)
			return
		}

		resp := map[string]any{
			"id":     "resp_test",
			"object": "response",
			"status": "completed",
			"output": []map[string]any{
				{
					"id":     "msg_1",
					"type":   "message",
					"role":   "assistant",
					"status": "completed",
					"content": []map[string]any{
						{"type": "output_text", "text": "Hi from Codex!"},
					},
				},
			},
			"usage": map[string]any{
				"input_tokens":          12,
				"output_tokens":         6,
				"total_tokens":          18,
				"input_tokens_details":  map[string]any{"cached_tokens": 0},
				"output_tokens_details": map[string]any{"reasoning_tokens": 0},
			},
		}
		writeCompletedSSE(w, resp)
	}))
	defer server.Close()

	provider := NewCodexProvider("test-token", "acc-123")
	provider.client = createOpenAITestClient(server.URL, "test-token", "acc-123")

	messages := []Message{{Role: "user", Content: "Hello"}}
	resp, err := provider.Chat(t.Context(), messages, nil, "gpt-4o", map[string]any{"max_tokens": 1024})
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if resp.Content != "Hi from Codex!" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hi from Codex!")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
	if resp.Usage.TotalTokens != 18 {
		t.Errorf("TotalTokens = %d, want 18", resp.Usage.TotalTokens)
	}
}

func TestCodexProvider_ChatRoundTrip_WebSearchDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			http.Error(w, "not found: "+r.URL.Path, http.StatusNotFound)
			return
		}

		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if _, ok := reqBody["tools"]; ok {
			http.Error(w, "tools should be absent when web search disabled", http.StatusBadRequest)
			return
		}

		resp := map[string]any{
			"id":     "resp_test",
			"object": "response",
			"status": "completed",
			"output": []map[string]any{
				{
					"id":     "msg_1",
					"type":   "message",
					"role":   "assistant",
					"status": "completed",
					"content": []map[string]any{
						{"type": "output_text", "text": "Hi from Codex!"},
					},
				},
			},
			"usage": map[string]any{
				"input_tokens":          4,
				"output_tokens":         3,
				"total_tokens":          7,
				"input_tokens_details":  map[string]any{"cached_tokens": 0},
				"output_tokens_details": map[string]any{"reasoning_tokens": 0},
			},
		}
		writeCompletedSSE(w, resp)
	}))
	defer server.Close()

	provider := NewCodexProvider("test-token", "acc-123")
	provider.enableWebSearch = false
	provider.client = createOpenAITestClient(server.URL, "test-token", "acc-123")

	messages := []Message{{Role: "user", Content: "Hello"}}
	resp, err := provider.Chat(t.Context(), messages, nil, "gpt-4o", map[string]any{})
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if resp.Content != "Hi from Codex!" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hi from Codex!")
	}
}

func TestCodexProvider_ChatRoundTrip_TokenSourceFallbackAccountID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			http.Error(w, "not found: "+r.URL.Path, http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer refreshed-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("Chatgpt-Account-Id") != "acc-123" {
			http.Error(w, "missing account id", http.StatusBadRequest)
			return
		}

		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if _, ok := reqBody["instructions"]; !ok {
			http.Error(w, "missing instructions", http.StatusBadRequest)
			return
		}
		if reqBody["instructions"] == "" {
			http.Error(w, "instructions must not be empty", http.StatusBadRequest)
			return
		}
		if _, ok := reqBody["temperature"]; ok {
			http.Error(w, "temperature is not supported", http.StatusBadRequest)
			return
		}
		if _, ok := reqBody["max_output_tokens"]; ok {
			http.Error(w, "max_output_tokens is not supported", http.StatusBadRequest)
			return
		}
		if reqBody["stream"] != true {
			http.Error(w, "stream must be true", http.StatusBadRequest)
			return
		}

		resp := map[string]any{
			"id":     "resp_test",
			"object": "response",
			"status": "completed",
			"output": []map[string]any{
				{
					"id":     "msg_1",
					"type":   "message",
					"role":   "assistant",
					"status": "completed",
					"content": []map[string]any{
						{"type": "output_text", "text": "Hi from Codex!"},
					},
				},
			},
			"usage": map[string]any{
				"input_tokens":          8,
				"output_tokens":         4,
				"total_tokens":          12,
				"input_tokens_details":  map[string]any{"cached_tokens": 0},
				"output_tokens_details": map[string]any{"reasoning_tokens": 0},
			},
		}
		writeCompletedSSE(w, resp)
	}))
	defer server.Close()

	provider := NewCodexProvider("stale-token", "acc-123")
	provider.client = createOpenAITestClient(server.URL, "stale-token", "")
	provider.tokenSource = func() (string, string, error) {
		return "refreshed-token", "", nil
	}

	messages := []Message{{Role: "user", Content: "Hello"}}
	resp, err := provider.Chat(t.Context(), messages, nil, "gpt-4o", map[string]any{"temperature": 0.7})
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if resp.Content != "Hi from Codex!" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hi from Codex!")
	}
}

func TestCodexProvider_ChatRoundTrip_ModelFallbackFromUnsupported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			http.Error(w, "not found: "+r.URL.Path, http.StatusNotFound)
			return
		}

		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if reqBody["model"] != codexDefaultModel {
			http.Error(w, "unsupported model", http.StatusBadRequest)
			return
		}
		if reqBody["stream"] != true {
			http.Error(w, "stream must be true", http.StatusBadRequest)
			return
		}
		if reqBody["instructions"] != codexDefaultInstructions {
			http.Error(w, "missing default instructions", http.StatusBadRequest)
			return
		}

		resp := map[string]any{
			"id":     "resp_test",
			"object": "response",
			"status": "completed",
			"output": []map[string]any{
				{
					"id":     "msg_1",
					"type":   "message",
					"role":   "assistant",
					"status": "completed",
					"content": []map[string]any{
						{"type": "output_text", "text": "Hi from Codex!"},
					},
				},
			},
			"usage": map[string]any{
				"input_tokens":          8,
				"output_tokens":         4,
				"total_tokens":          12,
				"input_tokens_details":  map[string]any{"cached_tokens": 0},
				"output_tokens_details": map[string]any{"reasoning_tokens": 0},
			},
		}
		writeCompletedSSE(w, resp)
	}))
	defer server.Close()

	provider := NewCodexProvider("test-token", "acc-123")
	provider.client = createOpenAITestClient(server.URL, "test-token", "acc-123")

	messages := []Message{{Role: "user", Content: "Hello"}}
	resp, err := provider.Chat(t.Context(), messages, nil, "gpt-5.2", nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if resp.Content != "Hi from Codex!" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hi from Codex!")
	}
}

func TestCodexProvider_GetDefaultModel(t *testing.T) {
	p := NewCodexProvider("test-token", "")
	if got := p.GetDefaultModel(); got != codexDefaultModel {
		t.Errorf("GetDefaultModel() = %q, want %q", got, codexDefaultModel)
	}
}

func TestResolveCodexModel(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantModel    string
		wantFallback bool
	}{
		{name: "empty", input: "", wantModel: codexDefaultModel, wantFallback: true},
		{
			name:         "unsupported namespace",
			input:        "anthropic/claude-3.5",
			wantModel:    codexDefaultModel,
			wantFallback: true,
		},
		{name: "non-openai prefixed", input: "glm-4.7", wantModel: codexDefaultModel, wantFallback: true},
		{name: "openai prefix", input: "openai/gpt-5.2", wantModel: "gpt-5.2", wantFallback: false},
		{name: "direct gpt", input: "gpt-4o", wantModel: "gpt-4o", wantFallback: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotModel, reason := resolveCodexModel(tt.input)
			if gotModel != tt.wantModel {
				t.Fatalf("resolveCodexModel(%q) model = %q, want %q", tt.input, gotModel, tt.wantModel)
			}
			if tt.wantFallback && reason == "" {
				t.Fatalf("resolveCodexModel(%q) expected fallback reason", tt.input)
			}
			if !tt.wantFallback && reason != "" {
				t.Fatalf("resolveCodexModel(%q) unexpected fallback reason: %q", tt.input, reason)
			}
		})
	}
}

func createOpenAITestClient(baseURL, token, accountID string) *openai.Client {
	opts := []openaiopt.RequestOption{
		openaiopt.WithBaseURL(baseURL),
		openaiopt.WithAPIKey(token),
	}
	if accountID != "" {
		opts = append(opts, openaiopt.WithHeader("Chatgpt-Account-Id", accountID))
	}
	c := openai.NewClient(opts...)
	return &c
}

func writeCompletedSSE(w http.ResponseWriter, response map[string]any) {
	event := map[string]any{
		"type":            "response.completed",
		"sequence_number": 1,
		"response":        response,
	}
	b, _ := json.Marshal(event)
	w.Header().Set("Content-Type", "text/event-stream")
	fmt.Fprintf(w, "event: response.completed\n")
	fmt.Fprintf(w, "data: %s\n\n", string(b))
	fmt.Fprintf(w, "data: [DONE]\n\n")
}
