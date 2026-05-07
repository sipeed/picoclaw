package seahorse

import (
	"context"
	"testing"
)

func TestSanitizeFTS5Query(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Basic tokens
		{"hello world", `"hello" "world"`},
		{"database", `"database"`},

		// FTS5 operators neutralized
		{"sub-agent", `"sub-agent"`},
		{"agent:main", `"agent:main"`},
		{"+required", `"+required"`},
		{"prefix*", `"prefix*"`},
		{"^initial", `"^initial"`},
		{"crash OR restart", `"crash" "OR" "restart"`},
		{"NOT excluded", `"NOT" "excluded"`},
		{"(grouped)", `"(grouped)"`},

		// User-quoted phrases preserved
		{`"exact phrase" other`, `"exact phrase" "other"`},
		{`before "middle phrase" after`, `"before" "middle phrase" "after"`},

		// Unmatched quotes stripped
		{`"unmatched`, `"unmatched"`},
		{`hello"world`, `"helloworld"`},

		// NEAR operator neutralized
		{"NEAR/2 agent", `"NEAR/2" "agent"`},

		// Empty input
		{"", ""},
		{"   ", ""},

		// CJK unaffected
		{"数据库连接", `"数据库连接"`},
		{"数据库 连接", `"数据库" "连接"`},
		{"sub-agent重启", `"sub-agent重启"`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SanitizeFTS5Query(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeFTS5Query(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestFTS5SpecialCharsShouldNotError verifies that user input containing
// FTS5 special characters does not cause errors when searching.
func TestFTS5SpecialCharsShouldNotError(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	conv, _ := s.GetOrCreateConversation(ctx, "test:fts5-sanitize")
	re := &RetrievalEngine{store: s}

	// Seed data with content containing special characters
	s.AddMessage(ctx, conv.ConversationID, "user", "the sub-agent restarted after crash", 10)
	s.AddMessage(ctx, conv.ConversationID, "assistant", "agent:main session restored successfully", 10)
	s.AddMessage(ctx, conv.ConversationID, "user", "use NOT operator in the query filter", 10)
	s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID,
		Kind:           SummaryKindLeaf,
		Depth:          0,
		Content:        "sub-agent crashed and was restarted by the orchestrator",
		TokenCount:     50,
	})
	s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID,
		Kind:           SummaryKindLeaf,
		Depth:          0,
		Content:        "agent:main handled the restart procedure",
		TokenCount:     50,
	})

	tests := []struct {
		name           string
		pattern        string
		wantSummaryMin int
		wantMessageMin int
	}{
		{
			name:           "hyphen in search term",
			pattern:        "sub-agent",
			wantSummaryMin: 1,
			wantMessageMin: 1,
		},
		{
			name:           "colon in search term",
			pattern:        "agent:main",
			wantSummaryMin: 1,
			wantMessageMin: 1,
		},
		{
			name:           "unmatched double quote",
			pattern:        `"sub-agent`,
			wantSummaryMin: 1,
			wantMessageMin: 1,
		},
		{
			name:           "plus sign",
			pattern:        "+agent",
			wantSummaryMin: 0,
			wantMessageMin: 0,
		},
		{
			name:           "parentheses",
			pattern:        "(agent)",
			wantSummaryMin: 0,
			wantMessageMin: 0,
		},
		{
			name:           "NOT keyword",
			pattern:        "NOT operator",
			wantSummaryMin: 0,
			wantMessageMin: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := re.Grep(ctx, GrepInput{
				Pattern: tt.pattern,
				Scope:   "both",
			})
			if err != nil {
				t.Fatalf("Grep(%q) returned error: %v", tt.pattern, err)
			}
			if len(result.Summaries) < tt.wantSummaryMin {
				t.Errorf("Grep(%q) summaries = %d, want >= %d",
					tt.pattern, len(result.Summaries), tt.wantSummaryMin)
			}
			if len(result.Messages) < tt.wantMessageMin {
				t.Errorf("Grep(%q) messages = %d, want >= %d",
					tt.pattern, len(result.Messages), tt.wantMessageMin)
			}
		})
	}
}

// TestFTS5OperatorsNotInterpreted verifies that FTS5 operators are treated
// as literal text, not as query syntax. Each case constructs data where
// boolean interpretation would produce different results than literal matching.
func TestFTS5OperatorsNotInterpreted(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	conv, _ := s.GetOrCreateConversation(ctx, "test:fts5-operators")
	re := &RetrievalEngine{store: s}

	// "restart only" — contains "restart" but NOT "crash".
	// If OR is treated as boolean, "crash OR restart" would match this.
	// With sanitization (literal AND), it should NOT match.
	s.AddMessage(ctx, conv.ConversationID, "user", "restart the service now please", 10)

	// "subcommand" — starts with "sub" but is not "sub-agent".
	// If * is treated as prefix wildcard, "sub*" would match this.
	// With sanitization (literal "sub*"), it should NOT match.
	s.AddMessage(ctx, conv.ConversationID, "user", "run the subcommand to deploy", 10)

	// "agent grouped" — contains "agent" but not "(agent)".
	// If () is treated as grouping, "(agent)" would match this.
	// With sanitization (literal "(agent)"), it should NOT match.
	s.AddMessage(ctx, conv.ConversationID, "user", "the agent processed the request", 10)

	// Same patterns in summaries
	s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID,
		Kind:           SummaryKindLeaf,
		Depth:          0,
		Content:        "restart procedure completed without any crash involvement",
		TokenCount:     50,
	})
	s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: conv.ConversationID,
		Kind:           SummaryKindLeaf,
		Depth:          0,
		Content:        "subprocess and subcommand management overview",
		TokenCount:     50,
	})

	t.Run("OR must not be boolean", func(t *testing.T) {
		// "crash OR restart" as literal means all three tokens must appear.
		// The message "restart the service now please" has "restart" but not "crash" or "OR".
		// Boolean OR would match it; literal AND should not.
		result, err := re.Grep(ctx, GrepInput{Pattern: "crash OR restart", Scope: "message"})
		if err != nil {
			t.Fatalf("Grep returned error: %v", err)
		}
		if len(result.Messages) != 0 {
			t.Errorf(
				"OR treated as boolean: got %d messages, want 0 (only-restart message should not match literal AND of 'crash','OR','restart')",
				len(result.Messages),
			)
		}
	})

	t.Run("asterisk must not be prefix wildcard", func(t *testing.T) {
		// "sub*" as literal means exact trigram match on "sub*".
		// The message "run the subcommand to deploy" contains "sub" as prefix.
		// Prefix wildcard would match it; literal should not.
		result, err := re.Grep(ctx, GrepInput{Pattern: "sub*", Scope: "message"})
		if err != nil {
			t.Fatalf("Grep returned error: %v", err)
		}
		if len(result.Messages) != 0 {
			t.Errorf(
				"asterisk treated as prefix wildcard: got %d messages, want 0 (literal 'sub*' does not appear in any message)",
				len(result.Messages),
			)
		}
	})

	t.Run("parentheses must not be grouping", func(t *testing.T) {
		// "(agent)" as literal means exact trigram match on "(agent)".
		// The message "the agent processed the request" contains "agent" without parens.
		// Grouping would match it; literal should not.
		result, err := re.Grep(ctx, GrepInput{Pattern: "(agent)", Scope: "message"})
		if err != nil {
			t.Fatalf("Grep returned error: %v", err)
		}
		if len(result.Messages) != 0 {
			t.Errorf(
				"parentheses treated as grouping: got %d messages, want 0 (literal '(agent)' does not appear in any message)",
				len(result.Messages),
			)
		}
	})
}
