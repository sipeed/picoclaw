package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func msg(role, content string) providers.Message {
	return providers.Message{Role: role, Content: content}
}

func assistantWithTools(toolIDs ...string) providers.Message {
	calls := make([]providers.ToolCall, len(toolIDs))
	for i, id := range toolIDs {
		calls[i] = providers.ToolCall{
			ID:   id,
			Type: "function",
			Name: "test_tool",
			Function: &providers.FunctionCall{
				Name:      "test_tool",
				Arguments: "{}",
			},
		}
	}
	return providers.Message{Role: "assistant", ToolCalls: calls}
}

func toolResult(id string) providers.Message {
	return providers.Message{Role: "tool", Content: "result", ToolCallID: id}
}

func TestSanitizeHistoryForProvider_EmptyHistory(t *testing.T) {
	result := sanitizeHistoryForProvider(nil)
	if len(result) != 0 {
		t.Fatalf("expected empty, got %d messages", len(result))
	}

	result = sanitizeHistoryForProvider([]providers.Message{})
	if len(result) != 0 {
		t.Fatalf("expected empty, got %d messages", len(result))
	}
}

func TestSanitizeHistoryForProvider_SingleToolCall(t *testing.T) {
	history := []providers.Message{
		msg("user", "hello"),
		assistantWithTools("A"),
		toolResult("A"),
		msg("assistant", "done"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(result))
	}
	assertRoles(t, result, "user", "assistant", "tool", "assistant")
}

func TestSanitizeHistoryForProvider_MultiToolCalls(t *testing.T) {
	history := []providers.Message{
		msg("user", "do two things"),
		assistantWithTools("A", "B"),
		toolResult("A"),
		toolResult("B"),
		msg("assistant", "both done"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 5 {
		t.Fatalf("expected 5 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "assistant", "tool", "tool", "assistant")
}

func TestSanitizeHistoryForProvider_AssistantToolCallAfterPlainAssistant(t *testing.T) {
	history := []providers.Message{
		msg("user", "hi"),
		msg("assistant", "thinking"),
		assistantWithTools("A"),
		toolResult("A"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "assistant")
}

func TestSanitizeHistoryForProvider_OrphanedLeadingTool(t *testing.T) {
	history := []providers.Message{
		toolResult("A"),
		msg("user", "hello"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user")
}

func TestSanitizeHistoryForProvider_ToolAfterUserDropped(t *testing.T) {
	history := []providers.Message{
		msg("user", "hello"),
		toolResult("A"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user")
}

func TestSanitizeHistoryForProvider_ToolAfterAssistantNoToolCalls(t *testing.T) {
	history := []providers.Message{
		msg("user", "hello"),
		msg("assistant", "hi"),
		toolResult("A"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "assistant")
}

func TestSanitizeHistoryForProvider_AssistantToolCallAtStart(t *testing.T) {
	history := []providers.Message{
		assistantWithTools("A"),
		toolResult("A"),
		msg("user", "hello"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user")
}

func TestSanitizeHistoryForProvider_MultiToolCallsThenNewRound(t *testing.T) {
	history := []providers.Message{
		msg("user", "do two things"),
		assistantWithTools("A", "B"),
		toolResult("A"),
		toolResult("B"),
		msg("assistant", "done"),
		msg("user", "hi"),
		assistantWithTools("C"),
		toolResult("C"),
		msg("assistant", "done again"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 9 {
		t.Fatalf("expected 9 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "assistant", "tool", "tool", "assistant", "user", "assistant", "tool", "assistant")
}

func TestSanitizeHistoryForProvider_ConsecutiveMultiToolRounds(t *testing.T) {
	history := []providers.Message{
		msg("user", "start"),
		assistantWithTools("A", "B"),
		toolResult("A"),
		toolResult("B"),
		assistantWithTools("C", "D"),
		toolResult("C"),
		toolResult("D"),
		msg("assistant", "all done"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 8 {
		t.Fatalf("expected 8 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "assistant", "tool", "tool", "assistant", "tool", "tool", "assistant")
}

func TestSanitizeHistoryForProvider_PlainConversation(t *testing.T) {
	history := []providers.Message{
		msg("user", "hello"),
		msg("assistant", "hi"),
		msg("user", "how are you"),
		msg("assistant", "fine"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(result))
	}
	assertRoles(t, result, "user", "assistant", "user", "assistant")
}

func roles(msgs []providers.Message) []string {
	r := make([]string, len(msgs))
	for i, m := range msgs {
		r[i] = m.Role
	}
	return r
}

func assertRoles(t *testing.T, msgs []providers.Message, expected ...string) {
	t.Helper()
	if len(msgs) != len(expected) {
		t.Fatalf("role count mismatch: got %v, want %v", roles(msgs), expected)
	}
	for i, exp := range expected {
		if msgs[i].Role != exp {
			t.Errorf("message[%d]: got role %q, want %q", i, msgs[i].Role, exp)
		}
	}
}

// TestSanitizeHistoryForProvider_IncompleteToolResults tests the forward validation
// that ensures assistant messages with tool_calls have ALL matching tool results.
// This fixes the DeepSeek error: "An assistant message with 'tool_calls' must be
// followed by tool messages responding to each 'tool_call_id'."
func TestSanitizeHistoryForProvider_IncompleteToolResults(t *testing.T) {
	// Assistant expects tool results for both A and B, but only A is present
	history := []providers.Message{
		msg("user", "do two things"),
		assistantWithTools("A", "B"),
		toolResult("A"),
		// toolResult("B") is missing - this would cause DeepSeek to fail
		msg("user", "next question"),
		msg("assistant", "answer"),
	}

	result := sanitizeHistoryForProvider(history)
	// The assistant message with incomplete tool results should be dropped,
	// along with its partial tool result. The remaining messages are:
	// user ("do two things"), user ("next question"), assistant ("answer")
	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "user", "assistant")
}

// TestSanitizeHistoryForProvider_MissingAllToolResults tests the case where
// an assistant message has tool_calls but no tool results follow at all.
func TestSanitizeHistoryForProvider_MissingAllToolResults(t *testing.T) {
	history := []providers.Message{
		msg("user", "do something"),
		assistantWithTools("A"),
		// No tool results at all
		msg("user", "hello"),
		msg("assistant", "hi"),
	}

	result := sanitizeHistoryForProvider(history)
	// The assistant message with no tool results should be dropped.
	// Remaining: user ("do something"), user ("hello"), assistant ("hi")
	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "user", "assistant")
}

// TestSanitizeHistoryForProvider_PartialToolResultsInMiddle tests that
// incomplete tool results in the middle of a conversation are properly handled.
func TestSanitizeHistoryForProvider_PartialToolResultsInMiddle(t *testing.T) {
	history := []providers.Message{
		msg("user", "first"),
		assistantWithTools("A"),
		toolResult("A"),
		msg("assistant", "done"),
		msg("user", "second"),
		assistantWithTools("B", "C"),
		toolResult("B"),
		// toolResult("C") is missing
		msg("user", "third"),
		assistantWithTools("D"),
		toolResult("D"),
		msg("assistant", "all done"),
	}

	result := sanitizeHistoryForProvider(history)
	// First round is complete (user, assistant+tools, tool, assistant),
	// second round is incomplete and dropped (assistant+tools, partial tool),
	// third round is complete (user, assistant+tools, tool, assistant).
	// Remaining: user, assistant, tool, assistant, user, user, assistant, tool, assistant
	if len(result) != 9 {
		t.Fatalf("expected 9 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "assistant", "tool", "assistant", "user", "user", "assistant", "tool", "assistant")
}

// TestSanitizeHistoryForProvider_EmptyToolCallName tests that tool calls with
// an empty name are dropped. This can occur when sessions are deserialized from
// JSON storage: ToolCall.Name is json:"-" so it is lost, but Function.Name is
// preserved. NormalizeToolCall restores Name from Function.Name; tool calls
// where both are empty are dropped to prevent "tooluse.name: String should
// have at least 1 character" errors from the Anthropic API.
func TestSanitizeHistoryForProvider_EmptyToolCallName(t *testing.T) {
	// Tool call with empty Name but Function.Name set — should be normalized and kept.
	toolCallRestorable := providers.ToolCall{
		ID:   "A",
		Type: "function",
		// Name intentionally empty (simulates post-JSON-deserialization state)
		Function: &providers.FunctionCall{Name: "my_tool", Arguments: "{}"},
	}
	// Tool call with both Name and Function.Name empty — truly invalid, should be dropped.
	toolCallInvalid := providers.ToolCall{
		ID:   "B",
		Type: "function",
		// Name empty, Function.Name empty
		Function: &providers.FunctionCall{Name: "", Arguments: "{}"},
	}

	history := []providers.Message{
		msg("user", "hello"),
		{Role: "assistant", ToolCalls: []providers.ToolCall{toolCallRestorable, toolCallInvalid}},
		toolResult("A"),
		toolResult("B"), // orphaned result for the dropped tool call
	}

	result := sanitizeHistoryForProvider(history)
	// toolCallRestorable: Name restored from Function.Name → kept.
	// toolCallInvalid: Name still empty after normalization → dropped.
	// toolResult("B") references the dropped call → also dropped.
	// Result: user, assistant (with only tc_A), tool_A
	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "assistant", "tool")
	if len(result[1].ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call in assistant, got %d", len(result[1].ToolCalls))
	}
	if result[1].ToolCalls[0].Name != "my_tool" {
		t.Errorf("expected tool call name %q, got %q", "my_tool", result[1].ToolCalls[0].Name)
	}
	if result[2].ToolCallID != "A" {
		t.Errorf("expected surviving tool result id %q, got %q", "A", result[2].ToolCallID)
	}
}

// TestSanitizeHistoryForProvider_EmptyToolCallID tests that tool calls with an
// empty ID are dropped, and that the resulting content-less assistant message
// is also dropped (avoiding an empty content array sent to the API).
func TestSanitizeHistoryForProvider_EmptyToolCallID(t *testing.T) {
	history := []providers.Message{
		msg("user", "hello"),
		{Role: "assistant", ToolCalls: []providers.ToolCall{
			{ID: "", Name: "my_tool", Type: "function", Function: &providers.FunctionCall{Name: "my_tool"}},
		}},
		// No tool result since the call has no ID to reference
	}

	result := sanitizeHistoryForProvider(history)
	// The tool call with empty ID is dropped; the assistant message then has no
	// tool calls and no text content → it is also dropped to avoid sending an
	// empty content array to the API.
	// Remaining: user ("hello") only.
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user")
}

// TestSanitizeHistoryForProvider_AllToolCallsDroppedEmptyContent tests that an
// assistant message whose every tool call is dropped AND whose text content is
// empty is itself dropped. Sending such a message would produce "content": []
// or an empty text block — both cause a generic Anthropic 400 "Error".
func TestSanitizeHistoryForProvider_AllToolCallsDroppedEmptyContent(t *testing.T) {
	history := []providers.Message{
		msg("user", "hello"),
		{Role: "assistant", Content: "", ToolCalls: []providers.ToolCall{
			{ID: "A", Type: "function", Function: &providers.FunctionCall{Name: "", Arguments: "{}"}},
		}},
		toolResult("A"),
		msg("user", "follow up"),
	}

	result := sanitizeHistoryForProvider(history)
	// The assistant message has no valid tool calls and no content → dropped.
	// Its tool result is orphaned and also dropped.
	// Remaining: user ("hello"), user ("follow up")
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "user")
}
