package session

import (
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// sessionBackend abstracts the common API shared by SessionManager and LegacyAdapter.

type sessionBackend interface { //nolint:interfacebloat // test helper mirrors SessionManager API

	GetOrCreate(key string) *Session

	AddMessage(sessionKey, role, content string)

	AddFullMessage(sessionKey string, msg providers.Message)

	GetHistory(key string) []providers.Message

	SetHistory(key string, history []providers.Message)

	GetSummary(key string) string

	SetSummary(key string, summary string)

	TruncateHistory(key string, keepLast int)

	MarkDirty(key string)

	FlushDirty()

	Save(key string) error

	Close()
}

func backends(t *testing.T) map[string]sessionBackend {
	t.Helper()

	jsonDir := t.TempDir()

	sm := NewSessionManager(jsonDir)

	t.Cleanup(func() { sm.Close() })

	dbPath := filepath.Join(t.TempDir(), "test.db")

	store, err := OpenSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLiteStore: %v", err)
	}

	la := NewLegacyAdapter(store)

	t.Cleanup(func() { la.Close() })

	return map[string]sessionBackend{
		"json": sm,

		"sqlite": la,
	}
}

func TestBackend_GetOrCreate(t *testing.T) {
	for name, be := range backends(t) {
		t.Run(name, func(t *testing.T) {
			s := be.GetOrCreate("k1")

			if s.Key != "k1" {
				t.Errorf("expected key k1, got %s", s.Key)
			}

			if len(s.Messages) != 0 {
				t.Errorf("expected empty messages, got %d", len(s.Messages))
			}

			// Second call returns existing

			be.AddMessage("k1", "user", "hello")

			s2 := be.GetOrCreate("k1")

			if s2.Key != "k1" {
				t.Errorf("expected key k1 on second call")
			}
		})
	}
}

func TestBackend_AddMessageAndGetHistory(t *testing.T) {
	for name, be := range backends(t) {
		t.Run(name, func(t *testing.T) {
			be.GetOrCreate("k1")

			be.AddMessage("k1", "user", "hello")

			be.AddMessage("k1", "assistant", "hi")

			history := be.GetHistory("k1")

			if len(history) != 2 {
				t.Fatalf("expected 2 messages, got %d", len(history))
			}

			if history[0].Role != "user" || history[0].Content != "hello" {
				t.Errorf("unexpected first message: %+v", history[0])
			}

			if history[1].Role != "assistant" || history[1].Content != "hi" {
				t.Errorf("unexpected second message: %+v", history[1])
			}
		})
	}
}

func TestBackend_AddFullMessage(t *testing.T) {
	for name, be := range backends(t) {
		t.Run(name, func(t *testing.T) {
			be.GetOrCreate("k1")

			be.AddFullMessage("k1", providers.Message{
				Role: "assistant",

				Content: "sure",

				ToolCalls: []providers.ToolCall{
					{ID: "call_1", Type: "function", Function: &providers.FunctionCall{Name: "exec", Arguments: `{}`}},
				},
			})

			be.AddFullMessage("k1", providers.Message{
				Role: "tool",

				Content: "ok",

				ToolCallID: "call_1",
			})

			history := be.GetHistory("k1")

			if len(history) != 2 {
				t.Fatalf("expected 2, got %d", len(history))
			}

			if history[0].ToolCalls[0].ID != "call_1" {
				t.Errorf("tool call ID mismatch")
			}

			if history[1].ToolCallID != "call_1" {
				t.Errorf("tool call result ID mismatch")
			}
		})
	}
}

func TestBackend_AddFullMessage_AutoCreates(t *testing.T) {
	for name, be := range backends(t) {
		t.Run(name, func(t *testing.T) {
			// AddFullMessage without prior GetOrCreate should still work

			be.AddFullMessage("auto", providers.Message{Role: "user", Content: "hi"})

			history := be.GetHistory("auto")

			if len(history) != 1 {
				t.Fatalf("expected 1, got %d", len(history))
			}
		})
	}
}

func TestBackend_SetHistory(t *testing.T) {
	for name, be := range backends(t) {
		t.Run(name, func(t *testing.T) {
			be.GetOrCreate("k1")

			be.AddMessage("k1", "user", "old")

			newHistory := []providers.Message{
				{Role: "user", Content: "new1"},

				{Role: "assistant", Content: "new2"},
			}

			be.SetHistory("k1", newHistory)

			got := be.GetHistory("k1")

			if len(got) != 2 || got[0].Content != "new1" || got[1].Content != "new2" {
				t.Errorf("unexpected history after SetHistory: %+v", got)
			}
		})
	}
}

func TestBackend_GetSetSummary(t *testing.T) {
	for name, be := range backends(t) {
		t.Run(name, func(t *testing.T) {
			be.GetOrCreate("k1")

			if s := be.GetSummary("k1"); s != "" {
				t.Errorf("expected empty summary, got %q", s)
			}

			be.SetSummary("k1", "test summary")

			if s := be.GetSummary("k1"); s != "test summary" {
				t.Errorf("expected 'test summary', got %q", s)
			}
		})
	}
}

func TestBackend_TruncateHistory(t *testing.T) {
	for name, be := range backends(t) {
		t.Run(name, func(t *testing.T) {
			be.GetOrCreate("k1")

			for i := range 10 {
				be.AddMessage("k1", "user", string(rune('a'+i)))
			}

			be.TruncateHistory("k1", 3)

			got := be.GetHistory("k1")

			if len(got) != 3 {
				t.Fatalf("expected 3, got %d", len(got))
			}

			if got[0].Content != "h" {
				t.Errorf("expected 'h', got %q", got[0].Content)
			}
		})
	}
}

func TestBackend_TruncateHistory_Zero(t *testing.T) {
	for name, be := range backends(t) {
		t.Run(name, func(t *testing.T) {
			be.GetOrCreate("k1")

			be.AddMessage("k1", "user", "hello")

			be.TruncateHistory("k1", 0)

			got := be.GetHistory("k1")

			if len(got) != 0 {
				t.Errorf("expected 0, got %d", len(got))
			}
		})
	}
}

func TestBackend_TruncateHistory_LargerThanLen(t *testing.T) {
	for name, be := range backends(t) {
		t.Run(name, func(t *testing.T) {
			be.GetOrCreate("k1")

			be.AddMessage("k1", "user", "hello")

			be.TruncateHistory("k1", 100)

			got := be.GetHistory("k1")

			if len(got) != 1 {
				t.Errorf("expected 1, got %d", len(got))
			}
		})
	}
}

func TestBackend_GetHistory_DefensiveCopy(t *testing.T) {
	for name, be := range backends(t) {
		t.Run(name, func(t *testing.T) {
			be.GetOrCreate("k1")

			be.AddMessage("k1", "user", "hello")

			h1 := be.GetHistory("k1")

			h1[0].Content = "modified"

			h2 := be.GetHistory("k1")

			if h2[0].Content != "hello" {
				t.Errorf("defensive copy failed: %q", h2[0].Content)
			}
		})
	}
}

func TestBackend_GetHistory_NonExistent(t *testing.T) {
	for name, be := range backends(t) {
		t.Run(name, func(t *testing.T) {
			got := be.GetHistory("nope")

			if got == nil || len(got) != 0 {
				t.Errorf("expected empty slice, got %v", got)
			}
		})
	}
}

func TestBackend_MarkDirtyAndFlush(t *testing.T) {
	for name, be := range backends(t) {
		t.Run(name, func(t *testing.T) {
			be.GetOrCreate("k1")

			be.AddMessage("k1", "user", "hello")

			be.MarkDirty("k1")

			be.FlushDirty()

			// Should not panic or error
		})
	}
}

func TestBackend_SaveAndReload(t *testing.T) {
	// Test that Save persists data that can be reloaded.

	// For JSON backend, we reload via new SessionManager.

	// For SQLite, we reload via new LegacyAdapter on same DB.

	t.Run("json", func(t *testing.T) {
		dir := t.TempDir()

		sm := NewSessionManager(dir)

		sm.GetOrCreate("k1")

		sm.AddMessage("k1", "user", "hello")

		sm.SetSummary("k1", "test")

		sm.Save("k1")

		sm.Close()

		sm2 := NewSessionManager(dir)

		defer sm2.Close()

		h := sm2.GetHistory("k1")

		if len(h) != 1 || h[0].Content != "hello" {
			t.Errorf("json reload: expected [hello], got %+v", h)
		}

		if s := sm2.GetSummary("k1"); s != "test" {
			t.Errorf("json reload summary: expected 'test', got %q", s)
		}
	})

	t.Run("sqlite", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "test.db")

		store, _ := OpenSQLiteStore(dbPath)

		la := NewLegacyAdapter(store)

		la.GetOrCreate("k1")

		la.AddMessage("k1", "user", "hello")

		la.SetSummary("k1", "test")

		la.Save("k1")

		la.Close()

		store2, _ := OpenSQLiteStore(dbPath)

		la2 := NewLegacyAdapter(store2)

		defer la2.Close()

		h := la2.GetHistory("k1")

		if len(h) != 1 || h[0].Content != "hello" {
			t.Errorf("sqlite reload: expected [hello], got %+v", h)
		}

		if s := la2.GetSummary("k1"); s != "test" {
			t.Errorf("sqlite reload summary: expected 'test', got %q", s)
		}
	})
}

func TestBackend_SaveAfterSetHistory(t *testing.T) {
	// Verify that Save after SetHistory (full replacement) works correctly

	t.Run("sqlite", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "test.db")

		store, _ := OpenSQLiteStore(dbPath)

		la := NewLegacyAdapter(store)

		la.GetOrCreate("k1")

		la.AddMessage("k1", "user", "old1")

		la.AddMessage("k1", "user", "old2")

		la.Save("k1")

		// Replace history

		la.SetHistory("k1", []providers.Message{
			{Role: "user", Content: "new1"},
		})

		la.Save("k1")

		la.Close()

		// Reload and verify

		store2, _ := OpenSQLiteStore(dbPath)

		la2 := NewLegacyAdapter(store2)

		defer la2.Close()

		h := la2.GetHistory("k1")

		if len(h) != 1 || h[0].Content != "new1" {
			t.Errorf("expected [new1], got %+v", h)
		}
	})
}

func TestBackend_IncrementalSave(t *testing.T) {
	// Verify that incremental saves only add new messages

	t.Run("sqlite", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "test.db")

		store, _ := OpenSQLiteStore(dbPath)

		la := NewLegacyAdapter(store)

		la.GetOrCreate("k1")

		la.AddMessage("k1", "user", "msg1")

		la.Save("k1")

		la.AddMessage("k1", "user", "msg2")

		la.Save("k1")

		la.Close()

		// Verify 2 turns were created (one per save)

		store2, _ := OpenSQLiteStore(dbPath)

		defer store2.Close()

		turns, _ := store2.Turns("k1", 0)

		if len(turns) != 2 {
			t.Errorf("expected 2 turns (incremental), got %d", len(turns))
		}

		// But total messages should be 2

		la2 := NewLegacyAdapter(store2)

		h := la2.GetHistory("k1")

		if len(h) != 2 {
			t.Errorf("expected 2 messages total, got %d", len(h))
		}
	})
}
