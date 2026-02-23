package tools

import (
	"context"
	"strings"
)

type executionContextKey string

const (
	executionContextChannelKey  executionContextKey = "tool_execution_channel"
	executionContextChatIDKey   executionContextKey = "tool_execution_chat_id"
	executionContextSenderIDKey executionContextKey = "tool_execution_sender_id"
)

func withExecutionContext(ctx context.Context, channel, chatID, senderID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(channel) != "" {
		ctx = context.WithValue(ctx, executionContextChannelKey, strings.TrimSpace(channel))
	}
	if strings.TrimSpace(chatID) != "" {
		ctx = context.WithValue(ctx, executionContextChatIDKey, strings.TrimSpace(chatID))
	}
	if strings.TrimSpace(senderID) != "" {
		ctx = context.WithValue(ctx, executionContextSenderIDKey, strings.TrimSpace(senderID))
	}
	return ctx
}

func toolExecutionChannel(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(executionContextChannelKey).(string)
	return strings.TrimSpace(v)
}

func toolExecutionChatID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(executionContextChatIDKey).(string)
	return strings.TrimSpace(v)
}

func toolExecutionSenderID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(executionContextSenderIDKey).(string)
	return strings.TrimSpace(v)
}
