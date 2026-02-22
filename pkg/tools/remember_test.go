// ABOUTME: Tests for the remember tool.
// ABOUTME: Uses a mock memory store to verify tool behavior without Ollama.
package tools

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/memory"
)

// mockStore implements memory.Store for testing.
type mockStore struct {
	available bool
	entries   []memory.MemoryEntry
}

func (m *mockStore) IsAvailable() bool { return m.available }
func (m *mockStore) Count() int        { return len(m.entries) }

func (m *mockStore) Remember(_ context.Context, entry memory.MemoryEntry) error {
	m.entries = append(m.entries, entry)
	return nil
}

func (m *mockStore) Recall(_ context.Context, query string, topK int) ([]memory.RecallResult, error) {
	var results []memory.RecallResult
	for i, e := range m.entries {
		if i >= topK {
			break
		}
		results = append(results, memory.RecallResult{
			MemoryEntry: e,
			Similarity:  0.9 - float32(i)*0.1,
		})
	}
	return results, nil
}

func TestRememberTool_Name(t *testing.T) {
	tool := NewRememberTool(&mockStore{available: true})
	if got := tool.Name(); got != "remember" {
		t.Errorf("Name() = %q, want %q", got, "remember")
	}
}

func TestRememberTool_Execute_Success(t *testing.T) {
	store := &mockStore{available: true}
	tool := NewRememberTool(store)

	result := tool.Execute(t.Context(), map[string]any{
		"content":  "User prefers dark mode",
		"category": "preference",
		"tags":     "ui,theme",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !result.Silent {
		t.Error("expected silent result")
	}
	if len(store.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(store.entries))
	}
	entry := store.entries[0]
	if entry.Content != "User prefers dark mode" {
		t.Errorf("content = %q, want %q", entry.Content, "User prefers dark mode")
	}
	if entry.Category != "preference" {
		t.Errorf("category = %q, want %q", entry.Category, "preference")
	}
	if len(entry.Tags) != 2 || entry.Tags[0] != "ui" || entry.Tags[1] != "theme" {
		t.Errorf("tags = %v, want [ui theme]", entry.Tags)
	}
	if entry.Source != "agent" {
		t.Errorf("source = %q, want %q", entry.Source, "agent")
	}
}

func TestRememberTool_Execute_DefaultCategory(t *testing.T) {
	store := &mockStore{available: true}
	tool := NewRememberTool(store)

	result := tool.Execute(t.Context(), map[string]any{
		"content": "some fact",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if store.entries[0].Category != "other" {
		t.Errorf("category = %q, want default %q", store.entries[0].Category, "other")
	}
}

func TestRememberTool_Execute_EmptyContent(t *testing.T) {
	tool := NewRememberTool(&mockStore{available: true})
	result := tool.Execute(t.Context(), map[string]any{
		"content": "",
	})
	if !result.IsError {
		t.Error("expected error for empty content")
	}
}

func TestRememberTool_Execute_Unavailable(t *testing.T) {
	tool := NewRememberTool(&mockStore{available: false})
	result := tool.Execute(t.Context(), map[string]any{
		"content": "test",
	})
	if !result.IsError {
		t.Error("expected error when store unavailable")
	}
}
