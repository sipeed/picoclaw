package tools

import (
	"context"
	"fmt"
	"strings"
)

type SendCallback func(channel, chatID, content string) error

// StateResolver provides access to persistent state for cross-channel routing.
type StateResolver interface {
	GetLastMainChannel() string
	GetChannelChatID(channel string) string
}

type MessageTool struct {
	sendCallback    SendCallback
	defaultChannel  string
	defaultChatID   string
	sentInRound     bool // Tracks whether a message was sent in the current processing round
	enabledChannels []string
	stateResolver   StateResolver
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

func (t *MessageTool) Parameters() map[string]interface{} {
	channelDesc := "Optional: target channel (telegram, whatsapp, etc.)"
	if len(t.enabledChannels) > 0 {
		channelDesc = fmt.Sprintf("Target channel. Available: %s, app (= current Android app session). Omit to reply on the current channel.",
			strings.Join(t.enabledChannels, ", "))
	}
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The message content to send",
			},
			"channel": map[string]interface{}{
				"type":        "string",
				"description": channelDesc,
			},
			"chat_id": map[string]interface{}{
				"type":        "string",
				"description": "Optional: target chat/user ID",
			},
		},
		"required": []string{"content"},
	}
}

// SetEnabledChannels updates the list of available channel names for parameter descriptions.
func (t *MessageTool) SetEnabledChannels(channels []string) {
	t.enabledChannels = channels
}

// SetStateResolver sets the state resolver for cross-channel alias resolution.
func (t *MessageTool) SetStateResolver(sr StateResolver) {
	t.stateResolver = sr
}

func (t *MessageTool) SetContext(channel, chatID string) {
	t.defaultChannel = channel
	t.defaultChatID = chatID
	t.sentInRound = false // Reset send tracking for new processing round
}

// HasSentInRound returns true if the message tool sent a message during the current round.
func (t *MessageTool) HasSentInRound() bool {
	return t.sentInRound
}

func (t *MessageTool) SetSendCallback(callback SendCallback) {
	t.sendCallback = callback
}

func (t *MessageTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	content, ok := args["content"].(string)
	if !ok {
		return &ToolResult{ForLLM: "content is required", IsError: true}
	}

	channel, _ := args["channel"].(string)
	chatID, _ := args["chat_id"].(string)

	// Resolve "app" alias to the last known Android app (main) WebSocket session
	if channel == "app" && t.stateResolver != nil {
		if mainCh := t.stateResolver.GetLastMainChannel(); mainCh != "" {
			// mainCh format: "websocket:ws:uuid"
			parts := strings.SplitN(mainCh, ":", 2)
			if len(parts) == 2 {
				channel = parts[0]
				chatID = parts[1]
			}
		}
	}

	if channel == "" {
		channel = t.defaultChannel
	}

	// Cross-channel send: AI specified a different channel but no chatID.
	// Look up the last known chatID for the target channel from state,
	// instead of using the current session's defaultChatID (which belongs
	// to a different channel and would cause API errors).
	isCrossChannel := channel != "" && channel != t.defaultChannel
	if chatID == "" && isCrossChannel && t.stateResolver != nil {
		chatID = t.stateResolver.GetChannelChatID(channel)
	}
	if chatID == "" && isCrossChannel {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Cannot send to %s: no known chat_id. A message must be received from %s first so the system can learn its chat_id.", channel, channel),
			IsError: true,
		}
	}

	if chatID == "" {
		chatID = t.defaultChatID
	}

	if channel == "" || chatID == "" {
		return &ToolResult{ForLLM: "No target channel/chat specified", IsError: true}
	}

	if t.sendCallback == nil {
		return &ToolResult{ForLLM: "Message sending not configured", IsError: true}
	}

	if err := t.sendCallback(channel, chatID, content); err != nil {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("sending message: %v", err),
			IsError: true,
			Err:     err,
		}
	}

	// Only mark as "sent in round" when the message went to the originating channel.
	// Cross-channel sends (e.g. WS→Discord) must NOT suppress the response
	// back to the sender's channel.
	if channel == t.defaultChannel {
		t.sentInRound = true
	}
	// Silent: user already received the message directly
	return &ToolResult{
		ForLLM: fmt.Sprintf("Message sent to %s:%s", channel, chatID),
		Silent: true,
	}
}
