// ABOUTME: Tests for the recall tool.
// ABOUTME: Uses a mock memory store to verify search and formatting behavior.
package tools

import (
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/memory"
)

func TestRecallTool_Name(t *testing.T) {
	tool := NewRecallTool(&mockStore{available: true})
	if got := tool.Name(); got != "recall" {
		t.Errorf("Name() = %q, want %q", got, "recall")
	}
}

func TestRecallTool_Execute_Success(t *testing.T) {
	store := &mockStore{
		available: true,
		entries: []memory.MemoryEntry{
			{
				Content:   "User prefers dark mode",
				Category:  "preference",
				Tags:      []string{"ui"},
				Timestamp: time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC),
			},
			{
				Content:  "Database is PostgreSQL",
				Category: "fact",
			},
		},
	}
	tool := NewRecallTool(store)

	result := tool.Execute(t.Context(), map[string]any{
		"query": "user preferences",
		"top_k": float64(5),
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !result.Silent {
		t.Error("expected silent result")
	}
	if !strings.Contains(result.ForLLM, "Found 2 memories") {
		t.Errorf("expected 'Found 2 memories' in result, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "dark mode") {
		t.Errorf("expected 'dark mode' in result, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "90% match") {
		t.Errorf("expected '90%% match' in result, got: %s", result.ForLLM)
	}
}

func TestRecallTool_Execute_NoResults(t *testing.T) {
	store := &mockStore{available: true, entries: nil}
	tool := NewRecallTool(store)

	result := tool.Execute(t.Context(), map[string]any{
		"query": "nonexistent",
	})

	if result.IsError {
		t.Fatalf("expected success (no results), got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "No memories found") {
		t.Errorf("expected 'No memories found' in result, got: %s", result.ForLLM)
	}
}

func TestRecallTool_Execute_EmptyQuery(t *testing.T) {
	tool := NewRecallTool(&mockStore{available: true})
	result := tool.Execute(t.Context(), map[string]any{
		"query": "",
	})
	if !result.IsError {
		t.Error("expected error for empty query")
	}
}

func TestRecallTool_Execute_Unavailable(t *testing.T) {
	tool := NewRecallTool(&mockStore{available: false})
	result := tool.Execute(t.Context(), map[string]any{
		"query": "test",
	})
	if !result.IsError {
		t.Error("expected error when store unavailable")
	}
}

func TestRecallTool_Execute_DefaultTopK(t *testing.T) {
	// Ensure default top_k of 5 is used when not specified
	store := &mockStore{
		available: true,
		entries: []memory.MemoryEntry{
			{Content: "a", Category: "fact"},
			{Content: "b", Category: "fact"},
			{Content: "c", Category: "fact"},
			{Content: "d", Category: "fact"},
			{Content: "e", Category: "fact"},
			{Content: "f", Category: "fact"},
		},
	}
	tool := NewRecallTool(store)

	result := tool.Execute(t.Context(), map[string]any{
		"query": "test",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Found 5 memories") {
		t.Errorf("expected default top_k of 5, got: %s", result.ForLLM)
	}
}
