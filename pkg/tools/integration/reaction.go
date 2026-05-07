package integrationtools

import (
	"context"
	"fmt"
)

type ReactionCallback func(ctx context.Context, channel, chatID, messageID string) error

type ReactionTool struct {
	reactionCallback ReactionCallback
}

func NewReactionTool() *ReactionTool {
	return &ReactionTool{}
}

func (t *ReactionTool) Name() string {
	return "reaction"
}

func (t *ReactionTool) Description() string {
	return "Add a reaction to a message. Defaults to the current inbound message when message_id is omitted."
}

func (t *ReactionTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message_id": map[string]any{
				"type":        "string",
				"description": "Optional: target message ID; defaults to the current inbound message",
			},
			"channel": map[string]any{
				"type":        "string",
				"description": "Optional: target channel (telegram, whatsapp, etc.)",
			},
			"chat_id": map[string]any{
				"type":        "string",
				"description": "Optional: target chat/user ID",
			},
		},
	}
}

func (t *ReactionTool) SetReactionCallback(callback ReactionCallback) {
	t.reactionCallback = callback
}

func (t *ReactionTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	channel, _ := args["channel"].(string)
	chatID, _ := args["chat_id"].(string)
	messageID, _ := args["message_id"].(string)

	if channel == "" {
		channel = ToolChannel(ctx)
	}
	if chatID == "" {
		chatID = ToolChatID(ctx)
	}
	if messageID == "" {
		messageID = ToolMessageID(ctx)
	}

	if channel == "" || chatID == "" {
		return &ToolResult{ForLLM: "No target channel/chat specified", IsError: true}
	}
	if messageID == "" {
		return &ToolResult{ForLLM: "message_id is required", IsError: true}
	}
	if t.reactionCallback == nil {
		return &ToolResult{ForLLM: "Reaction not configured", IsError: true}
	}

	if err := t.reactionCallback(ctx, channel, chatID, messageID); err != nil {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("adding reaction: %v", err),
			IsError: true,
			Err:     err,
		}
	}

	return &ToolResult{
		ForLLM: fmt.Sprintf("Reaction added to %s:%s message %s", channel, chatID, messageID),
		Silent: true,
	}
}
