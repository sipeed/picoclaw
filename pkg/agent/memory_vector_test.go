package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMemoryStore_SearchRelevant_BuildsVectorIndex(t *testing.T) {
	workspace := t.TempDir()
	ms := NewMemoryStore(workspace)
	ms.SetVectorSettings(MemoryVectorSettings{
		Enabled:         true,
		Dimensions:      128,
		TopK:            4,
		MinScore:        0.01,
		MaxContextChars: 1200,
		RecentDailyDays: 7,
	})

	memoryDir := filepath.Join(workspace, "memory")
	if err := os.MkdirAll(memoryDir, 0o755); err != nil {
		t.Fatalf("mkdir memory dir: %v", err)
	}

	memoryContent := `# MEMORY

## Open Threads
- Renew passport in March
- Compare flight options for Tokyo trip
`
	if err := os.WriteFile(filepath.Join(memoryDir, "MEMORY.md"), []byte(memoryContent), 0o644); err != nil {
		t.Fatalf("write MEMORY.md: %v", err)
	}

	day := time.Now().Format("20060102")
	dayPath := filepath.Join(memoryDir, day[:6], day+".md")
	if err := os.MkdirAll(filepath.Dir(dayPath), 0o755); err != nil {
		t.Fatalf("mkdir daily dir: %v", err)
	}
	daily := "# Daily\n\n- Follow up with passport office tomorrow\n"
	if err := os.WriteFile(dayPath, []byte(daily), 0o644); err != nil {
		t.Fatalf("write daily note: %v", err)
	}

	hits, err := ms.SearchRelevant("passport renewal", 3, 0.01)
	if err != nil {
		t.Fatalf("SearchRelevant failed: %v", err)
	}
	if len(hits) == 0 {
		t.Fatalf("expected semantic hits, got none")
	}

	joined := strings.ToLower(hits[0].Text)
	if !strings.Contains(joined, "passport") {
		t.Fatalf("expected top hit to mention passport, got %q", hits[0].Text)
	}

	indexPath := filepath.Join(memoryDir, "vector", "index.json")
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("expected vector index at %s: %v", indexPath, err)
	}
}

func TestMemoryStore_GetBySource(t *testing.T) {
	workspace := t.TempDir()
	ms := NewMemoryStore(workspace)
	ms.SetVectorSettings(MemoryVectorSettings{
		Enabled:         true,
		Dimensions:      128,
		TopK:            4,
		MinScore:        0.01,
		MaxContextChars: 1200,
		RecentDailyDays: 7,
	})

	memoryDir := filepath.Join(workspace, "memory")
	if err := os.MkdirAll(memoryDir, 0o755); err != nil {
		t.Fatalf("mkdir memory dir: %v", err)
	}

	memoryContent := `# MEMORY

## Open Threads
- Prepare tax documents by end of month
`
	if err := os.WriteFile(filepath.Join(memoryDir, "MEMORY.md"), []byte(memoryContent), 0o644); err != nil {
		t.Fatalf("write MEMORY.md: %v", err)
	}

	if _, err := ms.SearchRelevant("tax documents", 3, 0.01); err != nil {
		t.Fatalf("SearchRelevant failed: %v", err)
	}

	hit, found, err := ms.GetBySource("MEMORY.md#Open Threads")
	if err != nil {
		t.Fatalf("GetBySource failed: %v", err)
	}
	if !found {
		t.Fatal("expected source to be found")
	}
	if !strings.Contains(strings.ToLower(hit.Text), "tax") {
		t.Fatalf("expected hit text to include tax info, got: %q", hit.Text)
	}
}
