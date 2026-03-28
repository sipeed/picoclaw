package asr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// Ensure ElevenLabsTranscriber satisfies the Transcriber interface at compile time.
var _ Transcriber = (*ElevenLabsTranscriber)(nil)

func TestElevenLabsTranscriberName(t *testing.T) {
	tr := NewElevenLabsTranscriber("sk_test", "")
	if got := tr.Name(); got != "elevenlabs" {
		t.Errorf("Name() = %q, want %q", got, "elevenlabs")
	}
}

func TestElevenLabsTranscribe(t *testing.T) {
	tmpDir := t.TempDir()
	audioPath := filepath.Join(tmpDir, "clip.ogg")
	if err := os.WriteFile(audioPath, []byte("fake-audio-data"), 0o644); err != nil {
		t.Fatalf("failed to write fake audio file: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/speech-to-text" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Header.Get("Xi-Api-Key") != "sk_test" {
				t.Errorf("unexpected xi-api-key header: %s", r.Header.Get("Xi-Api-Key"))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(TranscriptionResponse{
				Text:     "hello from elevenlabs",
				Language: "en",
			})
		}))
		defer srv.Close()

		tr := NewElevenLabsTranscriber("sk_test", "")
		tr.apiBase = srv.URL

		resp, err := tr.Transcribe(context.Background(), audioPath)
		if err != nil {
			t.Fatalf("Transcribe() error: %v", err)
		}
		if resp.Text != "hello from elevenlabs" {
			t.Errorf("Text = %q, want %q", resp.Text, "hello from elevenlabs")
		}
		if resp.Language != "en" {
			t.Errorf("Language = %q, want %q", resp.Language, "en")
		}
	})

	t.Run("api error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"error":"invalid_api_key"}`, http.StatusUnauthorized)
		}))
		defer srv.Close()

		tr := NewElevenLabsTranscriber("sk_bad", "")
		tr.apiBase = srv.URL

		_, err := tr.Transcribe(context.Background(), audioPath)
		if err == nil {
			t.Fatal("expected error for non-200 response, got nil")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		tr := NewElevenLabsTranscriber("sk_test", "")
		_, err := tr.Transcribe(context.Background(), filepath.Join(tmpDir, "nonexistent.ogg"))
		if err == nil {
			t.Fatal("expected error for missing file, got nil")
		}
	})
}
