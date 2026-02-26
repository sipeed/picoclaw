package memory

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func newTestStore(t *testing.T) *JSONLStore {
	t.Helper()
	store, err := NewJSONLStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewJSONLStore: %v", err)
	}
	return store
}

func TestNewJSONLStore_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "sessions")
	store, err := NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore: %v", err)
	}
	defer store.Close()

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("expected directory, got file")
	}
}

func TestAddMessage_BasicRoundtrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.AddMessage(ctx, "s1", "user", "hello")
	if err != nil {
		t.Fatalf("AddMessage: %v", err)
	}
	err = store.AddMessage(ctx, "s1", "assistant", "hi there")
	if err != nil {
		t.Fatalf("AddMessage: %v", err)
	}

	history, err := store.GetHistory(ctx, "s1")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history))
	}
	if history[0].Role != "user" || history[0].Content != "hello" {
		t.Errorf("msg[0] = %+v", history[0])
	}
	if history[1].Role != "assistant" || history[1].Content != "hi there" {
		t.Errorf("msg[1] = %+v", history[1])
	}
}

func TestAddMessage_AutoCreatesSession(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Adding a message to a non-existent session should work.
	err := store.AddMessage(ctx, "new-session", "user", "first message")
	if err != nil {
		t.Fatalf("AddMessage: %v", err)
	}

	history, err := store.GetHistory(ctx, "new-session")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 message, got %d", len(history))
	}
}

func TestAddFullMessage_WithToolCalls(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	msg := providers.Message{
		Role:    "assistant",
		Content: "Let me search that.",
		ToolCalls: []providers.ToolCall{
			{
				ID:   "call_abc",
				Type: "function",
				Function: &providers.FunctionCall{
					Name:      "web_search",
					Arguments: `{"q":"golang jsonl"}`,
				},
			},
		},
	}

	err := store.AddFullMessage(ctx, "tc", msg)
	if err != nil {
		t.Fatalf("AddFullMessage: %v", err)
	}

	history, err := store.GetHistory(ctx, "tc")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1, got %d", len(history))
	}
	if len(history[0].ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(history[0].ToolCalls))
	}
	tc := history[0].ToolCalls[0]
	if tc.ID != "call_abc" {
		t.Errorf("tool call ID = %q", tc.ID)
	}
	if tc.Function == nil || tc.Function.Name != "web_search" {
		t.Errorf("tool call function = %+v", tc.Function)
	}
}

func TestAddFullMessage_ToolCallID(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	msg := providers.Message{
		Role:       "tool",
		Content:    "search results here",
		ToolCallID: "call_abc",
	}

	err := store.AddFullMessage(ctx, "tr", msg)
	if err != nil {
		t.Fatalf("AddFullMessage: %v", err)
	}

	history, err := store.GetHistory(ctx, "tr")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1, got %d", len(history))
	}
	if history[0].ToolCallID != "call_abc" {
		t.Errorf("ToolCallID = %q", history[0].ToolCallID)
	}
}

func TestGetHistory_EmptySession(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	history, err := store.GetHistory(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if history == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(history) != 0 {
		t.Errorf("expected 0 messages, got %d", len(history))
	}
}

func TestGetHistory_Ordering(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		err := store.AddMessage(
			ctx, "order",
			"user",
			string(rune('a'+i)),
		)
		if err != nil {
			t.Fatalf("AddMessage(%d): %v", i, err)
		}
	}

	history, err := store.GetHistory(ctx, "order")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 5 {
		t.Fatalf("expected 5, got %d", len(history))
	}
	for i := 0; i < 5; i++ {
		expected := string(rune('a' + i))
		if history[i].Content != expected {
			t.Errorf("msg[%d].Content = %q, want %q", i, history[i].Content, expected)
		}
	}
}

func TestSetSummary_GetSummary(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// No summary yet.
	summary, err := store.GetSummary(ctx, "s1")
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if summary != "" {
		t.Errorf("expected empty, got %q", summary)
	}

	// Set a summary.
	err = store.SetSummary(ctx, "s1", "talked about Go")
	if err != nil {
		t.Fatalf("SetSummary: %v", err)
	}

	summary, err = store.GetSummary(ctx, "s1")
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if summary != "talked about Go" {
		t.Errorf("summary = %q", summary)
	}

	// Update summary.
	err = store.SetSummary(ctx, "s1", "updated summary")
	if err != nil {
		t.Fatalf("SetSummary: %v", err)
	}

	summary, err = store.GetSummary(ctx, "s1")
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if summary != "updated summary" {
		t.Errorf("summary = %q", summary)
	}
}

func TestTruncateHistory_KeepLast(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		err := store.AddMessage(
			ctx, "trunc",
			"user",
			string(rune('a'+i)),
		)
		if err != nil {
			t.Fatalf("AddMessage: %v", err)
		}
	}

	err := store.TruncateHistory(ctx, "trunc", 4)
	if err != nil {
		t.Fatalf("TruncateHistory: %v", err)
	}

	history, err := store.GetHistory(ctx, "trunc")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 4 {
		t.Fatalf("expected 4, got %d", len(history))
	}
	// Should be the last 4: g, h, i, j
	if history[0].Content != "g" {
		t.Errorf("first kept = %q, want 'g'", history[0].Content)
	}
	if history[3].Content != "j" {
		t.Errorf("last kept = %q, want 'j'", history[3].Content)
	}
}

func TestTruncateHistory_KeepZero(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		err := store.AddMessage(ctx, "empty", "user", "msg")
		if err != nil {
			t.Fatalf("AddMessage: %v", err)
		}
	}

	err := store.TruncateHistory(ctx, "empty", 0)
	if err != nil {
		t.Fatalf("TruncateHistory: %v", err)
	}

	history, err := store.GetHistory(ctx, "empty")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("expected 0, got %d", len(history))
	}
}

func TestTruncateHistory_KeepMoreThanExists(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		err := store.AddMessage(ctx, "few", "user", "msg")
		if err != nil {
			t.Fatalf("AddMessage: %v", err)
		}
	}

	// Keep 100, but only 3 exist — should keep all.
	err := store.TruncateHistory(ctx, "few", 100)
	if err != nil {
		t.Fatalf("TruncateHistory: %v", err)
	}

	history, err := store.GetHistory(ctx, "few")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 3 {
		t.Errorf("expected 3, got %d", len(history))
	}
}

func TestSetHistory_ReplacesAll(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Add some initial messages.
	for i := 0; i < 5; i++ {
		err := store.AddMessage(ctx, "replace", "user", "old")
		if err != nil {
			t.Fatalf("AddMessage: %v", err)
		}
	}

	// Replace with new history.
	newHistory := []providers.Message{
		{Role: "user", Content: "new1"},
		{Role: "assistant", Content: "new2"},
	}
	err := store.SetHistory(ctx, "replace", newHistory)
	if err != nil {
		t.Fatalf("SetHistory: %v", err)
	}

	history, err := store.GetHistory(ctx, "replace")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2, got %d", len(history))
	}
	if history[0].Content != "new1" || history[1].Content != "new2" {
		t.Errorf("history = %+v", history)
	}
}

func TestSetHistory_ResetsSkip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Add messages and truncate.
	for i := 0; i < 10; i++ {
		err := store.AddMessage(ctx, "skip-reset", "user", "old")
		if err != nil {
			t.Fatalf("AddMessage: %v", err)
		}
	}
	err := store.TruncateHistory(ctx, "skip-reset", 3)
	if err != nil {
		t.Fatalf("TruncateHistory: %v", err)
	}

	// SetHistory should reset skip to 0.
	newHistory := []providers.Message{
		{Role: "user", Content: "fresh"},
	}
	err = store.SetHistory(ctx, "skip-reset", newHistory)
	if err != nil {
		t.Fatalf("SetHistory: %v", err)
	}

	history, err := store.GetHistory(ctx, "skip-reset")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1, got %d", len(history))
	}
	if history[0].Content != "fresh" {
		t.Errorf("content = %q", history[0].Content)
	}
}

func TestColonInKey(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.AddMessage(ctx, "telegram:123", "user", "hi")
	if err != nil {
		t.Fatalf("AddMessage: %v", err)
	}

	history, err := store.GetHistory(ctx, "telegram:123")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1, got %d", len(history))
	}

	// Verify the file is named with underscore.
	jsonlFile := filepath.Join(store.dir, "telegram_123.jsonl")
	if _, statErr := os.Stat(jsonlFile); statErr != nil {
		t.Errorf("expected file %s to exist: %v", jsonlFile, statErr)
	}
}

func TestCompact_RemovesSkippedMessages(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Write 10 messages, then truncate to keep last 3.
	for i := 0; i < 10; i++ {
		err := store.AddMessage(ctx, "compact", "user", string(rune('a'+i)))
		if err != nil {
			t.Fatalf("AddMessage: %v", err)
		}
	}
	err := store.TruncateHistory(ctx, "compact", 3)
	if err != nil {
		t.Fatalf("TruncateHistory: %v", err)
	}

	// Before compact: file still has 10 lines.
	allOnDisk, err := readMessages(store.jsonlPath("compact"), 0)
	if err != nil {
		t.Fatalf("readMessages: %v", err)
	}
	if len(allOnDisk) != 10 {
		t.Fatalf("before compact: expected 10 on disk, got %d", len(allOnDisk))
	}

	// Compact.
	err = store.Compact(ctx, "compact")
	if err != nil {
		t.Fatalf("Compact: %v", err)
	}

	// After compact: file should have only 3 lines.
	allOnDisk, err = readMessages(store.jsonlPath("compact"), 0)
	if err != nil {
		t.Fatalf("readMessages: %v", err)
	}
	if len(allOnDisk) != 3 {
		t.Fatalf("after compact: expected 3 on disk, got %d", len(allOnDisk))
	}

	// GetHistory should still return the same 3 messages.
	history, err := store.GetHistory(ctx, "compact")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("expected 3, got %d", len(history))
	}
	if history[0].Content != "h" || history[2].Content != "j" {
		t.Errorf("wrong content: %+v", history)
	}
}

func TestCompact_NoOpWhenNoSkip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		err := store.AddMessage(ctx, "noop", "user", "msg")
		if err != nil {
			t.Fatalf("AddMessage: %v", err)
		}
	}

	// Compact without prior truncation — should be a no-op.
	err := store.Compact(ctx, "noop")
	if err != nil {
		t.Fatalf("Compact: %v", err)
	}

	history, err := store.GetHistory(ctx, "noop")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 5 {
		t.Errorf("expected 5, got %d", len(history))
	}
}

func TestCompact_ThenAppend(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	for i := 0; i < 8; i++ {
		err := store.AddMessage(ctx, "cap", "user", string(rune('a'+i)))
		if err != nil {
			t.Fatalf("AddMessage: %v", err)
		}
	}

	err := store.TruncateHistory(ctx, "cap", 2)
	if err != nil {
		t.Fatalf("TruncateHistory: %v", err)
	}
	err = store.Compact(ctx, "cap")
	if err != nil {
		t.Fatalf("Compact: %v", err)
	}

	// Append after compaction should work correctly.
	err = store.AddMessage(ctx, "cap", "user", "new")
	if err != nil {
		t.Fatalf("AddMessage after compact: %v", err)
	}

	history, err := store.GetHistory(ctx, "cap")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("expected 3, got %d", len(history))
	}
	// g, h (kept from truncation), new (appended after compaction).
	if history[0].Content != "g" {
		t.Errorf("first = %q, want 'g'", history[0].Content)
	}
	if history[2].Content != "new" {
		t.Errorf("last = %q, want 'new'", history[2].Content)
	}
}

func TestCrashRecovery_PartialLine(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Write a valid message first.
	err := store.AddMessage(ctx, "crash", "user", "valid")
	if err != nil {
		t.Fatalf("AddMessage: %v", err)
	}

	// Simulate a crash by appending a partial JSON line directly.
	jsonlPath := store.jsonlPath("crash")
	f, err := os.OpenFile(jsonlPath, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatalf("open for append: %v", err)
	}
	_, err = f.WriteString(`{"role":"user","content":"incomple`)
	if err != nil {
		t.Fatalf("write partial: %v", err)
	}
	f.Close()

	// GetHistory should return only the valid message.
	history, err := store.GetHistory(ctx, "crash")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 valid message, got %d", len(history))
	}
	if history[0].Content != "valid" {
		t.Errorf("content = %q", history[0].Content)
	}
}

func TestPersistence_AcrossInstances(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	// Write with first instance.
	store1, err := NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore: %v", err)
	}
	err = store1.AddMessage(ctx, "persist", "user", "remember me")
	if err != nil {
		t.Fatalf("AddMessage: %v", err)
	}
	err = store1.SetSummary(ctx, "persist", "a test session")
	if err != nil {
		t.Fatalf("SetSummary: %v", err)
	}
	store1.Close()

	// Read with second instance.
	store2, err := NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore: %v", err)
	}
	defer store2.Close()

	history, err := store2.GetHistory(ctx, "persist")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 1 || history[0].Content != "remember me" {
		t.Errorf("history = %+v", history)
	}

	summary, err := store2.GetSummary(ctx, "persist")
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if summary != "a test session" {
		t.Errorf("summary = %q", summary)
	}
}

func TestConcurrent_AddAndRead(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	const goroutines = 10
	const msgsPerGoroutine = 20

	// Concurrent writes.
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < msgsPerGoroutine; i++ {
				_ = store.AddMessage(ctx, "concurrent", "user", "msg")
			}
		}()
	}
	wg.Wait()

	history, err := store.GetHistory(ctx, "concurrent")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	expected := goroutines * msgsPerGoroutine
	if len(history) != expected {
		t.Errorf("expected %d messages, got %d", expected, len(history))
	}
}

func TestConcurrent_SummarizeRace(t *testing.T) {
	// Simulates the #704 race: one goroutine adds messages while
	// another truncates + sets summary — like summarizeSession().
	store := newTestStore(t)
	ctx := context.Background()

	// Seed with some messages.
	for i := 0; i < 20; i++ {
		err := store.AddMessage(ctx, "race", "user", "seed")
		if err != nil {
			t.Fatalf("AddMessage: %v", err)
		}
	}

	var wg sync.WaitGroup

	// Writer goroutine (main agent loop).
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_ = store.AddMessage(ctx, "race", "user", "new")
		}
	}()

	// Summarizer goroutine (background task).
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			_ = store.SetSummary(ctx, "race", "summary")
			_ = store.TruncateHistory(ctx, "race", 5)
		}
	}()

	wg.Wait()

	// Verify the store is still in a consistent state.
	_, err := store.GetHistory(ctx, "race")
	if err != nil {
		t.Fatalf("GetHistory after race: %v", err)
	}
	_, err = store.GetSummary(ctx, "race")
	if err != nil {
		t.Fatalf("GetSummary after race: %v", err)
	}
}

func TestMultipleSessions_Isolation(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.AddMessage(ctx, "s1", "user", "msg for s1")
	if err != nil {
		t.Fatalf("AddMessage: %v", err)
	}
	err = store.AddMessage(ctx, "s2", "user", "msg for s2")
	if err != nil {
		t.Fatalf("AddMessage: %v", err)
	}

	h1, err := store.GetHistory(ctx, "s1")
	if err != nil {
		t.Fatalf("GetHistory s1: %v", err)
	}
	h2, err := store.GetHistory(ctx, "s2")
	if err != nil {
		t.Fatalf("GetHistory s2: %v", err)
	}

	if len(h1) != 1 || h1[0].Content != "msg for s1" {
		t.Errorf("s1 history = %+v", h1)
	}
	if len(h2) != 1 || h2[0].Content != "msg for s2" {
		t.Errorf("s2 history = %+v", h2)
	}
}

func BenchmarkAddMessage(b *testing.B) {
	dir := b.TempDir()
	store, err := NewJSONLStore(dir)
	if err != nil {
		b.Fatalf("NewJSONLStore: %v", err)
	}
	defer store.Close()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.AddMessage(ctx, "bench", "user", "benchmark message content")
	}
}

func BenchmarkGetHistory_100(b *testing.B) {
	dir := b.TempDir()
	store, err := NewJSONLStore(dir)
	if err != nil {
		b.Fatalf("NewJSONLStore: %v", err)
	}
	defer store.Close()
	ctx := context.Background()

	for i := 0; i < 100; i++ {
		_ = store.AddMessage(ctx, "bench", "user", "message content")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.GetHistory(ctx, "bench")
	}
}

func BenchmarkGetHistory_1000(b *testing.B) {
	dir := b.TempDir()
	store, err := NewJSONLStore(dir)
	if err != nil {
		b.Fatalf("NewJSONLStore: %v", err)
	}
	defer store.Close()
	ctx := context.Background()

	for i := 0; i < 1000; i++ {
		_ = store.AddMessage(ctx, "bench", "user", "message content")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.GetHistory(ctx, "bench")
	}
}
