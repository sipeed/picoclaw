package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestRepairOrphanedToolPairs_NoOrphans(t *testing.T) {
	msgs := []providers.Message{
		{Role: "user", Content: "hello"},
		{
			Role: "assistant", Content: "let me check",
			ToolCalls: []providers.ToolCall{{ID: "call_1", Name: "exec"}},
		},
		{Role: "tool", Content: "output", ToolCallID: "call_1"},
		{Role: "assistant", Content: "done"},
	}
	repaired := repairOrphanedToolPairs(msgs)
	if len(repaired) != 4 {
		t.Errorf("expected 4 messages, got %d", len(repaired))
	}
}

func TestRepairOrphanedToolPairs_OrphanToolUseAtEnd(t *testing.T) {
	msgs := []providers.Message{
		{Role: "user", Content: "hello"},
		{
			Role: "assistant", Content: "let me check",
			ToolCalls: []providers.ToolCall{{ID: "call_1", Name: "exec"}},
		},
	}
	repaired := repairOrphanedToolPairs(msgs)
	if len(repaired) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(repaired))
	}
	if repaired[2].Role != "tool" {
		t.Errorf("expected injected tool message, got role=%q", repaired[2].Role)
	}
	if repaired[2].ToolCallID != "call_1" {
		t.Errorf("expected ToolCallID=call_1, got %q", repaired[2].ToolCallID)
	}
}

func TestRepairOrphanedToolPairs_MultipleToolCallsPartialResults(t *testing.T) {
	msgs := []providers.Message{
		{Role: "user", Content: "hello"},
		{
			Role: "assistant", Content: "checking two things",
			ToolCalls: []providers.ToolCall{
				{ID: "call_1", Name: "exec"},
				{ID: "call_2", Name: "web_fetch"},
			},
		},
		{Role: "tool", Content: "output1", ToolCallID: "call_1"},
	}
	repaired := repairOrphanedToolPairs(msgs)
	if len(repaired) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(repaired))
	}
	if repaired[3].ToolCallID != "call_2" {
		t.Errorf("expected injected result for call_2, got ToolCallID=%q", repaired[3].ToolCallID)
	}
}

func TestRepairOrphanedToolPairs_OrphanToolResultDropped(t *testing.T) {
	msgs := []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "tool", Content: "orphaned output", ToolCallID: "call_orphan"},
		{Role: "assistant", Content: "hi"},
	}
	repaired := repairOrphanedToolPairs(msgs)
	if len(repaired) != 2 {
		t.Fatalf("expected 2 messages (user + assistant), got %d", len(repaired))
	}
	if repaired[0].Role != "user" || repaired[1].Role != "assistant" {
		t.Errorf("unexpected message roles: %q, %q", repaired[0].Role, repaired[1].Role)
	}
}

func TestSanitizeHistoryForProvider_OrphanToolUseRepaired(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "do something"},
		{
			Role: "assistant", Content: "calling tool",
			ToolCalls: []providers.ToolCall{{ID: "call_99", Name: "exec"}},
		},
	}
	sanitized := sanitizeHistoryForProvider(history)

	if len(sanitized) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(sanitized))
	}
	if sanitized[2].Role != "tool" || sanitized[2].ToolCallID != "call_99" {
		t.Errorf("expected synthetic tool_result for call_99, got role=%q id=%q",
			sanitized[2].Role, sanitized[2].ToolCallID)
	}
}

func TestRepairOrphanedToolPairs_EmptyInput(t *testing.T) {
	repaired := repairOrphanedToolPairs(nil)
	if len(repaired) != 0 {
		t.Errorf("expected 0 messages for nil input, got %d", len(repaired))
	}
	repaired = repairOrphanedToolPairs([]providers.Message{})
	if len(repaired) != 0 {
		t.Errorf("expected 0 messages for empty input, got %d", len(repaired))
	}
}
