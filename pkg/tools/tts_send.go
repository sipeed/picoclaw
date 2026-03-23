package tools

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/tts"
)

type SendTTSTool struct {
	provider   tts.TTSProvider
	mediaStore media.MediaStore
}

func NewSendTTSTool(provider tts.TTSProvider, store media.MediaStore) *SendTTSTool {
	return &SendTTSTool{
		provider:   provider,
		mediaStore: store,
	}
}

func (t *SendTTSTool) Name() string { return "send_tts" }

func (t *SendTTSTool) Description() string {
	return "Synthesize speech from text and send it as an audio file to the user."
}

func (t *SendTTSTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"text": map[string]any{
				"type":        "string",
				"description": "The text to synthesize into speech.",
			},
			"filename": map[string]any{
				"type":        "string",
				"description": "Optional filename for the audio file (e.g., response.ogg).",
			},
		},
		"required": []string{"text"},
	}
}

func (t *SendTTSTool) SetMediaStore(store media.MediaStore) {
	t.mediaStore = store
}

func (t *SendTTSTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	text, _ := args["text"].(string)
	text = strings.TrimSpace(text)
	if text == "" {
		return ErrorResult("text is required")
	}

	if t.provider == nil {
		return ErrorResult("tts provider is not configured")
	}
	if t.mediaStore == nil {
		return ErrorResult("media store not configured")
	}

	channel := ToolChannel(ctx)
	chatID := ToolChatID(ctx)
	if channel == "" || chatID == "" {
		return ErrorResult("no target channel/chat available")
	}

	stream, err := t.provider.Synthesize(ctx, text)
	if err != nil {
		return ErrorResult(fmt.Sprintf("tts synthesize failed: %v", err)).WithError(err)
	}
	defer stream.Close()

	err = os.MkdirAll(media.TempDir(), 0o755)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create media temp dir: %v", err)).WithError(err)
	}

	file, err := os.CreateTemp(media.TempDir(), "tts-*.ogg")
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create temp file: %v", err)).WithError(err)
	}
	defer file.Close()

	_, err = io.Copy(file, stream)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to write tts audio: %v", err)).WithError(err)
	}

	filename, _ := args["filename"].(string)
	filename = strings.TrimSpace(filename)
	if filename == "" {
		filename = fmt.Sprintf("tts-%d.ogg", time.Now().Unix())
	}
	if filepath.Ext(filename) == "" {
		filename += ".ogg"
	}

	scope := fmt.Sprintf("tool:send_tts:%s:%s:%d", channel, chatID, time.Now().UnixNano())
	ref, err := t.mediaStore.Store(file.Name(), media.MediaMeta{
		Filename:    filename,
		ContentType: "audio/ogg",
		Source:      "tool:send_tts",
	}, scope)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to register audio: %v", err)).WithError(err)
	}

	return MediaResult("TTS audio sent", []string{ref})
}
