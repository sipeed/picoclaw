package append_file

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAppendFileTool_AppendToExisting verifies appending to an existing file
func TestAppendFileTool_AppendToExisting(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("Hello World"), 0o644)

	tool := NewAppendFileTool(tmpDir, true)
	ctx := context.Background()
	args := map[string]any{
		"path":    testFile,
		"content": "\nAppended text",
	}

	result := tool.Execute(ctx, args)

	assert.False(t, result.IsError, "Expected success, got error: %s", result.ForLLM)
	assert.True(t, result.Silent, "Expected Silent=true for AppendFile")

	content, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "Appended text")
	assert.Contains(t, string(content), "Hello World")
}

// TestAppendFileTool_AppendToNonExistent verifies appending to a non-existent file creates it
func TestAppendFileTool_AppendToNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "newfile.txt")

	tool := NewAppendFileTool(tmpDir, true)
	ctx := context.Background()
	args := map[string]any{
		"path":    testFile,
		"content": "First content",
	}

	result := tool.Execute(ctx, args)

	assert.False(t, result.IsError, "Expected success, got error: %s", result.ForLLM)

	content, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, "First content", string(content))
}

// TestAppendFileTool_MissingPath verifies error handling for missing path
func TestAppendFileTool_MissingPath(t *testing.T) {
	tool := NewAppendFileTool("", false)
	ctx := context.Background()
	args := map[string]any{
		"content": "Some content",
	}

	result := tool.Execute(ctx, args)

	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "path is required")
}

// TestAppendFileTool_MissingContent verifies error handling for missing content
func TestAppendFileTool_MissingContent(t *testing.T) {
	tool := NewAppendFileTool("", false)
	ctx := context.Background()
	args := map[string]any{
		"path": "/tmp/test.txt",
	}

	result := tool.Execute(ctx, args)

	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "content is required")
}

// TestAppendFileTool_RestrictedMode verifies access control
func TestAppendFileTool_RestrictedMode(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("Original"), 0o644)

	tool := NewAppendFileTool(tmpDir, true)
	ctx := context.Background()

	// Try to append to a file outside the workspace
	args := map[string]any{
		"path":    "/etc/passwd",
		"content": "Malicious content",
	}

	result := tool.Execute(ctx, args)

	assert.True(t, result.IsError)
}
