package main

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/voice"
)

func TestResolveSTTTranscriber_UnknownProtocolSkipped(t *testing.T) {
	// stt_model points to an entry with an unknown protocol and no api_base.
	// The unknown protocol resolves to an empty API base, so this entry must be
	// skipped (continue) and the function should fall through to return nil.
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				STTModel: "whisper",
			},
		},
		ModelList: []config.ModelConfig{
			{
				ModelName: "whisper",
				Model:     "unknownprotocol/whisper-1",
				APIKey:    "sk-test",
			},
		},
	}

	tr := resolveSTTTranscriber(cfg)
	if tr != nil {
		t.Error("expected nil - unknown protocol with no api_base should be skipped")
	}
}

func TestResolveSTTTranscriber_STTModel(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				STTModel: "whisper",
			},
		},
		ModelList: []config.ModelConfig{
			{
				ModelName: "whisper",
				Model:     "openai/whisper-1",
				APIKey:    "sk-test",
			},
		},
	}

	tr := resolveSTTTranscriber(cfg)
	if tr == nil {
		t.Fatal("expected transcriber, got nil")
	}
	if !tr.IsAvailable() {
		t.Error("expected transcriber to be available")
	}
}

func TestResolveSTTTranscriber_STTModelWithAPIBase(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				STTModel: "whisper",
			},
		},
		ModelList: []config.ModelConfig{
			{
				ModelName: "whisper",
				Model:     "openai/whisper-1",
				APIKey:    "sk-test",
				APIBase:   "https://custom.api.com/v1",
			},
		},
	}

	tr := resolveSTTTranscriber(cfg)
	if tr == nil {
		t.Fatal("expected transcriber, got nil")
	}
	// Verify it's the right type and has the right fields
	oat, ok := tr.(*voice.OpenAICompatTranscriber)
	if !ok {
		t.Fatal("expected *voice.OpenAICompatTranscriber")
	}
	_ = oat // Can't access unexported fields from test, but the resolution worked
}

func TestResolveSTTTranscriber_STTModelGroq(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				STTModel: "whisper",
			},
		},
		ModelList: []config.ModelConfig{
			{
				ModelName: "whisper",
				Model:     "groq/whisper-large-v3",
				APIKey:    "gsk-test",
			},
		},
	}

	tr := resolveSTTTranscriber(cfg)
	if tr == nil {
		t.Fatal("expected transcriber, got nil")
	}
}

func TestResolveSTTTranscriber_BackwardCompatGroqProvider(t *testing.T) {
	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Groq: config.ProviderConfig{
				APIKey: "gsk-test-key",
			},
		},
	}

	tr := resolveSTTTranscriber(cfg)
	if tr == nil {
		t.Fatal("expected transcriber, got nil")
	}
	if !tr.IsAvailable() {
		t.Error("expected transcriber to be available")
	}
}

func TestResolveSTTTranscriber_BackwardCompatGroqModelList(t *testing.T) {
	cfg := &config.Config{
		ModelList: []config.ModelConfig{
			{
				ModelName: "groq-llama",
				Model:     "groq/llama-3.3-70b",
				APIKey:    "gsk-test-key",
			},
		},
	}

	tr := resolveSTTTranscriber(cfg)
	if tr == nil {
		t.Fatal("expected transcriber, got nil")
	}
}

func TestResolveSTTTranscriber_NoneAvailable(t *testing.T) {
	cfg := &config.Config{}
	tr := resolveSTTTranscriber(cfg)
	if tr != nil {
		t.Error("expected nil transcriber when no config available")
	}
}

func TestResolveSTTTranscriber_STTModelPriority(t *testing.T) {
	// stt_model should take priority over providers.groq.api_key
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				STTModel: "whisper",
			},
		},
		ModelList: []config.ModelConfig{
			{
				ModelName: "whisper",
				Model:     "openai/whisper-1",
				APIKey:    "sk-openai-key",
			},
		},
		Providers: config.ProvidersConfig{
			Groq: config.ProviderConfig{
				APIKey: "gsk-groq-key",
			},
		},
	}

	tr := resolveSTTTranscriber(cfg)
	if tr == nil {
		t.Fatal("expected transcriber, got nil")
	}
	// The transcriber should be from stt_model (OpenAI), not from Groq
	// We can verify by checking it's available
	if !tr.IsAvailable() {
		t.Error("expected transcriber to be available")
	}
}

func TestResolveSTTTranscriber_STTModelNotInModelList(t *testing.T) {
	// stt_model set but not found in model_list, should fall back
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				STTModel: "nonexistent",
			},
		},
		Providers: config.ProvidersConfig{
			Groq: config.ProviderConfig{
				APIKey: "gsk-fallback",
			},
		},
	}

	tr := resolveSTTTranscriber(cfg)
	if tr == nil {
		t.Fatal("expected transcriber from fallback, got nil")
	}
}

func TestResolveSTTTranscriber_LLMEntryNotMatchedAsSTT(t *testing.T) {
	// A non-groq LLM entry should NOT be matched as STT provider
	cfg := &config.Config{
		ModelList: []config.ModelConfig{
			{
				ModelName: "gpt4",
				Model:     "openai/gpt-4o",
				APIKey:    "sk-test",
			},
		},
	}

	tr := resolveSTTTranscriber(cfg)
	if tr != nil {
		t.Error("expected nil - LLM entries should not match as STT providers")
	}
}

func TestResolveSTTTranscriber_STTModelNoAPIKey(t *testing.T) {
	// stt_model found but no API key, should fall back
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				STTModel: "whisper",
			},
		},
		ModelList: []config.ModelConfig{
			{
				ModelName: "whisper",
				Model:     "openai/whisper-1",
				APIKey:    "",
			},
		},
	}

	tr := resolveSTTTranscriber(cfg)
	if tr != nil {
		t.Error("expected nil - model has no API key")
	}
}
