package tools

import (
	"context"
	"fmt"
	"sync"
)

type SendCallback func(channel, chatID, content, replyToMessageID string) error

// sentTarget records the channel+chatID that the message tool sent to.
type sentTarget struct {
	Channel string
	ChatID  string
}

type MessageTool struct {
	sendCallback SendCallback
	mu           sync.Mutex
	sentTargets  []sentTarget // Tracks all targets sent to in the current round
}

func NewMessageTool() *MessageTool {
	return &MessageTool{}
}

func (t *MessageTool) Name() string {
	return "message"
}

func (t *MessageTool) Description() string {
	return "Send a message to user on a chat channel. Use this when you want to communicate something."
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
			"reply_to_message_id": map[string]any{
				"type":        "string",
				"description": "Optional: reply target message ID for channels that support threaded replies",
			},
		},
		"required": []string{"content"},
	}
}

// ResetSentInRound resets the per-round send tracker.
// Called by the agent loop at the start of each inbound message processing round.
func (t *MessageTool) ResetSentInRound() {
	t.mu.Lock()
	t.sentTargets = t.sentTargets[:0]
	t.mu.Unlock()
}

// HasSentInRound returns true if the message tool sent a message during the current round.
func (t *MessageTool) HasSentInRound() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.sentTargets) > 0
}

// HasSentTo returns true if the message tool sent to the specific channel+chatID
// during the current round. Used by PublishResponseIfNeeded to avoid suppressing
// the final response when the message tool only sent to a different conversation.
func (t *MessageTool) HasSentTo(channel, chatID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, st := range t.sentTargets {
		if st.Channel == channel && st.ChatID == chatID {
			return true
		}
	}
	return false
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
	replyToMessageID, _ := args["reply_to_message_id"].(string)

	if channel == "" {
		channel = ToolChannel(ctx)
	}
	if chatID == "" {
		chatID = ToolChatID(ctx)
	}

	if channel == "" || chatID == "" {
		return &ToolResult{ForLLM: "No target channel/chat specified", IsError: true}
	}

	if t.sendCallback == nil {
		return &ToolResult{ForLLM: "Message sending not configured", IsError: true}
	}

	if err := t.sendCallback(channel, chatID, content, replyToMessageID); err != nil {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("sending message: %v", err),
			IsError: true,
			Err:     err,
		}
	}

	t.mu.Lock()
	t.sentTargets = append(t.sentTargets, sentTarget{Channel: channel, ChatID: chatID})
	t.mu.Unlock()

	// Silent: user already received the message directly
	return &ToolResult{
		ForLLM: fmt.Sprintf("Message sent to %s:%s", channel, chatID),
		Silent: true,
	}
}
