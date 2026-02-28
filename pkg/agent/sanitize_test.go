package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestSanitizeToolPairs_NoToolMessages(t *testing.T) {
	// Normal messages without tool pairs should pass through unchanged
	messages := []providers.Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "How are you?"},
		{Role: "assistant", Content: "I am good!"},
	}

	result := sanitizeToolPairs(messages)
	if len(result) != len(messages) {
		t.Errorf("expected %d messages, got %d", len(messages), len(result))
	}
	for i, m := range result {
		if m.Role != messages[i].Role || m.Content != messages[i].Content {
			t.Errorf("message %d: expected role=%s content=%q, got role=%s content=%q",
				i, messages[i].Role, messages[i].Content, m.Role, m.Content)
		}
	}
}

func TestSanitizeToolPairs_CompletePairs(t *testing.T) {
	// Complete tool_call/tool_result pairs should be preserved
	messages := []providers.Message{
		{Role: "user", Content: "What is the weather?"},
		{Role: "assistant", Content: "", ToolCalls: []providers.ToolCall{
			{ID: "call_1", Name: "get_weather"},
		}},
		{Role: "tool", Content: `{"temp": 72}`, ToolCallID: "call_1"},
		{Role: "assistant", Content: "The temperature is 72 degrees."},
	}

	result := sanitizeToolPairs(messages)
	if len(result) != 4 {
		t.Errorf("expected 4 messages, got %d", len(result))
	}
}

func TestSanitizeToolPairs_OrphanedToolResult(t *testing.T) {
	// Tool result without matching tool_call should be removed
	// This happens when forceCompression cuts after the assistant tool_call
	// but keeps the tool result
	messages := []providers.Message{
		{Role: "tool", Content: `{"temp": 72}`, ToolCallID: "call_1"},
		{Role: "assistant", Content: "The temperature is 72 degrees."},
		{Role: "user", Content: "Thanks!"},
	}

	result := sanitizeToolPairs(messages)
	if len(result) != 2 {
		t.Errorf("expected 2 messages (orphaned tool result removed), got %d", len(result))
	}
	if result[0].Role != "assistant" {
		t.Errorf("expected first message to be assistant, got %s", result[0].Role)
	}
	if result[1].Role != "user" {
		t.Errorf("expected second message to be user, got %s", result[1].Role)
	}
}

func TestSanitizeToolPairs_OrphanedToolCall(t *testing.T) {
	// Assistant message with tool_calls but no tool results should be removed
	// This happens when forceCompression cuts after the tool_call but before results
	messages := []providers.Message{
		{Role: "user", Content: "What is the weather?"},
		{Role: "assistant", Content: "", ToolCalls: []providers.ToolCall{
			{ID: "call_1", Name: "get_weather"},
		}},
		// tool result for call_1 was dropped by compression
		{Role: "user", Content: "Never mind"},
	}

	result := sanitizeToolPairs(messages)
	if len(result) != 2 {
		t.Errorf("expected 2 messages (orphaned assistant with tool_calls removed), got %d", len(result))
	}
	if result[0].Role != "user" || result[0].Content != "What is the weather?" {
		t.Errorf("expected first message to be user 'What is the weather?', got %s %q", result[0].Role, result[0].Content)
	}
	if result[1].Role != "user" || result[1].Content != "Never mind" {
		t.Errorf("expected second message to be user 'Never mind', got %s %q", result[1].Role, result[1].Content)
	}
}

func TestSanitizeToolPairs_OrphanedToolCallWithContent(t *testing.T) {
	// Assistant message with tool_calls AND text content: keep text, strip tool_calls
	messages := []providers.Message{
		{Role: "user", Content: "What is the weather?"},
		{Role: "assistant", Content: "Let me check the weather for you.", ToolCalls: []providers.ToolCall{
			{ID: "call_1", Name: "get_weather"},
		}},
		// tool result for call_1 was dropped by compression
		{Role: "user", Content: "Thanks"},
	}

	result := sanitizeToolPairs(messages)
	if len(result) != 3 {
		t.Errorf("expected 3 messages (assistant kept with text, tool_calls stripped), got %d", len(result))
	}
	if result[1].Role != "assistant" || result[1].Content != "Let me check the weather for you." {
		t.Errorf("expected assistant message with text content, got role=%s content=%q", result[1].Role, result[1].Content)
	}
	if len(result[1].ToolCalls) != 0 {
		t.Errorf("expected tool_calls to be stripped, got %d tool_calls", len(result[1].ToolCalls))
	}
}

func TestSanitizeToolPairs_MultipleConsecutivePairs(t *testing.T) {
	// Multiple complete tool pairs should all be preserved
	messages := []providers.Message{
		{Role: "user", Content: "Do two things"},
		{Role: "assistant", Content: "", ToolCalls: []providers.ToolCall{
			{ID: "call_1", Name: "task_a"},
			{ID: "call_2", Name: "task_b"},
		}},
		{Role: "tool", Content: "result_a", ToolCallID: "call_1"},
		{Role: "tool", Content: "result_b", ToolCallID: "call_2"},
		{Role: "assistant", Content: "Both tasks done."},
	}

	result := sanitizeToolPairs(messages)
	if len(result) != 5 {
		t.Errorf("expected 5 messages (all pairs complete), got %d", len(result))
	}
}

func TestSanitizeToolPairs_MultiToolCallPartialResults(t *testing.T) {
	// Assistant with 2 tool_calls but only 1 result
	messages := []providers.Message{
		{Role: "user", Content: "Do two things"},
		{Role: "assistant", Content: "", ToolCalls: []providers.ToolCall{
			{ID: "call_1", Name: "task_a"},
			{ID: "call_2", Name: "task_b"},
		}},
		{Role: "tool", Content: "result_a", ToolCallID: "call_1"},
		// call_2 result was dropped by compression
		{Role: "user", Content: "OK"},
	}

	result := sanitizeToolPairs(messages)
	// The assistant msg is dropped (not all tool_calls have results).
	// call_1's tool result stays because toolCallIDs includes call_1
	// from the original scan. This is a secondary orphan handled by
	// the existing Diegox-17 fix in BuildMessages (strips leading tool msgs).
	if len(result) != 3 {
		t.Errorf("expected 3 messages, got %d", len(result))
	}
	if result[0].Role != "user" {
		t.Errorf("expected first message to be user, got %s", result[0].Role)
	}
}

func TestSanitizeToolPairs_EmptyMessages(t *testing.T) {
	result := sanitizeToolPairs([]providers.Message{})
	if len(result) != 0 {
		t.Errorf("expected 0 messages, got %d", len(result))
	}
}

func TestSanitizeToolPairs_MixedOrphanedAndComplete(t *testing.T) {
	// Mix of orphaned and complete pairs after a simulated forceCompression cut
	messages := []providers.Message{
		// This is orphaned (its assistant tool_call was in the dropped half)
		{Role: "tool", Content: "old_result", ToolCallID: "call_old"},
		// This is a complete pair
		{Role: "assistant", Content: "", ToolCalls: []providers.ToolCall{
			{ID: "call_new", Name: "new_task"},
		}},
		{Role: "tool", Content: "new_result", ToolCallID: "call_new"},
		{Role: "assistant", Content: "Done with the new task."},
	}

	result := sanitizeToolPairs(messages)
	if len(result) != 3 {
		t.Errorf("expected 3 messages (orphaned tool result removed), got %d", len(result))
	}
	// First should be the assistant with tool_calls (complete pair)
	if result[0].Role != "assistant" || len(result[0].ToolCalls) != 1 {
		t.Errorf("expected assistant with tool_calls, got role=%s tool_calls=%d", result[0].Role, len(result[0].ToolCalls))
	}
	if result[1].Role != "tool" || result[1].ToolCallID != "call_new" {
		t.Errorf("expected tool result for call_new, got role=%s id=%s", result[1].Role, result[1].ToolCallID)
	}
	if result[2].Role != "assistant" || result[2].Content != "Done with the new task." {
		t.Errorf("expected final assistant message, got role=%s content=%q", result[2].Role, result[2].Content)
	}
}
