package integrationtools

import (
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/h2non/filetype"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
	fstools "github.com/sipeed/picoclaw/pkg/tools/fs"
)

type SendCallbackWithContext func(
	ctx context.Context,
	channel, chatID, content, replyToMessageID string,
	mediaParts []bus.MediaPart,
) error

type messageMediaArg struct {
	Path     string
	Type     string
	Filename string
}

// sentTarget records the channel+chatID that the message tool sent to.
type sentTarget struct {
	Channel string
	ChatID  string
}

type MessageTool struct {
	sendCallback SendCallbackWithContext
	workspace    string
	restrict     bool
	maxFileSize  int
	mediaStore   media.MediaStore
	allowPaths   []*regexp.Regexp
	mu           sync.Mutex
	sentTargets  map[string][]sentTarget
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
	return "Send a message to the user on a chat channel. Supports text-only, media-only, or text with media attachments."
}

func (t *MessageTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{
				"type":        "string",
				"description": "Optional message text. When media is present, this text is used as the caption/body for the media message.",
			},
			"media": map[string]any{
				"type":        "array",
				"description": "Optional local media attachments to send with the message.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "Path to the local file. Relative paths are resolved from workspace.",
						},
						"type": map[string]any{
							"type":        "string",
							"description": "Optional media type hint: image, audio, video, or file.",
						},
						"filename": map[string]any{
							"type":        "string",
							"description": "Optional display filename. Defaults to the basename of path.",
						},
					},
					"required": []string{"path"},
				},
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
		"anyOf": []map[string]any{
			{"required": []string{"content"}},
			{"required": []string{"media"}},
		},
	}
}

func (t *MessageTool) ConfigureLocalMedia(
	workspace string,
	restrict bool,
	maxFileSize int,
	allowPaths []*regexp.Regexp,
) {
	t.workspace = workspace
	t.restrict = restrict
	if maxFileSize <= 0 {
		maxFileSize = config.DefaultMaxMediaSize
	}
	t.maxFileSize = maxFileSize
	t.allowPaths = allowPaths
}

func (t *MessageTool) SetMediaStore(store media.MediaStore) {
	t.mediaStore = store
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
	content, _ := args["content"].(string)
	content = strings.TrimSpace(content)
	mediaArgs, err := parseMessageMediaArgs(args["media"])
	if err != nil {
		return &ToolResult{ForLLM: err.Error(), IsError: true}
	}
	if content == "" && len(mediaArgs) == 0 {
		return &ToolResult{ForLLM: "content or media is required", IsError: true}
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

	parts, err := t.buildMediaParts(channel, chatID, content, mediaArgs)
	if err != nil {
		return &ToolResult{ForLLM: err.Error(), IsError: true, Err: err}
	}

	if err := t.sendCallback(ctx, channel, chatID, content, replyToMessageID, parts); err != nil {
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

	status := fmt.Sprintf("Message sent to %s:%s", channel, chatID)
	if len(parts) > 0 {
		status = fmt.Sprintf("Message with %d media attachment(s) sent to %s:%s", len(parts), channel, chatID)
	}

	return &ToolResult{
		ForLLM: status,
		Silent: true,
	}
}

func parseMessageMediaArgs(raw any) ([]messageMediaArg, error) {
	if raw == nil {
		return nil, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("media must be an array")
	}
	result := make([]messageMediaArg, 0, len(items))
	for i, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("media[%d] must be an object", i)
		}
		path, _ := obj["path"].(string)
		path = strings.TrimSpace(path)
		if path == "" {
			return nil, fmt.Errorf("media[%d].path is required", i)
		}
		typ, _ := obj["type"].(string)
		filename, _ := obj["filename"].(string)
		result = append(result, messageMediaArg{
			Path:     path,
			Type:     strings.TrimSpace(typ),
			Filename: strings.TrimSpace(filename),
		})
	}
	return result, nil
}

func (t *MessageTool) buildMediaParts(
	channel, chatID, content string,
	mediaArgs []messageMediaArg,
) ([]bus.MediaPart, error) {
	if len(mediaArgs) == 0 {
		return nil, nil
	}
	if t.mediaStore == nil {
		return nil, fmt.Errorf("media store not configured")
	}
	if strings.TrimSpace(t.workspace) == "" {
		return nil, fmt.Errorf("message media delivery is not configured")
	}

	scope := fmt.Sprintf("tool:message:%s:%s", channel, chatID)
	parts := make([]bus.MediaPart, 0, len(mediaArgs))
	for i, item := range mediaArgs {
		resolved, err := fstools.ValidatePathWithAllowPaths(item.Path, t.workspace, t.restrict, t.allowPaths)
		if err != nil {
			return nil, fmt.Errorf("invalid media[%d].path: %w", i, err)
		}
		info, err := os.Stat(resolved)
		if err != nil {
			return nil, fmt.Errorf("media[%d] file not found: %w", i, err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("media[%d] path is a directory, expected a file", i)
		}
		if t.maxFileSize > 0 && info.Size() > int64(t.maxFileSize) {
			return nil, fmt.Errorf("media[%d] file too large: %d bytes (max %d bytes)", i, info.Size(), t.maxFileSize)
		}

		filename := item.Filename
		if filename == "" {
			filename = filepath.Base(resolved)
		}
		contentType := detectMessageMediaType(resolved)
		partType := normalizeMessageMediaType(item.Type, filename, contentType)
		ref, err := t.mediaStore.Store(resolved, media.MediaMeta{
			Filename:      filename,
			ContentType:   contentType,
			Source:        "tool:message",
			CleanupPolicy: media.CleanupPolicyForgetOnly,
		}, scope)
		if err != nil {
			return nil, fmt.Errorf("failed to register media[%d]: %w", i, err)
		}

		part := bus.MediaPart{
			Type:        partType,
			Ref:         ref,
			Filename:    filename,
			ContentType: contentType,
		}
		if i == 0 && content != "" {
			part.Caption = content
		}
		parts = append(parts, part)
	}
	return parts, nil
}

func detectMessageMediaType(path string) string {
	kind, err := filetype.MatchFile(path)
	if err == nil && kind != filetype.Unknown {
		return kind.MIME.Value
	}
	if ext := filepath.Ext(path); ext != "" {
		if t := mime.TypeByExtension(ext); t != "" {
			return t
		}
	}
	return "application/octet-stream"
}

func normalizeMessageMediaType(typeHint, filename, contentType string) string {
	switch strings.ToLower(strings.TrimSpace(typeHint)) {
	case "image", "audio", "video", "file":
		return strings.ToLower(strings.TrimSpace(typeHint))
	}

	ct := strings.ToLower(strings.TrimSpace(contentType))
	switch {
	case strings.HasPrefix(ct, "image/"):
		return "image"
	case strings.HasPrefix(ct, "audio/"):
		return "audio"
	case strings.HasPrefix(ct, "video/"):
		return "video"
	}

	switch strings.ToLower(filepath.Ext(filename)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp":
		return "image"
	case ".mp3", ".wav", ".ogg", ".oga", ".m4a", ".flac":
		return "audio"
	case ".mp4", ".mov", ".mkv", ".webm", ".avi":
		return "video"
	default:
		return "file"
	}
}
