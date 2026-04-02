package tts

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
)

func TestNewOpenAITTSProvider_APIBaseNormalization(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "empty base",
			input:  "",
			expect: "https://api.openai.com/v1/audio/speech",
		},
		{
			name:   "official host no path",
			input:  "https://api.openai.com",
			expect: "https://api.openai.com/v1/audio/speech",
		},
		{
			name:   "official host v1",
			input:  "https://api.openai.com/v1",
			expect: "https://api.openai.com/v1/audio/speech",
		},
		{
			name:   "official host v1 slash",
			input:  "https://api.openai.com/v1/",
			expect: "https://api.openai.com/v1/audio/speech",
		},
		{
			name:   "non-openai host preserves base path",
			input:  "https://proxy.example.com/base",
			expect: "https://proxy.example.com/base/audio/speech",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			provider := NewOpenAITTSProvider("key", tc.input, "", "")
			if provider.apiBase != tc.expect {
				t.Fatalf("apiBase mismatch: got %q, want %q", provider.apiBase, tc.expect)
			}
		})
	}
}

func TestOpenAITTSProvider_SynthesizeSuccess(t *testing.T) {
	t.Parallel()

	var gotPath string
	var gotAuth string
	var gotContentType string
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")

		bodyBytes, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		_ = json.Unmarshal(bodyBytes, &gotBody)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("audio-bytes"))
	}))
	defer server.Close()

	provider := NewOpenAITTSProvider("k123", server.URL, "", "")
	stream, err := provider.Synthesize(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Synthesize failed: %v", err)
	}
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("read stream failed: %v", err)
	}

	if gotPath != "/audio/speech" {
		t.Fatalf("request path mismatch: got %q", gotPath)
	}
	if gotAuth != "Bearer k123" {
		t.Fatalf("authorization mismatch: got %q", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Fatalf("content-type mismatch: got %q", gotContentType)
	}
	if gotBody["model"] != "tts-1" || gotBody["voice"] != "alloy" || gotBody["response_format"] != "opus" ||
		gotBody["input"] != "hello" {
		bodyJSON, _ := json.Marshal(gotBody)
		t.Fatalf("request body mismatch: %s", string(bodyJSON))
	}
	if string(data) != "audio-bytes" {
		t.Fatalf("response body mismatch: got %q", string(data))
	}
}

func TestOpenAITTSProvider_SynthesizeNon200(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("nope"))
	}))
	defer server.Close()

	provider := NewOpenAITTSProvider("k123", server.URL, "", "")
	_, err := provider.Synthesize(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "API error (status 500): nope") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewOpenAITTSProvider_UsesConfiguredModel(t *testing.T) {
	t.Parallel()

	provider := NewOpenAITTSProvider("key", "https://api.xiaomimimo.com/v1", "", "mimo-v2-tts")
	if provider.model != "mimo-v2-tts" {
		t.Fatalf("model mismatch: got %q, want %q", provider.model, "mimo-v2-tts")
	}
	if provider.apiBase != "https://api.xiaomimimo.com/v1/audio/speech" {
		t.Fatalf("apiBase mismatch: got %q", provider.apiBase)
	}
}

func TestDetectTTS_UsesMimoProviderForMimoModels(t *testing.T) {
	t.Parallel()

	provider := DetectTTS(&config.Config{
		Voice: config.VoiceConfig{TTSModelName: "mimo-tts"},
		ModelList: []*config.ModelConfig{
			{
				ModelName: "mimo-tts",
				Model:     "mimo/mimo-v2-tts",
				APIKeys:   config.SimpleSecureStrings("sk-mimo"),
			},
		},
	})

	ttsProvider, ok := provider.(*MimoTTSProvider)
	if !ok {
		t.Fatalf("DetectTTS() type = %T, want *MimoTTSProvider", provider)
	}
	if ttsProvider.model != "mimo-v2-tts" {
		t.Fatalf("model mismatch: got %q, want %q", ttsProvider.model, "mimo-v2-tts")
	}
	if ttsProvider.apiBase != "https://api.xiaomimimo.com/v1/chat/completions" {
		t.Fatalf("apiBase mismatch: got %q", ttsProvider.apiBase)
	}
}

type stubTTSProvider struct {
	name string
}

func (s stubTTSProvider) Name() string {
	return s.name
}

func (s stubTTSProvider) Synthesize(ctx context.Context, text string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("audio")), nil
}

func TestSynthesizeAndStore_UsesOggMetadataByDefault(t *testing.T) {
	t.Parallel()

	store := media.NewFileMediaStore()
	ref, err := SynthesizeAndStore(
		context.Background(),
		stubTTSProvider{name: "openai-tts"},
		store,
		"hello",
		"",
		"discord",
		"chat123",
	)
	if err != nil {
		t.Fatalf("SynthesizeAndStore failed: %v", err)
	}

	path, meta, err := store.ResolveWithMeta(ref)
	if err != nil {
		t.Fatalf("ResolveWithMeta failed: %v", err)
	}
	if meta.ContentType != "audio/ogg" {
		t.Fatalf("ContentType = %q, want %q", meta.ContentType, "audio/ogg")
	}
	if filepath.Ext(path) != ".ogg" {
		t.Fatalf("stored file extension = %q, want %q", filepath.Ext(path), ".ogg")
	}
	if filepath.Ext(meta.Filename) != ".ogg" {
		t.Fatalf("filename extension = %q, want %q", filepath.Ext(meta.Filename), ".ogg")
	}
}

func TestSynthesizeAndStore_UsesMp3MetadataForMimo(t *testing.T) {
	t.Parallel()

	store := media.NewFileMediaStore()
	ref, err := SynthesizeAndStore(
		context.Background(),
		stubTTSProvider{name: "mimo-tts"},
		store,
		"hello",
		"",
		"discord",
		"chat123",
	)
	if err != nil {
		t.Fatalf("SynthesizeAndStore failed: %v", err)
	}

	path, meta, err := store.ResolveWithMeta(ref)
	if err != nil {
		t.Fatalf("ResolveWithMeta failed: %v", err)
	}
	if meta.ContentType != "audio/mpeg" {
		t.Fatalf("ContentType = %q, want %q", meta.ContentType, "audio/mpeg")
	}
	if filepath.Ext(path) != ".mp3" {
		t.Fatalf("stored file extension = %q, want %q", filepath.Ext(path), ".mp3")
	}
	if filepath.Ext(meta.Filename) != ".mp3" {
		t.Fatalf("filename extension = %q, want %q", filepath.Ext(meta.Filename), ".mp3")
	}
}
