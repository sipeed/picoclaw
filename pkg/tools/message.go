package tools

import (
	"context"
	"fmt"
	"sync"
)

type SendCallbackWithContext func(ctx context.Context, channel, chatID, content, replyToMessageID string) error

// sentTarget records the channel+chatID that the message tool sent to.
type sentTarget struct {
	Channel string
	ChatID  string
}

type MessageTool struct {
	sendCallback SendCallbackWithContext
	mu           sync.Mutex
	// sentTargets tracks targets sent to in the current round, keyed by session key
	// to support parallel turns for different sessions.
	sentTargets map[string][]sentTarget
}

func NewMessageTool() *MessageTool {
	return &MessageTool{
		sentTargets: make(map[string][]sentTarget),
	}
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

// ResetSentInRound resets the per-round send tracker for the given session key.
// Called by the agent loop at the start of each inbound message processing round.
func (t *MessageTool) ResetSentInRound(sessionKey string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Delete the key entirely to prevent unbounded map growth over time
	// with many unique sessions. Truncating the slice keeps the key alive.
	delete(t.sentTargets, sessionKey)
}

// HasSentInRound returns true if the message tool sent a message during the current round.
func (t *MessageTool) HasSentInRound(sessionKey string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.sentTargets[sessionKey]) > 0
}

// HasSentTo returns true if the message tool sent to the specific channel+chatID
// during the current round. Used by PublishResponseIfNeeded to avoid suppressing
// the final response when the message tool only sent to a different conversation.
func (t *MessageTool) HasSentTo(sessionKey, channel, chatID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, st := range t.sentTargets[sessionKey] {
		if st.Channel == channel && st.ChatID == chatID {
			return true
		}
	}
	return false
}

func (t *MessageTool) SetSendCallback(callback SendCallbackWithContext) {
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

	if err := t.sendCallback(ctx, channel, chatID, content, replyToMessageID); err != nil {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("sending message: %v", err),
			IsError: true,
			Err:     err,
		}
	}

	sessionKey := ToolSessionKey(ctx)
	t.mu.Lock()
	t.sentTargets[sessionKey] = append(t.sentTargets[sessionKey], sentTarget{Channel: channel, ChatID: chatID})
	t.mu.Unlock()

	// Silent: user already received the message directly
	return &ToolResult{
		ForLLM: fmt.Sprintf("Message sent to %s:%s", channel, chatID),
		Silent: true,
	}
}
