package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/memory"
)

// --- MemorySaveTool tests ---

func TestMemorySave_NewNote(t *testing.T) {
	dir := t.TempDir()
	vault := memory.NewVault(dir)
	tool := NewMemorySaveTool(vault)
	ctx := context.Background()

	result := tool.Execute(ctx, map[string]any{
		"path":    "topics/test-note.md",
		"title":   "Test Note",
		"content": "This is the body.",
		"tags":    "go, testing",
	})

	if result.IsError {
		t.Fatalf("Expected success, got error: %s", result.ForLLM)
	}
	if !result.Silent {
		t.Error("Expected silent result for memory_save")
	}
	if !strings.Contains(result.ForLLM, "Saved") {
		t.Errorf("ForLLM = %q, expected to contain 'Saved'", result.ForLLM)
	}

	// Verify file exists and has correct frontmatter
	data, err := os.ReadFile(filepath.Join(dir, "topics", "test-note.md"))
	if err != nil {
		t.Fatalf("Note file not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "title: Test Note") {
		t.Error("Note missing title in frontmatter")
	}
	if !strings.Contains(content, "tags: [go, testing]") {
		t.Error("Note missing tags in frontmatter")
	}
	if !strings.Contains(content, "This is the body.") {
		t.Error("Note missing body content")
	}

	// Verify index was updated
	index := vault.ReadIndex()
	if !strings.Contains(index, "Test Note") {
		t.Error("Index not updated after save")
	}
}

func TestMemorySave_MissingRequired(t *testing.T) {
	dir := t.TempDir()
	vault := memory.NewVault(dir)
	tool := NewMemorySaveTool(vault)
	ctx := context.Background()

	// Missing path
	result := tool.Execute(ctx, map[string]any{
		"title":   "Test",
		"content": "Body",
	})
	if !result.IsError {
		t.Error("Expected error for missing path")
	}

	// Missing title
	result = tool.Execute(ctx, map[string]any{
		"path":    "test.md",
		"content": "Body",
	})
	if !result.IsError {
		t.Error("Expected error for missing title")
	}

	// Missing content
	result = tool.Execute(ctx, map[string]any{
		"path":  "test.md",
		"title": "Test",
	})
	if !result.IsError {
		t.Error("Expected error for missing content")
	}
}

// --- MemorySearchTool tests ---

func TestMemorySearch_ByTags(t *testing.T) {
	dir := t.TempDir()
	vault := memory.NewVault(dir)

	// Set up test notes
	vault.SaveNote("a.md", memory.NoteMeta{Title: "Go Errors", Tags: []string{"go", "errors"}}, "Content A.")
	vault.SaveNote("b.md", memory.NoteMeta{Title: "Go Testing", Tags: []string{"go", "testing"}}, "Content B.")
	vault.SaveNote("c.md", memory.NoteMeta{Title: "Python", Tags: []string{"python"}}, "Content C.")

	tool := NewMemorySearchTool(vault)
	ctx := context.Background()

	result := tool.Execute(ctx, map[string]any{
		"tags": "go",
	})

	if result.IsError {
		t.Fatalf("Expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Go Errors") {
		t.Error("Search result should contain 'Go Errors'")
	}
	if !strings.Contains(result.ForLLM, "Go Testing") {
		t.Error("Search result should contain 'Go Testing'")
	}
	if strings.Contains(result.ForLLM, "Python") {
		t.Error("Search result should not contain 'Python'")
	}
}

func TestMemorySearch_ByQuery(t *testing.T) {
	dir := t.TempDir()
	vault := memory.NewVault(dir)

	vault.SaveNote("a.md", memory.NoteMeta{Title: "Go Errors", Tags: []string{"go"}}, "Content.")
	vault.SaveNote("b.md", memory.NoteMeta{Title: "Python Basics", Tags: []string{"python"}}, "Content.")

	tool := NewMemorySearchTool(vault)
	ctx := context.Background()

	result := tool.Execute(ctx, map[string]any{
		"query": "Error",
	})

	if result.IsError {
		t.Fatalf("Expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Go Errors") {
		t.Error("Search result should contain 'Go Errors'")
	}
	if strings.Contains(result.ForLLM, "Python") {
		t.Error("Search result should not contain 'Python'")
	}
}

func TestMemorySearch_NoParams(t *testing.T) {
	dir := t.TempDir()
	vault := memory.NewVault(dir)
	tool := NewMemorySearchTool(vault)
	ctx := context.Background()

	result := tool.Execute(ctx, map[string]any{})
	if result.IsError {
		t.Error("Search with no params should not error (returns all notes)")
	}
}

// --- MemoryRecallTool tests ---

func TestMemoryRecall_ByPath(t *testing.T) {
	dir := t.TempDir()
	vault := memory.NewVault(dir)

	vault.SaveNote("test.md", memory.NoteMeta{Title: "Test Note", Tags: []string{"test"}}, "Full body content here.")

	tool := NewMemoryRecallTool(vault)
	ctx := context.Background()

	result := tool.Execute(ctx, map[string]any{
		"path": "test.md",
	})

	if result.IsError {
		t.Fatalf("Expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Full body content here.") {
		t.Error("Recall should return full note content")
	}
}

func TestMemoryRecall_ByTopic(t *testing.T) {
	dir := t.TempDir()
	vault := memory.NewVault(dir)

	vault.SaveNote("go-errors.md", memory.NoteMeta{Title: "Go Error Patterns", Tags: []string{"go"}}, "Error patterns body.")
	vault.SaveNote("python.md", memory.NoteMeta{Title: "Python Basics", Tags: []string{"python"}}, "Python body.")

	tool := NewMemoryRecallTool(vault)
	ctx := context.Background()

	result := tool.Execute(ctx, map[string]any{
		"topic": "Go Error",
	})

	if result.IsError {
		t.Fatalf("Expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Error patterns body.") {
		t.Error("Recall should return matching note content")
	}
}

func TestMemoryRecall_MissingNote(t *testing.T) {
	dir := t.TempDir()
	vault := memory.NewVault(dir)
	tool := NewMemoryRecallTool(vault)
	ctx := context.Background()

	result := tool.Execute(ctx, map[string]any{
		"path": "nonexistent.md",
	})

	if !result.IsError {
		t.Error("Expected error for missing note")
	}
}

func TestMemoryRecall_NoParams(t *testing.T) {
	dir := t.TempDir()
	vault := memory.NewVault(dir)
	tool := NewMemoryRecallTool(vault)
	ctx := context.Background()

	result := tool.Execute(ctx, map[string]any{})
	if !result.IsError {
		t.Error("Expected error when no path or topic provided")
	}
}
