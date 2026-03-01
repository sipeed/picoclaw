package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sipeed/picoclaw/pkg/agent/sandbox"
)

// TestFilesystemTool_ReadFile_Success verifies successful file reading
func TestFilesystemTool_ReadFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0o644)

	tool := NewReadFileTool("", false)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{
		fs: sandbox.NewHostSandbox(tmpDir, false).Fs(),
	})
	// We must ensure the mock FsBridge can actually read the TempDir.
	// but stubSandbox uses hostFs internally for Fs(). ReadFile so this just works if injected.
	args := map[string]any{
		"path": testFile,
	}

	result := tool.Execute(ctx, args)

	// Success should not be an error
	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	// ForLLM should contain file content
	if !strings.Contains(result.ForLLM, "test content") {
		t.Errorf("Expected ForLLM to contain 'test content', got: %s", result.ForLLM)
	}

	// ReadFile returns NewToolResult which only sets ForLLM, not ForUser
	// This is the expected behavior - file content goes to LLM, not directly to user
	if result.ForUser != "" {
		t.Errorf("Expected ForUser to be empty for NewToolResult, got: %s", result.ForUser)
	}
}

// TestFilesystemTool_ReadFile_NotFound verifies error handling for missing file
func TestFilesystemTool_ReadFile_NotFound(t *testing.T) {
	tool := NewReadFileTool("", false)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{
		err: fmt.Errorf("failed to read file: file not found"),
	})
	args := map[string]any{
		"path": "/nonexistent_file_12345.txt",
	}

	result := tool.Execute(ctx, args)

	// Failure should be marked as error
	if !result.IsError {
		t.Errorf("Expected error for missing file, got IsError=false")
	}

	// Should contain error message
	if !strings.Contains(result.ForLLM, "failed to read") && !strings.Contains(result.ForUser, "failed to read") {
		t.Errorf("Expected error message, got ForLLM: %s, ForUser: %s", result.ForLLM, result.ForUser)
	}
}

// TestFilesystemTool_ReadFile_MissingPath verifies error handling for missing path
func TestFilesystemTool_ReadFile_MissingPath(t *testing.T) {
	tool := &ReadFileTool{}
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{})
	args := map[string]any{}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error when path is missing")
	}

	// Should mention required parameter
	if !strings.Contains(result.ForLLM, "path is required") && !strings.Contains(result.ForUser, "path is required") {
		t.Errorf("Expected 'path is required' message, got ForLLM: %s", result.ForLLM)
	}
}

// TestFilesystemTool_WriteFile_Success verifies successful file writing
func TestFilesystemTool_WriteFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "newfile.txt")

	tool := NewWriteFileTool("", false)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{
		fs: sandbox.NewHostSandbox(tmpDir, false).Fs(),
	})
	args := map[string]any{
		"path":    testFile,
		"content": "hello world",
	}

	result := tool.Execute(ctx, args)

	// Success should not be an error
	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	// WriteFile returns SilentResult
	if !result.Silent {
		t.Errorf("Expected Silent=true for WriteFile, got false")
	}

	// ForUser should be empty (silent result)
	if result.ForUser != "" {
		t.Errorf("Expected ForUser to be empty for SilentResult, got: %s", result.ForUser)
	}

	// Verify file was actually written
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}
	if string(content) != "hello world" {
		t.Errorf("Expected file content 'hello world', got: %s", string(content))
	}
}

// TestFilesystemTool_WriteFile_CreateDir verifies directory creation
func TestFilesystemTool_WriteFile_CreateDir(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "subdir", "newfile.txt")

	tool := NewWriteFileTool("", false)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{
		fs: sandbox.NewHostSandbox(tmpDir, false).Fs(),
	})
	args := map[string]any{
		"path":    testFile,
		"content": "test",
	}

	result := tool.Execute(ctx, args)

	// Success should not be an error
	if result.IsError {
		t.Errorf("Expected success with directory creation, got IsError=true: %s", result.ForLLM)
	}

	// Verify directory was created and file written
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}
	if string(content) != "test" {
		t.Errorf("Expected file content 'test', got: %s", string(content))
	}
}

// TestFilesystemTool_WriteFile_MissingPath verifies error handling for missing path
func TestFilesystemTool_WriteFile_MissingPath(t *testing.T) {
	tool := NewWriteFileTool("", false)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{})
	args := map[string]any{
		"content": "test",
	}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error when path is missing")
	}
}

// TestFilesystemTool_WriteFile_MissingContent verifies error handling for missing content
func TestFilesystemTool_WriteFile_MissingContent(t *testing.T) {
	tool := NewWriteFileTool("", false)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{})
	args := map[string]any{
		"path": "/tmp/test.txt",
	}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error when content is missing")
	}

	// Should mention required parameter
	if !strings.Contains(result.ForLLM, "content is required") &&
		!strings.Contains(result.ForUser, "content is required") {
		t.Errorf("Expected 'content is required' message, got ForLLM: %s", result.ForLLM)
	}
}

// TestFilesystemTool_ListDir_Success verifies successful directory listing
func TestFilesystemTool_ListDir_Success(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content"), 0o644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0o755)

	tool := NewListDirTool(tmpDir, false)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{
		fs: sandbox.NewHostSandbox(tmpDir, false).Fs(),
	})
	args := map[string]any{
		"path": tmpDir,
	}

	result := tool.Execute(ctx, args)

	// Success should not be an error
	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	// Should list files and directories
	if !strings.Contains(result.ForLLM, "file1.txt") || !strings.Contains(result.ForLLM, "file2.txt") {
		t.Errorf("Expected files in listing, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "subdir") {
		t.Errorf("Expected subdir in listing, got: %s", result.ForLLM)
	}
}

// TestFilesystemTool_ListDir_NotFound verifies error handling for non-existent directory
func TestFilesystemTool_ListDir_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewListDirTool(tmpDir, false)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{
		fs:  sandbox.NewHostSandbox(tmpDir, false).Fs(),
		err: fmt.Errorf("failed to read directory: file not found"),
	})
	args := map[string]any{
		"path": "/nonexistent_directory_12345",
	}

	result := tool.Execute(ctx, args)

	// Failure should be marked as error
	if !result.IsError {
		t.Errorf("Expected error for non-existent directory, got IsError=false")
	}

	// Should contain error message
	if !strings.Contains(result.ForLLM, "failed to read") && !strings.Contains(result.ForUser, "failed to read") {
		t.Errorf("Expected error message, got ForLLM: %s, ForUser: %s", result.ForLLM, result.ForUser)
	}
}

// TestFilesystemTool_ListDir_DefaultPath verifies default to current directory
func TestFilesystemTool_ListDir_DefaultPath(t *testing.T) {
	tool := NewListDirTool(".", false)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{
		fs: sandbox.NewHostSandbox(".", false).Fs(),
	})
	args := map[string]any{}

	result := tool.Execute(ctx, args)

	// Should use "." as default path
	if result.IsError {
		t.Errorf("Expected success with default path '.', got IsError=true: %s", result.ForLLM)
	}
}

// Block paths that look inside workspace but point outside via symlink.
func TestFilesystemTool_ReadFile_RejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	secret := filepath.Join(root, "secret.txt")
	if err := os.WriteFile(secret, []byte("top secret"), 0o644); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	link := filepath.Join(workspace, "leak.txt")
	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}

	tool := NewReadFileTool(workspace, true)
	result := tool.Execute(sandbox.WithSandbox(context.Background(), &stubSandbox{
		fs: sandbox.NewHostSandbox(workspace, true).Fs(),
	}), map[string]any{
		"path": link,
	})

	if !result.IsError {
		t.Fatalf("expected symlink escape to be blocked")
	}
	// os.Root might return different errors depending on platform/implementation
	// but it definitely should error.
	// Our wrapper returns "access denied or file not found"
	if !strings.Contains(result.ForLLM, "access denied") && !strings.Contains(result.ForLLM, "file not found") &&
		!strings.Contains(result.ForLLM, "no such file") {
		t.Fatalf("expected symlink escape error, got: %s", result.ForLLM)
	}
}

func TestFilesystemTool_EmptyWorkspace_AccessDenied(t *testing.T) {
	tool := NewReadFileTool("", true) // restrict=true but workspace=""

	// Try to read a sensitive file (simulated by a temp file outside workspace)
	tmpDir := t.TempDir()
	secretFile := filepath.Join(tmpDir, "shadow")
	os.WriteFile(secretFile, []byte("secret data"), 0o600)

	result := tool.Execute(sandbox.WithSandbox(context.Background(), &stubSandbox{
		fs: sandbox.NewHostSandbox("", true).Fs(),
	}), map[string]any{
		"path": secretFile,
	})

	// We EXPECT IsError=true (access blocked due to empty workspace)
	assert.True(t, result.IsError, "Security Regression: Empty workspace allowed access! content: %s", result.ForLLM)

	// Verify it failed for the right reason
	assert.Contains(t, result.ForLLM, "workspace is not defined", "Expected 'workspace is not defined' error")
}

func TestFilesystemTool_EmptyWorkspace_UnrestrictedAllowed(t *testing.T) {
	tool := NewReadFileTool("", false) // restrict=false and workspace=""

	tmpDir := t.TempDir()
	secretFile := filepath.Join(tmpDir, "public.txt")
	if err := os.WriteFile(secretFile, []byte("public data"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result := tool.Execute(sandbox.WithSandbox(context.Background(), &stubSandbox{
		fs: sandbox.NewHostSandbox("", false).Fs(),
	}), map[string]any{
		"path": secretFile,
	})

	assert.False(t, result.IsError, "Expected unrestricted empty-workspace read to succeed, got: %s", result.ForLLM)
	assert.Contains(t, result.ForLLM, "public data")
}

// TestRootMkdirAll verifies that root.MkdirAll (used by atomicWriteFileInRoot) handles all cases:
// single dir, deeply nested dirs, already-existing dirs, and a file blocking a directory path.
func TestRootMkdirAll(t *testing.T) {
	workspace := t.TempDir()
	root, err := os.OpenRoot(workspace)
	if err != nil {
		t.Fatalf("failed to open root: %v", err)
	}
	defer root.Close()

	// Case 1: Single directory
	err = root.MkdirAll("dir1", 0o755)
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(workspace, "dir1"))
	assert.NoError(t, err)

	// Case 2: Deeply nested directory
	err = root.MkdirAll("a/b/c/d", 0o755)
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(workspace, "a/b/c/d"))
	assert.NoError(t, err)

	// Case 3: Already exists — must be idempotent
	err = root.MkdirAll("a/b/c/d", 0o755)
	assert.NoError(t, err)

	// Case 4: A regular file blocks directory creation — must error
	err = os.WriteFile(filepath.Join(workspace, "file_exists"), []byte("data"), 0o644)
	assert.NoError(t, err)
	err = root.MkdirAll("file_exists", 0o755)
	assert.Error(t, err, "expected error when a file exists at the directory path")
}

func TestFilesystemTool_WriteFile_Restricted_CreateDir(t *testing.T) {
	workspace := t.TempDir()
	tool := NewWriteFileTool(workspace, true)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{
		fs: sandbox.NewHostSandbox(workspace, true).Fs(),
	})

	testFile := "deep/nested/path/to/file.txt"
	content := "deep content"
	args := map[string]any{
		"path":    testFile,
		"content": content,
	}

	result := tool.Execute(ctx, args)
	assert.False(t, result.IsError, "Expected success, got: %s", result.ForLLM)

	// Verify file content
	actualPath := filepath.Join(workspace, testFile)
	data, err := os.ReadFile(actualPath)
	assert.NoError(t, err)
	assert.Equal(t, content, string(data))
}
