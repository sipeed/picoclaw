package tools

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type SendCallback func(msg bus.OutboundMessage) error

const (
	replyModeChat    = "chat"
	replyModeCurrent = "current"
	replyModeParent  = "parent"
)

type MessageTool struct {
	sendCallback SendCallback
	sentInRound  atomic.Bool // Tracks whether a message was sent in the current processing round
}

func NewMessageTool() *MessageTool {
	return &MessageTool{}
}

func (t *MessageTool) Name() string {
	return "message"
}

func (t *MessageTool) Description() string {
	return "Send an out-of-band message to a chat channel. Do not use this for the normal final reply in the current conversation."
}

func (t *MessageTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{
				"type":        "string",
				"description": "The message content to send",
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
		"required": []string{"content"},
	}
}

// ResetSentInRound resets the per-round send tracker.
// Called by the agent loop at the start of each inbound message processing round.
func (t *MessageTool) ResetSentInRound() {
	t.sentInRound.Store(false)
}

// HasSentInRound returns true if the message tool sent a message during the current round.
func (t *MessageTool) HasSentInRound() bool {
	return t.sentInRound.Load()
}

func (t *MessageTool) SetSendCallback(callback SendCallback) {
	t.sendCallback = callback
}

func (t *MessageTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	content, ok := args["content"].(string)
	if !ok {
		return &ToolResult{ForLLM: "content is required", IsError: true}
	}

	channel, _ := args["channel"].(string)
	chatID, _ := args["chat_id"].(string)

	if channel == "" {
		channel = ToolChannel(ctx)
	}
	if chatID == "" {
		chatID = ToolChatID(ctx)
	}

	if channel == "" || chatID == "" {
		return &ToolResult{ForLLM: "No target channel/chat specified", IsError: true}
	}

	currentChannel := ToolChannel(ctx)
	currentChatID := ToolChatID(ctx)
	replyMode, _ := args["reply_mode"].(string)
	replyMode = strings.ToLower(strings.TrimSpace(replyMode))
	explicitReplyTo, _ := args["reply_to_message_id"].(string)
	explicitReplyTo = strings.TrimSpace(explicitReplyTo)

	if replyMode != "" || explicitReplyTo != "" {
		logger.WarnCF("tool", "Message tool received deprecated reply routing args", map[string]any{
			"channel":             channel,
			"chat_id":             chatID,
			"reply_mode":          replyMode,
			"reply_to_message_id": explicitReplyTo,
		})
	}
	if currentChannel != "" && currentChatID != "" && channel == currentChannel && chatID == currentChatID {
		logger.InfoCF("tool", "Message tool targeting current conversation", map[string]any{
			"channel":     channel,
			"chat_id":     chatID,
			"content_len": len(content),
			"reply_mode":  replyMode,
			"same_target": true,
			"session_key": ToolSessionKey(ctx),
		})
	}

	replyToMessageID, err := resolveReplyTarget(ctx, args)
	if err != nil {
		return &ToolResult{
			ForLLM:  err.Error(),
			IsError: true,
			Err:     err,
		}
	}

	if t.sendCallback == nil {
		return &ToolResult{ForLLM: "Message sending not configured", IsError: true}
	}

	msg := bus.OutboundMessage{
		Channel:          channel,
		ChatID:           chatID,
		Content:          content,
		ReplyToMessageID: replyToMessageID,
	}
	if err := t.sendCallback(msg); err != nil {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("sending message: %v", err),
			IsError: true,
			Err:     err,
		}
	}

	t.sentInRound.Store(true)
	logger.InfoCF("tool", "Message tool sent outbound message", map[string]any{
		"channel":             channel,
		"chat_id":             chatID,
		"content_len":         len(content),
		"reply_to_message_id": replyToMessageID,
		"same_target":         currentChannel != "" && currentChatID != "" && channel == currentChannel && chatID == currentChatID,
	})

	// Silent: user already received the message directly
	status := fmt.Sprintf("Message sent to %s:%s", channel, chatID)
	if replyToMessageID != "" {
		status = fmt.Sprintf("%s in reply to %s", status, replyToMessageID)
	}
	return &ToolResult{
		ForLLM: status,
		Silent: true,
	}
}

func resolveReplyTarget(ctx context.Context, args map[string]any) (string, error) {
	replyToMessageID, _ := args["reply_to_message_id"].(string)
	replyToMessageID = strings.TrimSpace(replyToMessageID)
	if replyToMessageID != "" {
		return replyToMessageID, nil
	}

	replyMode, _ := args["reply_mode"].(string)
	replyMode = strings.ToLower(strings.TrimSpace(replyMode))

	switch replyMode {
	case "", replyModeChat:
		return "", nil
	case replyModeCurrent:
		if id := strings.TrimSpace(ToolCurrentMessageID(ctx)); id != "" {
			return id, nil
		}
		return "", fmt.Errorf("reply_mode=current requested but current message id is unavailable")
	case replyModeParent:
		if id := strings.TrimSpace(ToolParentMessageID(ctx)); id != "" {
			return id, nil
		}
		return "", fmt.Errorf("reply_mode=parent requested but parent message id is unavailable")
	default:
		return "", fmt.Errorf("unsupported reply_mode %q", replyMode)
	}
}
