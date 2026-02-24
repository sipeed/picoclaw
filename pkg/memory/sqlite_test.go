package memory

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func openTestDB(t *testing.T) *SQLiteStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("Open(%q): %v", dbPath, err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestOpen_CreatesDatabase(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sub", "sessions.db")
	store, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	// Verify schema tables exist.
	for _, table := range []string{"sessions", "messages"} {
		var name string
		err := store.db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestOpen_ExistingDatabase(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "persist.db")
	ctx := context.Background()

	// Write data, then close.
	store, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	err = store.AddMessage(ctx, "s1", "user", "hello")
	if err != nil {
		t.Fatalf("AddMessage: %v", err)
	}
	store.Close()

	// Re-open and verify persistence.
	store2, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open (reopen): %v", err)
	}
	defer store2.Close()

	history, err := store2.GetHistory(ctx, "s1")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 1 || history[0].Content != "hello" {
		t.Errorf("expected 1 message 'hello', got %v", history)
	}
}

func TestAddMessage_BasicRoundtrip(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	if err := store.AddMessage(ctx, "s1", "user", "hi"); err != nil {
		t.Fatalf("AddMessage: %v", err)
	}
	if err := store.AddMessage(ctx, "s1", "assistant", "hello"); err != nil {
		t.Fatalf("AddMessage: %v", err)
	}

	history, err := store.GetHistory(ctx, "s1")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history))
	}
	if history[0].Role != "user" || history[0].Content != "hi" {
		t.Errorf("msg[0] = %+v", history[0])
	}
	if history[1].Role != "assistant" || history[1].Content != "hello" {
		t.Errorf("msg[1] = %+v", history[1])
	}
}

func TestAddMessage_AutoCreatesSession(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	// Adding a message to a non-existent session should auto-create it.
	if err := store.AddMessage(ctx, "new-session", "user", "first"); err != nil {
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
	store := openTestDB(t)
	ctx := context.Background()

	msg := providers.Message{
		Role:    "assistant",
		Content: "Let me search.",
		ToolCalls: []providers.ToolCall{
			{
				ID:   "call_123",
				Type: "function",
				Function: &providers.FunctionCall{
					Name:      "web_search",
					Arguments: `{"query":"test"}`,
				},
			},
		},
	}
	if err := store.AddFullMessage(ctx, "s1", msg); err != nil {
		t.Fatalf("AddFullMessage: %v", err)
	}

	history, err := store.GetHistory(ctx, "s1")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 message, got %d", len(history))
	}

	got := history[0]
	if got.Content != "Let me search." {
		t.Errorf("content = %q", got.Content)
	}
	if len(got.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(got.ToolCalls))
	}
	tc := got.ToolCalls[0]
	if tc.ID != "call_123" || tc.Type != "function" {
		t.Errorf("tool call = %+v", tc)
	}
	if tc.Function == nil || tc.Function.Name != "web_search" {
		t.Errorf("function = %+v", tc.Function)
	}
	if tc.Function.Arguments != `{"query":"test"}` {
		t.Errorf("arguments = %q", tc.Function.Arguments)
	}
}

func TestAddFullMessage_ToolCallID(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	msg := providers.Message{
		Role:       "tool",
		Content:    "search result here",
		ToolCallID: "call_123",
	}
	if err := store.AddFullMessage(ctx, "s1", msg); err != nil {
		t.Fatalf("AddFullMessage: %v", err)
	}

	history, err := store.GetHistory(ctx, "s1")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 message, got %d", len(history))
	}
	if history[0].ToolCallID != "call_123" {
		t.Errorf("ToolCallID = %q, want %q", history[0].ToolCallID, "call_123")
	}
}

func TestGetHistory_EmptySession(t *testing.T) {
	store := openTestDB(t)
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
	store := openTestDB(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		content := string(rune('a' + i))
		if err := store.AddMessage(ctx, "s1", "user", content); err != nil {
			t.Fatalf("AddMessage(%d): %v", i, err)
		}
	}

	history, err := store.GetHistory(ctx, "s1")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 10 {
		t.Fatalf("expected 10 messages, got %d", len(history))
	}
	for i, msg := range history {
		expected := string(rune('a' + i))
		if msg.Content != expected {
			t.Errorf("msg[%d].Content = %q, want %q", i, msg.Content, expected)
		}
	}
}

func TestSetSummary_GetSummary(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	// No session yet — should return empty string.
	summary, err := store.GetSummary(ctx, "s1")
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if summary != "" {
		t.Errorf("expected empty summary, got %q", summary)
	}

	// Set summary (auto-creates session).
	err = store.SetSummary(ctx, "s1", "User asked about Go.")
	if err != nil {
		t.Fatalf("SetSummary: %v", err)
	}

	summary, err = store.GetSummary(ctx, "s1")
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if summary != "User asked about Go." {
		t.Errorf("summary = %q", summary)
	}

	// Overwrite summary.
	err = store.SetSummary(ctx, "s1", "Updated.")
	if err != nil {
		t.Fatalf("SetSummary: %v", err)
	}
	summary, err = store.GetSummary(ctx, "s1")
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if summary != "Updated." {
		t.Errorf("summary = %q", summary)
	}
}

func TestTruncateHistory_KeepLast(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		if err := store.AddMessage(ctx, "s1", "user", string(rune('a'+i))); err != nil {
			t.Fatalf("AddMessage: %v", err)
		}
	}

	if err := store.TruncateHistory(ctx, "s1", 4); err != nil {
		t.Fatalf("TruncateHistory: %v", err)
	}

	history, err := store.GetHistory(ctx, "s1")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(history))
	}
	// Should keep the last 4: g, h, i, j
	expected := []string{"g", "h", "i", "j"}
	for i, msg := range history {
		if msg.Content != expected[i] {
			t.Errorf("msg[%d].Content = %q, want %q", i, msg.Content, expected[i])
		}
	}
}

func TestTruncateHistory_KeepZero(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		if err := store.AddMessage(ctx, "s1", "user", "msg"); err != nil {
			t.Fatalf("AddMessage: %v", err)
		}
	}

	if err := store.TruncateHistory(ctx, "s1", 0); err != nil {
		t.Fatalf("TruncateHistory: %v", err)
	}

	history, err := store.GetHistory(ctx, "s1")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("expected 0 messages, got %d", len(history))
	}
}

func TestSetHistory_ReplacesAll(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	// Add some initial messages.
	for i := 0; i < 5; i++ {
		if err := store.AddMessage(ctx, "s1", "user", "old"); err != nil {
			t.Fatalf("AddMessage: %v", err)
		}
	}

	// Replace with new history.
	newHistory := []providers.Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "new question"},
		{Role: "assistant", Content: "new answer"},
	}
	if err := store.SetHistory(ctx, "s1", newHistory); err != nil {
		t.Fatalf("SetHistory: %v", err)
	}

	history, err := store.GetHistory(ctx, "s1")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(history))
	}
	for i, msg := range history {
		if msg.Role != newHistory[i].Role || msg.Content != newHistory[i].Content {
			t.Errorf("msg[%d] = %+v, want %+v", i, msg, newHistory[i])
		}
	}
}

func TestConcurrent_AddAndRead(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	const goroutines = 10
	const msgsPerGoroutine = 20

	var wg sync.WaitGroup
	wg.Add(goroutines * 2) // writers + readers

	// Writers
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < msgsPerGoroutine; i++ {
				err := store.AddMessage(ctx, "concurrent", "user", "msg")
				if err != nil {
					t.Errorf("goroutine %d: AddMessage: %v", id, err)
					return
				}
			}
		}(g)
	}

	// Readers
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < msgsPerGoroutine; i++ {
				_, err := store.GetHistory(ctx, "concurrent")
				if err != nil {
					t.Errorf("goroutine %d: GetHistory: %v", id, err)
					return
				}
			}
		}(g)
	}

	wg.Wait()

	// Final count should be exactly goroutines * msgsPerGoroutine.
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
	// Simulates the #704 race: one goroutine does TruncateHistory while another does AddFullMessage.
	store := openTestDB(t)
	ctx := context.Background()

	// Seed with some messages.
	for i := 0; i < 20; i++ {
		if err := store.AddMessage(ctx, "race", "user", "msg"); err != nil {
			t.Fatalf("AddMessage: %v", err)
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Summarize goroutine: set summary + truncate
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			_ = store.SetSummary(ctx, "race", "summary text")
			_ = store.TruncateHistory(ctx, "race", 4)
		}
	}()

	// Main loop goroutine: add messages
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			_ = store.AddMessage(ctx, "race", "user", "new msg")
			_ = store.AddMessage(ctx, "race", "assistant", "response")
		}
	}()

	wg.Wait()

	// No panic, no corruption — just verify we can still read.
	_, err := store.GetHistory(ctx, "race")
	if err != nil {
		t.Fatalf("GetHistory after race: %v", err)
	}
	_, err = store.GetSummary(ctx, "race")
	if err != nil {
		t.Fatalf("GetSummary after race: %v", err)
	}
}

func BenchmarkAddMessage(b *testing.B) {
	dbPath := filepath.Join(b.TempDir(), "bench.db")
	store, err := Open(context.Background(), dbPath)
	if err != nil {
		b.Fatalf("Open: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := store.AddMessage(ctx, "bench", "user", "benchmark message")
		if err != nil {
			b.Fatalf("AddMessage: %v", err)
		}
	}
}

func BenchmarkGetHistory_100(b *testing.B) {
	benchGetHistory(b, 100)
}

func BenchmarkGetHistory_1000(b *testing.B) {
	benchGetHistory(b, 1000)
}

func benchGetHistory(b *testing.B, count int) {
	dbPath := filepath.Join(b.TempDir(), "bench.db")
	store, err := Open(context.Background(), dbPath)
	if err != nil {
		b.Fatalf("Open: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	for i := 0; i < count; i++ {
		if err := store.AddMessage(ctx, "bench", "user", "message content"); err != nil {
			b.Fatalf("AddMessage: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := store.GetHistory(ctx, "bench"); err != nil {
			b.Fatalf("GetHistory: %v", err)
		}
	}
}
