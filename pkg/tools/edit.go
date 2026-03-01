package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sipeed/picoclaw/pkg/agent/sandbox"
)

// EditFileTool edits a file by replacing old_text with new_text.
// The old_text must exist exactly in the file.
type EditFileTool struct{}

// NewEditFileTool creates a new EditFileTool.
// The workspace and restrict parameters are kept for compatibility with tool registry signatures
// but are no longer used since filesystem access is entirely delegated to the Sandbox context.
func NewEditFileTool(workspace string, restrict bool) *EditFileTool {
	return &EditFileTool{}
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

	sb := sandbox.FromContext(ctx)
	if sb == nil {
		return ErrorResult("sandbox environment unavailable")
	}

	content, err := sb.Fs().ReadFile(ctx, path)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read file: %v", err))
	}

	newContent, err := replaceEditContent(content, oldText, newText)
	if err != nil {
		return ErrorResult(err.Error())
	}

	err = sb.Fs().WriteFile(ctx, path, newContent, true)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to write file: %v", err))
	}
	return SilentResult(fmt.Sprintf("File edited: %s", path))
}

type AppendFileTool struct{}

// NewAppendFileTool creates a new AppendFileTool.
// The workspace and restrict parameters are kept for compatibility with tool registry signatures
// but are no longer used since filesystem access is entirely delegated to the Sandbox context.
func NewAppendFileTool(workspace string, restrict bool) *AppendFileTool {
	return &AppendFileTool{}
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

	content, ok := args["content"].(string)
	if !ok {
		return ErrorResult("content is required")
	}

	sb := sandbox.FromContext(ctx)
	if sb == nil {
		return ErrorResult("sandbox environment unavailable")
	}

	// Implement Append using Read + Write
	oldContent, err := sb.Fs().ReadFile(ctx, path)
	if err != nil && !os.IsNotExist(err) && !strings.Contains(err.Error(), "not found") {
		return ErrorResult(fmt.Sprintf("failed to read file for append: %v", err))
	}
	newContent := append(oldContent, []byte(content)...)
	err = sb.Fs().WriteFile(ctx, path, newContent, true)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to append (write) to file: %v", err))
	}
	return SilentResult(fmt.Sprintf("Appended to %s", path))
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
