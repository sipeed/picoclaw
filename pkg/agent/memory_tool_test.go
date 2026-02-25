package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMemorySearchTool_Execute(t *testing.T) {
	workspace := t.TempDir()
	ms := NewMemoryStore(workspace)
	ms.SetVectorSettings(MemoryVectorSettings{
		Enabled:         true,
		Dimensions:      128,
		TopK:            5,
		MinScore:        0.01,
		MaxContextChars: 1200,
		RecentDailyDays: 7,
	})

	if err := os.MkdirAll(filepath.Join(workspace, "memory"), 0o755); err != nil {
		t.Fatalf("mkdir memory dir: %v", err)
	}
	content := `# MEMORY

## Long-term Facts
- Favorite editor: neovim
`
	if err := os.WriteFile(filepath.Join(workspace, "memory", "MEMORY.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write memory: %v", err)
	}

	tool := NewMemorySearchTool(ms, 3, 0.01)
	result := tool.Execute(context.Background(), map[string]any{
		"query": "what editor do I use",
	})

	if result.IsError {
		t.Fatalf("expected successful tool result, got error: %s", result.ForLLM)
	}
	if !result.Silent {
		t.Fatalf("expected memory_search to be silent")
	}
	if !strings.Contains(strings.ToLower(result.ForLLM), "neovim") {
		t.Fatalf("expected tool output to include retrieved memory, got: %s", result.ForLLM)
	}
}

func TestMemoryGetTool_Execute(t *testing.T) {
	workspace := t.TempDir()
	ms := NewMemoryStore(workspace)
	ms.SetVectorSettings(MemoryVectorSettings{
		Enabled:         true,
		Dimensions:      128,
		TopK:            5,
		MinScore:        0.01,
		MaxContextChars: 1200,
		RecentDailyDays: 7,
	})

	if err := os.MkdirAll(filepath.Join(workspace, "memory"), 0o755); err != nil {
		t.Fatalf("mkdir memory dir: %v", err)
	}
	content := `# MEMORY

## Long-term Facts
- Favorite editor: neovim
`
	if err := os.WriteFile(filepath.Join(workspace, "memory", "MEMORY.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write memory: %v", err)
	}

	search := NewMemorySearchTool(ms, 3, 0.01)
	searchResult := search.Execute(context.Background(), map[string]any{
		"query": "favorite editor",
	})
	if searchResult.IsError {
		t.Fatalf("memory_search failed: %s", searchResult.ForLLM)
	}

	get := NewMemoryGetTool(ms)
	getResult := get.Execute(context.Background(), map[string]any{
		"source": "MEMORY.md#Long-term Facts",
	})
	if getResult.IsError {
		t.Fatalf("memory_get failed: %s", getResult.ForLLM)
	}
	if !strings.Contains(strings.ToLower(getResult.ForLLM), "neovim") {
		t.Fatalf("expected memory_get output to include neovim, got: %s", getResult.ForLLM)
	}
}
