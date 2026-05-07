package asr

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Ensure ElevenLabsTranscriber satisfies the Transcriber interface at compile time.
var _ Transcriber = (*ElevenLabsTranscriber)(nil)

func TestElevenLabsTranscriberName(t *testing.T) {
	tr := NewElevenLabsTranscriber("sk_test", "", "scribe_v1")
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
			mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil {
				t.Fatalf("ParseMediaType() error = %v", err)
			}
			if mediaType != "multipart/form-data" {
				t.Fatalf("content-type = %q, want multipart/form-data", mediaType)
			}
			reader := multipart.NewReader(r.Body, params["boundary"])
			var gotModelID string
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("NextPart() error = %v", err)
				}
				if part.FormName() != "model_id" {
					continue
				}
				body, err := io.ReadAll(part)
				if err != nil {
					t.Fatalf("ReadAll(part) error = %v", err)
				}
				gotModelID = strings.TrimSpace(string(body))
			}
			if gotModelID != "scribe_v1" {
				t.Fatalf("model_id = %q, want %q", gotModelID, "scribe_v1")
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(TranscriptionResponse{
				Text:     "hello from elevenlabs",
				Language: "en",
			})
		}))
		defer srv.Close()

		tr := NewElevenLabsTranscriber("sk_test", "", "scribe_v1")
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

		tr := NewElevenLabsTranscriber("sk_bad", "", "scribe_v1")
		tr.apiBase = srv.URL

		_, err := tr.Transcribe(context.Background(), audioPath)
		if err == nil {
			t.Fatal("expected error for non-200 response, got nil")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		tr := NewElevenLabsTranscriber("sk_test", "", "scribe_v1")
		_, err := tr.Transcribe(context.Background(), filepath.Join(tmpDir, "nonexistent.ogg"))
		if err == nil {
			t.Fatal("expected error for missing file, got nil")
		}
	})

	t.Run("unsupported model falls back to scribe_v1", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil {
				t.Fatalf("ParseMediaType() error = %v", err)
			}
			if mediaType != "multipart/form-data" {
				t.Fatalf("content-type = %q, want multipart/form-data", mediaType)
			}
			reader := multipart.NewReader(r.Body, params["boundary"])
			var gotModelID string
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("NextPart() error = %v", err)
				}
				if part.FormName() != "model_id" {
					continue
				}
				body, err := io.ReadAll(part)
				if err != nil {
					t.Fatalf("ReadAll(part) error = %v", err)
				}
				gotModelID = strings.TrimSpace(string(body))
			}
			if gotModelID != "scribe_v1" {
				t.Fatalf("model_id = %q, want runtime fallback to %q", gotModelID, "scribe_v1")
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(TranscriptionResponse{Text: "ok"})
		}))
		defer srv.Close()

		tr := NewElevenLabsTranscriber("sk_test", "", "unsupported-model")
		tr.apiBase = srv.URL

		if _, err := tr.Transcribe(context.Background(), audioPath); err != nil {
			t.Fatalf("Transcribe() error: %v", err)
		}
	})
}
