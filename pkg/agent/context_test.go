package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestSanitizeHistory_LeadingToolMessages(t *testing.T) {
	history := []providers.Message{
		{Role: "tool", Content: "orphaned result", ToolCallID: "call_1"},
		{Role: "tool", Content: "orphaned result 2", ToolCallID: "call_2"},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}

	result := sanitizeHistory(history)

	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	if result[0].Role != "user" {
		t.Errorf("expected first message to be user, got %s", result[0].Role)
	}
}

func TestSanitizeHistory_LeadingAssistantWithToolCalls(t *testing.T) {
	history := []providers.Message{
		{
			Role: "assistant",
			ToolCalls: []providers.ToolCall{
				{ID: "call_1", Type: "function", Function: &providers.FunctionCall{Name: "test"}},
			},
		},
		{Role: "tool", Content: "result", ToolCallID: "call_1"},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}

	result := sanitizeHistory(history)

	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	if result[0].Role != "user" {
		t.Errorf("expected first message to be user, got %s", result[0].Role)
	}
}

func TestSanitizeHistory_ConsecutiveUsers(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "first"},
		{Role: "user", Content: "second"},
		{Role: "user", Content: "third"},
		{Role: "assistant", Content: "response"},
	}

	result := sanitizeHistory(history)

	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	if result[0].Content != "third" {
		t.Errorf("expected last user message 'third', got %q", result[0].Content)
	}
}

func TestSanitizeHistory_ValidHistory(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "hello"},
		{
			Role: "assistant",
			ToolCalls: []providers.ToolCall{
				{ID: "call_1", Type: "function", Function: &providers.FunctionCall{Name: "test"}},
			},
		},
		{Role: "tool", Content: "result", ToolCallID: "call_1"},
		{Role: "assistant", Content: "done"},
		{Role: "user", Content: "thanks"},
		{Role: "assistant", Content: "welcome"},
	}

	result := sanitizeHistory(history)

	if len(result) != len(history) {
		t.Fatalf("expected %d messages (unchanged), got %d", len(history), len(result))
	}
	for i := range result {
		if result[i].Role != history[i].Role {
			t.Errorf("message %d: expected role %s, got %s", i, history[i].Role, result[i].Role)
		}
	}
}

func TestSanitizeHistory_TrailingOrphanedToolCalls(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
		{Role: "user", Content: "run tools"},
		{
			Role: "assistant",
			ToolCalls: []providers.ToolCall{
				{ID: "call_1", Type: "function", Function: &providers.FunctionCall{Name: "tool1"}},
				{ID: "call_2", Type: "function", Function: &providers.FunctionCall{Name: "tool2"}},
			},
		},
		// Only one tool result for two tool calls — incomplete
		{Role: "tool", Content: "result1", ToolCallID: "call_1"},
	}

	result := sanitizeHistory(history)

	// Should remove the incomplete tool-call sequence (assistant + partial tool results)
	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d: %+v", len(result), result)
	}
	if result[2].Role != "user" {
		t.Errorf("expected last message to be user, got %s", result[2].Role)
	}
}

func TestSanitizeHistory_Empty(t *testing.T) {
	result := sanitizeHistory(nil)
	if len(result) != 0 {
		t.Fatalf("expected empty result, got %d messages", len(result))
	}
}

func TestSanitizeHistory_TrailingAssistantWithToolCallsNoResults(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
		{Role: "user", Content: "do something"},
		{
			Role: "assistant",
			ToolCalls: []providers.ToolCall{
				{ID: "call_1", Type: "function", Function: &providers.FunctionCall{Name: "tool1"}},
			},
		},
	}

	result := sanitizeHistory(history)

	// Should remove trailing assistant with unanswered tool_calls
	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result))
	}
	if result[2].Role != "user" {
		t.Errorf("expected last message to be user, got %s", result[2].Role)
	}
}
