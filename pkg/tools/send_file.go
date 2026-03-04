package tools

import (
	"context"
	"fmt"
	"os"
)

// SendMediaCallback is called to deliver a file to a chat channel.
type SendMediaCallback func(channel, chatID, localPath, caption string) error

// SendFileTool allows the LLM to send files to the user on a chat channel.
type SendFileTool struct {
	mediaCallback  SendMediaCallback
	defaultChannel string
	defaultChatID  string
}

func NewSendFileTool() *SendFileTool {
	return &SendFileTool{}
}

func (t *SendFileTool) Name() string {
	return "send_file"
}

func (t *SendFileTool) Description() string {
	return "Send a file to the user on a chat channel. Use this to deliver files, images, documents, audio, video, etc."
}

func (t *SendFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Absolute path to the file on disk to send",
			},
			"caption": map[string]any{
				"type":        "string",
				"description": "Optional caption to include with the file",
			},
			"channel": map[string]any{
				"type":        "string",
				"description": "Optional: target channel (telegram, discord, etc.)",
			},
			"chat_id": map[string]any{
				"type":        "string",
				"description": "Optional: target chat/user ID",
			},
		},
		"required": []string{"path"},
	}
}

func (t *SendFileTool) SetContext(channel, chatID string) {
	t.defaultChannel = channel
	t.defaultChatID = chatID
}

func (t *SendFileTool) SetMediaCallback(callback SendMediaCallback) {
	t.mediaCallback = callback
}

func (t *SendFileTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return ErrorResult("path is required")
	}

	if _, err := os.Stat(path); err != nil {
		return ErrorResult(fmt.Sprintf("file not accessible: %v", err))
	}

	caption, _ := args["caption"].(string)
	channel, _ := args["channel"].(string)
	chatID, _ := args["chat_id"].(string)

	if channel == "" {
		channel = t.defaultChannel
	}
	if chatID == "" {
		chatID = t.defaultChatID
	}

	if channel == "" || chatID == "" {
		return ErrorResult("no target channel/chat specified")
	}

	if t.mediaCallback == nil {
		return ErrorResult("file sending not configured")
	}

	if err := t.mediaCallback(channel, chatID, path, caption); err != nil {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("sending file: %v", err),
			IsError: true,
			Err:     err,
		}
	}

	return SilentResult(fmt.Sprintf("File sent to %s:%s: %s", channel, chatID, path))
}
