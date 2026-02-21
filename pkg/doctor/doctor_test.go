package doctor

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestCheckSessionMessages_Clean(t *testing.T) {
	msgs := []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
	}
	problems := checkSessionMessages(msgs)
	if len(problems) != 0 {
		t.Errorf("expected no problems, got %v", problems)
	}
}

func TestCheckSessionMessages_OrphanToolCall(t *testing.T) {
	msgs := []providers.Message{
		{
			Role:    "assistant",
			Content: "let me check",
			ToolCalls: []providers.ToolCall{
				{ID: "call_123", Name: "exec"},
			},
		},
		{Role: "user", Content: "hi"},
	}
	problems := checkSessionMessages(msgs)
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d: %v", len(problems), problems)
	}
	if problems[0] == "" {
		t.Error("problem message should not be empty")
	}
}

func TestCheckSessionMessages_MatchedToolCall(t *testing.T) {
	msgs := []providers.Message{
		{
			Role:    "assistant",
			Content: "let me check",
			ToolCalls: []providers.ToolCall{
				{ID: "call_123", Name: "exec"},
			},
		},
		{Role: "tool", Content: "output", ToolCallID: "call_123"},
		{Role: "assistant", Content: "done"},
	}
	problems := checkSessionMessages(msgs)
	if len(problems) != 0 {
		t.Errorf("expected no problems, got %v", problems)
	}
}

func TestCheckSessionMessages_OrphanToolResult(t *testing.T) {
	msgs := []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "tool", Content: "output", ToolCallID: "call_orphan"},
	}
	problems := checkSessionMessages(msgs)
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d: %v", len(problems), problems)
	}
}

func TestCheckSessionMessages_EmptyAssistant(t *testing.T) {
	msgs := []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: ""},
	}
	problems := checkSessionMessages(msgs)
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d: %v", len(problems), problems)
	}
}

func TestCheckSessionMessages_ConsecutiveUserMessages(t *testing.T) {
	msgs := []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "user", Content: "hello again"},
	}
	problems := checkSessionMessages(msgs)
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d: %v", len(problems), problems)
	}
}

func TestRepairSessionMessages_InjectsResult(t *testing.T) {
	msgs := []providers.Message{
		{Role: "user", Content: "hello"},
		{
			Role: "assistant", Content: "checking",
			ToolCalls: []providers.ToolCall{{ID: "call_1", Name: "exec"}},
		},
		{Role: "assistant", Content: "done"},
	}
	repaired := repairSessionMessages(msgs)
	if len(repaired) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(repaired))
	}
	if repaired[2].Role != "tool" || repaired[2].ToolCallID != "call_1" {
		t.Errorf("expected injected tool_result at index 2, got role=%q id=%q",
			repaired[2].Role, repaired[2].ToolCallID)
	}
}

func TestRepairSessionMessages_DropsOrphanResult(t *testing.T) {
	msgs := []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "tool", Content: "orphaned", ToolCallID: "call_x"},
		{Role: "assistant", Content: "hi"},
	}
	repaired := repairSessionMessages(msgs)
	if len(repaired) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(repaired))
	}
}

func TestRepairSessionMessages_AlreadyClean(t *testing.T) {
	msgs := []providers.Message{
		{Role: "user", Content: "hello"},
		{
			Role: "assistant", Content: "checking",
			ToolCalls: []providers.ToolCall{{ID: "call_1", Name: "exec"}},
		},
		{Role: "tool", Content: "output", ToolCallID: "call_1"},
		{Role: "assistant", Content: "done"},
	}
	repaired := repairSessionMessages(msgs)
	if len(repaired) != 4 {
		t.Errorf("clean messages should be unchanged, got %d", len(repaired))
	}
}

func TestSeverityString(t *testing.T) {
	tests := []struct {
		sev  Severity
		want string
	}{
		{SeverityInfo, "info"},
		{SeverityWarn, "warn"},
		{SeverityError, "ERROR"},
	}
	for _, tt := range tests {
		if got := tt.sev.String(); got != tt.want {
			t.Errorf("Severity(%d).String() = %q, want %q", tt.sev, got, tt.want)
		}
	}
}
