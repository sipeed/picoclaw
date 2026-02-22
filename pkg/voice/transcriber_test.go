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

func TestNewOpenAICompatTranscriber(t *testing.T) {
	tr := NewOpenAICompatTranscriber("test-key", "https://api.example.com/v1", "whisper-1")
	if tr.apiKey != "test-key" {
		t.Errorf("expected apiKey 'test-key', got %q", tr.apiKey)
	}
	if tr.apiBase != "https://api.example.com/v1" {
		t.Errorf("expected apiBase 'https://api.example.com/v1', got %q", tr.apiBase)
	}
	if tr.model != "whisper-1" {
		t.Errorf("expected model 'whisper-1', got %q", tr.model)
	}
}

func TestOpenAICompatTranscriber_IsAvailable(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected bool
	}{
		{"with key", "test-key", true},
		{"empty key", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewOpenAICompatTranscriber(tt.apiKey, "https://example.com", "model")
			if got := tr.IsAvailable(); got != tt.expected {
				t.Errorf("IsAvailable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestOpenAICompatTranscriber_ImplementsInterface(t *testing.T) {
	var _ Transcriber = (*OpenAICompatTranscriber)(nil)
}

func TestOpenAICompatTranscriber_Transcribe(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it hits the right endpoint
		if r.URL.Path != "/audio/transcriptions" {
			t.Errorf("expected path /audio/transcriptions, got %s", r.URL.Path)
		}
		// Verify auth header
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		// Verify it's multipart
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("failed to parse multipart form: %v", err)
		}
		// Verify model field
		if model := r.FormValue("model"); model != "whisper-1" {
			t.Errorf("expected model 'whisper-1', got %q", model)
		}
		// Return mock response
		resp := TranscriptionResponse{
			Text:     "Hello world",
			Language: "en",
			Duration: 1.5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tr := NewOpenAICompatTranscriber("test-key", server.URL, "whisper-1")

	// Create a temp audio file
	tmpDir := t.TempDir()
	audioFile := filepath.Join(tmpDir, "test.ogg")
	if err := os.WriteFile(audioFile, []byte("fake audio data"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	result, err := tr.Transcribe(context.Background(), audioFile)
	if err != nil {
		t.Fatalf("Transcribe() error: %v", err)
	}
	if result.Text != "Hello world" {
		t.Errorf("expected text 'Hello world', got %q", result.Text)
	}
	if result.Language != "en" {
		t.Errorf("expected language 'en', got %q", result.Language)
	}
}

func TestOpenAICompatTranscriber_TranscribeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid api key"}`))
	}))
	defer server.Close()

	tr := NewOpenAICompatTranscriber("bad-key", server.URL, "whisper-1")

	tmpDir := t.TempDir()
	audioFile := filepath.Join(tmpDir, "test.ogg")
	os.WriteFile(audioFile, []byte("fake audio data"), 0644)

	_, err := tr.Transcribe(context.Background(), audioFile)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
