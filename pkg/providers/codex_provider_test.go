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
	params := buildCodexParams(messages, nil, "gpt-4o", map[string]interface{}{
		"max_tokens":  2048,
		"temperature": 0.7,
	})
	if params.Model != "gpt-4o" {
		t.Errorf("Model = %q, want %q", params.Model, "gpt-4o")
	}
	if !params.Instructions.Valid() {
		t.Fatal("Instructions should be set")
	}
	if params.Instructions.Or("") != defaultCodexInstructions {
		t.Errorf("Instructions = %q, want %q", params.Instructions.Or(""), defaultCodexInstructions)
	}
}

func TestBuildCodexParams_SystemAsInstructions(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "You are helpful"},
		{Role: "user", Content: "Hi"},
	}
	params := buildCodexParams(messages, nil, "gpt-4o", map[string]interface{}{})
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
				{ID: "call_1", Name: "get_weather", Arguments: map[string]interface{}{"city": "SF"}},
			},
		},
		{Role: "tool", Content: `{"temp": 72}`, ToolCallID: "call_1"},
	}
	params := buildCodexParams(messages, nil, "gpt-4o", map[string]interface{}{})
	if params.Input.OfInputItemList == nil {
		t.Fatal("Input.OfInputItemList should not be nil")
	}
	if len(params.Input.OfInputItemList) != 3 {
		t.Errorf("len(Input items) = %d, want 3", len(params.Input.OfInputItemList))
	}
}

func TestBuildCodexParams_WithTools(t *testing.T) {
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "get_weather",
				Description: "Get weather",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"city": map[string]interface{}{"type": "string"},
					},
				},
			},
		},
	}
	params := buildCodexParams([]Message{{Role: "user", Content: "Hi"}}, tools, "gpt-4o", map[string]interface{}{})
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
	params := buildCodexParams([]Message{{Role: "user", Content: "Hi"}}, nil, "gpt-4o", map[string]interface{}{})
	if !params.Store.Valid() || params.Store.Or(true) != false {
		t.Error("Store should be explicitly set to false")
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

		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if reqBody["stream"] != true {
			http.Error(w, "stream must be true", http.StatusBadRequest)
			return
		}

		resp := map[string]interface{}{
			"id":     "resp_test",
			"object": "response",
			"status": "completed",
			"output": []map[string]interface{}{
				{
					"id":     "msg_1",
					"type":   "message",
					"role":   "assistant",
					"status": "completed",
					"content": []map[string]interface{}{
						{"type": "output_text", "text": "Hi from Codex!"},
					},
				},
			},
			"usage": map[string]interface{}{
				"input_tokens":          12,
				"output_tokens":         6,
				"total_tokens":          18,
				"input_tokens_details":  map[string]interface{}{"cached_tokens": 0},
				"output_tokens_details": map[string]interface{}{"reasoning_tokens": 0},
			},
		}
		writeCompletedSSE(w, resp)
	}))
	defer server.Close()

	provider := NewCodexProvider("test-token", "acc-123")
	provider.client = createOpenAITestClient(server.URL, "test-token", "acc-123")

	messages := []Message{{Role: "user", Content: "Hello"}}
	resp, err := provider.Chat(t.Context(), messages, nil, "gpt-4o", map[string]interface{}{"max_tokens": 1024})
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

		var reqBody map[string]interface{}
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
		if reqBody["stream"] != true {
			http.Error(w, "stream must be true", http.StatusBadRequest)
			return
		}

		resp := map[string]interface{}{
			"id":     "resp_test",
			"object": "response",
			"status": "completed",
			"output": []map[string]interface{}{
				{
					"id":     "msg_1",
					"type":   "message",
					"role":   "assistant",
					"status": "completed",
					"content": []map[string]interface{}{
						{"type": "output_text", "text": "Hi from Codex!"},
					},
				},
			},
			"usage": map[string]interface{}{
				"input_tokens":          8,
				"output_tokens":         4,
				"total_tokens":          12,
				"input_tokens_details":  map[string]interface{}{"cached_tokens": 0},
				"output_tokens_details": map[string]interface{}{"reasoning_tokens": 0},
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
	resp, err := provider.Chat(t.Context(), messages, nil, "gpt-4o", map[string]interface{}{"temperature": 0.7})
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

		var reqBody map[string]interface{}
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

		resp := map[string]interface{}{
			"id":     "resp_test",
			"object": "response",
			"status": "completed",
			"output": []map[string]interface{}{
				{
					"id":     "msg_1",
					"type":   "message",
					"role":   "assistant",
					"status": "completed",
					"content": []map[string]interface{}{
						{"type": "output_text", "text": "Hi from Codex!"},
					},
				},
			},
			"usage": map[string]interface{}{
				"input_tokens":          8,
				"output_tokens":         4,
				"total_tokens":          12,
				"input_tokens_details":  map[string]interface{}{"cached_tokens": 0},
				"output_tokens_details": map[string]interface{}{"reasoning_tokens": 0},
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
		{name: "unsupported namespace", input: "anthropic/claude-3.5", wantModel: codexDefaultModel, wantFallback: true},
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

func writeCompletedSSE(w http.ResponseWriter, response map[string]interface{}) {
	event := map[string]interface{}{
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
