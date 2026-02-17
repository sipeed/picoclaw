package tools

import (
	"context"
	"fmt"
	"os"
)

// SendCallback sends a plain-text message to a channel/chat.
type SendCallback func(channel, chatID, content string) error

// SendMediaCallback sends one or more local media files to a channel/chat.
// The callback owns the call; the caller is responsible for cleaning up files afterward.
type SendMediaCallback func(ctx context.Context, channel, chatID string, filePaths []string) error

// SynthesizeCallback converts text to an audio file and returns the local path.
// The caller must delete the file when done.
type SynthesizeCallback func(ctx context.Context, text string) (filePath string, err error)

type MessageTool struct {
	sendCallback     SendCallback
	sendMediaCallback SendMediaCallback
	synthesizeCallback SynthesizeCallback
	defaultChannel   string
	defaultChatID    string
	sentInRound      bool
}

func NewMessageTool() *MessageTool {
	return &MessageTool{}
}

func (t *MessageTool) Name() string {
	return "message"
}

func (t *MessageTool) Description() string {
	return `Send a message or voice reply to the user.
Set voice=true to reply with audio (uses TTS). Use voice when the user sent a voice message or explicitly asks for audio.
Default is text. voice=true requires the TTS service to be available.`
}

func (t *MessageTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The message text to send (also used as TTS input when voice=true)",
			},
			"voice": map[string]interface{}{
				"type":        "boolean",
				"description": "Set to true to send a voice/audio message via TTS instead of text",
			},
			"channel": map[string]interface{}{
				"type":        "string",
				"description": "Optional: target channel override",
			},
			"chat_id": map[string]interface{}{
				"type":        "string",
				"description": "Optional: target chat ID override",
			},
		},
		"required": []string{"content"},
	}
}

func (t *MessageTool) SetContext(channel, chatID string) {
	t.defaultChannel = channel
	t.defaultChatID = chatID
	t.sentInRound = false
}

func (t *MessageTool) HasSentInRound() bool {
	return t.sentInRound
}

func (t *MessageTool) SetSendCallback(callback SendCallback) {
	t.sendCallback = callback
}

func (t *MessageTool) SetSendMediaCallback(callback SendMediaCallback) {
	t.sendMediaCallback = callback
}

func (t *MessageTool) SetSynthesizeCallback(callback SynthesizeCallback) {
	t.synthesizeCallback = callback
}

func (t *MessageTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	content, ok := args["content"].(string)
	if !ok || content == "" {
		return &ToolResult{ForLLM: "content is required", IsError: true}
	}

	voice, _ := args["voice"].(bool)
	channel, _ := args["channel"].(string)
	chatID, _ := args["chat_id"].(string)

	if channel == "" {
		channel = t.defaultChannel
	}
	if chatID == "" {
		chatID = t.defaultChatID
	}

	if channel == "" || chatID == "" {
		return &ToolResult{ForLLM: "No target channel/chat specified", IsError: true}
	}

	// Voice path
	if voice {
		if t.synthesizeCallback == nil || t.sendMediaCallback == nil {
			return &ToolResult{ForLLM: "TTS not available — sending as text instead", IsError: false}
		}

		audioPath, err := t.synthesizeCallback(ctx, content)
		if err != nil {
			return &ToolResult{
				ForLLM:  fmt.Sprintf("TTS synthesis failed: %v — falling back to text", err),
				IsError: false,
			}
		}
		defer os.Remove(audioPath)

		if err := t.sendMediaCallback(ctx, channel, chatID, []string{audioPath}); err != nil {
			return &ToolResult{
				ForLLM:  fmt.Sprintf("failed to send audio: %v", err),
				IsError: true,
				Err:     err,
			}
		}

		t.sentInRound = true
		return &ToolResult{
			ForLLM: fmt.Sprintf("Voice message sent to %s:%s", channel, chatID),
			Silent: true,
		}
	}

	// Text path
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

	t.sentInRound = true
	return &ToolResult{
		ForLLM: fmt.Sprintf("Message sent to %s:%s", channel, chatID),
		Silent: true,
	}
}
