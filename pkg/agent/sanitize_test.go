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

func TestSanitizeHistoryForProvider_MultiToolCallsPreserved(t *testing.T) {
	// Reproduces the actual bug: assistant with 2 tool_calls, both have results,
	// but the sanitizer was dropping the second tool_result because it only checked
	// if the immediately preceding message was an assistant (not another tool).
	history := []providers.Message{
		{Role: "user", Content: "do something"},
		{
			Role: "assistant", Content: "checking two things",
			ToolCalls: []providers.ToolCall{
				{ID: "call_A", Name: "read_file"},
				{ID: "call_B", Name: "read_file"},
			},
		},
		{Role: "tool", Content: "file contents A", ToolCallID: "call_A"},
		{Role: "tool", Content: "file contents B", ToolCallID: "call_B"},
		{Role: "assistant", Content: "here are the results"},
	}
	sanitized := sanitizeHistoryForProvider(history)

	if len(sanitized) != 5 {
		t.Fatalf("expected 5 messages (all preserved), got %d", len(sanitized))
	}
	// Verify both tool results are present
	if sanitized[2].Role != "tool" || sanitized[2].ToolCallID != "call_A" {
		t.Errorf("message[2]: expected tool result for call_A, got role=%q id=%q",
			sanitized[2].Role, sanitized[2].ToolCallID)
	}
	if sanitized[3].Role != "tool" || sanitized[3].ToolCallID != "call_B" {
		t.Errorf("message[3]: expected tool result for call_B, got role=%q id=%q",
			sanitized[3].Role, sanitized[3].ToolCallID)
	}
}

func TestSanitizeHistoryForProvider_RealSessionRegression(t *testing.T) {
	// Reproduces the exact pattern from the user's corrupt session:
	// assistant(2 calls) -> tool -> tool -> assistant -> user ->
	// assistant(2 calls) -> tool -> tool -> assistant(1 call) -> tool -> ...
	history := []providers.Message{
		{Role: "assistant", Content: "", ToolCalls: []providers.ToolCall{
			{ID: "tc1", Name: "read_file"}, {ID: "tc2", Name: "read_file"},
		}},
		{Role: "tool", Content: "file1", ToolCallID: "tc1"},
		{Role: "tool", Content: "file2", ToolCallID: "tc2"},
		{Role: "assistant", Content: "summary"},
		{Role: "user", Content: "do it"},
		{Role: "assistant", Content: "checking", ToolCalls: []providers.ToolCall{
			{ID: "tc3", Name: "list_dir"}, {ID: "tc4", Name: "read_file"},
		}},
		{Role: "tool", Content: "denied", ToolCallID: "tc3"},
		{Role: "tool", Content: "denied", ToolCallID: "tc4"},
		{Role: "assistant", Content: "", ToolCalls: []providers.ToolCall{
			{ID: "tc5", Name: "exec"},
		}},
		{Role: "tool", Content: "output", ToolCallID: "tc5"},
	}
	sanitized := sanitizeHistoryForProvider(history)

	// Count tool_use and tool_result IDs — must be balanced
	toolCallIDs := map[string]bool{}
	toolResultIDs := map[string]bool{}
	for _, m := range sanitized {
		for _, tc := range m.ToolCalls {
			toolCallIDs[tc.ID] = true
		}
		if m.Role == "tool" && m.ToolCallID != "" {
			toolResultIDs[m.ToolCallID] = true
		}
	}
	for id := range toolCallIDs {
		if !toolResultIDs[id] {
			t.Errorf("orphaned tool_call %q — no matching tool_result after sanitize", id)
		}
	}
	for id := range toolResultIDs {
		if !toolCallIDs[id] {
			t.Errorf("orphaned tool_result %q — no matching tool_call after sanitize", id)
		}
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
