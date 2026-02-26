package list_dir

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
