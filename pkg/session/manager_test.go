package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"telegram:123456", "telegram_123456"},
		{"discord:987654321", "discord_987654321"},
		{"slack:C01234", "slack_C01234"},
		{"no-colons-here", "no-colons-here"},
		{"multiple:colons:here", "multiple_colons_here"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSave_WithColonInKey(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	// Create a session with a key containing colon (typical channel session key).
	key := "telegram:123456"
	sm.GetOrCreate(key)
	sm.AddMessage(key, "user", "hello")

	// Save should succeed even though the key contains ':'
	if err := sm.Save(key); err != nil {
		t.Fatalf("Save(%q) failed: %v", key, err)
	}

	// The file on disk should use sanitized name.
	expectedFile := filepath.Join(tmpDir, "telegram_123456.json")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("expected session file %s to exist", expectedFile)
	}

	// Load into a fresh manager and verify the session round-trips.
	sm2 := NewSessionManager(tmpDir)
	history := sm2.GetHistory(key)
	if len(history) != 1 {
		t.Fatalf("expected 1 message after reload, got %d", len(history))
	}
	if history[0].Content != "hello" {
		t.Errorf("expected message content %q, got %q", "hello", history[0].Content)
	}
}

func TestSave_RejectsPathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	badKeys := []string{"", ".", "..", "foo/bar", "foo\\bar"}
	for _, key := range badKeys {
		sm.GetOrCreate(key)
		if err := sm.Save(key); err == nil {
			t.Errorf("Save(%q) should have failed but didn't", key)
		}
	}
}

func TestTruncateHistory_PreservesToolPairs(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	key := "test-truncate"
	sm.GetOrCreate(key)

	// Build: user, assistant+2tools, tool1, tool2, user, assistant = 6 messages
	sm.AddFullMessage(key, providers.Message{Role: "user", Content: "q1"})
	sm.AddFullMessage(key, providers.Message{
		Role:    "assistant",
		Content: "checking",
		ToolCalls: []providers.ToolCall{
			{ID: "c1", Name: "exec"},
			{ID: "c2", Name: "web"},
		},
	})
	sm.AddFullMessage(key, providers.Message{Role: "tool", Content: "r1", ToolCallID: "c1"})
	sm.AddFullMessage(key, providers.Message{Role: "tool", Content: "r2", ToolCallID: "c2"})
	sm.AddFullMessage(key, providers.Message{Role: "user", Content: "q2"})
	sm.AddFullMessage(key, providers.Message{Role: "assistant", Content: "done"})

	// keepLast=4 naively keeps: [tool2, user, assistant_done] or similar
	// which orphans tool messages. Should snap to include/exclude full group.
	sm.TruncateHistory(key, 4)
	history := sm.GetHistory(key)

	// Verify no orphaned tool messages
	toolCallIDs := map[string]bool{}
	toolResultIDs := map[string]bool{}
	for _, m := range history {
		for _, tc := range m.ToolCalls {
			toolCallIDs[tc.ID] = true
		}
		if m.Role == "tool" && m.ToolCallID != "" {
			toolResultIDs[m.ToolCallID] = true
		}
	}

	for id := range toolResultIDs {
		if !toolCallIDs[id] {
			t.Errorf("orphaned tool_result %q after truncation", id)
		}
	}
	for id := range toolCallIDs {
		if !toolResultIDs[id] {
			t.Errorf("orphaned tool_call %q after truncation", id)
		}
	}
}
