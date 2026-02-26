package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveEditPath_AbsoluteBlocked(t *testing.T) {
	workspace := t.TempDir()
	_, err := resolveEditPath("/etc/passwd", workspace, workspace)
	if err == nil {
		t.Fatal("Expected absolute path outside workspace to be blocked")
	}
}

func TestResolveEditPath_TildeIsWorkspace(t *testing.T) {
	workspace := t.TempDir()
	// Create a file so the path resolves
	os.WriteFile(filepath.Join(workspace, "test.txt"), []byte("hi"), 0o644)

	path, err := resolveEditPath("~/test.txt", workspace, workspace)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	expected := filepath.Join(workspace, "test.txt")
	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}

func TestResolveEditPath_BareTildeIsWorkspace(t *testing.T) {
	workspace := t.TempDir()
	path, err := resolveEditPath("~", workspace, workspace)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if path != workspace {
		t.Errorf("Expected %s, got %s", workspace, path)
	}
}

func TestResolveEditPath_TraversalBlocked(t *testing.T) {
	workspace := t.TempDir()
	_, err := resolveEditPath("../../etc/passwd", workspace, workspace)
	if err == nil {
		t.Fatal("Expected path traversal to be blocked")
	}
}

func TestResolveEditPath_SymlinkBlocked(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	os.MkdirAll(workspace, 0o755)
	secret := filepath.Join(root, "secret.txt")
	os.WriteFile(secret, []byte("secret"), 0o644)
	link := filepath.Join(workspace, "link.txt")
	if err := os.Symlink(secret, link); err != nil {
		t.Skip("symlinks not supported")
	}
	_, err := resolveEditPath("link.txt", workspace, workspace)
	if err == nil {
		t.Fatal("Expected symlink escape to be blocked")
	}
}

func TestResolveEditPath_ValidRelative(t *testing.T) {
	workspace := t.TempDir()
	subdir := filepath.Join(workspace, "subdir")
	os.MkdirAll(subdir, 0o755)
	testFile := filepath.Join(subdir, "test.txt")
	os.WriteFile(testFile, []byte("content"), 0o644)

	path, err := resolveEditPath("subdir/test.txt", workspace, workspace)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if path != testFile {
		t.Errorf("Expected %s, got %s", testFile, path)
	}
}

func TestShortenHomePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get home dir")
	}

	tests := []struct {
		input    string
		expected string
	}{
		{home, "~"},
		{filepath.Join(home, "projects"), "~/projects"},
		{"/tmp/other", "/tmp/other"},
	}

	for _, tt := range tests {
		result := shortenHomePath(tt.input)
		if result != tt.expected {
			t.Errorf("shortenHomePath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
