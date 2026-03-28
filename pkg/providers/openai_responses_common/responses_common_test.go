package openai_responses_common

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/openai/openai-go/v3/responses"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

// --- TranslateMessages tests ---

func TestTranslateMessages_SystemExtractedAsInstructions(t *testing.T) {
	msgs := []protocoltypes.Message{
		{Role: "system", Content: "You are helpful"},
		{Role: "user", Content: "Hi"},
	}
	input, instructions := TranslateMessages(msgs)
	if instructions != "You are helpful" {
		t.Errorf("instructions = %q, want %q", instructions, "You are helpful")
	}
	if len(input) != 1 {
		t.Fatalf("len(input) = %d, want 1", len(input))
	}
	if input[0].OfMessage == nil {
		t.Fatal("expected user message item")
	}
}

func TestTranslateMessages_UserTextMessage(t *testing.T) {
	msgs := []protocoltypes.Message{
		{Role: "user", Content: "Hello"},
	}
	input, instructions := TranslateMessages(msgs)
	if instructions != "" {
		t.Errorf("instructions = %q, want empty", instructions)
	}
	if len(input) != 1 {
		t.Fatalf("len(input) = %d, want 1", len(input))
	}
	if input[0].OfMessage == nil {
		t.Fatal("expected EasyInputMessage")
	}
	if string(input[0].OfMessage.Role) != "user" {
		t.Errorf("role = %q, want %q", input[0].OfMessage.Role, "user")
	}
}

func TestTranslateMessages_UserWithToolCallID(t *testing.T) {
	msgs := []protocoltypes.Message{
		{Role: "user", Content: `{"temp":72}`, ToolCallID: "call_1"},
	}
	input, _ := TranslateMessages(msgs)
	if len(input) != 1 {
		t.Fatalf("len(input) = %d, want 1", len(input))
	}
	if input[0].OfFunctionCallOutput == nil {
		t.Fatal("expected FunctionCallOutput for user with ToolCallID")
	}
	if input[0].OfFunctionCallOutput.CallID != "call_1" {
		t.Errorf("CallID = %q, want %q", input[0].OfFunctionCallOutput.CallID, "call_1")
	}
}

func TestTranslateMessages_UserWithMedia(t *testing.T) {
	msgs := []protocoltypes.Message{
		{Role: "user", Content: "Describe this", Media: []string{"data:image/png;base64,abc123"}},
	}
	input, _ := TranslateMessages(msgs)
	if len(input) != 1 {
		t.Fatalf("len(input) = %d, want 1", len(input))
	}
	if input[0].OfInputMessage == nil {
		t.Fatal("expected InputMessage for multipart content")
	}
	if input[0].OfInputMessage.Role != "user" {
		t.Errorf("role = %q, want %q", input[0].OfInputMessage.Role, "user")
	}
}

func TestTranslateMessages_AssistantWithToolCalls(t *testing.T) {
	msgs := []protocoltypes.Message{
		{Role: "user", Content: "Weather?"},
		{
			Role:    "assistant",
			Content: "Let me check",
			ToolCalls: []protocoltypes.ToolCall{
				{ID: "call_1", Name: "get_weather", Arguments: map[string]any{"city": "SF"}},
			},
		},
		{Role: "tool", Content: `{"temp":72}`, ToolCallID: "call_1"},
	}
	input, _ := TranslateMessages(msgs)
	// user + assistant text + function_call + tool output = 4 items
	if len(input) != 4 {
		t.Fatalf("len(input) = %d, want 4", len(input))
	}
	// item[1] = assistant text
	if input[1].OfMessage == nil {
		t.Fatal("expected assistant text message")
	}
	// item[2] = function call
	if input[2].OfFunctionCall == nil {
		t.Fatal("expected function call")
	}
	if input[2].OfFunctionCall.Name != "get_weather" {
		t.Errorf("function name = %q, want %q", input[2].OfFunctionCall.Name, "get_weather")
	}
	// item[3] = tool output
	if input[3].OfFunctionCallOutput == nil {
		t.Fatal("expected function call output")
	}
}

func TestTranslateMessages_AssistantWithoutToolCalls(t *testing.T) {
	msgs := []protocoltypes.Message{
		{Role: "assistant", Content: "Sure thing"},
	}
	input, _ := TranslateMessages(msgs)
	if len(input) != 1 {
		t.Fatalf("len(input) = %d, want 1", len(input))
	}
	if input[0].OfMessage == nil {
		t.Fatal("expected EasyInputMessage for assistant without tool calls")
	}
}

func TestTranslateMessages_ToolMessage(t *testing.T) {
	msgs := []protocoltypes.Message{
		{Role: "tool", Content: "result data", ToolCallID: "call_99"},
	}
	input, _ := TranslateMessages(msgs)
	if len(input) != 1 {
		t.Fatalf("len(input) = %d, want 1", len(input))
	}
	if input[0].OfFunctionCallOutput == nil {
		t.Fatal("expected FunctionCallOutput")
	}
	if input[0].OfFunctionCallOutput.CallID != "call_99" {
		t.Errorf("CallID = %q, want %q", input[0].OfFunctionCallOutput.CallID, "call_99")
	}
}

// --- ResolveToolCall tests ---

func TestResolveToolCall_FromNameAndArguments(t *testing.T) {
	tc := protocoltypes.ToolCall{
		Name:      "get_weather",
		Arguments: map[string]any{"city": "SF"},
	}
	name, args, ok := ResolveToolCall(tc)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if name != "get_weather" {
		t.Errorf("name = %q, want %q", name, "get_weather")
	}
	if !strings.Contains(args, "SF") {
		t.Errorf("args = %q, want to contain SF", args)
	}
}

func TestResolveToolCall_FromFunctionField(t *testing.T) {
	tc := protocoltypes.ToolCall{
		ID: "call_1",
		Function: &protocoltypes.FunctionCall{
			Name:      "read_file",
			Arguments: `{"path":"README.md"}`,
		},
	}
	name, args, ok := ResolveToolCall(tc)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if name != "read_file" {
		t.Errorf("name = %q, want %q", name, "read_file")
	}
	if args != `{"path":"README.md"}` {
		t.Errorf("args = %q, want %q", args, `{"path":"README.md"}`)
	}
}

func TestResolveToolCall_EmptyName(t *testing.T) {
	tc := protocoltypes.ToolCall{}
	_, _, ok := ResolveToolCall(tc)
	if ok {
		t.Error("expected ok=false for empty tool call")
	}
}

func TestResolveToolCall_NoArgsFallsBackToEmptyObject(t *testing.T) {
	tc := protocoltypes.ToolCall{Name: "do_something"}
	name, args, ok := ResolveToolCall(tc)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if name != "do_something" {
		t.Errorf("name = %q, want %q", name, "do_something")
	}
	if args != "{}" {
		t.Errorf("args = %q, want %q", args, "{}")
	}
}

// --- TranslateTools tests ---

func TestTranslateTools_FunctionTools(t *testing.T) {
	tools := []protocoltypes.ToolDefinition{
		{
			Type: "function",
			Function: protocoltypes.ToolFunctionDefinition{
				Name:        "get_weather",
				Description: "Get weather",
				Parameters:  map[string]any{"type": "object"},
			},
		},
	}
	result := TranslateTools(tools, false)
	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}
	if result[0].OfFunction == nil {
		t.Fatal("expected function tool")
	}
	if result[0].OfFunction.Name != "get_weather" {
		t.Errorf("name = %q, want %q", result[0].OfFunction.Name, "get_weather")
	}
}

func TestTranslateTools_SkipsNonFunction(t *testing.T) {
	tools := []protocoltypes.ToolDefinition{
		{Type: "not_function"},
	}
	result := TranslateTools(tools, false)
	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0", len(result))
	}
}

func TestTranslateTools_WebSearchAppended(t *testing.T) {
	result := TranslateTools(nil, true)
	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}
	if result[0].OfWebSearch == nil {
		t.Fatal("expected web_search tool")
	}
}

func TestTranslateTools_WebSearchReplacesUserDefined(t *testing.T) {
	tools := []protocoltypes.ToolDefinition{
		{
			Type: "function",
			Function: protocoltypes.ToolFunctionDefinition{
				Name:       "web_search",
				Parameters: map[string]any{"type": "object"},
			},
		},
		{
			Type: "function",
			Function: protocoltypes.ToolFunctionDefinition{
				Name:       "read_file",
				Parameters: map[string]any{"type": "object"},
			},
		},
	}
	result := TranslateTools(tools, true)
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
	if result[0].OfFunction == nil || result[0].OfFunction.Name != "read_file" {
		t.Errorf("first tool should be read_file, got %v", result[0])
	}
	if result[1].OfWebSearch == nil {
		t.Error("second tool should be web_search")
	}
}

func TestTranslateTools_DescriptionOmittedWhenEmpty(t *testing.T) {
	tools := []protocoltypes.ToolDefinition{
		{
			Type: "function",
			Function: protocoltypes.ToolFunctionDefinition{
				Name:       "no_desc",
				Parameters: map[string]any{"type": "object"},
			},
		},
	}
	result := TranslateTools(tools, false)
	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}
	if result[0].OfFunction.Description.Valid() {
		t.Error("Description should not be set when empty")
	}
}

// --- ParseResponseBody tests ---

func TestParseResponseBody_TextOutput(t *testing.T) {
	body := strings.NewReader(fmt.Sprintf(`{
		"id": "resp_123",
		"object": "response",
		"status": "%s",
		"output": [
			{
				"type": "message",
				"content": [{"type": "output_text", "text": "Hello!"}]
			}
		],
		"usage": {
			"input_tokens": 10,
			"output_tokens": 5,
			"total_tokens": 15,
			"input_tokens_details": {"cached_tokens": 0},
			"output_tokens_details": {"reasoning_tokens": 0}
		}
	}`, string(responses.ResponseStatusCompleted)))

	result, err := ParseResponseBody(body)
	if err != nil {
		t.Fatalf("ParseResponseBody error: %v", err)
	}
	if result.Content != "Hello!" {
		t.Errorf("Content = %q, want %q", result.Content, "Hello!")
	}
	if result.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", result.FinishReason, "stop")
	}
	if result.Usage.TotalTokens != 15 {
		t.Errorf("TotalTokens = %d, want 15", result.Usage.TotalTokens)
	}
}

func TestParseResponseBody_FunctionCall(t *testing.T) {
	body := strings.NewReader(fmt.Sprintf(`{
		"id": "resp_456",
		"object": "response",
		"status": "%s",
		"output": [
			{
				"type": "function_call",
				"call_id": "call_abc",
				"name": "get_weather",
				"arguments": "{\"city\":\"SF\"}"
			}
		],
		"usage": {
			"input_tokens": 10,
			"output_tokens": 8,
			"total_tokens": 18,
			"input_tokens_details": {"cached_tokens": 0},
			"output_tokens_details": {"reasoning_tokens": 0}
		}
	}`, string(responses.ResponseStatusCompleted)))

	result, err := ParseResponseBody(body)
	if err != nil {
		t.Fatalf("ParseResponseBody error: %v", err)
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(result.ToolCalls))
	}
	if result.ToolCalls[0].Name != "get_weather" {
		t.Errorf("Name = %q, want %q", result.ToolCalls[0].Name, "get_weather")
	}
	if result.ToolCalls[0].ID != "call_abc" {
		t.Errorf("ID = %q, want %q", result.ToolCalls[0].ID, "call_abc")
	}
	if result.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want %q", result.FinishReason, "tool_calls")
	}
}

func TestParseResponseBody_Reasoning(t *testing.T) {
	body := strings.NewReader(fmt.Sprintf(`{
		"id": "resp_789",
		"object": "response",
		"status": "%s",
		"output": [
			{
				"type": "reasoning",
				"id": "rs_1",
				"summary": [{"type": "summary_text", "text": "Thinking about it..."}]
			},
			{
				"type": "message",
				"content": [{"type": "output_text", "text": "The answer is 42."}]
			}
		],
		"usage": {
			"input_tokens": 10,
			"output_tokens": 20,
			"total_tokens": 30,
			"input_tokens_details": {"cached_tokens": 0},
			"output_tokens_details": {"reasoning_tokens": 10}
		}
	}`, string(responses.ResponseStatusCompleted)))

	result, err := ParseResponseBody(body)
	if err != nil {
		t.Fatalf("ParseResponseBody error: %v", err)
	}
	if result.Content != "The answer is 42." {
		t.Errorf("Content = %q, want %q", result.Content, "The answer is 42.")
	}
	if result.ReasoningContent != "Thinking about it..." {
		t.Errorf("ReasoningContent = %q, want %q", result.ReasoningContent, "Thinking about it...")
	}
}

func TestParseResponseBody_Refusal(t *testing.T) {
	body := strings.NewReader(fmt.Sprintf(`{
		"id": "resp_ref",
		"object": "response",
		"status": "%s",
		"output": [
			{
				"type": "message",
				"content": [{"type": "refusal", "refusal": "I cannot help with that."}]
			}
		],
		"usage": {
			"input_tokens": 5,
			"output_tokens": 5,
			"total_tokens": 10,
			"input_tokens_details": {"cached_tokens": 0},
			"output_tokens_details": {"reasoning_tokens": 0}
		}
	}`, string(responses.ResponseStatusCompleted)))

	result, err := ParseResponseBody(body)
	if err != nil {
		t.Fatalf("ParseResponseBody error: %v", err)
	}
	if result.Content != "I cannot help with that." {
		t.Errorf("Content = %q, want %q", result.Content, "I cannot help with that.")
	}
}

func TestParseResponseBody_IncompleteStatus(t *testing.T) {
	body := strings.NewReader(fmt.Sprintf(`{
		"id": "resp_inc",
		"object": "response",
		"status": "%s",
		"output": [
			{
				"type": "message",
				"content": [{"type": "output_text", "text": "partial"}]
			}
		],
		"usage": {"input_tokens": 5, "output_tokens": 2, "total_tokens": 7,
			"input_tokens_details": {"cached_tokens": 0},
			"output_tokens_details": {"reasoning_tokens": 0}}
	}`, string(responses.ResponseStatusIncomplete)))

	result, err := ParseResponseBody(body)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.FinishReason != "length" {
		t.Errorf("FinishReason = %q, want %q", result.FinishReason, "length")
	}
}

func TestParseResponseBody_FailedStatus(t *testing.T) {
	body := strings.NewReader(fmt.Sprintf(`{
		"id": "resp_fail",
		"object": "response",
		"status": "%s",
		"output": [],
		"usage": {"input_tokens": 0, "output_tokens": 0, "total_tokens": 0,
			"input_tokens_details": {"cached_tokens": 0},
			"output_tokens_details": {"reasoning_tokens": 0}}
	}`, string(responses.ResponseStatusFailed)))

	result, err := ParseResponseBody(body)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.FinishReason != "error" {
		t.Errorf("FinishReason = %q, want %q", result.FinishReason, "error")
	}
}

func TestParseResponseBody_CanceledStatus(t *testing.T) {
	body := strings.NewReader(fmt.Sprintf(`{
		"id": "resp_cancel",
		"object": "response",
		"status": "%s",
		"output": [],
		"usage": {"input_tokens": 0, "output_tokens": 0, "total_tokens": 0,
			"input_tokens_details": {"cached_tokens": 0},
			"output_tokens_details": {"reasoning_tokens": 0}}
	}`, string(responses.ResponseStatusCancelled)))

	result, err := ParseResponseBody(body)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.FinishReason != "canceled" {
		t.Errorf("FinishReason = %q, want %q", result.FinishReason, "canceled")
	}
}

// --- ParseDataAudioURL tests ---

func TestParseDataAudioURL_Valid(t *testing.T) {
	format, data, ok := ParseDataAudioURL("data:audio/mp3;base64,SGVsbG8=")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if format != "mp3" {
		t.Errorf("format = %q, want %q", format, "mp3")
	}
	if data != "SGVsbG8=" {
		t.Errorf("data = %q, want %q", data, "SGVsbG8=")
	}
}

func TestParseDataAudioURL_NotAudio(t *testing.T) {
	_, _, ok := ParseDataAudioURL("data:image/png;base64,abc")
	if ok {
		t.Error("expected ok=false for non-audio URL")
	}
}

func TestParseDataAudioURL_MalformedNoComma(t *testing.T) {
	_, _, ok := ParseDataAudioURL("data:audio/mp3;base64")
	if ok {
		t.Error("expected ok=false for malformed URL")
	}
}

func TestParseDataAudioURL_EmptyData(t *testing.T) {
	_, _, ok := ParseDataAudioURL("data:audio/mp3;base64,")
	if ok {
		t.Error("expected ok=false for empty data")
	}
}

// --- BuildMultipartContent tests ---

func TestBuildMultipartContent_TextOnly(t *testing.T) {
	parts := BuildMultipartContent("hello", nil)
	if len(parts) != 1 {
		t.Fatalf("len(parts) = %d, want 1", len(parts))
	}
	if parts[0].OfInputText == nil {
		t.Fatal("expected text part")
	}
}

func TestBuildMultipartContent_TextAndImage(t *testing.T) {
	parts := BuildMultipartContent("describe", []string{"data:image/png;base64,abc"})
	if len(parts) != 2 {
		t.Fatalf("len(parts) = %d, want 2", len(parts))
	}
	if parts[0].OfInputText == nil {
		t.Error("first part should be text")
	}
	if parts[1].OfInputImage == nil {
		t.Error("second part should be image")
	}
}

func TestBuildMultipartContent_AudioFile(t *testing.T) {
	parts := BuildMultipartContent("", []string{"data:audio/wav;base64,AAAA"})
	if len(parts) != 1 {
		t.Fatalf("len(parts) = %d, want 1", len(parts))
	}
	if parts[0].OfInputFile == nil {
		t.Fatal("expected file part for audio")
	}
}

func TestBuildMultipartContent_EmptyTextSkipped(t *testing.T) {
	parts := BuildMultipartContent("", []string{"data:image/png;base64,abc"})
	if len(parts) != 1 {
		t.Fatalf("len(parts) = %d, want 1", len(parts))
	}
	if parts[0].OfInputImage == nil {
		t.Error("should only have image part")
	}
}

// --- JSON serialization sanity checks ---

func TestTranslateTools_SerializesToJSON(t *testing.T) {
	tools := []protocoltypes.ToolDefinition{
		{
			Type: "function",
			Function: protocoltypes.ToolFunctionDefinition{
				Name:        "test_tool",
				Description: "A test",
				Parameters:  map[string]any{"type": "object"},
			},
		},
	}
	result := TranslateTools(tools, true)
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "test_tool") {
		t.Errorf("JSON should contain test_tool, got: %s", s)
	}
	if !strings.Contains(s, "web_search") {
		t.Errorf("JSON should contain web_search, got: %s", s)
	}
}
