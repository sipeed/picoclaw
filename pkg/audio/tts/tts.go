package tts

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type TTSProvider interface {
	Name() string
	Synthesize(ctx context.Context, text string) (io.ReadCloser, error)
}

type ttsAudioMetaProvider interface {
	AudioFileMeta() (fileExt string, contentType string)
}

func providerFromModelConfig(mc *config.ModelConfig) TTSProvider {
	if mc == nil || mc.APIKey() == "" {
		return nil
	}

	protocol, modelID := providers.ExtractProtocol(mc)
	if modelID == "" {
		modelID = strings.TrimSpace(mc.Model)
	}

	switch protocol {
	case "mimo":
		return NewMimoTTSProvider(mc.APIKey(), providers.ResolveAPIBase(mc), modelID, mc.Proxy)
	case "dashscope":
		voice := ""
		if mc.ExtraBody != nil {
			if v, ok := mc.ExtraBody["voice"].(string); ok {
				voice = strings.TrimSpace(v)
			}
		}
		workspace := extractWorkspaceFromBaseURL(mc.APIBase)
		if workspace == "" {
			workspace = extractWorkspaceFromBaseURL(providers.ResolveAPIBase(mc))
		}
		return NewDashScopeTTSProvider(mc.APIKey(), workspace, modelID, voice)
	default:
		return NewOpenAITTSProviderWithOptions(
			mc.APIKey(),
			providers.ResolveAPIBase(mc),
			mc.Proxy,
			modelID,
			openAITTSOptionsFromModelConfig(mc),
		)
	}
}

func openAITTSOptionsFromModelConfig(mc *config.ModelConfig) OpenAITTSOptions {
	options := OpenAITTSOptions{}
	if mc == nil || mc.ExtraBody == nil {
		return options
	}

	if voice, ok := mc.ExtraBody["voice"].(string); ok {
		options.Voice = strings.TrimSpace(voice)
	}
	if responseFormat, ok := mc.ExtraBody["response_format"].(string); ok {
		options.ResponseFormat = strings.TrimSpace(responseFormat)
	}
	return options
}

func DetectTTS(cfg *config.Config) TTSProvider {
	if cfg == nil {
		return nil
	}

	if modelName := strings.TrimSpace(cfg.Voice.TTSModelName); modelName != "" {
		if mc, err := cfg.GetModelConfig(modelName); err == nil {
			if provider := providerFromModelConfig(mc); provider != nil {
				return provider
			}
		}
	}

	for _, mc := range cfg.ModelList {
		if strings.Contains(strings.ToLower(mc.Model), "tts") && mc.APIKey() != "" {
			if provider := providerFromModelConfig(mc); provider != nil {
				return provider
			}
		}
	}
	return nil
}

// SynthesizeAndStore synthesizes text to speech and registers it in the media store, returning the media reference.
func SynthesizeAndStore(
	ctx context.Context,
	provider TTSProvider,
	store media.MediaStore,
	text string,
	filename string,
	channel string,
	chatID string,
) (string, error) {
	if provider == nil {
		return "", fmt.Errorf("tts provider is not configured")
	}
	if store == nil {
		return "", fmt.Errorf("media store not configured")
	}
	if channel == "" || chatID == "" {
		return "", fmt.Errorf("no target channel/chat available")
	}
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("text is required")
	}

	stream, err := provider.Synthesize(ctx, text)
	if err != nil {
		return "", fmt.Errorf("tts synthesize failed: %w", err)
	}
	defer stream.Close()

	err = os.MkdirAll(media.TempDir(), 0o700)
	if err != nil {
		return "", fmt.Errorf("failed to create media temp dir: %w", err)
	}

	fileExt := ".ogg"
	contentType := "audio/ogg"
	if provider.Name() == "mimo-tts" {
		fileExt = ".mp3"
		contentType = "audio/mpeg"
	}
	if metaProvider, ok := stream.(ttsAudioMetaProvider); ok {
		if ext, ct := metaProvider.AudioFileMeta(); ext != "" && ct != "" {
			fileExt = ext
			contentType = ct
		}
	}

	file, err := os.CreateTemp(media.TempDir(), "tts-*"+fileExt)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(file.Name())
		}
	}()

	_, err = io.Copy(file, stream)
	if err != nil {
		_ = file.Close()
		return "", fmt.Errorf("failed to write tts audio: %w", err)
	}

	err = file.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close tts audio file: %w", err)
	}

	filename = strings.TrimSpace(filename)
	if filename == "" {
		filename = fmt.Sprintf("tts-%d%s", time.Now().Unix(), fileExt)
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		filename += fileExt
	} else if ext != fileExt {
		filename = strings.TrimSuffix(filename, filepath.Ext(filename)) + fileExt
	}

	scope := fmt.Sprintf("tool:send_tts:%s:%s:%d", channel, chatID, time.Now().UnixNano())
	ref, err := store.Store(file.Name(), media.MediaMeta{
		Filename:    filename,
		ContentType: contentType,
		Source:      "tool:send_tts",
	}, scope)
	if err != nil {
		return "", fmt.Errorf("failed to register audio: %w", err)
	}
	removeTemp = false

	return ref, nil
}
