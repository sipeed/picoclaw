package tools

import (
	"context"
	"strings"

	"github.com/sipeed/picoclaw/pkg/audio/tts"
	"github.com/sipeed/picoclaw/pkg/media"
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
				"description": "The text to synthesize into speech. NOTE: Reply in a highly concise, conversational, oral style suitable for text-to-speech. Do not use markdown, emojis, asterisks, or code blocks. Speak naturally.",
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

	channel := ToolChannel(ctx)
	chatID := ToolChatID(ctx)
	filename, _ := args["filename"].(string)

	ref, err := tts.SynthesizeAndStore(
		ctx,
		t.provider,
		t.mediaStore,
		text,
		filename,
		channel,
		chatID,
	)
	if err != nil {
		return ErrorResult(err.Error()).WithError(err)
	}

	// Return with ForUser set to original text, Media containing the audio ref,
	// and mark as ResponseHandled so the audio is sent immediately without LLM intervention.
	return &ToolResult{
		ForLLM:          "TTS audio sent",
		ForUser:         text,
		Media:           []string{ref},
		ResponseHandled: true,
	}
}
