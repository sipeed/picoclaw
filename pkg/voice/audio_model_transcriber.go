package voice

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type AudioModelTranscriber struct {
	provider providers.LLMProvider
	modelID  string
	prompt   string
}

const (
	defaultTranscriptionPrompt = "Transcribe this audio."
)

func NewAudioModelTranscriber(modelCfg *config.ModelConfig) *AudioModelTranscriber {
	if modelCfg == nil {
		return nil
	}

	logger.DebugCF("voice", "Creating audio model transcriber", map[string]any{
		"has_api_key": modelCfg.APIKey != "",
		"api_base":    modelCfg.APIBase,
		"model":       modelCfg.Model,
	})

	provider, modelID, err := providers.CreateProviderFromConfig(modelCfg)
	if err != nil {
		logger.ErrorCF("voice", "Failed to create audio model provider", map[string]any{"error": err})
		return nil
	}

	return &AudioModelTranscriber{
		provider: provider,
		modelID:  modelID,
		prompt:   defaultTranscriptionPrompt,
	}
}

func (t *AudioModelTranscriber) Transcribe(ctx context.Context, audioFilePath string) (*TranscriptionResponse, error) {
	logger.InfoCF("voice", "Starting audio model transcription", map[string]any{
		"audio_file": audioFilePath,
		"model":      t.modelID,
	})

	audioBytes, err := os.ReadFile(audioFilePath)
	if err != nil {
		logger.ErrorCF("voice", "Failed to read audio file", map[string]any{"path": audioFilePath, "error": err})
		return nil, fmt.Errorf("failed to read audio file: %w", err)
	}

	format, err := utils.AudioFormat(audioFilePath)
	if err != nil {
		logger.ErrorCF("voice", "Failed to detect audio format", map[string]any{"path": audioFilePath, "error": err})
		return nil, err
	}

	resp, err := t.provider.Chat(ctx, []providers.Message{
		{
			Role:    "user",
			Content: t.prompt,
			Media: []string{
				fmt.Sprintf("data:audio/%s;base64,%s", format, base64.StdEncoding.EncodeToString(audioBytes)),
			},
		},
	}, nil, t.modelID, map[string]any{
		"temperature": 0,
	})
	if err != nil {
		logger.ErrorCF("voice", "Audio model transcription request failed", map[string]any{"error": err})
		return nil, fmt.Errorf("transcription request failed: %w", err)
	}

	text := strings.TrimSpace(resp.Content)
	logger.InfoCF("voice", "Audio model transcription completed successfully", map[string]any{
		"text_length":           len(text),
		"transcription_preview": utils.Truncate(text, 50),
	})

	return &TranscriptionResponse{Text: text}, nil
}

func (t *AudioModelTranscriber) Name() string {
	return "audio-model"
}
