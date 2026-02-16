package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
)

type SendFileCallback func(msg bus.OutboundMessage) error

type SendFileTool struct {
	workspace      string
	restrict       bool
	maxFileBytes   int64
	sendCallback   SendFileCallback
	defaultChannel string
	defaultChatID  string
}

func NewSendFileTool(workspace string, restrict bool, maxFileBytes int64) *SendFileTool {
	return &SendFileTool{
		workspace:    workspace,
		restrict:     restrict,
		maxFileBytes: maxFileBytes,
	}
}

func (t *SendFileTool) Name() string {
	return "send_file"
}

func (t *SendFileTool) Description() string {
	return "Send a local file to the user on the current chat channel."
}

func (t *SendFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the local file. Relative paths are resolved from workspace.",
			},
			"caption": map[string]interface{}{
				"type":        "string",
				"description": "Optional text to include with the file.",
			},
			"attachment_type": map[string]interface{}{
				"type":        "string",
				"description": "Optional attachment type: image or file.",
			},
			"filename": map[string]interface{}{
				"type":        "string",
				"description": "Optional custom filename shown to user.",
			},
			"mime_type": map[string]interface{}{
				"type":        "string",
				"description": "Optional mime type hint, for example image/png.",
			},
			"channel": map[string]interface{}{
				"type":        "string",
				"description": "Optional target channel override.",
			},
			"chat_id": map[string]interface{}{
				"type":        "string",
				"description": "Optional target chat id override.",
			},
		},
		"required": []string{"path"},
	}
}

func (t *SendFileTool) SetContext(channel, chatID string) {
	t.defaultChannel = channel
	t.defaultChatID = chatID
}

func (t *SendFileTool) SetSendCallback(callback SendFileCallback) {
	t.sendCallback = callback
}

func (t *SendFileTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	path, ok := args["path"].(string)
	if !ok || strings.TrimSpace(path) == "" {
		return ErrorResult("path is required")
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
		return ErrorResult("No target channel/chat specified")
	}
	if t.sendCallback == nil {
		return ErrorResult("File sending not configured")
	}

	resolvedPath, err := validatePath(path, t.workspace, t.restrict)
	if err != nil {
		return ErrorResult(err.Error())
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to stat file: %v", err))
	}
	if info.IsDir() {
		return ErrorResult("path points to a directory, expected a file")
	}
	if t.maxFileBytes > 0 && info.Size() > t.maxFileBytes {
		return ErrorResult(fmt.Sprintf("file too large: %d bytes (limit %d bytes)", info.Size(), t.maxFileBytes))
	}

	attachmentType, _ := args["attachment_type"].(string)
	if attachmentType == "" {
		attachmentType = detectAttachmentType(resolvedPath)
	}
	if attachmentType != "image" && attachmentType != "file" {
		attachmentType = "file"
	}

	fileName, _ := args["filename"].(string)
	if fileName == "" {
		fileName = filepath.Base(resolvedPath)
	}
	mimeType, _ := args["mime_type"].(string)
	caption, _ := args["caption"].(string)

	outbound := bus.OutboundMessage{
		Channel: channel,
		ChatID:  chatID,
		Content: caption,
		Attachments: []bus.Attachment{
			{
				Type:     attachmentType,
				Path:     resolvedPath,
				FileName: fileName,
				MIMEType: mimeType,
			},
		},
	}
	if err := t.sendCallback(outbound); err != nil {
		return ErrorResult(fmt.Sprintf("sending file: %v", err))
	}

	return SilentResult(fmt.Sprintf("File sent to %s:%s (%s)", channel, chatID, fileName))
}

func detectAttachmentType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp":
		return "image"
	default:
		return "file"
	}
}
