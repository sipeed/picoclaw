package shell

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSandboxedOpenHandler_AllowsInsideWorkspace(t *testing.T) {
	workspace := t.TempDir()
	handler := SandboxedOpenHandler(workspace)

	testFile := filepath.Join(workspace, "test.txt")
	os.WriteFile(testFile, []byte("hello"), 0o644)

	f, err := handler(context.Background(), testFile, os.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("expected open inside workspace to succeed: %v", err)
	}
	f.Close()
}

func TestSandboxedOpenHandler_BlocksOutsideWorkspace(t *testing.T) {
	workspace := t.TempDir()
	handler := SandboxedOpenHandler(workspace)

	outsideFile := filepath.Join(t.TempDir(), "secret.txt")
	os.WriteFile(outsideFile, []byte("secret"), 0o644)

	_, err := handler(context.Background(), outsideFile, os.O_RDONLY, 0)
	if err == nil {
		t.Fatal("expected open outside workspace to be blocked")
	}
}

func TestSandboxedOpenHandler_AllowsSafePaths(t *testing.T) {
	workspace := t.TempDir()
	handler := SandboxedOpenHandler(workspace)

	f, err := handler(context.Background(), "/dev/null", os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("expected /dev/null to be allowed: %v", err)
	}
	f.Close()
}

func TestSandboxedOpenHandler_BlocksSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	secretDir := filepath.Join(root, "secret")
	os.MkdirAll(workspace, 0o755)
	os.MkdirAll(secretDir, 0o755)
	os.WriteFile(filepath.Join(secretDir, "data.txt"), []byte("secret"), 0o644)

	link := filepath.Join(workspace, "escape")
	if err := os.Symlink(secretDir, link); err != nil {
		t.Skipf("symlinks not supported: %v", err)
	}

	handler := SandboxedOpenHandler(workspace)

	target := filepath.Join(link, "data.txt")
	_, err := handler(context.Background(), target, os.O_RDONLY, 0)
	if err == nil {
		t.Fatal("expected symlink escape to be blocked")
	}
}

func TestSandboxedOpenHandler_AllowsNewFileInWorkspace(t *testing.T) {
	workspace := t.TempDir()
	handler := SandboxedOpenHandler(workspace)

	newFile := filepath.Join(workspace, "new_output.txt")
	f, err := handler(context.Background(), newFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		t.Fatalf("expected new file creation inside workspace to succeed: %v", err)
	}
	f.Close()
	os.Remove(newFile)
}

// TestSandboxedOpenHandler_AllowsDottedFiles verifies that files with
// names starting with ".." (like ".../file", "....txt", "..something")
// are NOT incorrectly blocked by the escape check.
// Regression test for bug where rel[:2] == ".." would match these.
func TestSandboxedOpenHandler_AllowsDottedFiles(t *testing.T) {
	workspace := t.TempDir()
	handler := SandboxedOpenHandler(workspace)

	// Create files with names that start with ".." but don't escape
	testCases := []string{
		".../test.txt",
		"....txt",
		"..something.txt",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			fullPath := filepath.Join(workspace, tc)

			// Create parent directories if needed
			if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
				t.Fatal(err)
			}

			// Create the file
			if err := os.WriteFile(fullPath, []byte("test"), 0o644); err != nil {
				t.Fatal(err)
			}
			defer os.Remove(fullPath)

			// Try to open via sandbox handler
			f, err := handler(context.Background(), fullPath, os.O_RDONLY, 0)
			if err != nil {
				t.Errorf("file %q should be allowed within workspace, got error: %v", tc, err)
			}
			if f != nil {
				f.Close()
			}
		})
	}
}
