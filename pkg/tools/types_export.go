package tools

import (
	"github.com/sipeed/picoclaw/pkg/tools/common"
)

type Tool = common.Tool
type ToolResult = common.ToolResult
type ContextualTool = common.ContextualTool
type AsyncTool = common.AsyncTool
type AsyncCallback = common.AsyncCallback
type FileSystem = common.FileSystem
type HostFs = common.HostFs
type SandboxFs = common.SandboxFs

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
