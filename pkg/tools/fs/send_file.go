package fstools

import (
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/h2non/filetype"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
)

// SendFileTool allows the LLM to send a local file (image, document, etc.)
// to the user on the current chat channel via the MediaStore pipeline.
type SendFileTool struct {
	workspace   string
	restrict    bool
	maxFileSize int
	mediaStore  media.MediaStore
	allowPaths  []*regexp.Regexp

	defaultChannel string
	defaultChatID  string
}

func NewSendFileTool(
	workspace string,
	restrict bool,
	maxFileSize int,
	store media.MediaStore,
	allowPaths ...[]*regexp.Regexp,
) *SendFileTool {
	if maxFileSize <= 0 {
		maxFileSize = config.DefaultMaxMediaSize
	}
	var patterns []*regexp.Regexp
	if len(allowPaths) > 0 {
		patterns = allowPaths[0]
	}
	return &SendFileTool{
		workspace:   workspace,
		restrict:    restrict,
		maxFileSize: maxFileSize,
		mediaStore:  store,
		allowPaths:  patterns,
	}
}

func (t *SendFileTool) Name() string { return "send_file" }
func (t *SendFileTool) Description() string {
	return "Send a local file (image, document, etc.) to the user on the current chat channel."
}

func (t *SendFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the local file. Relative paths are resolved from workspace.",
			},
			"filename": map[string]any{
				"type":        "string",
				"description": "Optional display filename. Defaults to the basename of path.",
			},
		},
		"required": []string{"path"},
	}
}

func (t *SendFileTool) SetContext(channel, chatID string) {
	t.defaultChannel = channel
	t.defaultChatID = chatID
}

func (t *SendFileTool) SetMediaStore(store media.MediaStore) {
	t.mediaStore = store
}

func (t *SendFileTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, _ := args["path"].(string)
	if strings.TrimSpace(path) == "" {
		return ErrorResult("path is required")
	}

	// Prefer context-injected channel/chatID (set by ExecuteWithContext), fall back to SetContext values.
	channel := ToolChannel(ctx)
	if channel == "" {
		channel = t.defaultChannel
	}
	chatID := ToolChatID(ctx)
	if chatID == "" {
		chatID = t.defaultChatID
	}
	if channel == "" || chatID == "" {
		return ErrorResult("no target channel/chat available")
	}

	if t.mediaStore == nil {
		return ErrorResult("media store not configured")
	}

	resolved, err := validatePathWithAllowPaths(path, t.workspace, t.restrict, t.allowPaths)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid path: %v", err))
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return ErrorResult(fmt.Sprintf("file not found: %v", err))
	}
	if info.IsDir() {
		return ErrorResult("path is a directory, expected a file")
	}
	if info.Size() > int64(t.maxFileSize) {
		return ErrorResult(fmt.Sprintf(
			"file too large: %d bytes (max %d bytes)",
			info.Size(), t.maxFileSize,
		))
	}

	filename, _ := args["filename"].(string)
	if filename == "" {
		filename = filepath.Base(resolved)
	}

	mediaType := detectMediaType(resolved)
	scope := fmt.Sprintf("tool:send_file:%s:%s", channel, chatID)

	ref, err := t.mediaStore.Store(resolved, media.MediaMeta{
		Filename:      filename,
		ContentType:   mediaType,
		Source:        "tool:send_file",
		CleanupPolicy: media.CleanupPolicyForgetOnly,
	}, scope)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to register media: %v", err))
	}

	return MediaResult(fmt.Sprintf("File %q sent to user", filename), []string{ref}).WithResponseHandled()
}

// detectMediaType determines the MIME type of a file.
// Uses magic-bytes detection (h2non/filetype) first, then falls back to
// extension-based lookup via mime.TypeByExtension.
func detectMediaType(path string) string {
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
