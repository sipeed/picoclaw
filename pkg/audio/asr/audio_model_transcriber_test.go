package asr

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

var _ Transcriber = (*AudioModelTranscriber)(nil)

type fakeLLMProvider struct {
	chatFunc func(
		ctx context.Context,
		messages []providers.Message,
		tools []providers.ToolDefinition,
		model string,
		options map[string]any,
	) (*providers.LLMResponse, error)
}

func (p *fakeLLMProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	if p.chatFunc == nil {
		return nil, nil
	}
	return p.chatFunc(ctx, messages, tools, model, options)
}

func (p *fakeLLMProvider) GetDefaultModel() string {
	return ""
}

func TestAudioModelTranscriberName(t *testing.T) {
	tr := &AudioModelTranscriber{}
	if got := tr.Name(); got != "audio-model" {
		t.Errorf("Name() = %q, want %q", got, "audio-model")
	}
}

func TestNewAudioModelTranscriberInvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.ModelConfig
	}{
		{
			name: "nil config",
			cfg:  nil,
		},
		{
			name: "missing api key",
			cfg: &config.ModelConfig{
				Model: "gemini/gemini-2.5-flash",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tr := NewAudioModelTranscriber(tt.cfg); tr != nil {
				t.Fatalf("NewAudioModelTranscriber() = %#v, want nil", tr)
			}
		})
	}
}

func TestAudioModelTranscriberTranscribe(t *testing.T) {
	tmpDir := t.TempDir()
	audioPath := filepath.Join(tmpDir, "clip.ogg")
	audioData := []byte("fake-audio-data")
	if err := os.WriteFile(audioPath, audioData, 0o644); err != nil {
		t.Fatalf("failed to write fake audio file: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		tr := &AudioModelTranscriber{
			provider: &fakeLLMProvider{
				chatFunc: func(
					ctx context.Context,
					messages []providers.Message,
					tools []providers.ToolDefinition,
					model string,
					options map[string]any,
				) (*providers.LLMResponse, error) {
					if ctx == nil {
						t.Fatal("context should not be nil")
					}
					if tools != nil {
						t.Fatalf("tools = %#v, want nil", tools)
					}
					if model != "gemini-2.5-flash" {
						t.Fatalf("model = %q, want %q", model, "gemini-2.5-flash")
					}
					if len(messages) != 1 {
						t.Fatalf("len(messages) = %d, want 1", len(messages))
					}
					msg := messages[0]
					if msg.Role != "user" {
						t.Fatalf("role = %q, want %q", msg.Role, "user")
					}
					if msg.Content != defaultTranscriptionPrompt {
						t.Fatalf("prompt = %q, want %q", msg.Content, defaultTranscriptionPrompt)
					}
					if len(msg.Media) != 1 {
						t.Fatalf("len(media) = %d, want 1", len(msg.Media))
					}
					wantMedia := "data:audio/ogg;base64," + base64.StdEncoding.EncodeToString(audioData)
					if msg.Media[0] != wantMedia {
						t.Fatalf("media = %q, want %q", msg.Media[0], wantMedia)
					}
					if len(options) != 1 {
						t.Fatalf("options = %#v, want only temperature", options)
					}
					if got := options["temperature"]; got != 0 {
						t.Fatalf("temperature = %#v, want 0", got)
					}

					return &providers.LLMResponse{Content: "  hello from gemini \n"}, nil
				},
			},
			modelID: "gemini-2.5-flash",
			prompt:  defaultTranscriptionPrompt,
		}

		resp, err := tr.Transcribe(context.Background(), audioPath)
		if err != nil {
			t.Fatalf("Transcribe() error: %v", err)
		}
		if resp.Text != "hello from gemini" {
			t.Fatalf("Text = %q, want %q", resp.Text, "hello from gemini")
		}
	})

	t.Run("provider error", func(t *testing.T) {
		tr := &AudioModelTranscriber{
			provider: &fakeLLMProvider{
				chatFunc: func(
					ctx context.Context,
					messages []providers.Message,
					tools []providers.ToolDefinition,
					model string,
					options map[string]any,
				) (*providers.LLMResponse, error) {
					return nil, errors.New("upstream failure")
				},
			},
			modelID: "gemini-2.5-flash",
			prompt:  defaultTranscriptionPrompt,
		}

		_, err := tr.Transcribe(context.Background(), audioPath)
		if err == nil {
			t.Fatal("expected error for provider failure, got nil")
		}
		if got := err.Error(); got != "transcription request failed: upstream failure" {
			t.Fatalf("error = %q, want %q", got, "transcription request failed: upstream failure")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		tr := &AudioModelTranscriber{
			provider: &fakeLLMProvider{},
			modelID:  "gemini-2.5-flash",
			prompt:   defaultTranscriptionPrompt,
		}

		_, err := tr.Transcribe(context.Background(), filepath.Join(tmpDir, "nonexistent.ogg"))
		if err == nil {
			t.Fatal("expected error for missing file, got nil")
		}
	})

	t.Run("unsupported audio format", func(t *testing.T) {
		badPath := filepath.Join(tmpDir, "clip.txt")
		if err := os.WriteFile(badPath, []byte("not-audio"), 0o644); err != nil {
			t.Fatalf("failed to write fake file: %v", err)
		}

		tr := &AudioModelTranscriber{
			provider: &fakeLLMProvider{},
			modelID:  "gemini-2.5-flash",
			prompt:   defaultTranscriptionPrompt,
		}

		_, err := tr.Transcribe(context.Background(), badPath)
		if err == nil {
			t.Fatal("expected error for unsupported audio format, got nil")
		}
		if got := err.Error(); got != `unsupported audio format for "`+badPath+`"` {
			t.Fatalf("error = %q, want unsupported format error", got)
		}
	})
}
