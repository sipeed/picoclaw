package voice

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestDetectTranscriber(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		wantNil  bool
		wantName string
	}{
		{
			name:    "no config",
			cfg:     &config.Config{},
			wantNil: true,
		},
		{
			name: "voice model name selects audio model transcriber",
			cfg: &config.Config{
				Voice: config.VoiceConfig{ModelName: "voice-gemini"},
				ModelList: []*config.ModelConfig{
					{
						ModelName: "voice-gemini",
						Model:     "gemini/gemini-2.5-flash",
						APIKeys:   config.SimpleSecureStrings("sk-gemini-model"),
					},
				},
			},
			wantName: "audio-model",
		},
		{
			name: "groq via model list",
			cfg: &config.Config{
				ModelList: []*config.ModelConfig{
					{ModelName: "openai", Model: "openai/gpt-4o", APIKeys: config.SimpleSecureStrings("sk-openai")},
					{
						ModelName: "groq",
						Model:     "groq/llama-3.3-70b",
						APIKeys:   config.SimpleSecureStrings("sk-groq-model"),
					},
				},
			},
			wantName: "groq",
		},
		{
			name: "voice model name selects non-gemini audio model transcriber",
			cfg: &config.Config{
				Voice: config.VoiceConfig{ModelName: "voice-openai-audio"},
				ModelList: []*config.ModelConfig{
					{
						ModelName: "voice-openai-audio",
						Model:     "openai/gpt-4o-audio-preview",
						APIKeys:   config.SimpleSecureStrings("sk-openai"),
					},
				},
			},
			wantName: "audio-model",
		},
		{
			name: "voice model name selects azure audio model transcriber",
			cfg: &config.Config{
				Voice: config.VoiceConfig{ModelName: "voice-azure-audio"},
				ModelList: []*config.ModelConfig{
					{
						ModelName: "voice-azure-audio",
						Model:     "azure/my-audio-deployment", APIKeys: config.SimpleSecureStrings("sk-azure"),
						APIBase: "https://example.openai.azure.com",
					},
				},
			},
			wantName: "audio-model",
		},
		{
			name: "voice model name with non openai compatible protocol does not select audio model transcriber",
			cfg: &config.Config{
				Voice: config.VoiceConfig{ModelName: "voice-anthropic"},
				ModelList: []*config.ModelConfig{
					{
						ModelName: "voice-anthropic",
						Model:     "anthropic/claude-sonnet-4.6",
						APIKeys:   config.SimpleSecureStrings("sk-anthropic"),
					},
				},
			},
			wantNil: true,
		},
		{
			name: "groq model list entry without key is skipped",
			cfg: &config.Config{
				ModelList: []*config.ModelConfig{
					{Model: "groq/llama-3.3-70b"},
				},
			},
			wantNil: true,
		},
		{
			name: "provider key takes priority over model list",
			cfg: &config.Config{
				ModelList: []*config.ModelConfig{
					{
						ModelName: "groq",
						Model:     "groq/llama-3.3-70b",
						APIKeys:   config.SimpleSecureStrings("sk-groq-model"),
					},
				},
			},
			wantName: "groq",
		},
		{
			name: "missing voice model name config returns nil",
			cfg: &config.Config{
				Voice: config.VoiceConfig{ModelName: "missing"},
				ModelList: []*config.ModelConfig{
					{
						ModelName: "other",
						Model:     "gemini/gemini-2.5-flash",
						APIKeys:   config.SimpleSecureStrings("sk-other-model"),
					},
				},
			},
			wantNil: true,
		},
		{
			name: "elevenlabs voice config key",
			cfg: &config.Config{
				Voice: config.VoiceConfig{ElevenLabsAPIKey: "sk_elevenlabs_test"},
			},
			wantName: "elevenlabs",
		},
		{
			name: "elevenlabs takes priority over groq model list",
			cfg: &config.Config{
				Voice: config.VoiceConfig{ElevenLabsAPIKey: "sk_elevenlabs_test"},
				ModelList: []*config.ModelConfig{
					{
						ModelName: "groq",
						Model:     "groq/llama-3.3-70b",
						APIKeys:   config.SimpleSecureStrings("sk-groq-model"),
					},
				},
			},
			wantName: "elevenlabs",
		},
		{
			name: "voice model name takes priority over elevenlabs",
			cfg: &config.Config{
				Voice: config.VoiceConfig{
					ModelName:        "voice-gemini",
					ElevenLabsAPIKey: "sk_elevenlabs_test",
				},
				ModelList: []*config.ModelConfig{
					{
						ModelName: "voice-gemini",
						Model:     "gemini/gemini-2.5-flash",
						APIKeys:   config.SimpleSecureStrings("sk-gemini-model"),
					},
				},
			},
			wantName: "audio-model",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tr := DetectTranscriber(tc.cfg)
			if tc.wantNil {
				if tr != nil {
					t.Errorf("DetectTranscriber() = %v, want nil", tr)
				}
				return
			}
			if tr == nil {
				t.Fatal("DetectTranscriber() = nil, want non-nil")
			}
			if got := tr.Name(); got != tc.wantName {
				t.Errorf("Name() = %q, want %q", got, tc.wantName)
			}
		})
	}
}
