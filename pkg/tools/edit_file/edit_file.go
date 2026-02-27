package edit_file

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/tools/common"
)

// EditFileTool edits a file by replacing old_text with new_text.
// The old_text must exist exactly in the file.
type EditFileTool struct {
	fs common.FileSystem
}

// NewEditFileTool creates a new EditFileTool with optional directory restriction.
func NewEditFileTool(workspace string, restrict bool) *EditFileTool {
	var fs common.FileSystem
	if restrict {
		fs = &common.SandboxFs{Workspace: workspace}
	} else {
		fs = &common.HostFs{}
	}
	return &EditFileTool{fs: fs}
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

func (t *EditFileTool) Execute(ctx context.Context, args map[string]any) *common.ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		return common.ErrorResult("path is required")
	}

	oldText, ok := args["old_text"].(string)
	if !ok {
		return common.ErrorResult("old_text is required")
	}

	newText, ok := args["new_text"].(string)
	if !ok {
		return common.ErrorResult("new_text is required")
	}

	if err := editFile(t.fs, path, oldText, newText); err != nil {
		return common.ErrorResult(err.Error())
	}
	return common.SilentResult(fmt.Sprintf("File edited: %s", path))
}

// editFile reads the file via sysFs, performs the replacement, and writes back.
// It uses a common.FileSystem interface, allowing the same logic for both restricted and unrestricted modes.
func editFile(sysFs common.FileSystem, path, oldText, newText string) error {
	content, err := sysFs.ReadFile(path)
	if err != nil {
		return err
	}

	newContent, err := replaceEditContent(content, oldText, newText)
	if err != nil {
		return err
	}

	return sysFs.WriteFile(path, newContent)
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
