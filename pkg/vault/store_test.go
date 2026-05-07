package vault

import (
	"testing"
	"path/filepath"
	"os"
)

func TestCreateNote(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "test-vault")
	defer os.RemoveAll(tmpDir)

	store := NewVaultStore(tmpDir)
	
	err := store.CreateNote("test-note", map[string]interface{}{
		"title": "Test Note",
		"tags":  []string{"memory", "test"},
	}, "## Content here")
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	notePath := filepath.Join(tmpDir, "test-note.md")
	if _, err := os.Stat(notePath); os.IsNotExist(err) {
		t.Errorf("expected note file to be created at %s", notePath)
	}
}

func TestReadNoteWithFrontmatter(t *testing.T) {
	store := NewVaultStore("/tmp/test-vault")
	
	// Create note with frontmatter
	store.CreateNote("MEMORY", map[string]interface{}{
		"title": "Agent Memory",
		"tags":  []string{"memory"},
	}, "# Content")
	
	fm, body, err := store.ReadNote("MEMORY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm["title"] != "Agent Memory" {
		t.Errorf("expected title in frontmatter")
	}
	if !contains(body, "# Content") {
		t.Errorf("expected body to contain content")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
