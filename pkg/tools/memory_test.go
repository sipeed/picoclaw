package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMemSearchTool_BasicSearch(t *testing.T) {
	// Setup temp workspace
	tmpDir := t.TempDir()
	memDir := filepath.Join(tmpDir, "memory")
	os.MkdirAll(memDir, 0755)

	// Create MEMORY.md with structured content
	memContent := `# Memory

## Preferences

- User prefers dark mode for all editors _2026-01-15_
- Language: Portuguese (Brazil) _2026-01-10_

## Facts

- User works at Acme Corp _2026-02-01_
- Main project is PicoClaw _2026-02-10_

## Projects

- PicoClaw: personal AI agent in Go _2026-02-10_
`
	os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte(memContent), 0644)

	tool := NewMemSearchTool(tmpDir)

	// Test basic search
	result := tool.Execute(context.Background(), map[string]interface{}{
		"query": "dark mode",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "dark mode") {
		t.Errorf("expected result to contain 'dark mode', got: %s", result.ForLLM)
	}

	// Test category filter
	result = tool.Execute(context.Background(), map[string]interface{}{
		"query":    "PicoClaw",
		"category": "projects",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "PicoClaw") {
		t.Errorf("expected result to contain 'PicoClaw', got: %s", result.ForLLM)
	}

	// Test no results
	result = tool.Execute(context.Background(), map[string]interface{}{
		"query": "nonexistent keyword xyz",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "No results") {
		t.Errorf("expected 'No results', got: %s", result.ForLLM)
	}
}

func TestMemSaveTool_SaveAndRetrieve(t *testing.T) {
	tmpDir := t.TempDir()
	memDir := filepath.Join(tmpDir, "memory")
	os.MkdirAll(memDir, 0755)

	saveTool := NewMemSaveTool(tmpDir)

	// Save a preference
	result := saveTool.Execute(context.Background(), map[string]interface{}{
		"category": "preferences",
		"content":  "User prefers vim keybindings",
		"tags":     "editor,vim",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	// Verify file was created with correct structure
	data, err := os.ReadFile(filepath.Join(memDir, "MEMORY.md"))
	if err != nil {
		t.Fatalf("failed to read MEMORY.md: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "## Preferences") {
		t.Error("expected ## Preferences section")
	}
	if !strings.Contains(content, "vim keybindings") {
		t.Error("expected saved content")
	}
	if !strings.Contains(content, "[editor,vim]") {
		t.Error("expected tags")
	}

	// Save another entry in same category
	result = saveTool.Execute(context.Background(), map[string]interface{}{
		"category": "preferences",
		"content":  "Prefers dark theme",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	data, _ = os.ReadFile(filepath.Join(memDir, "MEMORY.md"))
	content = string(data)
	if !strings.Contains(content, "vim keybindings") || !strings.Contains(content, "dark theme") {
		t.Error("expected both entries to be present")
	}

	// Save to a different category
	result = saveTool.Execute(context.Background(), map[string]interface{}{
		"category": "facts",
		"content":  "User's name is John",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	data, _ = os.ReadFile(filepath.Join(memDir, "MEMORY.md"))
	content = string(data)
	if !strings.Contains(content, "## Facts") {
		t.Error("expected ## Facts section")
	}

	// Now verify mem_search can find the saved entries
	searchTool := NewMemSearchTool(tmpDir)
	result = searchTool.Execute(context.Background(), map[string]interface{}{
		"query": "vim",
	})
	if !strings.Contains(result.ForLLM, "vim keybindings") {
		t.Errorf("expected to find saved entry via search, got: %s", result.ForLLM)
	}
}

func TestMemIndexTool_BuildIndex(t *testing.T) {
	tmpDir := t.TempDir()
	memDir := filepath.Join(tmpDir, "memory")
	os.MkdirAll(memDir, 0755)

	// Create structured memory
	memContent := `# Memory

## Preferences

- Dark mode enabled
- Portuguese language
- Vim keybindings

## Facts

- Works at Acme Corp

## Projects

- PicoClaw: AI agent
- Website: portfolio site
`
	os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte(memContent), 0644)

	tool := NewMemIndexTool(tmpDir)
	result := tool.Execute(context.Background(), nil)

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	// Check index contains category summaries
	if !strings.Contains(result.ForLLM, "Preferences") {
		t.Error("expected Preferences in index")
	}
	if !strings.Contains(result.ForLLM, "3 entries") {
		t.Errorf("expected '3 entries' for Preferences, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Facts") {
		t.Error("expected Facts in index")
	}
	if !strings.Contains(result.ForLLM, "Projects") {
		t.Error("expected Projects in index")
	}
}

func TestMemSaveTool_InvalidCategory(t *testing.T) {
	tmpDir := t.TempDir()
	saveTool := NewMemSaveTool(tmpDir)

	result := saveTool.Execute(context.Background(), map[string]interface{}{
		"category": "invalid_category",
		"content":  "test",
	})
	if !result.IsError {
		t.Error("expected error for invalid category")
	}
}

func TestMemSearchTool_EmptyQuery(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewMemSearchTool(tmpDir)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"query": "",
	})
	if !result.IsError {
		t.Error("expected error for empty query")
	}
}
