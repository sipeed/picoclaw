package read_file

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/tools/common"
)

type ReadFileTool struct {
	fs common.FileSystem
}

func NewReadFileTool(workspace string, restrict bool) *ReadFileTool {
	var fs common.FileSystem
	if restrict {
		fs = &common.SandboxFs{Workspace: workspace}
	} else {
		fs = &common.HostFs{}
	}
	return &ReadFileTool{fs: fs}
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

func (t *ReadFileTool) Execute(ctx context.Context, args map[string]any) *common.ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		return common.ErrorResult("path is required")
	}

	content, err := t.fs.ReadFile(path)
	if err != nil {
		return common.ErrorResult(err.Error())
	}
	return common.NewToolResult(string(content))
}
