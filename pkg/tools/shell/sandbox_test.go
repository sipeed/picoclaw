package shell

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

// TestSandboxedOpenHandler_RelativePathUsesInterpreterCwd verifies that
// relative paths in shell redirections resolve against the interpreter's
// working directory (from interp.HandlerCtx), not the process CWD.
func TestSandboxedOpenHandler_RelativePathUsesInterpreterCwd(t *testing.T) {
	workspace := t.TempDir()

	// Write to a relative path inside the workspace via the interpreter.
	// The interpreter's Dir is set to workspace, so "output.txt" should
	// resolve to workspace/output.txt regardless of the process CWD.
	result := Run(context.Background(), RunConfig{
		Command:       "echo sandbox_relative > output.txt",
		Dir:           workspace,
		Timeout:       5 * time.Second,
		Restrict:      true,
		WorkspaceDir:  workspace,
		RiskThreshold: RiskMedium,
	})

	if result.IsError {
		t.Fatalf("relative redirect inside workspace should succeed: %s", result.Output)
	}

	content, err := os.ReadFile(filepath.Join(workspace, "output.txt"))
	if err != nil {
		t.Fatalf("output.txt should exist in workspace: %v", err)
	}
	if !strings.Contains(string(content), "sandbox_relative") {
		t.Errorf("expected 'sandbox_relative' in file, got: %s", content)
	}
}

// TestSandboxedOpenHandler_RelativePathBlocksEscapeViaCd verifies that
// if a script uses cd to move outside the workspace, a subsequent relative
// redirect is blocked by the sandbox.
func TestSandboxedOpenHandler_RelativePathBlocksEscapeViaCd(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	outside := filepath.Join(root, "outside")
	os.MkdirAll(workspace, 0o755)
	os.MkdirAll(outside, 0o755)

	// cd to outside dir, then try to write a relative path.
	// The sandbox should block because the resolved path is outside workspace.
	result := Run(context.Background(), RunConfig{
		Command:       "cd " + outside + " && echo escaped > leak.txt",
		Dir:           workspace,
		Timeout:       5 * time.Second,
		Restrict:      true,
		WorkspaceDir:  workspace,
		RiskThreshold: RiskHigh, // allow cd
	})

	if !result.IsError {
		// If it didn't error, check that the file was NOT written outside
		if _, err := os.Stat(filepath.Join(outside, "leak.txt")); err == nil {
			t.Fatal("sandbox should have blocked write outside workspace")
		}
	}

	// Verify nothing leaked
	if _, err := os.Stat(filepath.Join(outside, "leak.txt")); err == nil {
		t.Error("leak.txt should not exist outside workspace")
	}
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
