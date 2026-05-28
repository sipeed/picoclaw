package seahorse

import (
	"context"
	"testing"
)

// =============================================================================
// CompactUntilUnder iteration cap
// =============================================================================

func TestCompactUntilUnderIterationCap(t *testing.T) {
	// Setup: create a conversation with so many tokens that compaction
	// will never reach the budget. The iteration cap prevents infinite loops.
	//
	// We use a mock CompleteFn that always returns the same content,
	// and a budget of 0 which tokens can never reach.
	// Without the cap, this would loop forever.

	db := openTestDB(t)
	if err := runSchema(db); err != nil {
		t.Fatalf("migration: %v", err)
	}
	s := &Store{db: db}

	conv, _ := s.GetOrCreateConversation(context.Background(), "agent:iter-cap")
	convID := conv.ConversationID

	// Add many messages to ensure there's plenty to compact
	for i := 0; i < 40; i++ {
		m, _ := s.AddMessage(context.Background(), convID, "user",
			"this is a long message with lots of tokens to push context over budget", 100)
		s.AppendContextMessage(context.Background(), convID, m.ID)
	}

	// A completeFn that always succeeds but returns non-reducing content
	mockComplete := func(ctx context.Context, prompt string, opts CompleteOptions) (string, error) {
		return "Summary that doesn't reduce tokens much.", nil
	}

	ce, cancel := newTestCompactionEngineWithStore(s, mockComplete)
	defer cancel()

	// Use budget=1 so tokens can never reach budget
	// (each message is 100 tokens, so 40 messages = 4000 tokens, budget 1 is unreachable)
	// The function should stop after maxCompactIterations, not loop forever
	ce.config = Config{} // ensure defaults

	result, err := ce.CompactUntilUnder(context.Background(), convID, 1)
	if err != nil {
		// Should not error — should stop gracefully
		t.Fatalf("CompactUntilUnder with budget=0: %v", err)
	}

	// The function should have completed within reasonable time
	// If it exceeded the cap, it would still return (not hang)
	_ = result
}
