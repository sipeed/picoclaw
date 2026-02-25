package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMemoryOrganizeWriteback_DedupesAndSections(t *testing.T) {
	workspace := t.TempDir()
	ms := NewMemoryStore(workspace)

	initial := `# MEMORY

## Long-term Facts
- Likes tea
- Likes tea

## Open Threads
- Renew passport
`
	if err := os.WriteFile(filepath.Join(workspace, "memory", "MEMORY.md"), []byte(initial), 0o644); err != nil {
		t.Fatalf("write initial memory: %v", err)
	}

	update := `## Long-term Facts
- Likes tea
- Works remotely

## Active Goals
- Finish migration
`
	if err := ms.OrganizeWriteback(update); err != nil {
		t.Fatalf("OrganizeWriteback: %v", err)
	}

	got := ms.ReadLongTerm()
	if strings.Count(got, "- Likes tea") != 1 {
		t.Fatalf("expected deduped fact, got:\n%s", got)
	}
	if !strings.Contains(got, "## Active Goals") {
		t.Fatalf("expected Active Goals section, got:\n%s", got)
	}
	if !strings.Contains(got, "- Finish migration") {
		t.Fatalf("expected merged goal, got:\n%s", got)
	}
}
