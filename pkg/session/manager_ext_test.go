package session

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestSanitizeHistory_OrphanedToolCall(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "hello"},

		{Role: "assistant", Content: "sure", ToolCalls: []providers.ToolCall{
			{ID: "call_1", Name: "exec"},

			{ID: "call_2", Name: "list_dir"},
		}},

		{Role: "tool", Content: "ok", ToolCallID: "call_1"},
	}

	sanitized, removed := SanitizeHistory(history)

	if removed == 0 {
		t.Fatal("expected orphaned messages to be removed")
	}

	if len(sanitized) != 1 || sanitized[0].Role != "user" {
		t.Errorf("expected [user], got %d messages", len(sanitized))
	}
}

func TestSanitizeHistory_InterleavedMessages(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "first"},

		{Role: "assistant", Content: "ok", ToolCalls: []providers.ToolCall{
			{ID: "call_1", Name: "exec"},
		}},

		{Role: "user", Content: "collision!"},

		{Role: "tool", Content: "ok", ToolCallID: "call_1"},

		{Role: "assistant", Content: "done"},
	}

	sanitized, removed := SanitizeHistory(history)

	if removed == 0 {
		t.Fatal("expected interleaved messages to be removed")
	}

	if len(sanitized) != 3 {
		t.Errorf("expected 3 messages, got %d", len(sanitized))

		for i, m := range sanitized {
			t.Logf("  [%d] role=%s content=%q", i, m.Role, m.Content)
		}
	}
}

func TestSanitizeHistory_CleanHistory(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "hello"},

		{Role: "assistant", Content: "sure", ToolCalls: []providers.ToolCall{
			{ID: "call_1", Name: "exec"},
		}},

		{Role: "tool", Content: "ok", ToolCallID: "call_1"},

		{Role: "assistant", Content: "done"},
	}

	sanitized, removed := SanitizeHistory(history)

	if removed != 0 {
		t.Errorf("expected 0 removed, got %d", removed)
	}

	if len(sanitized) != 4 {
		t.Errorf("expected 4 messages, got %d", len(sanitized))
	}
}

func TestSanitizeHistory_MultipleToolCalls(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "hello"},

		{Role: "assistant", Content: "", ToolCalls: []providers.ToolCall{
			{ID: "call_1", Name: "exec"},

			{ID: "call_2", Name: "read_file"},
		}},

		{Role: "tool", Content: "ok", ToolCallID: "call_1"},

		{Role: "tool", Content: "content", ToolCallID: "call_2"},

		{Role: "assistant", Content: "all done"},
	}

	sanitized, removed := SanitizeHistory(history)

	if removed != 0 {
		t.Errorf("expected 0 removed, got %d", removed)
	}

	if len(sanitized) != 5 {
		t.Errorf("expected 5 messages, got %d", len(sanitized))
	}
}

func TestSanitizeHistory_Empty(t *testing.T) {
	sanitized, removed := SanitizeHistory(nil)

	if removed != 0 || sanitized != nil {
		t.Errorf("expected nil/0, got %v/%d", sanitized, removed)
	}
}
