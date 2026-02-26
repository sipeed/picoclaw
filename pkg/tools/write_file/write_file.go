package write_file

import (
	"context"
	"fmt"

	"github.com/sipeed/picoclaw/pkg/tools/common"
)

type WriteFileTool struct {
	fs common.FileSystem
}

func NewWriteFileTool(workspace string, restrict bool) *WriteFileTool {
	var fs common.FileSystem
	if restrict {
		fs = &common.SandboxFs{Workspace: workspace}
	} else {
		fs = &common.HostFs{}
	}
	return &WriteFileTool{fs: fs}
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

func (t *WriteFileTool) Execute(ctx context.Context, args map[string]any) *common.ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		return common.ErrorResult("path is required")
	}

	content, ok := args["content"].(string)
	if !ok {
		return common.ErrorResult("content is required")
	}

	if err := t.fs.WriteFile(path, []byte(content)); err != nil {
		return common.ErrorResult(err.Error())
	}

	return common.SilentResult(fmt.Sprintf("File written: %s", path))
}
