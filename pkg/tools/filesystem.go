package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sipeed/picoclaw/pkg/agent/sandbox"
)

type ReadFileTool struct{}

// NewReadFileTool creates a new ReadFileTool.
// The workspace and restrict parameters are kept for compatibility with tool registry signatures
// but are no longer used since filesystem access is entirely delegated to the Sandbox context.
func NewReadFileTool(workspace string, restrict bool) *ReadFileTool {
	return &ReadFileTool{}
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "Read the contents of a file"
}

func (t *ReadFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the file to read",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		return ErrorResult("path is required")
	}

	sb := sandbox.FromContext(ctx)
	if sb == nil {
		return ErrorResult("sandbox environment unavailable")
	}

	content, err := sb.Fs().ReadFile(ctx, path)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read file: %v", err))
	}
	return NewToolResult(string(content))
}

type WriteFileTool struct{}

func NewWriteFileTool(workspace string, restrict bool) *WriteFileTool {
	return &WriteFileTool{}
}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Description() string {
	return "Write content to a file"
}

func (t *WriteFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the file to write",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Content to write to the file",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *WriteFileTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
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

	if err := sb.Fs().WriteFile(ctx, path, []byte(content), true); err != nil {
		return ErrorResult(fmt.Sprintf("failed to write file: %v", err))
	}
	return SilentResult(fmt.Sprintf("File written: %s", path))
}

type ListDirTool struct{}

func NewListDirTool(workspace string, restrict bool) *ListDirTool {
	return &ListDirTool{}
}

func (t *ListDirTool) Name() string {
	return "list_dir"
}

func (t *ListDirTool) Description() string {
	return "List files and directories in a path"
}

func (t *ListDirTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to list",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ListDirTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		path = "."
	}

	sb := sandbox.FromContext(ctx)
	if sb == nil {
		return ErrorResult("sandbox environment unavailable")
	}

	entries, err := sb.Fs().ReadDir(ctx, path)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read directory: %v", err))
	}
	return formatDirEntries(entries)
}

func formatDirEntries(entries []os.DirEntry) *ToolResult {
	var result strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			result.WriteString("DIR:  " + entry.Name() + "\n")
		} else {
			result.WriteString("FILE: " + entry.Name() + "\n")
		}
	}
	return NewToolResult(result.String())
}
