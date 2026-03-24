package tools

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFilesystemTool_ReadFile_Success verifies successful file reading
func TestFilesystemTool_ReadFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0o644)

	tool := NewReadFileTool("", false, MaxReadFileSize)
	ctx := context.Background()
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
	ctx := context.Background()
	args := map[string]any{
		"path": "/nonexistent_file_12345.txt",
	}

	result := tool.Execute(ctx, args)

	// Failure should be marked as error
	if !result.IsError {
		t.Errorf("Expected error for missing file, got IsError=false")
	}

	// Should contain error message
	if !strings.Contains(result.ForLLM, "failed to open file") && !strings.Contains(result.ForUser, "failed to read") {
		t.Errorf("Expected error message, got ForLLM: %s, ForUser: %s", result.ForLLM, result.ForUser)
	}
}

// TestFilesystemTool_ReadFile_MissingPath verifies error handling for missing path
func TestFilesystemTool_ReadFile_MissingPath(t *testing.T) {
	tool := &ReadFileTool{}
	ctx := context.Background()
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
	ctx := context.Background()
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
	ctx := context.Background()
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
	ctx := context.Background()
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
	ctx := context.Background()
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

// TestFilesystemTool_WriteFile_OverwriteDefaultBlocked verifies that writing to an
// existing file without overwrite=true returns an error.
func TestFilesystemTool_WriteFile_OverwriteDefaultBlocked(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "existing.txt")
	os.WriteFile(testFile, []byte("original"), 0o644)

	tool := NewWriteFileTool("", false)
	result := tool.Execute(context.Background(), map[string]any{
		"path":    testFile,
		"content": "new content",
	})

	assert.True(t, result.IsError, "expected error when overwriting without overwrite=true")
	assert.Contains(t, result.ForLLM, "already exists")
	assert.Contains(t, result.ForLLM, "overwrite=true")

	// Original content must be untouched
	data, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, "original", string(data))
}

// TestFilesystemTool_WriteFile_OverwriteExplicitAllowed verifies that setting
// overwrite=true replaces the existing file.
func TestFilesystemTool_WriteFile_OverwriteExplicitAllowed(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "existing.txt")
	os.WriteFile(testFile, []byte("original"), 0o644)

	tool := NewWriteFileTool("", false)
	result := tool.Execute(context.Background(), map[string]any{
		"path":      testFile,
		"content":   "replaced",
		"overwrite": true,
	})

	assert.False(t, result.IsError, "expected success with overwrite=true, got: %s", result.ForLLM)

	data, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, "replaced", string(data))
}

// TestFilesystemTool_WriteFile_NewFileNoOverwriteFlag verifies that a new (non-existing)
// file can be written without setting overwrite=true.
func TestFilesystemTool_WriteFile_NewFileNoOverwriteFlag(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "newfile.txt")

	tool := NewWriteFileTool("", false)
	result := tool.Execute(context.Background(), map[string]any{
		"path":    testFile,
		"content": "brand new",
	})

	assert.False(t, result.IsError, "expected success for new file, got: %s", result.ForLLM)

	data, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, "brand new", string(data))
}

// TestFilesystemTool_WriteFile_OverwriteFalseExplicitBlocked verifies that
// explicitly passing overwrite=false also blocks overwriting.
func TestFilesystemTool_WriteFile_OverwriteFalseExplicitBlocked(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "existing.txt")
	os.WriteFile(testFile, []byte("original"), 0o644)

	tool := NewWriteFileTool("", false)
	result := tool.Execute(context.Background(), map[string]any{
		"path":      testFile,
		"content":   "new content",
		"overwrite": false,
	})

	assert.True(t, result.IsError, "expected error when overwrite=false")
	assert.Contains(t, result.ForLLM, "already exists")

	data, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, "original", string(data))
}

// TestFilesystemTool_WriteFile_OverwriteSandboxed verifies the overwrite guard
// works correctly in restricted (sandbox) mode.
func TestFilesystemTool_WriteFile_OverwriteSandboxed(t *testing.T) {
	workspace := t.TempDir()
	testFile := "file.txt"
	os.WriteFile(filepath.Join(workspace, testFile), []byte("original"), 0o644)

	tool := NewWriteFileTool(workspace, true)

	// Without overwrite=true → blocked
	result := tool.Execute(context.Background(), map[string]any{
		"path":    testFile,
		"content": "new content",
	})
	assert.True(t, result.IsError, "expected error in sandbox mode without overwrite=true")
	assert.Contains(t, result.ForLLM, "already exists")

	// With overwrite=true → allowed
	result = tool.Execute(context.Background(), map[string]any{
		"path":      testFile,
		"content":   "replaced in sandbox",
		"overwrite": true,
	})
	assert.False(t, result.IsError, "expected success in sandbox mode with overwrite=true, got: %s", result.ForLLM)

	data, err := os.ReadFile(filepath.Join(workspace, testFile))
	assert.NoError(t, err)
	assert.Equal(t, "replaced in sandbox", string(data))
}

// TestFilesystemTool_ListDir_Success verifies successful directory listing
func TestFilesystemTool_ListDir_Success(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content"), 0o644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0o755)

	tool := NewListDirTool("", false)
	ctx := context.Background()
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
	ctx := context.Background()
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
	ctx := context.Background()
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
	result := tool.Execute(context.Background(), map[string]any{
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

	result := tool.Execute(context.Background(), map[string]any{
		"path": secretFile,
	})

	// We EXPECT IsError=true (access blocked due to empty workspace)
	assert.True(t, result.IsError, "Security Regression: Empty workspace allowed access! content: %s", result.ForLLM)

	// Verify it failed for the right reason
	assert.Contains(t, result.ForLLM, "workspace is not defined", "Expected 'workspace is not defined' error")
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
	ctx := context.Background()

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

// TestHostRW_Read_PermissionDenied verifies that hostRW.Read surfaces access denied errors.
func TestHostRW_Read_PermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test: running as root")
	}
	tmpDir := t.TempDir()
	protected := filepath.Join(tmpDir, "protected.txt")
	err := os.WriteFile(protected, []byte("secret"), 0o000)
	assert.NoError(t, err)
	defer os.Chmod(protected, 0o644) // ensure cleanup

	_, err = (&hostFs{}).ReadFile(protected)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

// TestHostRW_Read_Directory verifies that hostRW.Read returns an error when given a directory path.
func TestHostRW_Read_Directory(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := (&hostFs{}).ReadFile(tmpDir)
	assert.Error(t, err, "expected error when reading a directory as a file")
}

// TestRootRW_Read_Directory verifies that rootRW.Read returns an error when given a directory.
func TestRootRW_Read_Directory(t *testing.T) {
	workspace := t.TempDir()
	root, err := os.OpenRoot(workspace)
	assert.NoError(t, err)
	defer root.Close()

	// Create a subdirectory
	err = root.Mkdir("subdir", 0o755)
	assert.NoError(t, err)

	_, err = (&sandboxFs{workspace: workspace}).ReadFile("subdir")
	assert.Error(t, err, "expected error when reading a directory as a file")
}

// TestHostRW_Write_ParentDirMissing verifies that hostRW.Write creates parent dirs automatically.
func TestHostRW_Write_ParentDirMissing(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "a", "b", "c", "file.txt")

	err := (&hostFs{}).WriteFile(target, []byte("hello"))
	assert.NoError(t, err)

	data, err := os.ReadFile(target)
	assert.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}

// TestRootRW_Write_ParentDirMissing verifies that rootRW.Write creates
// nested parent directories automatically within the sandbox.
func TestRootRW_Write_ParentDirMissing(t *testing.T) {
	workspace := t.TempDir()

	relPath := "x/y/z/file.txt"
	err := (&sandboxFs{workspace: workspace}).WriteFile(relPath, []byte("nested"))
	assert.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(workspace, relPath))
	assert.NoError(t, err)
	assert.Equal(t, "nested", string(data))
}

// TestHostRW_Write verifies the hostRW.Write helper function
func TestHostRW_Write(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "atomic_test.txt")
	testData := []byte("atomic test content")

	err := (&hostFs{}).WriteFile(testFile, testData)
	assert.NoError(t, err)

	content, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, testData, content)

	// Verify it overwrites correctly
	newData := []byte("new atomic content")
	err = (&hostFs{}).WriteFile(testFile, newData)
	assert.NoError(t, err)

	content, err = os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, newData, content)
}

// TestRootRW_Write verifies the rootRW.Write helper function
func TestRootRW_Write(t *testing.T) {
	tmpDir := t.TempDir()

	relPath := "atomic_root_test.txt"
	testData := []byte("atomic root test content")

	erw := &sandboxFs{workspace: tmpDir}
	err := erw.WriteFile(relPath, testData)
	assert.NoError(t, err)

	root, err := os.OpenRoot(tmpDir)
	assert.NoError(t, err)
	defer root.Close()

	f, err := root.Open(relPath)
	assert.NoError(t, err)
	defer f.Close()

	content, err := io.ReadAll(f)
	assert.NoError(t, err)
	assert.Equal(t, testData, content)

	// Verify it overwrites correctly
	newData := []byte("new root atomic content")
	err = erw.WriteFile(relPath, newData)
	assert.NoError(t, err)

	f2, err := root.Open(relPath)
	assert.NoError(t, err)
	defer f2.Close()

	content, err = io.ReadAll(f2)
	assert.NoError(t, err)
	assert.Equal(t, newData, content)
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
	result := tool.Execute(context.Background(), map[string]any{"path": outsideFile})
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

	result = tool.Execute(context.Background(), map[string]any{"path": otherFile})
	if !result.IsError {
		t.Errorf("expected non-whitelisted path to be blocked, got: %s", result.ForLLM)
	}
}

func TestWhitelistFs_BlocksSymlinkEscapeInAllowedDir(t *testing.T) {
	workspace := t.TempDir()
	allowedDir := t.TempDir()
	secretDir := t.TempDir()
	secretFile := filepath.Join(secretDir, "secret.txt")
	if err := os.WriteFile(secretFile, []byte("top secret"), 0o644); err != nil {
		t.Fatalf("WriteFile(secretFile) error = %v", err)
	}

	linkPath := filepath.Join(allowedDir, "link_out")
	if err := os.Symlink(secretDir, linkPath); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}

	patterns := []*regexp.Regexp{regexp.MustCompile(`^` + regexp.QuoteMeta(allowedDir))}
	tool := NewReadFileTool(workspace, true, MaxReadFileSize, patterns)

	result := tool.Execute(context.Background(), map[string]any{"path": filepath.Join(linkPath, "secret.txt")})
	if !result.IsError {
		t.Fatalf("expected symlink escape from allowed dir to be blocked, got: %s", result.ForLLM)
	}
}

func TestWhitelistFs_WriteAllowsNewFileUnderAllowedDir(t *testing.T) {
	workspace := t.TempDir()
	rootDir := t.TempDir()
	allowedDir := filepath.Join(rootDir, "allowed")
	targetFile := filepath.Join(allowedDir, "nested", "file.txt")

	patterns := []*regexp.Regexp{regexp.MustCompile(`^` + regexp.QuoteMeta(allowedDir))}
	tool := NewWriteFileTool(workspace, true, patterns)

	result := tool.Execute(context.Background(), map[string]any{
		"path":    targetFile,
		"content": "outside write",
	})
	if result.IsError {
		t.Fatalf("expected whitelisted write to succeed, got: %s", result.ForLLM)
	}

	data, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("ReadFile(targetFile) error = %v", err)
	}
	if string(data) != "outside write" {
		t.Fatalf("target file content = %q, want %q", string(data), "outside write")
	}
}

func TestWhitelistFs_AllowsResolvedAllowedRootAlias(t *testing.T) {
	workspace := t.TempDir()
	realDir := t.TempDir()
	linkParent := t.TempDir()
	allowedAlias := filepath.Join(linkParent, "allowed-link")

	if err := os.Symlink(realDir, allowedAlias); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}

	targetFile := filepath.Join(allowedAlias, "nested", "alias.txt")
	if err := os.MkdirAll(filepath.Dir(targetFile), 0o755); err != nil {
		t.Fatalf("MkdirAll(targetFile dir) error = %v", err)
	}
	if err := os.WriteFile(targetFile, []byte("through alias"), 0o644); err != nil {
		t.Fatalf("WriteFile(targetFile) error = %v", err)
	}

	patterns := []*regexp.Regexp{
		regexp.MustCompile(
			"^" + regexp.QuoteMeta(filepath.Clean(allowedAlias)) +
				"(?:" + regexp.QuoteMeta(string(os.PathSeparator)) + "|$)",
		),
	}
	tool := NewReadFileTool(workspace, true, MaxReadFileSize, patterns)

	result := tool.Execute(context.Background(), map[string]any{"path": targetFile})
	if result.IsError {
		t.Fatalf("expected symlink-backed allowed root to be readable, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "through alias") {
		t.Fatalf("expected file content, got: %s", result.ForLLM)
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
	ctx := context.Background()

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
	ctx := context.Background()

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
