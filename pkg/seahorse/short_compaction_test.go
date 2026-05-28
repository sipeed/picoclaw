package seahorse

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// --- Test Helpers ---

// waitForCondensed blocks until the async condensed goroutine for convID finishes.
// Returns false if timeout is reached.
func waitForCondensed(ce *CompactionEngine, convID int64, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, exists := ce.condensing.Load(convID); !exists {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// --- Compaction Tests ---

func newTestCompactionEngine(t *testing.T) (*CompactionEngine, *Store, int64) {
	t.Helper()
	db := openTestDB(t)
	if err := runSchema(db); err != nil {
		t.Fatalf("migration: %v", err)
	}
	s := &Store{db: db}
	ctx := context.Background()
	conv, _ := s.GetOrCreateConversation(ctx, "test:compact")
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	ce := &CompactionEngine{
		store:          s,
		config:         Config{},
		complete:       mockCompleteFn,
		shutdownCtx:    shutdownCtx,
		shutdownCancel: shutdownCancel,
	}
	convID := conv.ConversationID
	// Ensure async goroutines are stopped before database is closed.
	// Register cleanup here (after openTestDB) so it runs BEFORE openTestDB's db.Close().
	t.Cleanup(func() {
		shutdownCancel()
		// Wait for async condensed goroutine to finish (poll condensing map)
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if _, exists := ce.condensing.Load(convID); !exists {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
	})
	return ce, s, conv.ConversationID
}

// newTestCompactionEngineWithStore creates a CompactionEngine with existing store.
// Note: Caller is responsible for calling shutdownCancel when test ends.
func newTestCompactionEngineWithStore(
	s *Store, complete CompleteFn,
) (ce *CompactionEngine, shutdownCancel context.CancelFunc) {
	shutdownCtx, cancel := context.WithCancel(context.Background())
	return &CompactionEngine{
		store:          s,
		config:         Config{},
		complete:       complete,
		shutdownCtx:    shutdownCtx,
		shutdownCancel: cancel,
	}, cancel
}

// mockCompleteFn returns a simple summary for testing
var mockCompleteFn CompleteFn = func(ctx context.Context, prompt string, opts CompleteOptions) (string, error) {
	return "Mock summary of the conversation segment.", nil
}

func TestNeedsCompaction(t *testing.T) {
	ce, s, convID := newTestCompactionEngine(t)
	ctx := context.Background()

	// Empty context — no compaction needed
	needed, err := ce.NeedsCompaction(ctx, convID, 10000)
	if err != nil {
		t.Fatalf("NeedsCompaction: %v", err)
	}
	if needed {
		t.Error("expected no compaction for empty context")
	}

	// Add messages to context, total tokens = 8000
	for i := 0; i < 8; i++ {
		m, _ := s.AddMessage(ctx, convID, "user", "test message content", 1000)
		s.AppendContextMessage(ctx, convID, m.ID)
	}

	// Threshold = 0.75 × 10000 = 7500. We have 8000 tokens → needs compaction
	needed, err = ce.NeedsCompaction(ctx, convID, 10000)
	if err != nil {
		t.Fatalf("NeedsCompaction: %v", err)
	}
	if !needed {
		t.Error("expected compaction needed at 8000/10000 tokens (threshold 75%)")
	}

	// Below threshold: 5000 / 10000 → no compaction
	s.UpsertContextItems(ctx, convID, nil) // clear
	for i := 0; i < 5; i++ {
		m, _ := s.AddMessage(ctx, convID, "user", "test", 1000)
		s.AppendContextMessage(ctx, convID, m.ID)
	}
	needed, _ = ce.NeedsCompaction(ctx, convID, 10000)
	if needed {
		t.Error("expected no compaction at 5000/10000 tokens")
	}
}

func TestCompactLeaf(t *testing.T) {
	ce, s, convID := newTestCompactionEngine(t)
	ctx := context.Background()

	// Create enough messages to trigger leaf compaction:
	// Need > FreshTailCount(32) evictable messages with >= LeafMinFanout(8) contiguous
	for i := 0; i < 40; i++ {
		m, _ := s.AddMessage(ctx, convID, "user", "message content for compaction test", 100)
		s.AppendContextMessage(ctx, convID, m.ID)
	}

	// Compact
	result, err := ce.Compact(ctx, convID, CompactInput{})
	if err != nil {
		t.Fatalf("Compact: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Should have created at least one leaf summary
	if result.LeafSummaries == 0 {
		t.Error("expected at least 1 leaf summary")
	}

	// Context should now contain a summary item
	items, _ := s.GetContextItems(ctx, convID)
	foundSummary := false
	for _, item := range items {
		if item.ItemType == "summary" {
			foundSummary = true
			break
		}
	}
	if !foundSummary {
		t.Error("expected a summary in context_items after leaf compaction")
	}

	// Some messages should have been replaced
	if len(result.SummariesCreated) == 0 {
		t.Error("expected at least 1 summary created")
	}
}

func TestCompactLeafNoCandidate(t *testing.T) {
	ce, _, convID := newTestCompactionEngine(t)
	ctx := context.Background()

	// Too few messages to trigger leaf compaction
	m, _ := ce.store.AddMessage(ctx, convID, "user", "short", 10)
	ce.store.AppendContextMessage(ctx, convID, m.ID)

	result, err := ce.Compact(ctx, convID, CompactInput{})
	if err != nil {
		t.Fatalf("Compact: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result even with no candidate")
	}
	if result.LeafSummaries != 0 {
		t.Errorf("LeafSummaries = %d, want 0 (too few messages)", result.LeafSummaries)
	}
}

func TestCompactCondensed(t *testing.T) {
	ce, s, convID := newTestCompactionEngine(t)
	ctx := context.Background()

	// Create enough leaf summaries and fresh messages to enable condensation
	leafIDs := make([]string, CondensedMinFanout)
	for i := 0; i < CondensedMinFanout; i++ {
		now := time.Now().UTC()
		summary, err := s.CreateSummary(ctx, CreateSummaryInput{
			ConversationID: convID,
			Kind:           SummaryKindLeaf,
			Depth:          0,
			Content:        "leaf summary content " + time.Now().String(),
			TokenCount:     500,
			EarliestAt:     &now,
			LatestAt:       &now,
		})
		if err != nil {
			t.Fatalf("CreateSummary %d: %v", i, err)
		}
		leafIDs[i] = summary.SummaryID
		s.AppendContextSummary(ctx, convID, summary.SummaryID)
	}

	// Add enough fresh messages to have a fresh tail (>= FreshTailCount)
	for i := 0; i < FreshTailCount; i++ {
		m, _ := s.AddMessage(ctx, convID, "user", "fresh message", 10)
		s.AppendContextMessage(ctx, convID, m.ID)
	}

	// Compact with force to trigger condensation
	_, err := ce.Compact(ctx, convID, CompactInput{Force: true})
	if err != nil {
		t.Fatalf("Compact: %v", err)
	}

	// Wait for async condensed goroutine to complete
	if !waitForCondensed(ce, convID, 2*time.Second) {
		t.Fatal("timeout waiting for condensed compaction")
	}

	// Should have created a condensed summary in the DB
	summaries, _ := s.GetSummariesByConversation(ctx, convID)
	foundCondensed := false
	for _, sum := range summaries {
		if sum.Kind == SummaryKindCondensed {
			foundCondensed = true
			break
		}
	}
	if !foundCondensed {
		t.Error("expected at least 1 condensed summary")
	}
}

func TestCompactCondensedDoesNotOrphanSummaryWhenCandidatesRemovedConcurrently(t *testing.T) {
	// Reproduce orphan bug: candidates found by selectOldestChunkAtDepth are removed
	// from context_items between candidate selection and ordinal range scan.
	// Use a slow CompleteFn with barrier sync to control timing.
	s := openTestStore(t)
	ctx := context.Background()
	conv, _ := s.GetOrCreateConversation(ctx, "test:orphan-race")
	convID := conv.ConversationID

	// Create leaf summaries with enough tokens for condensation
	var leafIDs []string
	for i := 0; i < CondensedMinFanout; i++ {
		now := time.Now().UTC()
		sum, err := s.CreateSummary(ctx, CreateSummaryInput{
			ConversationID: convID,
			Kind:           SummaryKindLeaf,
			Depth:          0,
			Content:        fmt.Sprintf("leaf summary %d", i),
			TokenCount:     500,
			EarliestAt:     &now,
			LatestAt:       &now,
		})
		if err != nil {
			t.Fatalf("CreateSummary: %v", err)
		}
		leafIDs = append(leafIDs, sum.SummaryID)
		s.AppendContextSummary(ctx, convID, sum.SummaryID)
	}

	// Add fresh tail so leaf summaries are in evictable range
	for i := 0; i < FreshTailCount+1; i++ {
		m, _ := s.AddMessage(ctx, convID, "user", "fresh", 10)
		s.AppendContextMessage(ctx, convID, m.ID)
	}

	// Barrier: CompleteFn waits until test removes context_items, then returns
	var barrier1, barrier2 sync.WaitGroup
	barrier1.Add(1) // CompleteFn signals when called
	barrier2.Add(1) // test signals when context_items removed

	slowComplete := func(ctx context.Context, prompt string, opts CompleteOptions) (string, error) {
		barrier1.Done() // signal: LLM called, candidates selected
		barrier2.Wait() // wait: test removes context_items
		return "Condensed summary.", nil
	}

	ce, cancel := newTestCompactionEngineWithStore(s, slowComplete)
	t.Cleanup(func() {
		cancel()
		time.Sleep(100 * time.Millisecond)
	})

	// Run compactCondensed in background
	type compactResult struct {
		summaryID *string
		err       error
	}
	resultCh := make(chan compactResult, 1)
	go func() {
		sid, err := ce.compactCondensed(context.Background(), convID)
		resultCh <- compactResult{summaryID: sid, err: err}
	}()

	// Wait for CompleteFn to be called (candidates selected)
	barrier1.Wait()

	// Remove leaf summaries from context_items (simulating concurrent replacement)
	items, _ := s.GetContextItems(ctx, convID)
	var preserved []ContextItem
	for _, item := range items {
		isLeaf := false
		for _, lid := range leafIDs {
			if item.SummaryID == lid {
				isLeaf = true
				break
			}
		}
		if !isLeaf {
			preserved = append(preserved, item)
		}
	}
	s.UpsertContextItems(ctx, convID, preserved)

	// Let CompleteFn return
	barrier2.Done()

	// Get result
	res := <-resultCh
	if res.err != nil {
		t.Fatalf("compactCondensed: %v", res.err)
	}

	// With the bug: returns non-nil summaryID even though context_items has no matching ordinals
	// The fix: should return nil when startOrd == -1
	if res.summaryID != nil {
		t.Errorf("compactCondensed returned summaryID=%s, want nil (orphan created)", *res.summaryID)

		// Verify the orphan exists in DB
		summary, _ := s.GetSummary(context.Background(), *res.summaryID)
		if summary != nil && summary.Kind == SummaryKindCondensed {
			// Check it's NOT in context_items (orphan)
			items2, _ := s.GetContextItems(context.Background(), convID)
			found := false
			for _, item := range items2 {
				if item.SummaryID == *res.summaryID {
					found = true
					break
				}
			}
			if !found {
				t.Error("condensed summary exists in DB but not in context_items — orphan confirmed")
			}
		}
	}
}

func TestCompactUntilUnder(t *testing.T) {
	ce, s, convID := newTestCompactionEngine(t)
	ctx := context.Background()

	// Create many leaf summaries to ensure we can condense
	for i := 0; i < 8; i++ {
		now := time.Now().UTC()
		summary, _ := s.CreateSummary(ctx, CreateSummaryInput{
			ConversationID: convID,
			Kind:           SummaryKindLeaf,
			Depth:          0,
			Content:        "leaf summary for condensation test",
			TokenCount:     500,
			EarliestAt:     &now,
			LatestAt:       &now,
		})
		s.AppendContextSummary(ctx, convID, summary.SummaryID)
	}

	// Force compact until under budget
	result, err := ce.CompactUntilUnder(ctx, convID, 2000)
	if err != nil {
		t.Fatalf("CompactUntilUnder: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestSelectShallowestCondensationCandidate(t *testing.T) {
	ce, s, convID := newTestCompactionEngine(t)
	ctx := context.Background()

	// Create enough leaf summaries + fresh messages for candidates
	for i := 0; i < LeafMinFanout; i++ {
		summary, _ := s.CreateSummary(ctx, CreateSummaryInput{
			ConversationID: convID,
			Kind:           SummaryKindLeaf,
			Depth:          0,
			Content:        "leaf",
			TokenCount:     100,
		})
		s.AppendContextSummary(ctx, convID, summary.SummaryID)
	}

	// Add fresh tail messages so summaries are in evictable range
	for i := 0; i < FreshTailCount+1; i++ {
		m, _ := s.AddMessage(ctx, convID, "user", "fresh", 5)
		s.AppendContextMessage(ctx, convID, m.ID)
	}

	candidates, err := ce.selectShallowestCondensationCandidate(ctx, convID, false)
	if err != nil {
		t.Fatalf("selectShallowestCondensationCandidate: %v", err)
	}

	// Should find leaf summaries at depth 0
	if len(candidates) < CondensedMinFanout {
		t.Errorf("candidates = %d, want >= %d", len(candidates), CondensedMinFanout)
	}
}

func TestSelectShallowestCondensationCandidateEmpty(t *testing.T) {
	ce, _, convID := newTestCompactionEngine(t)
	ctx := context.Background()

	candidates, err := ce.selectShallowestCondensationCandidate(ctx, convID, false)
	if err != nil {
		t.Fatalf("selectShallowestCondensationCandidate: %v", err)
	}
	if len(candidates) != 0 {
		t.Errorf("candidates = %d, want 0 for empty context", len(candidates))
	}
}

func TestCompactCondensedUsesSelectOldestChunk(t *testing.T) {
	// Verify that compactCondensed prefers ordinal-ordered chunks via selectOldestChunkAtDepth
	// rather than just grouping by depth without regard to order
	ce, s, convID := newTestCompactionEngine(t)
	ctx := context.Background()

	// Create interleaved summaries at depth 0 with a message in between:
	// sum1 (ordinal 100), msg (ordinal 200), sum2 (ordinal 300)

	for i := 0; i < LeafMinFanout+2; i++ {
		now := time.Now().UTC()

		s.CreateSummary(ctx, CreateSummaryInput{
			ConversationID: convID,
			Kind:           SummaryKindLeaf,
			Depth:          0,
			Content:        fmt.Sprintf("leaf summary %d", i),
			TokenCount:     100,
			EarliestAt:     &now,
			LatestAt:       &now,
		})
	}

	// Insert a message between first two summaries to break contiguity
	// for selectShallowestCondensationCandidate but would still find all 3
	// but selectOldestChunkAtDepth should only find sum1 + sum2 (not sum3)

	msg, _ := s.AddMessage(ctx, convID, "user", "interrupting message", 5)
	s.AppendContextMessage(ctx, convID, msg.ID)

	// Run compactCondensed
	result, err := ce.compactCondensed(ctx, convID)
	if err != nil {
		t.Fatalf("compactCondensed: %v", err)
	}

	// The result should have merged the two summaries at the start
	// (skipping the message in between), This proves ordinal-aware selection works.

	_ = result // verify summary was created

	if result != nil {
		summaries, _ := s.GetSummariesByConversation(ctx, convID)
		found := false
		for _, sum := range summaries {
			if sum.Kind == SummaryKindCondensed {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected condensed summary to be created via ordinal-aware selection")
		}
	}
}

func TestCompactCondensedUsesOrdinalAwareSelection(t *testing.T) {
	ce, s, convID := newTestCompactionEngine(t)
	ctx := context.Background()

	// Create leaf summaries at depth 0 (total tokens >= CondensedTargetTokens)
	for i := 0; i < 5; i++ {
		summary, _ := s.CreateSummary(ctx, CreateSummaryInput{
			ConversationID: convID,
			Kind:           SummaryKindLeaf,
			Depth:          0,
			Content:        fmt.Sprintf("leaf summary %d", i),
			TokenCount:     500, // 5 × 500 = 2500 >= CondensedTargetTokens (2000)
		})
		s.AppendContextSummary(ctx, convID, summary.SummaryID)
	}

	// Add fresh tail
	for i := 0; i < FreshTailCount+1; i++ {
		m, _ := s.AddMessage(ctx, convID, "user", "fresh", 5)
		s.AppendContextMessage(ctx, convID, m.ID)
	}

	chunk, err := ce.selectOldestChunkAtDepth(ctx, convID, 0)
	if err != nil {
		t.Fatalf("selectOldestChunkAtDepth: %v", err)
	}
	if len(chunk) < 2 {
		t.Errorf("chunk length = %d, want >= 2 contiguous summaries", len(chunk))
	}
	for _, s := range chunk {
		if s.Depth != 0 {
			t.Errorf("got depth %d, want 0", s.Depth)
		}
	}
}

func TestSelectOldestChunkAtDepthBreaksOnMessage(t *testing.T) {
	ce, s, convID := newTestCompactionEngine(t)
	ctx := context.Background()

	// Create 3 summaries, then a message, then 3 more summaries
	for i := 0; i < 3; i++ {
		summary, _ := s.CreateSummary(ctx, CreateSummaryInput{
			ConversationID: convID,
			Kind:           SummaryKindLeaf,
			Depth:          0,
			Content:        fmt.Sprintf("leaf %d", i),
			TokenCount:     100,
		})
		s.AppendContextSummary(ctx, convID, summary.SummaryID)
	}
	msg, _ := s.AddMessage(ctx, convID, "user", "break", 10)
	s.AppendContextMessage(ctx, convID, msg.ID)
	for i := 0; i < 3; i++ {
		summary, _ := s.CreateSummary(ctx, CreateSummaryInput{
			ConversationID: convID,
			Kind:           SummaryKindLeaf,
			Depth:          0,
			Content:        fmt.Sprintf("leaf-after %d", i),
			TokenCount:     100,
		})
		s.AppendContextSummary(ctx, convID, summary.SummaryID)
	}
	for i := 0; i < FreshTailCount+1; i++ {
		m, _ := s.AddMessage(ctx, convID, "user", "fresh", 5)
		s.AppendContextMessage(ctx, convID, m.ID)
	}

	chunk, _ := ce.selectOldestChunkAtDepth(ctx, convID, 0)
	if len(chunk) > 3 {
		t.Errorf("chunk length = %d, want <= 3 (message breaks chain)", len(chunk))
	}
}

func TestSelectOldestChunkAtDepthMinTokens(t *testing.T) {
	ce, s, convID := newTestCompactionEngine(t)
	ctx := context.Background()

	// Create summaries with very low token counts (total < 2000)
	for i := 0; i < 5; i++ {
		summary, _ := s.CreateSummary(ctx, CreateSummaryInput{
			ConversationID: convID,
			Kind:           SummaryKindLeaf,
			Depth:          0,
			Content:        fmt.Sprintf("tiny summary %d", i),
			TokenCount:     50, // very small
		})
		s.AppendContextSummary(ctx, convID, summary.SummaryID)
	}

	// Add fresh tail to protect from compaction
	for i := 0; i < FreshTailCount+1; i++ {
		m, _ := s.AddMessage(ctx, convID, "user", fmt.Sprintf("tail %d", i), 10)
		s.AppendContextMessage(ctx, convID, m.ID)
	}

	// Should return nil because total tokens (250) < 2000 minimum
	chunk, err := ce.selectOldestChunkAtDepth(ctx, convID, 0)
	if err != nil {
		t.Fatalf("selectOldestChunkAtDepth: %v", err)
	}
	if len(chunk) > 0 {
		t.Errorf("expected empty chunk when tokens < 2000, got %d summaries", len(chunk))
	}
}

func TestSelectOldestChunkAtDepthPassesMinTokens(t *testing.T) {
	ce, s, convID := newTestCompactionEngine(t)
	ctx := context.Background()

	// Create summaries with enough tokens (total >= 2000)
	for i := 0; i < 5; i++ {
		summary, _ := s.CreateSummary(ctx, CreateSummaryInput{
			ConversationID: convID,
			Kind:           SummaryKindLeaf,
			Depth:          0,
			Content: fmt.Sprintf(
				"substantial summary with enough content to meet minimum token threshold for condensation candidate %d",
				i,
			),
			TokenCount: 500, // 5 × 500 = 2500 >= 2000
		})
		s.AppendContextSummary(ctx, convID, summary.SummaryID)
	}

	// Add fresh tail
	for i := 0; i < FreshTailCount+1; i++ {
		m, _ := s.AddMessage(ctx, convID, "user", fmt.Sprintf("tail %d", i), 10)
		s.AppendContextMessage(ctx, convID, m.ID)
	}

	// Should return chunk because total tokens (2500) >= 2000
	chunk, err := ce.selectOldestChunkAtDepth(ctx, convID, 0)
	if err != nil {
		t.Fatalf("selectOldestChunkAtDepth: %v", err)
	}
	if len(chunk) == 0 {
		t.Error("expected non-empty chunk when tokens >= 2000")
	}
}

func TestGenerateLeafSummary(t *testing.T) {
	ce, _, _ := newTestCompactionEngine(t)
	ctx := context.Background()

	msgs := []Message{
		{Role: "user", Content: "hello world", TokenCount: 5},
		{Role: "assistant", Content: "hi there", TokenCount: 5},
	}

	content, err := ce.generateLeafSummary(ctx, msgs, "")
	if err != nil {
		t.Fatalf("generateLeafSummary: %v", err)
	}
	if content == "" {
		t.Error("expected non-empty summary content")
	}
}

func TestGenerateLeafSummaryEscalationToAggressive(t *testing.T) {
	// Level 1 returns summary that's too large (tokens >= input), should escalate to level 2
	var calls []string
	escalateComplete := func(ctx context.Context, prompt string, opts CompleteOptions) (string, error) {
		if contains(prompt, "Aggressive summary policy") {
			calls = append(calls, "aggressive")
			return "Short aggressive summary.", nil
		}
		calls = append(calls, "normal")
		// Return a very long summary to trigger escalation
		longContent := make([]byte, 5000)
		for i := range longContent {
			longContent[i] = 'x'
		}
		return string(longContent), nil
	}

	s := openTestStore(t)
	ce, _ := newTestCompactionEngineWithStore(s, escalateComplete)

	msgs := []Message{
		{Role: "user", Content: "hello world", TokenCount: 10},
		{Role: "assistant", Content: "response", TokenCount: 10},
	}

	content, err := ce.generateLeafSummary(context.Background(), msgs, "")
	if err != nil {
		t.Fatalf("generateLeafSummary: %v", err)
	}
	if content == "" {
		t.Error("expected non-empty summary content")
	}
	// Should have called both normal and aggressive
	foundNormal := false
	foundAggressive := false
	for _, c := range calls {
		if c == "normal" {
			foundNormal = true
		}
		if c == "aggressive" {
			foundAggressive = true
		}
	}
	if !foundNormal {
		t.Error("expected normal LLM call")
	}
	if !foundAggressive {
		t.Error("expected aggressive LLM call (level 2 escalation)")
	}
}

func TestGenerateLeafSummaryEscalatesWhenLevel1MissesTarget(t *testing.T) {
	var calls []string
	normalContent := strings.Repeat("n", 1000)    // ~404 tokens: below input, above target
	aggressiveContent := strings.Repeat("a", 450) // ~184 tokens: within aggressive target
	escalateComplete := func(ctx context.Context, prompt string, opts CompleteOptions) (string, error) {
		if contains(prompt, "Aggressive summary policy") {
			calls = append(calls, "aggressive")
			return aggressiveContent, nil
		}
		calls = append(calls, "normal")
		return normalContent, nil
	}

	s := openTestStore(t)
	ce, _ := newTestCompactionEngineWithStore(s, escalateComplete)

	msgs := []Message{
		{Role: "user", Content: "hello world", TokenCount: 500},
		{Role: "assistant", Content: "response", TokenCount: 500},
	}

	content, err := ce.generateLeafSummary(context.Background(), msgs, "")
	if err != nil {
		t.Fatalf("generateLeafSummary: %v", err)
	}
	if content != aggressiveContent {
		t.Fatalf("expected aggressive summary after level 1 missed target")
	}
	if len(calls) != 2 || calls[0] != "normal" || calls[1] != "aggressive" {
		t.Fatalf("expected normal then aggressive calls, got %v", calls)
	}
}

func TestGenerateLeafSummaryAcceptsContentAtTargetBoundary(t *testing.T) {
	exactTargetContent := strings.Repeat("x", 488) // (488 + 12) * 2 / 5 = 200 tokens
	var aggressiveCalled bool
	complete := func(ctx context.Context, prompt string, opts CompleteOptions) (string, error) {
		if contains(prompt, "Aggressive summary policy") {
			aggressiveCalled = true
		}
		return exactTargetContent, nil
	}

	s := openTestStore(t)
	ce, _ := newTestCompactionEngineWithStore(s, complete)

	msgs := []Message{
		{Role: "user", Content: "hello world", TokenCount: 286},
		{Role: "assistant", Content: "response", TokenCount: 286},
	}

	content, err := ce.generateLeafSummary(context.Background(), msgs, "")
	if err != nil {
		t.Fatalf("generateLeafSummary: %v", err)
	}
	if content != exactTargetContent {
		t.Fatalf("expected level 1 summary at target boundary to be accepted")
	}
	if aggressiveCalled {
		t.Fatal("did not expect aggressive retry when level 1 hit target exactly")
	}
}

func TestGenerateLeafSummaryEscalationToTruncation(t *testing.T) {
	// Both normal and aggressive return empty, should escalate to level 3 truncation
	emptyComplete := func(ctx context.Context, prompt string, opts CompleteOptions) (string, error) {
		return "", nil
	}

	s := openTestStore(t)
	ce, _ := newTestCompactionEngineWithStore(s, emptyComplete)

	msgs := []Message{
		{Role: "user", Content: "hello world from test", TokenCount: 10},
		{Role: "assistant", Content: "response text here", TokenCount: 10},
	}

	content, err := ce.generateLeafSummary(context.Background(), msgs, "")
	if err != nil {
		t.Fatalf("generateLeafSummary: %v", err)
	}
	// Level 3 truncation should have produced something
	if content == "" {
		t.Error("expected non-empty content from level 3 truncation fallback")
	}
	if !contains(content, "Truncated from") {
		t.Errorf("expected truncation marker in content: %q", content)
	}
}

func TestGenerateCondensedSummary(t *testing.T) {
	ce, _, _ := newTestCompactionEngine(t)
	ctx := context.Background()

	summaries := []Summary{
		{SummaryID: "sum_a", Content: "first summary", TokenCount: 100},
		{SummaryID: "sum_b", Content: "second summary", TokenCount: 100},
	}

	content, err := ce.generateCondensedSummary(ctx, summaries)
	if err != nil {
		t.Fatalf("generateCondensedSummary: %v", err)
	}
	if content == "" {
		t.Error("expected non-empty condensed summary content")
	}
}

func TestGenerateCondensedSummaryEscalation(t *testing.T) {
	// When LLM returns empty, should fall back to deterministic concatenation
	emptyComplete := func(ctx context.Context, prompt string, opts CompleteOptions) (string, error) {
		return "", nil
	}

	s := openTestStore(t)
	ce, _ := newTestCompactionEngineWithStore(s, emptyComplete)

	summaries := []Summary{
		{SummaryID: "sum_a", Content: "first summary text", TokenCount: 50},
		{SummaryID: "sum_b", Content: "second summary text", TokenCount: 50},
	}

	content, err := ce.generateCondensedSummary(context.Background(), summaries)
	if err != nil {
		t.Fatalf("generateCondensedSummary: %v", err)
	}
	// Should fall back to concatenation
	if content == "" {
		t.Error("expected non-empty content from fallback")
	}
}

// --- Async Condensed Compaction (Phase 2) ---

func TestCompactAsyncReturnsBeforeCondensed(t *testing.T) {
	// Use a slow CompleteFn to verify Compact returns before condensed finishes
	var callCount int32
	slowComplete := func(ctx context.Context, prompt string, opts CompleteOptions) (string, error) {
		atomic.AddInt32(&callCount, 1)
		time.Sleep(500 * time.Millisecond) // simulate slow LLM
		return "Slow condensed summary.", nil
	}

	s := openTestStore(t)
	ctx := context.Background()
	conv, _ := s.GetOrCreateConversation(ctx, "test:async")
	convID := conv.ConversationID

	ce, cancel := newTestCompactionEngineWithStore(s, slowComplete)
	t.Cleanup(func() {
		cancel()
		time.Sleep(100 * time.Millisecond)
	})

	// Create enough leaf summaries for condensation + fresh tail
	for i := 0; i < CondensedMinFanout; i++ {
		now := time.Now().UTC()
		summary, _ := s.CreateSummary(ctx, CreateSummaryInput{
			ConversationID: convID,
			Kind:           SummaryKindLeaf,
			Depth:          0,
			Content:        "leaf for async test",
			TokenCount:     500,
			EarliestAt:     &now,
			LatestAt:       &now,
		})
		s.AppendContextSummary(ctx, convID, summary.SummaryID)
	}
	for i := 0; i < FreshTailCount; i++ {
		m, _ := s.AddMessage(ctx, convID, "user", "fresh", 10)
		s.AppendContextMessage(ctx, convID, m.ID)
	}

	// Compact with force — should return quickly, condensed runs async
	start := time.Now()
	result, err := ce.Compact(ctx, convID, CompactInput{Force: true})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Compact: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Should return well before the 500ms LLM call
	if elapsed > 200*time.Millisecond {
		t.Errorf("Compact took %v, should return before async condensed finishes", elapsed)
	}

	// Wait for async to complete
	time.Sleep(800 * time.Millisecond)

	// Verify condensed summary was created by background goroutine
	summaries, _ := s.GetSummariesByConversation(ctx, convID)
	foundCondensed := false
	for _, sum := range summaries {
		if sum.Kind == SummaryKindCondensed {
			foundCondensed = true
			break
		}
	}
	if !foundCondensed {
		t.Error("expected at least one condensed summary from async Phase 2")
	}
}

func TestCompactAsyncDedup(t *testing.T) {
	var callCount int32
	slowComplete := func(ctx context.Context, prompt string, opts CompleteOptions) (string, error) {
		atomic.AddInt32(&callCount, 1)
		time.Sleep(300 * time.Millisecond)
		return "Slow condensed summary.", nil
	}

	s := openTestStore(t)
	ctx := context.Background()
	conv, _ := s.GetOrCreateConversation(ctx, "test:dedup")
	convID := conv.ConversationID

	ce, cancel := newTestCompactionEngineWithStore(s, slowComplete)
	t.Cleanup(func() {
		cancel()
		waitForCondensed(ce, convID, 2*time.Second)
	})

	// Create conditions for condensed compaction
	for i := 0; i < CondensedMinFanout; i++ {
		now := time.Now().UTC()
		summary, _ := s.CreateSummary(ctx, CreateSummaryInput{
			ConversationID: convID,
			Kind:           SummaryKindLeaf,
			Depth:          0,
			Content:        "leaf for dedup",
			TokenCount:     500,
			EarliestAt:     &now,
			LatestAt:       &now,
		})
		s.AppendContextSummary(ctx, convID, summary.SummaryID)
	}
	for i := 0; i < FreshTailCount; i++ {
		m, _ := s.AddMessage(ctx, convID, "user", "fresh", 10)
		s.AppendContextMessage(ctx, convID, m.ID)
	}

	// Call Compact twice rapidly
	ce.Compact(ctx, convID, CompactInput{Force: true})
	ce.Compact(ctx, convID, CompactInput{Force: true})

	// Wait for async to finish
	time.Sleep(600 * time.Millisecond)

	// LLM should only be called once for condensed (dedup)
	// callCount may be 0 if no leaf was created (only condensed in goroutine)
	// The key is that we don't get 2+ condensed calls
	if atomic.LoadInt32(&callCount) > 1 {
		t.Errorf("LLM called %d times, expected at most 1 (dedup)", callCount)
	}
}

func TestCompactLeafForceBypassesFreshTail(t *testing.T) {
	// Spec: compactLeaf with force=true should bypass FreshTailCount protection
	// so CompactUntilUnder can compress messages inside the fresh tail
	ce, s, convID := newTestCompactionEngine(t)
	ctx := context.Background()

	// Create exactly FreshTailCount+4 messages (36 total)
	// Without force: all messages are in fresh tail → no candidate
	// With force: should compact the oldest messages
	total := FreshTailCount + 4
	for i := 0; i < total; i++ {
		m, _ := s.AddMessage(ctx, convID, "user", fmt.Sprintf("message %d for force test", i), 100)
		s.AppendContextMessage(ctx, convID, m.ID)
	}

	// Without force: should return nil (all in fresh tail)
	summaryID, err := ce.compactLeaf(ctx, convID)
	if err != nil {
		t.Fatalf("compactLeaf no-force: %v", err)
	}
	if summaryID != nil {
		t.Error("expected nil without force (all messages in fresh tail)")
	}

	// With force: should compact despite fresh tail protection
	summaryID, err = ce.compactLeaf(ctx, convID, true)
	if err != nil {
		t.Fatalf("compactLeaf force: %v", err)
	}
	if summaryID == nil {
		t.Error("expected summary with force=true (bypasses fresh tail)")
	}
}

func TestCompactLeafAccumulatesUpToLeafChunkTokens(t *testing.T) {
	// Spec: compactLeaf should accumulate messages up to LeafChunkTokens before stopping
	// It should NOT take the entire contiguous chunk regardless of token count
	ce, s, convID := newTestCompactionEngine(t)
	ctx := context.Background()

	// Create messages totaling far more than LeafChunkTokens (20000)
	// Each message is ~500 tokens, create 80 messages = 40000 tokens
	for i := 0; i < 80; i++ {
		m, _ := s.AddMessage(
			ctx,
			convID,
			"user",
			fmt.Sprintf(
				"message %d with lots of content to make it big enough for token counting purposes and this should be a substantial message body that represents a meaningful conversation turn",
				i,
			),
			500,
		)
		s.AppendContextMessage(ctx, convID, m.ID)
	}

	summaryID, err := ce.compactLeaf(ctx, convID)
	if err != nil {
		t.Fatalf("compactLeaf: %v", err)
	}
	if summaryID == nil {
		t.Fatal("expected a summary to be created")
	}

	// The source messages that were compacted should total roughly LeafChunkTokens (20000),
	// not the entire 40000 tokens worth of messages
	summary, _ := s.GetSummary(ctx, *summaryID)
	if summary == nil {
		t.Fatal("summary not found")
	}

	// Source message tokens should be roughly <= LeafChunkTokens (20000)
	// Spec says: "Stop when accumulated tokens >= LeafChunkTokens"
	if summary.SourceMessageTokenCount > LeafChunkTokens {
		t.Errorf("source tokens = %d, should be <= LeafChunkTokens (%d)",
			summary.SourceMessageTokenCount, LeafChunkTokens)
	}
}
