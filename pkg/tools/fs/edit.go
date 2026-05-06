package fstools

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// EditFileTool edits a file by replacing old_text with new_text.
// The old_text must exist exactly in the file.
type EditFileTool struct {
	fs                  fileSystem
	workspace           string
	restrictToWorkspace bool
	permissionCache    permissionCache
	askPermission      bool
}

// permissionCache is an interface for checking permissions, implemented by tools.PermissionCache
type permissionCache interface {
	Check(path string) string
}

// NewEditFileTool creates a new EditFileTool with optional directory restriction.
func NewEditFileTool(workspace string, restrict bool, allowPaths ...[]*regexp.Regexp) *EditFileTool {
	var patterns []*regexp.Regexp
	if len(allowPaths) > 0 {
		patterns = allowPaths[0]
	}
	return &EditFileTool{
		fs:                  buildFs(workspace, restrict, patterns),
		workspace:           workspace,
		restrictToWorkspace: restrict,
	}
}

// NewEditFileToolWithPermission creates a new EditFileTool with permission checking enabled.
func NewEditFileToolWithPermission(
	workspace string,
	restrict bool,
	permCache permissionCache,
	allowPaths ...[]*regexp.Regexp,
) *EditFileTool {
	tool := NewEditFileTool(workspace, restrict, allowPaths...)
	tool.permissionCache = permCache
	tool.askPermission = permCache != nil
	return tool
}

func (t *EditFileTool) Name() string {
	return "edit_file"
}

func (t *EditFileTool) isOutsideWorkspace(path string) bool {
	if t.workspace == "" {
		return false
	}
	absWorkspace, _ := filepath.Abs(t.workspace)
	absPath, _ := filepath.Abs(path)
	return !strings.HasPrefix(absPath, absWorkspace)
}

func (t *EditFileTool) Description() string {
	return "Edit a file by replacing old_text with new_text. The old_text must exist exactly in the file. Standard JSON escaping applies: \\n for newline and \\\\n for literal backslash-n."
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
				"description": "The exact text to find and replace. Standard JSON escaping applies: \\n for newline and \\\\n for literal backslash-n.",
			},
			"new_text": map[string]any{
				"type":        "string",
				"description": "The text to replace with. Standard JSON escaping applies: \\n for newline and \\\\n for literal backslash-n.",
			},
		},
		"required": []string{"path", "old_text", "new_text"},
	}
}

func (t *EditFileTool) checkPermission(path string) string {
	if !t.restrictToWorkspace || !t.askPermission || t.permissionCache == nil {
		return "granted"
	}

	if !t.isOutsideWorkspace(path) {
		return "granted"
	}

	if perm := t.permissionCache.Check(path); perm != "" {
		if perm == "denied" {
			return "denied"
		}
		return "granted"
	}

	return "needs_permission"
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

	switch t.checkPermission(path) {
	case "needs_permission":
		logger.InfoCF("edit_file", "Permission needed", map[string]any{"path": path})
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Permission needed for path: %s. Call request_permission tool with path='%s'.", path, path),
			ForUser: fmt.Sprintf("⚠️ Permission required to edit %s", path),
		}
	case "denied":
		return ErrorResult(fmt.Sprintf("Access to %s was denied", path))
	}

	if err := editFile(t.fs, path, oldText, newText); err != nil {
		return ErrorResult(err.Error())
	}
	return SilentResult(fmt.Sprintf("File edited: %s", path))
}

type AppendFileTool struct {
	fs                  fileSystem
	workspace           string
	restrictToWorkspace bool
	permissionCache    interface{ Check(path string) string }
	askPermission      bool
}

func NewAppendFileTool(workspace string, restrict bool, permCache any, allowPaths ...[]*regexp.Regexp) *AppendFileTool {
	var patterns []*regexp.Regexp
	if len(allowPaths) > 0 {
		patterns = allowPaths[0]
	}
	askPerm := false
	var cache interface{ Check(path string) string }
	if permCache != nil {
		cache = permCache.(interface{ Check(path string) string })
		askPerm = true
	}
	return &AppendFileTool{
		fs:                  buildFs(workspace, restrict, patterns),
		workspace:           workspace,
		restrictToWorkspace: restrict,
		permissionCache:    cache,
		askPermission:      askPerm,
	}
}

func (t *AppendFileTool) isOutsideWorkspace(path string) bool {
	if t.workspace == "" {
		return false
	}
	absWorkspace, _ := filepath.Abs(t.workspace)
	absPath, _ := filepath.Abs(path)
	return !strings.HasPrefix(absPath, absWorkspace)
}

func (t *AppendFileTool) checkPermission(path string) string {
	if !t.restrictToWorkspace || !t.askPermission || t.permissionCache == nil {
		return "granted"
	}

	if !t.isOutsideWorkspace(path) {
		return "granted"
	}

	if perm := t.permissionCache.Check(path); perm != "" {
		if perm == "denied" {
			return "denied"
		}
		return "granted"
	}

	return "needs_permission"
}

func (t *AppendFileTool) Name() string {
	return "append_file"
}

func (t *AppendFileTool) Description() string {
	return "Append content to the end of a file. Standard JSON escaping applies: \\n for newline and \\\\n for literal backslash-n."
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
				"description": "The content to append. Standard JSON escaping applies: \\n for newline and \\\\n for literal backslash-n.",
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

	switch t.checkPermission(path) {
	case "needs_permission":
		logger.InfoCF("append_file", "Permission needed", map[string]any{"path": path})
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Permission needed for path: %s. Call request_permission tool with path='%s'.", path, path),
			ForUser: fmt.Sprintf("⚠️ Permission required to append to %s", path),
		}
	case "denied":
		return ErrorResult(fmt.Sprintf("Access to %s was denied", path))
	}

	content, ok := args["content"].(string)
	if !ok {
		return ErrorResult("content is required")
	}

	if err := appendFile(t.fs, path, content); err != nil {
		return ErrorResult(err.Error())
	}
	return SilentResult(fmt.Sprintf("Appended to %s", path))
}

// editFile reads the file via sysFs, performs the replacement, and writes back.
// It uses a fileSystem interface, allowing the same logic for both restricted and unrestricted modes.
func editFile(sysFs fileSystem, path, oldText, newText string) error {
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

// appendFile reads the existing content (if any) via sysFs, appends new content, and writes back.
func appendFile(sysFs fileSystem, path, appendContent string) error {
	content, err := sysFs.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	newContent := append(content, []byte(appendContent)...)
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