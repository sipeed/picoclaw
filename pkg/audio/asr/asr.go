package asr

import (
	"context"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type Transcriber interface {
	Name() string
	Transcribe(ctx context.Context, audioFilePath string) (*TranscriptionResponse, error)
}

type TranscriptionResponse struct {
	Text     string  `json:"text"`
	Language string  `json:"language,omitempty"`
	Duration float64 `json:"duration,omitempty"`
}

func supportsAudioTranscription(model string) bool {
	protocol, _ := providers.ExtractProtocol(model)

	switch protocol {
	case "openai", "azure", "azure-openai",
		"litellm", "openrouter", "groq", "zhipu", "gemini", "nvidia",
		"ollama", "moonshot", "shengsuanyun", "deepseek", "cerebras",
		"vivgrid", "volcengine", "vllm", "qwen", "qwen-intl", "qwen-international", "dashscope-intl",
		"qwen-us", "dashscope-us", "mistral", "avian", "minimax", "longcat", "modelscope", "novita",
		"coding-plan", "alibaba-coding", "qwen-coding":
		// These protocols all go through the OpenAI-compatible or Azure provider path in
		// providers.CreateProviderFromConfig, so they are the only ones that can supply
		// the audio media payload shape expected by NewAudioModelTranscriber.

		// TODO: Further restrict this by modelID, since not every model under these
		// protocols supports audio transcription.
		return true
	default:
		return false
	}
}

func supportsWhisperTranscription(model string) bool {
	protocol, _ := providers.ExtractProtocol(model)

	switch protocol {
	case "openai", "litellm", "openrouter", "groq", "zhipu", "gemini", "nvidia",
		"ollama", "moonshot", "shengsuanyun", "deepseek", "cerebras",
		"vivgrid", "volcengine", "vllm", "qwen", "qwen-intl", "qwen-international", "dashscope-intl",
		"qwen-us", "dashscope-us", "mistral", "avian", "minimax", "longcat", "modelscope", "novita",
		"coding-plan", "alibaba-coding", "qwen-coding", "mimo":
		return true
	default:
		return false
	}
}

func whisperModelID(modelCfg *config.ModelConfig) string {
	if modelCfg == nil || modelCfg.APIKey() == "" {
		return ""
	}

	if !supportsWhisperTranscription(modelCfg.Model) {
		return ""
	}

	_, modelID := providers.ExtractProtocol(strings.TrimSpace(modelCfg.Model))
	if strings.Contains(strings.ToLower(modelID), "whisper") {
		return modelID
	}
	return ""
}

func transcriberFromModelConfig(modelCfg *config.ModelConfig) Transcriber {
	if modelCfg == nil {
		return nil
	}

	protocol, _ := providers.ExtractProtocol(modelCfg.Model)
	if protocol == "elevenlabs" && modelCfg.APIKey() != "" {
		return NewElevenLabsTranscriber(modelCfg.APIKey(), modelCfg.APIBase)
	}
	if modelID := whisperModelID(modelCfg); modelID != "" {
		return NewWhisperTranscriber(modelCfg)
	}
	if supportsAudioTranscription(modelCfg.Model) {
		return NewAudioModelTranscriber(modelCfg)
	}
	return nil
}

func fallbackTranscriberFromModelConfig(modelCfg *config.ModelConfig) Transcriber {
	if modelCfg == nil {
		return nil
	}

	protocol, _ := providers.ExtractProtocol(modelCfg.Model)
	if protocol == "elevenlabs" && modelCfg.APIKey() != "" {
		return NewElevenLabsTranscriber(modelCfg.APIKey(), modelCfg.APIBase)
	}
	if modelID := whisperModelID(modelCfg); modelID != "" {
		return NewWhisperTranscriber(modelCfg)
	}
	return nil
}

// DetectTranscriber inspects cfg and returns the appropriate Transcriber, or
// nil if no supported transcription provider is configured.
func DetectTranscriber(cfg *config.Config) Transcriber {
	if cfg == nil {
		return nil
	}

	if modelName := strings.TrimSpace(cfg.Voice.ModelName); modelName != "" {
		modelCfg, err := cfg.GetModelConfig(modelName)
		if err == nil {
			if tr := transcriberFromModelConfig(modelCfg); tr != nil {
				return tr
			}
		}
	}

	// Fall back to compatibility scanning for legacy auto-detected ASR providers.
	for _, mc := range cfg.ModelList {
		if tr := fallbackTranscriberFromModelConfig(mc); tr != nil {
			return tr
		}
	}
	return nil
}
