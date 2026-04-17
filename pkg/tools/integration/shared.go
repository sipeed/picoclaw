package integrationtools

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/session"
	toolshared "github.com/sipeed/picoclaw/pkg/tools/shared"
)

type (
	Tool          = toolshared.Tool
	ToolResult    = toolshared.ToolResult
	AsyncCallback = toolshared.AsyncCallback
)

func WithToolContext(ctx context.Context, channel, chatID string) context.Context {
	return toolshared.WithToolContext(ctx, channel, chatID)
}

func WithToolInboundContext(
	ctx context.Context,
	channel, chatID, messageID, replyToMessageID string,
) context.Context {
	return toolshared.WithToolInboundContext(ctx, channel, chatID, messageID, replyToMessageID)
}

func WithToolSessionContext(
	ctx context.Context,
	agentID, sessionKey string,
	scope *session.SessionScope,
) context.Context {
	return toolshared.WithToolSessionContext(ctx, agentID, sessionKey, scope)
}

func ToolChannel(ctx context.Context) string {
	return toolshared.ToolChannel(ctx)
}

func ToolChatID(ctx context.Context) string {
	return toolshared.ToolChatID(ctx)
}

func ToolMessageID(ctx context.Context) string {
	return toolshared.ToolMessageID(ctx)
}

func ToolAgentID(ctx context.Context) string {
	return toolshared.ToolAgentID(ctx)
}

func ToolSessionKey(ctx context.Context) string {
	return toolshared.ToolSessionKey(ctx)
}

func ToolSessionScope(ctx context.Context) *session.SessionScope {
	return toolshared.ToolSessionScope(ctx)
}

func ErrorResult(message string) *ToolResult {
	return toolshared.ErrorResult(message)
}

func SilentResult(forLLM string) *ToolResult {
	return toolshared.SilentResult(forLLM)
}

func NewToolResult(forLLM string) *ToolResult {
	return toolshared.NewToolResult(forLLM)
}

func UserResult(content string) *ToolResult {
	return toolshared.UserResult(content)
}

func MediaResult(forLLM string, mediaRefs []string) *ToolResult {
	return toolshared.MediaResult(forLLM, mediaRefs)
}
