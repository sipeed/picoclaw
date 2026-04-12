package commands

import (
	"context"
	"strings"
	"testing"
)

func TestCompactCommand_PassesInstructionsToRuntime(t *testing.T) {
	var gotInstructions string
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), &Runtime{
		CompactContext: func(instructions string) (int, error) {
			gotInstructions = instructions
			return 3, nil
		},
	})

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/compact focus on decisions",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if gotInstructions != "focus on decisions" {
		t.Fatalf("instructions=%q, want %q", gotInstructions, "focus on decisions")
	}
	if !strings.Contains(reply, "Messages summarized: 3") {
		t.Fatalf("reply=%q, want summarized count", reply)
	}
}

func TestStatusCommand_ShowsSummaryAndCompactions(t *testing.T) {
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), &Runtime{
		GetModelInfo: func() (string, string) {
			return "gpt-test", "openai"
		},
		GetSessionStats: func() SessionStats {
			return SessionStats{
				Version:        "v1.2.3",
				TokenEstimate:  1200,
				ContextWindow:  8000,
				ContextPercent: 15,
				MessageCount:   6,
				SessionKey:     "agent:main:test",
				ThinkEnabled:   true,
				HasSummary:     true,
			}
		},
		GetCompactionCount: func() int { return 2 },
	})

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/status",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	for _, want := range []string{
		"PicoClaw v1.2.3",
		"gpt-test",
		"History:* 6 messages",
		"Summary:* present",
		"Compactions:* 2",
		"Runtime:* direct",
		"Think:* on",
	} {
		if !strings.Contains(reply, want) {
			t.Fatalf("reply=%q, want substring %q", reply, want)
		}
	}
}
