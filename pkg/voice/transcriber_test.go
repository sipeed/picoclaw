package voice

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// Ensure GroqTranscriber satisfies the Transcriber interface at compile time.
var _ Transcriber = (*GroqTranscriber)(nil)

func TestIsAvailable(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
		want   bool
	}{
		{"with key", "sk-test-key", true},
		{"empty key", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tr := NewGroqTranscriber(tc.apiKey)
			if got := tr.IsAvailable(); got != tc.want {
				t.Errorf("IsAvailable() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTranscribe(t *testing.T) {
	// Write a minimal fake audio file so the transcriber can open and send it.
	tmpDir := t.TempDir()
	audioPath := filepath.Join(tmpDir, "clip.ogg")
	if err := os.WriteFile(audioPath, []byte("fake-audio-data"), 0o644); err != nil {
		t.Fatalf("failed to write fake audio file: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/audio/transcriptions" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Header.Get("Authorization") != "Bearer sk-test" {
				t.Errorf("unexpected Authorization header: %s", r.Header.Get("Authorization"))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(TranscriptionResponse{
				Text:     "hello world",
				Language: "en",
				Duration: 1.5,
			})
		}))
		defer srv.Close()

		tr := NewGroqTranscriber("sk-test")
		tr.apiBase = srv.URL

		resp, err := tr.Transcribe(context.Background(), audioPath)
		if err != nil {
			t.Fatalf("Transcribe() error: %v", err)
		}
		if resp.Text != "hello world" {
			t.Errorf("Text = %q, want %q", resp.Text, "hello world")
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

		tr := NewGroqTranscriber("sk-bad")
		tr.apiBase = srv.URL

		_, err := tr.Transcribe(context.Background(), audioPath)
		if err == nil {
			t.Fatal("expected error for non-200 response, got nil")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		tr := NewGroqTranscriber("sk-test")
		_, err := tr.Transcribe(context.Background(), filepath.Join(tmpDir, "nonexistent.ogg"))
		if err == nil {
			t.Fatal("expected error for missing file, got nil")
		}
	})
}
