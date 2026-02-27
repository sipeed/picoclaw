package tools

import (
	"github.com/sipeed/picoclaw/pkg/tools/common"
)

type (
	Tool           = common.Tool
	ToolResult     = common.ToolResult
	ContextualTool = common.ContextualTool
	AsyncTool      = common.AsyncTool
	AsyncCallback  = common.AsyncCallback
	FileSystem     = common.FileSystem
	HostFs         = common.HostFs
	SandboxFs      = common.SandboxFs
)

func NewToolResult(forLLM string) *ToolResult {
	return common.NewToolResult(forLLM)
}

func SilentResult(forLLM string) *ToolResult {
	return common.SilentResult(forLLM)
}

func AsyncResult(forLLM string) *ToolResult {
	return common.AsyncResult(forLLM)
}

func ErrorResult(message string) *ToolResult {
	return common.ErrorResult(message)
}

func UserResult(content string) *ToolResult {
	return common.UserResult(content)
}

func ValidatePath(path, workspace string, restrict bool) (string, error) {
	return common.ValidatePath(path, workspace, restrict)
}
