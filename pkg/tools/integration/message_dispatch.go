package integrationtools

import (
	"context"
	"fmt"
	"sync"
)

type SendCallbackWithContext func(ctx context.Context, channel, chatID, content, replyToMessageID string) error

// MessageOption is kept for external callers that reference it directly.
// New code should use OptionItem in message.go instead.
type MessageOption = OptionItem

type SendStructuredCallbackWithContext func(ctx context.Context, channel, chatID, content, replyToMessageID string, structured any) error

// sentTarget records the channel+chatID that a message-like tool sent to.
type sentTarget struct {
	Channel string
	ChatID  string
}

type messageDispatchTool struct {
	sendCallback SendStructuredCallbackWithContext
	mu           sync.Mutex
	// sentTargets tracks targets sent to in the current round, keyed by session key
	// to support parallel turns for different sessions.
	sentTargets map[string][]sentTarget
}

func newMessageDispatchTool() messageDispatchTool {
	return messageDispatchTool{
		sentTargets: make(map[string][]sentTarget),
	}
}

func (t *messageDispatchTool) SetSendCallback(callback SendCallbackWithContext) {
	if callback == nil {
		t.sendCallback = nil
		return
	}
	t.sendCallback = func(ctx context.Context, channel, chatID, content, replyToMessageID string, structured any) error {
		return callback(ctx, channel, chatID, content, replyToMessageID)
	}
}

func (t *messageDispatchTool) SetStructuredSendCallback(callback SendStructuredCallbackWithContext) {
	t.sendCallback = callback
}

// ResetSentInRound resets the per-round send tracker for the given session key.
// Called by the agent loop at the start of each inbound message processing round.
func (t *messageDispatchTool) ResetSentInRound(sessionKey string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Delete the key entirely to prevent unbounded map growth over time
	// with many unique sessions. Truncating the slice keeps the key alive.
	delete(t.sentTargets, sessionKey)
}

// HasSentInRound returns true if the tool sent a message during the current round.
func (t *messageDispatchTool) HasSentInRound(sessionKey string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.sentTargets[sessionKey]) > 0
}

// HasSentTo returns true if the tool sent to the specific channel+chatID during the current round.
func (t *messageDispatchTool) HasSentTo(sessionKey, channel, chatID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, st := range t.sentTargets[sessionKey] {
		if st.Channel == channel && st.ChatID == chatID {
			return true
		}
	}
	return false
}

func (t *messageDispatchTool) executeSend(
	ctx context.Context,
	args map[string]any,
	content string,
	structured any,
) *ToolResult {
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

	if err := t.sendCallback(ctx, channel, chatID, content, replyToMessageID, structured); err != nil {
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

	return &ToolResult{
		ForLLM: fmt.Sprintf("Message sent to %s:%s", channel, chatID),
		Silent: true,
	}
}
