//go:build !windows

package memory

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestJSONLStore_DirectoryPermissions verifies the session store directory
// is created with 0700 (owner-only) to protect private chat history.
func TestJSONLStore_DirectoryPermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sessions")
	store, err := NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore: %v", err)
	}
	defer store.Close()

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("directory permissions = %04o, want 0700", perm)
	}
}

// TestJSONLStore_FilePermissions verifies that session data files (.jsonl)
// and metadata files (.meta.json) are created with 0600 (owner-only).
func TestJSONLStore_FilePermissions(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.AddMessage(ctx, "perms", "user", "secret conversation")
	if err != nil {
		t.Fatalf("AddMessage: %v", err)
	}

	// Check .jsonl file permissions
	jsonlPath := store.jsonlPath("perms")
	info, err := os.Stat(jsonlPath)
	if err != nil {
		t.Fatalf("Stat jsonl: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("jsonl permissions = %04o, want 0600", perm)
	}

	// Check .meta.json file permissions
	metaPath := store.metaPath("perms")
	info, err = os.Stat(metaPath)
	if err != nil {
		t.Fatalf("Stat meta: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("meta permissions = %04o, want 0600", perm)
	}
}

// TestJSONLStore_RewritePreservesPermissions verifies that SetHistory
// (which rewrites the JSONL file) maintains restrictive permissions.
func TestJSONLStore_RewritePreservesPermissions(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Write initial data
	err := store.AddMessage(ctx, "rewrite", "user", "old message")
	if err != nil {
		t.Fatalf("AddMessage: %v", err)
	}

	// Rewrite via SetHistory
	err = store.SetHistory(ctx, "rewrite", nil)
	if err != nil {
		t.Fatalf("SetHistory: %v", err)
	}

	// Verify the rewritten files still have restrictive permissions
	jsonlPath := store.jsonlPath("rewrite")
	info, err := os.Stat(jsonlPath)
	if err != nil {
		t.Fatalf("Stat jsonl after rewrite: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("jsonl permissions after rewrite = %04o, want 0600", perm)
	}

	metaPath := store.metaPath("rewrite")
	info, err = os.Stat(metaPath)
	if err != nil {
		t.Fatalf("Stat meta after rewrite: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("meta permissions after rewrite = %04o, want 0600", perm)
	}
}
