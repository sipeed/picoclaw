package session

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func newTestManager(t *testing.T, fn SummarizeFunc, cfg config.SummarizationConfig) *SessionManager {
	t.Helper()
	return NewSessionManager(t.TempDir(), WithSummarizer(fn, cfg))
}

func noopSummarize(_ context.Context, _ []providers.Message, _ string) (string, error) {
	return "summary", nil
}

// --- EstimateTokens ---

func TestEstimateTokens_Latin(t *testing.T) {
	sm := newTestManager(t, noopSummarize, config.SummarizationConfig{})

	msgs := []providers.Message{
		{Role: "user", Content: "hello world"},    // 11 runes
		{Role: "assistant", Content: "hi there!"}, // 9 runes
	}
	// 20 runes / 2.5 = 8 tokens
	got := sm.EstimateTokens(msgs)
	if got != 8 {
		t.Errorf("EstimateTokens = %d, want 8", got)
	}
}

func TestEstimateTokens_CJK(t *testing.T) {
	sm := newTestManager(t, noopSummarize, config.SummarizationConfig{})

	msgs := []providers.Message{
		{Role: "user", Content: "你好世界"}, //nolint:gosmopolitan // CJK text "你好世界" (4 runes)
	}
	// 4 / 2.5 = 1.6 → 1
	got := sm.EstimateTokens(msgs)
	if got != 1 {
		t.Errorf("EstimateTokens = %d, want 1", got)
	}
}

func TestEstimateTokens_CustomCharsPerToken(t *testing.T) {
	cfg := config.SummarizationConfig{CharsPerToken: 5.0}
	sm := newTestManager(t, noopSummarize, cfg)

	msgs := []providers.Message{
		{Role: "user", Content: "12345678901234567890"}, // 20 runes
	}
	// 20 / 5.0 = 4
	got := sm.EstimateTokens(msgs)
	if got != 4 {
		t.Errorf("EstimateTokens = %d, want 4", got)
	}
}

// --- MaybeSummarize ---

func TestMaybeSummarize_BelowThreshold(t *testing.T) {
	called := false
	fn := func(_ context.Context, _ []providers.Message, _ string) (string, error) {
		called = true
		return "summary", nil
	}

	cfg := config.SummarizationConfig{}
	cfg.MessageThreshold = 100
	cfg.TokenPercent = 99
	sm := newTestManager(t, fn, cfg)

	// Add a few messages — well below thresholds.
	for i := 0; i < 5; i++ {
		sm.AddMessage("test", "user", "hello")
	}

	sm.MaybeSummarize("test")
	time.Sleep(50 * time.Millisecond) // give goroutine time to fire (it shouldn't)

	if called {
		t.Error("SummarizeFunc was called despite being below thresholds")
	}
}

func TestMaybeSummarize_AboveMessageThreshold(t *testing.T) {
	done := make(chan struct{})
	fn := func(_ context.Context, _ []providers.Message, _ string) (string, error) {
		defer func() {
			select {
			case done <- struct{}{}:
			default:
			}
		}()
		return "summary of conversation", nil
	}

	cfg := config.SummarizationConfig{}
	cfg.MessageThreshold = 5
	cfg.KeepLastMessages = 2
	sm := newTestManager(t, fn, cfg)

	for i := 0; i < 10; i++ {
		sm.AddMessage("test", "user", "msg")
		sm.AddMessage("test", "assistant", "reply")
	}

	sm.MaybeSummarize("test")

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for summarization")
	}

	// Wait a bit for ApplySummarization to complete.
	time.Sleep(100 * time.Millisecond)

	summary := sm.GetSummary("test")
	if summary == "" {
		t.Error("expected non-empty summary after summarization")
	}

	history := sm.GetHistory("test")
	if len(history) > cfg.KeepLastMessages+2 {
		// Some slack for messages added during the goroutine window.
		t.Errorf("expected at most %d messages after summarization, got %d",
			cfg.KeepLastMessages+2, len(history))
	}
}

func TestMaybeSummarize_DeduplicatesConcurrent(t *testing.T) {
	goroutineCount := 0
	var mu sync.Mutex
	blocker := make(chan struct{})

	fn := func(_ context.Context, _ []providers.Message, _ string) (string, error) {
		mu.Lock()
		goroutineCount++
		mu.Unlock()
		<-blocker
		return "summary", nil
	}

	cfg := config.SummarizationConfig{}
	cfg.MessageThreshold = 3
	cfg.KeepLastMessages = 2
	sm := newTestManager(t, fn, cfg)

	for i := 0; i < 10; i++ {
		sm.AddMessage("test", "user", "msg")
		sm.AddMessage("test", "assistant", "reply")
	}

	// Trigger twice — only one goroutine should enter summarizeSession.
	sm.MaybeSummarize("test")
	sm.MaybeSummarize("test")

	// Let the blocker hold briefly so the second call sees inflight=true.
	time.Sleep(50 * time.Millisecond)

	close(blocker)
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	got := goroutineCount
	mu.Unlock()

	// summarizeSession may call Summarize multiple times internally
	// (multi-part batching via parallel goroutines), but only ONE
	// goroutine should have entered summarizeSession.
	// With 18 valid messages and threshold=10: 3 calls (part1+part2+merge).
	// If dedup failed, we'd see 6.
	if got > 3 {
		t.Errorf("SummarizeFunc called %d times, want at most 3 (single goroutine with multi-part)", got)
	}
}

// --- ForceCompression ---

func TestForceCompression_DropsOldestHalf(t *testing.T) {
	sm := newTestManager(t, noopSummarize, config.SummarizationConfig{})

	// [system, msg1, msg2, msg3, msg4, msg5, trigger]
	sm.AddFullMessage("test", providers.Message{Role: "system", Content: "system prompt"})
	for i := 1; i <= 5; i++ {
		sm.AddMessage("test", "user", "old message")
	}
	sm.AddMessage("test", "user", "trigger")

	sm.ForceCompression("test")

	history := sm.GetHistory("test")
	if len(history) == 0 {
		t.Fatal("history is empty after compression")
	}

	// System prompt should be first and contain the compression note.
	if history[0].Role != "system" {
		t.Errorf("first message role = %q, want system", history[0].Role)
	}
	if !strings.Contains(history[0].Content, "Emergency compression") {
		t.Error("system prompt missing compression note")
	}

	// Trigger message should be last.
	last := history[len(history)-1]
	if last.Content != "trigger" {
		t.Errorf("last message = %q, want trigger", last.Content)
	}
}

func TestForceCompression_TooFewMessages(t *testing.T) {
	sm := newTestManager(t, noopSummarize, config.SummarizationConfig{})

	sm.AddMessage("test", "user", "one")
	sm.AddMessage("test", "assistant", "two")

	sm.ForceCompression("test")

	history := sm.GetHistory("test")
	if len(history) != 2 {
		t.Errorf("expected 2 messages unchanged, got %d", len(history))
	}
}

func TestForceCompression_WithoutSummarizer_UsesDefaultMinMessages(t *testing.T) {
	sm := NewSessionManager(t.TempDir())

	// 3 messages is below the default ForceCompressionMinMessages (4),
	// so compression should not run even when no summarizer is configured.
	sm.AddFullMessage("test", providers.Message{Role: "system", Content: "system prompt"})
	sm.AddMessage("test", "user", "one")
	sm.AddMessage("test", "assistant", "two")

	sm.ForceCompression("test")

	history := sm.GetHistory("test")
	if len(history) != 3 {
		t.Errorf("expected 3 messages unchanged, got %d", len(history))
	}
}

// --- ApplySummarization ---

func TestApplySummarization_PreservesNewMessages(t *testing.T) {
	sm := NewSessionManager(t.TempDir())

	// Seed 10 messages.
	for i := 0; i < 10; i++ {
		sm.AddMessage("test", "user", "original")
	}
	snapshotLen := 10

	// Simulate the main loop appending 3 new messages after the snapshot.
	sm.AddMessage("test", "user", "new-1")
	sm.AddMessage("test", "assistant", "new-2")
	sm.AddMessage("test", "user", "new-3")

	applied := sm.ApplySummarization("test", "the summary", snapshotLen, 4)
	if !applied {
		t.Fatal("ApplySummarization returned false")
	}

	history := sm.GetHistory("test")
	// Should have: 4 kept from snapshot + 3 new = 7
	if len(history) != 7 {
		t.Errorf("len(history) = %d, want 7", len(history))
	}

	// Verify the new messages are at the end.
	if history[4].Content != "new-1" {
		t.Errorf("history[4] = %q, want new-1", history[4].Content)
	}
	if history[6].Content != "new-3" {
		t.Errorf("history[6] = %q, want new-3", history[6].Content)
	}

	// Summary should be set.
	summary := sm.GetSummary("test")
	if summary != "the summary" {
		t.Errorf("summary = %q, want 'the summary'", summary)
	}
}

func TestApplySummarization_StaleSnapshot(t *testing.T) {
	sm := NewSessionManager(t.TempDir())

	for i := 0; i < 10; i++ {
		sm.AddMessage("test", "user", "original")
	}
	snapshotLen := 10

	// Simulate another compression reducing messages below snapshot.
	sm.SetHistory("test", []providers.Message{
		{Role: "user", Content: "compressed-1"},
		{Role: "assistant", Content: "compressed-2"},
	})

	applied := sm.ApplySummarization("test", "stale summary", snapshotLen, 4)
	if applied {
		t.Error("ApplySummarization should return false for stale snapshot")
	}

	// Summary should NOT have changed.
	summary := sm.GetSummary("test")
	if summary == "stale summary" {
		t.Error("stale summary was applied despite stale snapshot")
	}
}

func TestApplySummarization_NoNewMessages(t *testing.T) {
	sm := NewSessionManager(t.TempDir())

	for i := 0; i < 10; i++ {
		sm.AddMessage("test", "user", "original")
	}

	applied := sm.ApplySummarization("test", "the summary", 10, 4)
	if !applied {
		t.Fatal("ApplySummarization returned false")
	}

	history := sm.GetHistory("test")
	// Should have exactly 4 kept.
	if len(history) != 4 {
		t.Errorf("len(history) = %d, want 4", len(history))
	}
}

// --- SummarizeSession Multi-Part ---

func TestSummarizeSession_MultiPart(t *testing.T) {
	callCount := 0
	var mu sync.Mutex
	fn := func(_ context.Context, msgs []providers.Message, existing string) (string, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		return "partial summary", nil
	}

	cfg := config.SummarizationConfig{}
	cfg.MultiPartBatchThreshold = 5
	cfg.KeepLastMessages = 2
	sm := newTestManager(t, fn, cfg)

	// Add 12 user+assistant messages (well above threshold of 5).
	for i := 0; i < 12; i++ {
		sm.AddMessage("test", "user", "hello")
		sm.AddMessage("test", "assistant", "world")
	}

	sm.summarizeSession("test")

	mu.Lock()
	got := callCount
	mu.Unlock()

	// Should be 3 calls: part1 + part2 + merge.
	if got != 3 {
		t.Errorf("SummarizeFunc called %d times, want 3 (part1 + part2 + merge)", got)
	}

	summary := sm.GetSummary("test")
	if summary == "" {
		t.Error("expected non-empty summary after multi-part summarization")
	}
}

// --- Concurrent safety: summarize + main loop writes ---

func TestConcurrent_SummarizeAndWrite(t *testing.T) {
	// Simulates the core race scenario: background summarization runs
	// while the main loop adds new messages.
	snapshotTaken := make(chan struct{})
	blocker := make(chan struct{})
	firstCall := true
	var mu sync.Mutex

	fn := func(_ context.Context, _ []providers.Message, _ string) (string, error) {
		mu.Lock()
		if firstCall {
			firstCall = false
			mu.Unlock()
			// Signal that summarizeSession has taken its snapshot and entered summarize.
			close(snapshotTaken)
			<-blocker // block until new messages are added
		} else {
			mu.Unlock()
		}
		return "concurrent summary", nil
	}

	cfg := config.SummarizationConfig{}
	cfg.KeepLastMessages = 2
	cfg.MultiPartBatchThreshold = 100 // force single-batch path
	sm := newTestManager(t, fn, cfg)

	// Seed 20 messages.
	for i := 0; i < 20; i++ {
		sm.AddMessage("test", "user", "seed")
	}

	// Start summarization in background.
	done := make(chan struct{})
	go func() {
		sm.summarizeSession("test")
		close(done)
	}()

	// Wait for the summarizer to take its snapshot and enter the LLM call.
	<-snapshotTaken

	// While summarization is blocked, add 5 new messages.
	for i := 0; i < 5; i++ {
		sm.AddMessage("test", "user", "new-during-summarize")
	}

	// Unblock the summarizer.
	close(blocker)
	<-done

	// All 5 new messages must survive.
	history := sm.GetHistory("test")
	newCount := 0
	for _, m := range history {
		if m.Content == "new-during-summarize" {
			newCount++
		}
	}
	if newCount != 5 {
		t.Errorf("expected 5 new messages preserved, got %d (total history: %d)",
			newCount, len(history))
	}

	summary := sm.GetSummary("test")
	if summary == "" {
		t.Error("expected non-empty summary")
	}
}

// --- BuildSummarizationPrompt ---

func TestBuildSummarizationPrompt(t *testing.T) {
	msgs := []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}

	prompt := BuildSummarizationPrompt(msgs, "")
	if !strings.Contains(prompt, "concise summary") {
		t.Error("prompt missing summary instruction")
	}
	if !strings.Contains(prompt, "user: hello") {
		t.Error("prompt missing user message")
	}

	// With existing summary.
	prompt2 := BuildSummarizationPrompt(msgs, "previous context")
	if !strings.Contains(prompt2, "Existing context: previous context") {
		t.Error("prompt missing existing summary")
	}
}
