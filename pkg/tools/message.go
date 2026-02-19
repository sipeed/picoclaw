package tools

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/sipeed/picoclaw/pkg/bus"
)

type SendCallback func(channel, chatID, content string, attachments []bus.Attachment) error

type MessageTool struct {
	allowedDir     string
	restrict       bool
	sendCallback   SendCallback
	defaultChannel string
	defaultChatID  string
	sentInRound    bool // Tracks whether a message was sent in the current processing round
}

// NewMessageTool creates a new MessageTool with optional uploading directory restriction.
func NewMessageTool(allowedDir string, restrict bool) *MessageTool {
	return &MessageTool{
		allowedDir: allowedDir,
		restrict:   restrict,
	}
}

func (t *MessageTool) Name() string {
	return "message"
}

func (t *MessageTool) Description() string {
	return "Send a message to user on a chat channel. Use this when you want to communicate something."
}

func (t *MessageTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The message content to send",
			},
			"channel": map[string]interface{}{
				"type":        "string",
				"description": "Optional: target channel (telegram, whatsapp, etc.)",
			},
			"chat_id": map[string]interface{}{
				"type":        "string",
				"description": "Optional: target chat/user ID",
			},
			"attachments": map[string]interface{}{
				"type":        "array",
				"description": "Optional: files to attach to the message",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "File path to attach (absolute or relative to workspace)",
						},
						"filename": map[string]interface{}{
							"type":        "string",
							"description": "Filename to use for the attachment (defaults to the base name of the path)",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		"required": []string{"content"},
	}
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

	if channel == "" {
		channel = t.defaultChannel
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

	// Parse attachments if provided
	var attachments []bus.Attachment
	if attachmentsRaw, ok := args["attachments"].([]interface{}); ok {
		for i, attachRaw := range attachmentsRaw {
			attachMap, ok := attachRaw.(map[string]interface{})
			if !ok {
				return ErrorResult(fmt.Sprintf("attachments[%d]: expected an object, got %T", i, attachRaw))
			}

			path, pathOk := attachMap["path"].(string)
			if !pathOk || path == "" {
				return ErrorResult(fmt.Sprintf("attachments[%d]: missing or invalid \"path\" field", i))
			}

			filename, filenameOk := attachMap["filename"].(string)
			if !filenameOk || filename == "" {
				filename = filepath.Base(path) // Default to the base name of the file if filename not provided
			}

			resolvedPath, err := validatePath(path, t.allowedDir, t.restrict)
			if err != nil {
				return ErrorResult(fmt.Sprintf("attachments[%d]: %v", i, err))
			}

			attachments = append(attachments, bus.Attachment{
				Path:     resolvedPath,
				Filename: filename,
			})
		}
	}

	if err := t.sendCallback(channel, chatID, content, attachments); err != nil {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("sending message: %v", err),
			IsError: true,
			Err:     err,
		}
	}

	t.sentInRound = true
	// Silent: user already received the message directly
	return &ToolResult{
		ForLLM: fmt.Sprintf("Message sent to %s:%s", channel, chatID),
		Silent: true,
	}
}
