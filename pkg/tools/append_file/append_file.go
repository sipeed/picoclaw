package append_file

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/sipeed/picoclaw/pkg/tools/common"
)

type AppendFileTool struct {
	fs common.FileSystem
}

func NewAppendFileTool(workspace string, restrict bool) *AppendFileTool {
	var fs common.FileSystem
	if restrict {
		fs = &common.SandboxFs{Workspace: workspace}
	} else {
		fs = &common.HostFs{}
	}
	return &AppendFileTool{fs: fs}
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

func (t *AppendFileTool) Execute(ctx context.Context, args map[string]any) *common.ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		return common.ErrorResult("path is required")
	}

	content, ok := args["content"].(string)
	if !ok {
		return common.ErrorResult("content is required")
	}

	if err := appendFile(t.fs, path, content); err != nil {
		return common.ErrorResult(err.Error())
	}
	return common.SilentResult(fmt.Sprintf("Appended to %s", path))
}

// appendFile reads the existing content (if any) via sysFs, appends new content, and writes back.
func appendFile(sysFs common.FileSystem, path, appendContent string) error {
	content, err := sysFs.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	newContent := append(content, []byte(appendContent)...)
	return sysFs.WriteFile(path, newContent)
}
