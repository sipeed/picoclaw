package main

import (
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/voice"
)

// resolveSTTTranscriber resolves the STT transcriber using a 3-tier fallback strategy:
// 1. agents.defaults.stt_model → model_list lookup
// 2. providers.groq.api_key (backward compat)
// 3. groq/ prefix in model_list (backward compat)
func resolveSTTTranscriber(cfg *config.Config) voice.Transcriber {
	// 1. Resolve from agents.defaults.stt_model → model_list lookup
	if cfg.Agents.Defaults.STTModel != "" {
		for _, mc := range cfg.ModelList {
			if mc.ModelName == cfg.Agents.Defaults.STTModel && mc.APIKey != "" {
				protocol, modelID := providers.ExtractProtocol(mc.Model)
				apiBase := mc.APIBase
				if apiBase == "" {
					apiBase = getDefaultSTTBase(protocol)
				}
				return voice.NewOpenAICompatTranscriber(mc.APIKey, apiBase, modelID)
			}
		}
	}

	// 2. Backward compat: providers.groq.api_key
	if cfg.Providers.Groq.APIKey != "" {
		return voice.NewOpenAICompatTranscriber(
			cfg.Providers.Groq.APIKey,
			"https://api.groq.com/openai/v1",
			"whisper-large-v3",
		)
	}

	// 3. Backward compat: groq/ in model_list (no stt_model set)
	for _, mc := range cfg.ModelList {
		if strings.HasPrefix(mc.Model, "groq/") && mc.APIKey != "" {
			return voice.NewOpenAICompatTranscriber(
				mc.APIKey,
				"https://api.groq.com/openai/v1",
				"whisper-large-v3",
			)
		}
	}

	return nil
}
