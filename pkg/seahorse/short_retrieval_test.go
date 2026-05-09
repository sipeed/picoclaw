package seahorse

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// --- Retrieval Tests ---

func newTestRetrieval(t *testing.T) (*RetrievalEngine, *Store, int64) {
	t.Helper()
	s := openTestStore(t)
	ctx := context.Background()
	conv, _ := s.GetOrCreateConversation(ctx, "test:retrieval")
	return &RetrievalEngine{store: s}, s, conv.ConversationID
}

func TestRetrievalGrepSummaries(t *testing.T) {
	r, s, convID := newTestRetrieval(t)
	ctx := context.Background()

	s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: convID,
		Kind:           SummaryKindLeaf,
		Depth:          0,
		Content:        "数据库连接配置说明",
		TokenCount:     50,
	})
	s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: convID,
		Kind:           SummaryKindLeaf,
		Depth:          0,
		Content:        "API endpoint documentation",
		TokenCount:     50,
	})

	// FTS5 search (trigram, needs >= 3 chars)
	results, err := r.Grep(ctx, GrepInput{
		Pattern: "数据库连",
	})
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}
	if len(results.Summaries) == 0 {
		t.Error("expected at least 1 FTS result")
	}

	// LIKE search with wildcard
	results, err = r.Grep(ctx, GrepInput{
		Pattern: "%endpoint%",
	})
	if err != nil {
		t.Fatalf("Grep LIKE: %v", err)
	}
	if len(results.Summaries) == 0 {
		t.Error("expected at least 1 LIKE result")
	}
}

func TestRetrievalGrepMessages(t *testing.T) {
	r, s, convID := newTestRetrieval(t)
	ctx := context.Background()

	s.AddMessage(ctx, convID, "user", "find this message about testing", 5)
	s.AddMessage(ctx, convID, "user", "unrelated content here", 5)

	results, err := r.Grep(ctx, GrepInput{
		Pattern: "testing",
	})
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}
	if len(results.Messages) == 0 {
		t.Error("expected at least 1 result for 'testing'")
	}
}

func TestRetrievalExpandMessages(t *testing.T) {
	r, s, convID := newTestRetrieval(t)
	ctx := context.Background()

	msg, _ := s.AddMessage(ctx, convID, "user", "expand this message", 10)

	result, err := r.ExpandMessages(ctx, []int64{msg.ID})
	if err != nil {
		t.Fatalf("ExpandMessages: %v", err)
	}
	if len(result.Messages) != 1 {
		t.Errorf("Messages = %d, want 1", len(result.Messages))
	}
	if result.Messages[0].Content != "expand this message" {
		t.Errorf("Content = %q, want 'expand this message'", result.Messages[0].Content)
	}
}

func TestRetrievalExpandMultipleMessages(t *testing.T) {
	r, s, convID := newTestRetrieval(t)
	ctx := context.Background()

	msg1, _ := s.AddMessage(ctx, convID, "user", "first message", 10)
	msg2, _ := s.AddMessage(ctx, convID, "assistant", "second message", 10)
	msg3, _ := s.AddMessage(ctx, convID, "user", "third message", 10)

	result, err := r.ExpandMessages(ctx, []int64{msg1.ID, msg2.ID, msg3.ID})
	if err != nil {
		t.Fatalf("ExpandMessages: %v", err)
	}
	if len(result.Messages) != 3 {
		t.Errorf("Messages = %d, want 3", len(result.Messages))
	}
	if result.TokenCount != 30 {
		t.Errorf("TokenCount = %d, want 30", result.TokenCount)
	}
}

func TestRetrievalGrepWithTimeFilter(t *testing.T) {
	r, s, convID := newTestRetrieval(t)
	ctx := context.Background()

	now := time.Now().UTC()
	before := now.Add(-2 * time.Hour)

	// Create messages at different times
	s.AddMessage(ctx, convID, "user", "old message about auth", 5)
	s.AddMessage(ctx, convID, "user", "recent message about auth", 5)

	// Search with time filter
	results, err := r.Grep(ctx, GrepInput{
		Pattern: "auth",
		Since:   &before,
	})
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}
	_ = results // Just verify no error
}

func TestRetrievalGrepAllConversations(t *testing.T) {
	r, s, _ := newTestRetrieval(t)
	ctx := context.Background()

	// Create another conversation
	conv2, _ := s.GetOrCreateConversation(ctx, "test:retrieval2")

	// Add messages to both
	s.AddMessage(ctx, conv2.ConversationID, "user", "unique keyword xyz", 5)

	// Search all conversations
	results, err := r.Grep(ctx, GrepInput{
		Pattern:          "xyz",
		AllConversations: true,
	})
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}
	if len(results.Messages) == 0 {
		t.Error("expected to find message in other conversation")
	}
}

// --- Last Duration Parsing Tests ---

func TestParseLastDuration(t *testing.T) {
	tests := []struct {
		input   string
		wantDur time.Duration
		wantErr bool
	}{
		{"6h", 6 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"2w", 14 * 24 * time.Hour, false},
		{"1m", 30 * 24 * time.Hour, false}, // month = 30 days
		{"3m", 90 * 24 * time.Hour, false},
		{"", 0, true},
		{"invalid", 0, true},
		{"5x", 0, true}, // unknown unit
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseLastDuration(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tt.wantDur {
					t.Errorf("ParseLastDuration(%q) = %v, want %v", tt.input, got, tt.wantDur)
				}
			}
		})
	}
}

// --- Role Filter Tests ---

func TestRetrievalGrepRoleFilter(t *testing.T) {
	r, s, convID := newTestRetrieval(t)
	ctx := context.Background()

	s.AddMessage(ctx, convID, "user", "user message about alpha", 5)
	s.AddMessage(ctx, convID, "assistant", "assistant reply about alpha", 5)
	s.AddMessage(ctx, convID, "user", "another user message", 5)

	// Search all roles
	allResults, err := r.Grep(ctx, GrepInput{
		Pattern: "alpha",
	})
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}
	if len(allResults.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(allResults.Messages))
	}

	// Search user only
	userResults, err := r.Grep(ctx, GrepInput{
		Pattern: "alpha",
		Role:    "user",
	})
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}
	if len(userResults.Messages) != 1 {
		t.Errorf("expected 1 user message, got %d", len(userResults.Messages))
	}
	if userResults.Messages[0].Role != "user" {
		t.Errorf("expected role=user, got %s", userResults.Messages[0].Role)
	}

	// Search assistant only
	assistantResults, err := r.Grep(ctx, GrepInput{
		Pattern: "alpha",
		Role:    "assistant",
	})
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}
	if len(assistantResults.Messages) != 1 {
		t.Errorf("expected 1 assistant message, got %d", len(assistantResults.Messages))
	}
}

// --- Last Parameter Tests ---

func TestRetrievalGrepWithLast(t *testing.T) {
	r, s, convID := newTestRetrieval(t)
	ctx := context.Background()

	// Add messages (we can't control timestamps in SQLite easily,
	// but we can verify the parameter is parsed correctly)
	s.AddMessage(ctx, convID, "user", "recent message about testing", 5)

	// Test that Last parameter is converted to Since
	results, err := r.Grep(ctx, GrepInput{
		Pattern: "testing",
		Last:    "1d", // last 1 day
	})
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}
	// Should still find the message since it's recent
	if len(results.Messages) == 0 {
		t.Error("expected to find recent message")
	}
}

// TestRetrievalGrepRoleFilterWithSummaries tests that role filter works when
// searching both summaries and messages (summaries don't have role column).
func TestRetrievalGrepRoleFilterWithSummaries(t *testing.T) {
	r, s, convID := newTestRetrieval(t)
	ctx := context.Background()

	// Create a summary (no role column)
	s.CreateSummary(ctx, CreateSummaryInput{
		ConversationID: convID,
		Kind:           SummaryKindLeaf,
		Depth:          0,
		Content:        "summary about testing",
		TokenCount:     50,
	})

	// Add messages with different roles
	s.AddMessage(ctx, convID, "user", "user message about testing", 5)
	s.AddMessage(ctx, convID, "assistant", "assistant reply about testing", 5)

	// Search with role filter and scope=both (default), using LIKE mode (%)
	// This should NOT error even though summaries don't have role column
	bothResults, err := r.Grep(ctx, GrepInput{
		Pattern: "%testing%", // LIKE mode to trigger the bug
		Role:    "user",
		Scope:   "both",
	})
	if err != nil {
		t.Fatalf("Grep with role and scope=both: %v", err)
	}

	// Should only return user messages, not summaries or assistant messages
	if len(bothResults.Messages) != 1 {
		t.Errorf("expected 1 user message, got %d", len(bothResults.Messages))
	}
	if len(bothResults.Messages) > 0 && bothResults.Messages[0].Role != "user" {
		t.Errorf("expected role=user, got %s", bothResults.Messages[0].Role)
	}

	// Summaries should be empty since they don't have roles to filter
	// (or we could return all summaries - either is acceptable)
}

// TestRetrievalGrepTotalCounts tests that grep returns total counts.
func TestRetrievalGrepTotalCounts(t *testing.T) {
	r, s, convID := newTestRetrieval(t)
	ctx := context.Background()

	// Create 3 summaries
	for i := 0; i < 3; i++ {
		s.CreateSummary(ctx, CreateSummaryInput{
			ConversationID: convID,
			Kind:        SummaryKindLeaf,
			Depth:       0,
			Content:    fmt.Sprintf("summary about testing %d", i),
			TokenCount: 50,
		})
	}

	// Add 5 messages
	for i := 0; i < 5; i++ {
		s.AddMessage(ctx, convID, "user", fmt.Sprintf("message about testing %d", i), 5)
	}

	// Search with limit smaller than total
	results, err := r.Grep(ctx, GrepInput{
		Pattern: "%testing%", // LIKE mode
		Scope:   "both",
		Limit:   2,
	})
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}

	// Should return limited results
	if len(results.Summaries) > 2 {
		t.Errorf("expected at most 2 summaries, got %d", len(results.Summaries))
	}
	if len(results.Messages) > 2 {
		t.Errorf("expected at most 2 messages, got %d", len(results.Messages))
	}

	// But total counts should reflect all matches
	if results.TotalSummaries != 3 {
		t.Errorf("expected TotalSummaries=3, got %d", results.TotalSummaries)
	}
	if results.TotalMessages != 5 {
		t.Errorf("expected TotalMessages=5, got %d", results.TotalMessages)
	}
}

func TestSearchSessionsByTag(t *testing.T) {
	r, _, _ := newTestRetrieval(t)
	results, err := r.SearchByTag("coding")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for placeholder, got %d", len(results))
	}
}

	// Add 5 messages
	for i := 0; i < 5; i++ {
		s.AddMessage(ctx, convID, "user", fmt.Sprintf("message about testing %d", i), 5)
	}

	// Search with limit smaller than total
	results, err := r.Grep(ctx, GrepInput{
		Pattern: "%testing%", // LIKE mode
		Scope:   "both",
		Limit:   2,
	})
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}

	// Should return limited results
	if len(results.Summaries) > 2 {
		t.Errorf("expected at most 2 summaries, got %d", len(results.Summaries))
	}
	if len(results.Messages) > 2 {
		t.Errorf("expected at most 2 messages, got %d", len(results.Messages))
	}

	// But total counts should reflect all matches
	if results.TotalSummaries != 3 {
		t.Errorf("expected TotalSummaries=3, got %d", results.TotalSummaries)
	}
	if results.TotalMessages != 5 {
		t.Errorf("expected TotalMessages=5, got %d", results.TotalMessages)
	}
}
