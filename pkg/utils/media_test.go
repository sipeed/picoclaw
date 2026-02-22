package utils

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMediaCleanerRemovesOldFiles(t *testing.T) {
	// Setup: create a temp media directory with old and new files
	mediaDir := filepath.Join(os.TempDir(), MediaDir)
	if err := os.MkdirAll(mediaDir, 0o700); err != nil {
		t.Fatalf("failed to create media dir: %v", err)
	}

	// Create an "old" file and backdate its modification time
	oldFile := filepath.Join(mediaDir, "test_old_file.jpg")
	if err := os.WriteFile(oldFile, []byte("old"), 0o600); err != nil {
		t.Fatalf("failed to create old file: %v", err)
	}
	oldTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatalf("failed to set old file time: %v", err)
	}

	// Create a "new" file (just created, so modtime is now)
	newFile := filepath.Join(mediaDir, "test_new_file.jpg")
	if err := os.WriteFile(newFile, []byte("new"), 0o600); err != nil {
		t.Fatalf("failed to create new file: %v", err)
	}

	// Cleanup test files at end
	defer os.Remove(oldFile)
	defer os.Remove(newFile)

	// Run cleanup directly
	mc := NewMediaCleaner(5, 30)
	mc.cleanup()

	// Old file should be gone
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Errorf("expected old file to be removed, but it still exists")
	}

	// New file should still exist
	if _, err := os.Stat(newFile); err != nil {
		t.Errorf("expected new file to still exist, got error: %v", err)
	}
}

func TestMediaCleanerStartStop(t *testing.T) {
	mc := NewMediaCleaner(5, 30)

	// Start should not panic
	mc.Start()

	// Second Start should be idempotent (sync.Once)
	mc.Start()

	// Stop should not panic
	mc.Stop()

	// Second Stop should be idempotent
	mc.Stop()
}
