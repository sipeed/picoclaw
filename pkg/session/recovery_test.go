package session

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestSanitizeRecoveredHistory_DropsDanglingToolCallTail(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "working", ToolCalls: []providers.ToolCall{{ID: "call_1"}}},
		{Role: "user", Content: "?"},
	}

	got := sanitizeRecoveredHistory(history)
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].Content != "hello" {
		t.Fatalf("got[0].Content = %q, want hello", got[0].Content)
	}
}

func TestSanitizeRecoveredHistory_KeepsCompletedToolCallSequence(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "working", ToolCalls: []providers.ToolCall{{ID: "call_1"}, {ID: "call_2"}}},
		{Role: "tool", ToolCallID: "call_1", Content: "done 1"},
		{Role: "tool", ToolCallID: "call_2", Content: "done 2"},
		{Role: "assistant", Content: "all set"},
	}

	got := sanitizeRecoveredHistory(history)
	if len(got) != len(history) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(history))
	}
	if got[len(got)-1].Content != "all set" {
		t.Fatalf("last content = %q, want all set", got[len(got)-1].Content)
	}
}

func TestSanitizeRecoveredHistory_DropsPartialToolResultsAndFollowingMessages(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "working", ToolCalls: []providers.ToolCall{{ID: "call_1"}, {ID: "call_2"}}},
		{Role: "tool", ToolCallID: "call_1", Content: "done 1"},
		{Role: "user", Content: "still there?"},
	}

	got := sanitizeRecoveredHistory(history)
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].Content != "hello" {
		t.Fatalf("got[0].Content = %q, want hello", got[0].Content)
	}
}
