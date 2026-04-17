package fstools

import (
	"context"

	toolshared "github.com/sipeed/picoclaw/pkg/tools/shared"
)

type ToolResult = toolshared.ToolResult

func WithToolContext(ctx context.Context, channel, chatID string) context.Context {
	return toolshared.WithToolContext(ctx, channel, chatID)
}

func ToolChannel(ctx context.Context) string {
	return toolshared.ToolChannel(ctx)
}

func ToolChatID(ctx context.Context) string {
	return toolshared.ToolChatID(ctx)
}

func ErrorResult(message string) *ToolResult {
	return toolshared.ErrorResult(message)
}

func NewToolResult(forLLM string) *ToolResult {
	return toolshared.NewToolResult(forLLM)
}

func SilentResult(forLLM string) *ToolResult {
	return toolshared.SilentResult(forLLM)
}

func MediaResult(forLLM string, mediaRefs []string) *ToolResult {
	return toolshared.MediaResult(forLLM, mediaRefs)
}
