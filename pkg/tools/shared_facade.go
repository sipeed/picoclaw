package tools

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/session"
	toolshared "github.com/sipeed/picoclaw/pkg/tools/shared"
)

type (
	Message                = toolshared.Message
	ToolCall               = toolshared.ToolCall
	FunctionCall           = toolshared.FunctionCall
	LLMResponse            = toolshared.LLMResponse
	UsageInfo              = toolshared.UsageInfo
	LLMProvider            = toolshared.LLMProvider
	ToolDefinition         = toolshared.ToolDefinition
	ToolFunctionDefinition = toolshared.ToolFunctionDefinition
	ExecRequest            = toolshared.ExecRequest
	ExecResponse           = toolshared.ExecResponse
	SessionInfo            = toolshared.SessionInfo
	Tool                   = toolshared.Tool
	AsyncCallback          = toolshared.AsyncCallback
	AsyncExecutor          = toolshared.AsyncExecutor
	ToolResult             = toolshared.ToolResult
)

const (
	handledToolLLMNote   = toolshared.HandledToolLLMNote
	artifactPathsLLMNote = toolshared.ArtifactPathsLLMNote
)

func WithToolContext(ctx context.Context, channel, chatID string) context.Context {
	return toolshared.WithToolContext(ctx, channel, chatID)
}

func WithToolMessageContext(ctx context.Context, messageID, replyToMessageID string) context.Context {
	return toolshared.WithToolMessageContext(ctx, messageID, replyToMessageID)
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

func ToolReplyToMessageID(ctx context.Context) string {
	return toolshared.ToolReplyToMessageID(ctx)
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

func ToolToSchema(tool Tool) map[string]any {
	return toolshared.ToolToSchema(tool)
}

func NewToolResult(forLLM string) *ToolResult {
	return toolshared.NewToolResult(forLLM)
}

func SilentResult(forLLM string) *ToolResult {
	return toolshared.SilentResult(forLLM)
}

func AsyncResult(forLLM string) *ToolResult {
	return toolshared.AsyncResult(forLLM)
}

func ErrorResult(message string) *ToolResult {
	return toolshared.ErrorResult(message)
}

func UserResult(content string) *ToolResult {
	return toolshared.UserResult(content)
}

func MediaResult(forLLM string, mediaRefs []string) *ToolResult {
	return toolshared.MediaResult(forLLM, mediaRefs)
}
