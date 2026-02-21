package tools

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

// EditFileTool edits a file by replacing old_text with new_text.
// The old_text must exist exactly in the file.
type EditFileTool struct {
	allowedDir string
	restrict   bool
}

// NewEditFileTool creates a new EditFileTool with optional directory restriction.
func NewEditFileTool(allowedDir string, restrict bool) *EditFileTool {
	return &EditFileTool{
		allowedDir: allowedDir,
		restrict:   restrict,
	}
}

func (t *EditFileTool) Name() string {
	return "edit_file"
}

func (t *EditFileTool) Description() string {
	return "Edit a file by replacing old_text with new_text. The old_text must exist exactly in the file."
}

func (t *EditFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The file path to edit",
			},
			"old_text": map[string]any{
				"type":        "string",
				"description": "The exact text to find and replace",
			},
			"new_text": map[string]any{
				"type":        "string",
				"description": "The text to replace with",
			},
		},
		"required": []string{"path", "old_text", "new_text"},
	}
}

func (t *EditFileTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		return ErrorResult("path is required")
	}

	oldText, ok := args["old_text"].(string)
	if !ok {
		return ErrorResult("old_text is required")
	}

	newText, ok := args["new_text"].(string)
	if !ok {
		return ErrorResult("new_text is required")
	}

	if t.restrict {
		return executeInRoot(t.allowedDir, path, func(root *os.Root, relPath string) (*ToolResult, error) {
			if err := editFileInRoot(root, relPath, oldText, newText); err != nil {
				return nil, err
			}
			return SilentResult(fmt.Sprintf("File edited: %s", path)), nil
		})
	}

	if err := editFile(&hostRW{}, path, oldText, newText); err != nil {
		return ErrorResult(err.Error())
	}
	return SilentResult(fmt.Sprintf("File edited: %s", path))
}

type AppendFileTool struct {
	workspace string
	restrict  bool
}

func NewAppendFileTool(workspace string, restrict bool) *AppendFileTool {
	return &AppendFileTool{workspace: workspace, restrict: restrict}
}

func (t *AppendFileTool) Name() string {
	return "append_file"
}

func (t *AppendFileTool) Description() string {
	return "Append content to the end of a file"
}

func (t *AppendFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The file path to append to",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to append",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *AppendFileTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		return ErrorResult("path is required")
	}

	appendContent, ok := args["content"].(string)
	if !ok {
		return ErrorResult("content is required")
	}

	var rw fileReadWriter
	if t.restrict {
		return executeInRoot(t.workspace, path, func(root *os.Root, relPath string) (*ToolResult, error) {
			if err := appendFileWithRW(&rootRW{root: root}, relPath, appendContent); err != nil {
				return nil, err
			}
			return SilentResult(fmt.Sprintf("Appended to %s", path)), nil
		})
	}

	rw = &hostRW{}
	if err := appendFileWithRW(rw, path, appendContent); err != nil {
		return ErrorResult(err.Error())
	}
	return SilentResult(fmt.Sprintf("Appended to %s", path))
}

// editFile reads the file via rw, performs the replacement, and writes back.
// It uses a fileReadWriter, allowing the same logic for both restricted and unrestricted modes.
func editFile(rw fileReadWriter, path, oldText, newText string) error {
	content, err := rw.Read(path)
	if err != nil {
		return err
	}

	newContent, err := replaceEditContent(content, oldText, newText)
	if err != nil {
		return err
	}

	return rw.Write(path, newContent)
}

// editFileInRoot performs an in-place edit within an os.Root using a single open call.
// By opening with O_RDWR and reusing the same file descriptor for both read and write,
// we narrow the TOCTOU window compared to two separate open calls.
func editFileInRoot(root *os.Root, relPath, oldText, newText string) error {
	f, err := root.OpenFile(relPath, os.O_RDWR, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("failed to read file: file not found: %w", err)
		}
		if os.IsPermission(err) || strings.Contains(err.Error(), "escapes from parent") {
			return fmt.Errorf("failed to read file: access denied: %w", err)
		}
		return fmt.Errorf("failed to open file for editing: %w", err)
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("failed to read file content: %w", err)
	}

	newContent, err := replaceEditContent(content, oldText, newText)
	if err != nil {
		return err
	}

	// Truncate the file and seek back to the beginning before writing.
	if err := f.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate file for in-place edit: %w", err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to beginning of file: %w", err)
	}

	if _, err := f.Write(newContent); err != nil {
		return fmt.Errorf("failed to write edited content: %w", err)
	}
	return nil
}

// appendFileWithRW reads the existing content (if any) via rw, appends new content, and writes back.
func appendFileWithRW(rw fileReadWriter, path, appendContent string) error {
	content, err := rw.Read(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	newContent := append(content, []byte(appendContent)...)
	return rw.Write(path, newContent)
}

// replaceEditContent handles the core logic of finding and replacing a single occurrence of oldText.
func replaceEditContent(content []byte, oldText, newText string) ([]byte, error) {
	contentStr := string(content)

	if !strings.Contains(contentStr, oldText) {
		return nil, fmt.Errorf("old_text not found in file. Make sure it matches exactly")
	}

	count := strings.Count(contentStr, oldText)
	if count > 1 {
		return nil, fmt.Errorf("old_text appears %d times. Please provide more context to make it unique", count)
	}

	newContent := strings.Replace(contentStr, oldText, newText, 1)
	return []byte(newContent), nil
}
