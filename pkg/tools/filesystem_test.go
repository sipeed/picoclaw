package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/security"
)

// TestFilesystemTool_ReadFile_Success verifies successful file reading
func TestFilesystemTool_ReadFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	tool := &ReadFileTool{}
	ctx := context.Background()
	args := map[string]interface{}{
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
	tool := &ReadFileTool{}
	ctx := context.Background()
	args := map[string]interface{}{
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
	ctx := context.Background()
	args := map[string]interface{}{}

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

	tool := &WriteFileTool{}
	ctx := context.Background()
	args := map[string]interface{}{
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

	tool := &WriteFileTool{}
	ctx := context.Background()
	args := map[string]interface{}{
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
	tool := &WriteFileTool{}
	ctx := context.Background()
	args := map[string]interface{}{
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
	tool := &WriteFileTool{}
	ctx := context.Background()
	args := map[string]interface{}{
		"path": "/tmp/test.txt",
	}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error when content is missing")
	}

	// Should mention required parameter
	if !strings.Contains(result.ForLLM, "content is required") && !strings.Contains(result.ForUser, "content is required") {
		t.Errorf("Expected 'content is required' message, got ForLLM: %s", result.ForLLM)
	}
}

// TestFilesystemTool_ListDir_Success verifies successful directory listing
func TestFilesystemTool_ListDir_Success(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	tool := &ListDirTool{}
	ctx := context.Background()
	args := map[string]interface{}{
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
	tool := &ListDirTool{}
	ctx := context.Background()
	args := map[string]interface{}{
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
	tool := &ListDirTool{}
	ctx := context.Background()
	args := map[string]interface{}{}

	result := tool.Execute(ctx, args)

	// Should use "." as default path
	if result.IsError {
		t.Errorf("Expected success with default path '.', got IsError=true: %s", result.ForLLM)
	}
}

// TestValidatePath_SymlinkEscape verifies that symlinks pointing outside workspace are blocked
// when path validation mode is "block" (enhanced symlink resolution).
func TestValidatePath_SymlinkEscape(t *testing.T) {
	workspace := t.TempDir()
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "secret.txt")
	os.WriteFile(outsideFile, []byte("secret"), 0644)

	symlinkPath := filepath.Join(workspace, "escape")
	if err := os.Symlink(outsideDir, symlinkPath); err != nil {
		t.Skipf("Cannot create symlink: %v", err)
	}

	_, err := validatePathWithMode("escape/secret.txt", workspace, true, security.ModeBlock, nil, "", "")
	if err == nil {
		t.Error("Expected symlink escape to be blocked, but it was allowed")
	}
}

func TestValidatePath_PrefixCollision(t *testing.T) {
	baseDir := t.TempDir()
	workspace := filepath.Join(baseDir, "workspace")
	otherDir := filepath.Join(baseDir, "workspace2")
	os.MkdirAll(workspace, 0755)
	os.MkdirAll(otherDir, 0755)

	_, err := validatePath(otherDir, workspace, true)
	if err == nil {
		t.Error("Expected prefix collision path to be blocked, but it was allowed")
	}
}

func TestValidatePath_AllowsWorkspaceItself(t *testing.T) {
	workspace := t.TempDir()

	path, err := validatePathWithMode(".", workspace, true, security.ModeBlock, nil, "", "")
	if err != nil {
		t.Errorf("Expected workspace root access to be allowed, got error: %v", err)
	}

	expectedPath, _ := filepath.EvalSymlinks(workspace)
	if path != expectedPath {
		t.Errorf("Expected path to be %s, got %s", expectedPath, path)
	}
}

func TestValidatePath_EmptyWorkspace(t *testing.T) {
	path, err := validatePath("/some/path", "", true)
	if err != nil {
		t.Errorf("Expected no error with empty workspace, got: %v", err)
	}
	if path != "/some/path" {
		t.Errorf("Expected path returned as-is, got: %s", path)
	}
}

func TestValidatePath_ModeOff_NoSymlinkResolution(t *testing.T) {
	workspace := t.TempDir()
	testFile := filepath.Join(workspace, "file.txt")
	os.WriteFile(testFile, []byte("data"), 0644)

	path, err := validatePathWithMode("file.txt", workspace, true, security.ModeOff, nil, "", "")
	if err != nil {
		t.Errorf("Expected success, got: %v", err)
	}
	if path == "" {
		t.Error("Expected non-empty path")
	}
}

func TestReadFileTool_SetContext(t *testing.T) {
	tool := NewReadFileToolWithPolicy("", false, PathPolicyOpts{})
	tool.SetContext("telegram", "chat-1")
	if tool.channel != "telegram" || tool.chatID != "chat-1" {
		t.Errorf("SetContext failed: channel=%q, chatID=%q", tool.channel, tool.chatID)
	}
}

func TestWriteFileTool_SetContext(t *testing.T) {
	tool := NewWriteFileToolWithPolicy("", false, PathPolicyOpts{})
	tool.SetContext("feishu", "chat-2")
	if tool.channel != "feishu" || tool.chatID != "chat-2" {
		t.Errorf("SetContext failed: channel=%q, chatID=%q", tool.channel, tool.chatID)
	}
}

func TestListDirTool_SetContext(t *testing.T) {
	tool := NewListDirToolWithPolicy("", false, PathPolicyOpts{})
	tool.SetContext("dingtalk", "chat-3")
	if tool.channel != "dingtalk" || tool.chatID != "chat-3" {
		t.Errorf("SetContext failed: channel=%q, chatID=%q", tool.channel, tool.chatID)
	}
}

func TestNewReadFileToolWithPolicy(t *testing.T) {
	opts := PathPolicyOpts{PathMode: security.ModeBlock}
	tool := NewReadFileToolWithPolicy("/workspace", true, opts)
	if tool.workspace != "/workspace" || !tool.restrict || tool.pathMode != security.ModeBlock {
		t.Error("WithPolicy constructor did not set fields correctly")
	}
}

func TestNewWriteFileToolWithPolicy(t *testing.T) {
	opts := PathPolicyOpts{PathMode: security.ModeApprove}
	tool := NewWriteFileToolWithPolicy("/ws", true, opts)
	if tool.workspace != "/ws" || !tool.restrict || tool.pathMode != security.ModeApprove {
		t.Error("WithPolicy constructor did not set fields correctly")
	}
}

func TestNewListDirToolWithPolicy(t *testing.T) {
	opts := PathPolicyOpts{PathMode: security.ModeBlock}
	tool := NewListDirToolWithPolicy("/ws", false, opts)
	if tool.workspace != "/ws" || tool.restrict || tool.pathMode != security.ModeBlock {
		t.Error("WithPolicy constructor did not set fields correctly")
	}
}

func TestFilesystemTool_ReadFile_RejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	secret := filepath.Join(root, "secret.txt")
	if err := os.WriteFile(secret, []byte("top secret"), 0644); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	link := filepath.Join(workspace, "leak.txt")
	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}

	tool := NewReadFileToolWithPolicy(workspace, true, PathPolicyOpts{PathMode: security.ModeBlock})
	relPath, _ := filepath.Rel(workspace, link)
	result := tool.Execute(context.Background(), map[string]interface{}{
		"path": relPath,
	})

	if !result.IsError {
		t.Fatalf("expected symlink escape to be blocked")
	}
	if !strings.Contains(result.ForLLM, "symlink resolves outside workspace") {
		t.Fatalf("expected symlink escape error, got: %s", result.ForLLM)
	}
}
