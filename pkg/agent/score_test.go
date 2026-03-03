package agent

import (
	"strings"
	"testing"
)

func TestCalcTurnScore_BasicRules(t *testing.T) {
	tests := []struct {
		name     string
		input    RuntimeInput
		wantMin  int
		wantMax  int
		wantExact *int
	}{
		{
			name:  "empty chat",
			input: RuntimeInput{Intent: "chat", UserMessage: "ok", AssistantReply: "ok"},
			// score = 0 (chat) -2 (< 80 chars total) = -2
			wantExact: intPtr(-2),
		},
		{
			name:  "question intent, short",
			input: RuntimeInput{Intent: "question", UserMessage: "hi", AssistantReply: "hello"},
			// score = 1 (question) -2 (short) = -1
			wantExact: intPtr(-1),
		},
		{
			name: "task with tool call",
			input: RuntimeInput{
				Intent:         "task",
				UserMessage:    "do something important",
				AssistantReply: "done",
				ToolCalls:      []ToolCallRecord{{Name: "exec"}},
			},
			// +3 (task) +3 (has tool) +3 ("important" keyword) -2 (short) = 7
			wantExact: intPtr(7),
		},
		{
			name: "code with write tool",
			input: RuntimeInput{
				Intent:         "code",
				UserMessage:    "fix the bug",
				AssistantReply: "fixed",
				ToolCalls:      []ToolCallRecord{{Name: "write_file"}},
			},
			// +3 (code) +3 (has tool) +2 (write tool) -2 (short) = 6
			wantExact: intPtr(6),
		},
		{
			name: "many tools",
			input: RuntimeInput{
				Intent:         "debug",
				UserMessage:    "debug it",
				AssistantReply: "ok",
				ToolCalls: []ToolCallRecord{
					{Name: "exec"},
					{Name: "read_file"},
					{Name: "list_dir"},
					{Name: "exec"},
				},
			},
			// +3 (debug) +3 (has tool) +2 (>3 tools) -2 (short) = 6
			wantExact: intPtr(6),
		},
		{
			name: "long reply",
			input: RuntimeInput{
				Intent:         "question",
				UserMessage:    "explain",
				AssistantReply: strings.Repeat("a", 600),
			},
			// +1 (question) +2 (long reply)  [total<80 does not apply because reply is 600]
			// total chars = 7 + 600 = 607 >= 80
			wantExact: intPtr(3),
		},
		{
			name: "explicit remember keyword",
			input: RuntimeInput{
				Intent:         "chat",
				UserMessage:    "记住这个地址 localhost:3000",
				AssistantReply: strings.Repeat("a", 600),
			},
			// 0(chat) +3 (记住) +2 (long reply) = 5
			wantExact: intPtr(5),
		},
		{
			name: "explicit important keyword",
			input: RuntimeInput{
				Intent:         "question",
				UserMessage:    "this is IMPORTANT: use port 8080",
				AssistantReply: "ok",
			},
			// 1 (question) + 3 (important) - 2 (short) = 2
			wantExact: intPtr(2),
		},
		{
			name: "always_keep threshold: full scoring",
			input: RuntimeInput{
				Intent:         "task",
				UserMessage:    "run the deployment pipeline for staging and fix it",
				AssistantReply: strings.Repeat("a", 600),
				ToolCalls: []ToolCallRecord{
					{Name: "edit_file"},
					{Name: "exec"},
					{Name: "exec"},
					{Name: "exec"},
				},
			},
			// +3(task) +3(tool) +2(write/edit) +2(>3 tools) +2(long reply) = 12
			wantExact: intPtr(12),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CalcTurnScore(tc.input)
			if tc.wantExact != nil {
				if got != *tc.wantExact {
					t.Errorf("CalcTurnScore() = %d, want %d", got, *tc.wantExact)
				}
			} else if got < tc.wantMin || (tc.wantMax > 0 && got > tc.wantMax) {
				t.Errorf("CalcTurnScore() = %d, want [%d, %d]", got, tc.wantMin, tc.wantMax)
			}
		})
	}
}

func TestAlwaysKeepThreshold(t *testing.T) {
	// High-value turn must meet or exceed the threshold.
	highValue := RuntimeInput{
		Intent:         "task",
		UserMessage:    "deploy staging",
		AssistantReply: strings.Repeat("a", 600),
		ToolCalls:      []ToolCallRecord{{Name: "edit_file"}, {Name: "exec"}},
	}
	score := CalcTurnScore(highValue)
	if score < alwaysKeepThreshold {
		t.Errorf("expected score %d >= alwaysKeepThreshold %d", score, alwaysKeepThreshold)
	}
}

func intPtr(i int) *int { return &i }
