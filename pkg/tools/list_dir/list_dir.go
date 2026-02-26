package list_dir

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sipeed/picoclaw/pkg/tools/common"
)

type ListDirTool struct {
	fs common.FileSystem
}

func NewListDirTool(workspace string, restrict bool) *ListDirTool {
	var fs common.FileSystem
	if restrict {
		fs = &common.SandboxFs{Workspace: workspace}
	} else {
		fs = &common.HostFs{}
	}
	return &ListDirTool{fs: fs}
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

func (t *ListDirTool) Execute(ctx context.Context, args map[string]any) *common.ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		path = "."
	}

	entries, err := t.fs.ReadDir(path)
	if err != nil {
		return common.ErrorResult(fmt.Sprintf("failed to read directory: %v", err))
	}
	return formatDirEntries(entries)
}

func formatDirEntries(entries []os.DirEntry) *common.ToolResult {
	var result strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			result.WriteString("DIR:  " + entry.Name() + "\n")
		} else {
			result.WriteString("FILE: " + entry.Name() + "\n")
		}
	}
	return common.NewToolResult(result.String())
}
