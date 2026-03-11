package tools

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sipeed/picoclaw/pkg/agent/sandbox"
)

func hostSandboxCtx(workspace string, restrict bool) context.Context {
	return sandbox.WithSandbox(context.Background(), sandbox.NewHostSandbox(workspace, restrict))
}

// TestFilesystemTool_ReadFile_Success verifies successful file reading
func TestFilesystemTool_ReadFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0o644)

	tool := NewReadFileTool("", false, MaxReadFileSize)
	ctx := hostSandboxCtx("", false)
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
	tool := NewReadFileTool("", false, MaxReadFileSize)
	ctx := hostSandboxCtx("", false)
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
	ctx := hostSandboxCtx("", false)
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
	ctx := hostSandboxCtx("", false)
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
	ctx := hostSandboxCtx("", false)
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
	ctx := hostSandboxCtx("", false)
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
	ctx := hostSandboxCtx("", false)
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
	ctx := hostSandboxCtx("", false)
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
	tool := NewListDirTool("", false)
	ctx := hostSandboxCtx("", false)
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
	tool := NewListDirTool("", false)
	ctx := hostSandboxCtx("", false)
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

	tool := NewReadFileTool(workspace, true, MaxReadFileSize)
	result := tool.Execute(hostSandboxCtx(workspace, true), map[string]any{
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
	tool := NewReadFileTool("", true, MaxReadFileSize) // restrict=true but workspace=""

	// Try to read a sensitive file (simulated by a temp file outside workspace)
	tmpDir := t.TempDir()
	secretFile := filepath.Join(tmpDir, "shadow")
	os.WriteFile(secretFile, []byte("secret data"), 0o600)

	result := tool.Execute(hostSandboxCtx("", true), map[string]any{
		"path": secretFile,
	})

	// We EXPECT IsError=true (access blocked due to empty workspace)
	assert.True(t, result.IsError, "Security Regression: Empty workspace allowed access! content: %s", result.ForLLM)

	// Verify it failed for the right reason
	assert.Contains(t, result.ForLLM, "workspace is not defined", "Expected 'workspace is not defined' error")
}

func TestFilesystemTool_WriteFile_Restricted_CreateDir(t *testing.T) {
	workspace := t.TempDir()
	tool := NewWriteFileTool(workspace, true)
	ctx := hostSandboxCtx(workspace, true)

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

// TestWhitelistFs_AllowsMatchingPaths verifies that whitelistFs allows access to
// paths matching the whitelist patterns while blocking non-matching paths.
func TestWhitelistFs_AllowsMatchingPaths(t *testing.T) {
	workspace := t.TempDir()
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "allowed.txt")
	os.WriteFile(outsideFile, []byte("outside content"), 0o644)

	// Pattern allows access to the outsideDir.
	patterns := []*regexp.Regexp{regexp.MustCompile(`^` + regexp.QuoteMeta(outsideDir))}

	tool := NewReadFileTool(workspace, true, MaxReadFileSize, patterns)

	// Read from whitelisted path should succeed.
	result := tool.Execute(hostSandboxCtx(workspace, true), map[string]any{"path": outsideFile})
	if result.IsError {
		t.Errorf("expected whitelisted path to be readable, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "outside content") {
		t.Errorf("expected file content, got: %s", result.ForLLM)
	}

	// Read from non-whitelisted path outside workspace should fail.
	otherDir := t.TempDir()
	otherFile := filepath.Join(otherDir, "blocked.txt")
	os.WriteFile(otherFile, []byte("blocked"), 0o644)

	result = tool.Execute(hostSandboxCtx(workspace, true), map[string]any{"path": otherFile})
	if !result.IsError {
		t.Errorf("expected non-whitelisted path to be blocked, got: %s", result.ForLLM)
	}
}

// TestReadFileTool_ChunkedReading verifies the pagination logic of the tool
// by reading a file in multiple chunks using 'offset' and 'length'.
func TestReadFileTool_ChunkedReading(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "pagination_test.txt")

	// Create a test file with exactly 26 bytes of content
	fullContent := "abcdefghijklmnopqrstuvwxyz"
	err := os.WriteFile(testFile, []byte(fullContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tool := NewReadFileTool(tmpDir, false, MaxReadFileSize)
	ctx := hostSandboxCtx("", false)

	// --- Step 1: Read the first chunk (10 bytes) ---
	args1 := map[string]any{
		"path":   testFile,
		"offset": 0,
		"length": 10,
	}
	result1 := tool.Execute(ctx, args1)

	if result1.IsError {
		t.Fatalf("Chunk 1 failed: %s", result1.ForLLM)
	}

	// Expect the first 10 characters
	if !strings.Contains(result1.ForLLM, "abcdefghij") {
		t.Errorf("Chunk 1 should contain 'abcdefghij', got: %s", result1.ForLLM)
	}
	// Expect the header to indicate the file is truncated
	if !strings.Contains(result1.ForLLM, "[TRUNCATED") {
		t.Errorf("Chunk 1 header should indicate truncation, got: %s", result1.ForLLM)
	}
	// Expect the header to suggest the next offset (10)
	if !strings.Contains(result1.ForLLM, "offset=10") {
		t.Errorf("Chunk 1 header should suggest next offset=10, got: %s", result1.ForLLM)
	}

	// Step 2: Read the second chunk (10 bytes) ---
	args2 := map[string]any{
		"path":   testFile,
		"offset": 10,
		"length": 10,
	}
	result2 := tool.Execute(ctx, args2)

	if result2.IsError {
		t.Fatalf("Chunk 2 failed: %s", result2.ForLLM)
	}

	// Expect the next 10 characters
	if !strings.Contains(result2.ForLLM, "klmnopqrst") {
		t.Errorf("Chunk 2 should contain 'klmnopqrst', got: %s", result2.ForLLM)
	}
	// Expect the header to suggest the next offset (20)
	if !strings.Contains(result2.ForLLM, "offset=20") {
		t.Errorf("Chunk 2 header should suggest next offset=20, got: %s", result2.ForLLM)
	}

	// Step 3: Read the final chunk (remaining 6 bytes) ---
	// We ask for 10 bytes, but only 6 are left in the file
	args3 := map[string]any{
		"path":   testFile,
		"offset": 20,
		"length": 10,
	}
	result3 := tool.Execute(ctx, args3)

	if result3.IsError {
		t.Fatalf("Chunk 3 failed: %s", result3.ForLLM)
	}

	// Expect the last 6 characters
	if !strings.Contains(result3.ForLLM, "uvwxyz") {
		t.Errorf("Chunk 3 should contain 'uvwxyz', got: %s", result3.ForLLM)
	}
	// Expect the header to indicate the end of the file
	if !strings.Contains(result3.ForLLM, "[END OF FILE") {
		t.Errorf("Chunk 3 header should indicate end of file, got: %s", result3.ForLLM)
	}

	// Ensure no TRUNCATED message is present in the final chunk
	if strings.Contains(result3.ForLLM, "[TRUNCATED") {
		t.Errorf("Chunk 3 header should NOT indicate truncation, got: %s", result3.ForLLM)
	}
}

// TestReadFileTool_OffsetBeyondEOF checks the behavior when requesting
// An offset that exceeds the total file size.
func TestReadFileTool_OffsetBeyondEOF(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "short.txt")

	// create a file of only 5 bytes
	err := os.WriteFile(testFile, []byte("12345"), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tool := NewReadFileTool(tmpDir, false, MaxReadFileSize)
	ctx := hostSandboxCtx("", false)

	args := map[string]any{
		"path":   testFile,
		"offset": int64(100), // Offset beyond the end of the file
	}

	result := tool.Execute(ctx, args)

	// It should not be classified as a tool execution error
	if result.IsError {
		t.Errorf("A mistake was not expected, obtained IsError=true: %s", result.ForLLM)
	}

	// Must return EXACTLY the string provided in the code
	expectedMsg := "[END OF FILE - no content at this offset]"
	if result.ForLLM != expectedMsg {
		t.Errorf("The message %q was expected, obtained: %q", expectedMsg, result.ForLLM)
	}
}
