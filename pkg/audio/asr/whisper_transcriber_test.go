package asr

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestWhisperTranscriberTranscribeDataUsesConfiguredModel(t *testing.T) {
	var gotModel string
	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if got := r.Header.Get("Authorization"); got != "Bearer sk-openai-test" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer sk-openai-test")
		}

		reader, err := r.MultipartReader()
		if err != nil {
			t.Fatalf("MultipartReader() error: %v", err)
		}

		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("NextPart() error: %v", err)
			}

			data, err := io.ReadAll(part)
			if err != nil {
				t.Fatalf("ReadAll() error: %v", err)
			}

			if part.FormName() == "model" {
				gotModel = string(data)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(TranscriptionResponse{Text: "hello from whisper"}); err != nil {
			t.Fatalf("Encode() error: %v", err)
		}
	}))
	defer server.Close()

	tr := NewWhisperTranscriber(&config.ModelConfig{
		Model:   "openai/whisper-1",
		APIBase: server.URL,
		APIKeys: config.SimpleSecureStrings("sk-openai-test"),
	})
	tr.httpClient = server.Client()

	resp, err := tr.TranscribeData(context.Background(), []byte("audio"), "clip.ogg")
	if err != nil {
		t.Fatalf("TranscribeData() error: %v", err)
	}
	if resp.Text != "hello from whisper" {
		t.Errorf("Text = %q, want %q", resp.Text, "hello from whisper")
	}
	if gotModel != "whisper-1" {
		t.Errorf("model field = %q, want %q", gotModel, "whisper-1")
	}
	if gotPath != "/audio/transcriptions" {
		t.Errorf("path = %q, want %q", gotPath, "/audio/transcriptions")
	}
}

func TestWhisperTranscriberUsesEndpointAPIBaseWithoutDoubleAppend(t *testing.T) {
	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(TranscriptionResponse{Text: "ok"}); err != nil {
			t.Fatalf("Encode() error: %v", err)
		}
	}))
	defer server.Close()

	tr := NewWhisperTranscriber(&config.ModelConfig{
		Model:   "groq/whisper-large-v3",
		APIBase: server.URL + "/audio/transcriptions",
		APIKeys: config.SimpleSecureStrings("sk-groq-test"),
	})
	tr.httpClient = server.Client()

	if _, err := tr.TranscribeData(context.Background(), []byte("audio"), "clip.ogg"); err != nil {
		t.Fatalf("TranscribeData() error: %v", err)
	}
	if gotPath != "/audio/transcriptions" {
		t.Errorf("path = %q, want %q", gotPath, "/audio/transcriptions")
	}
}

func TestWhisperTranscriberUsesOAuthTokenSource(t *testing.T) {
	var gotAuth string
	var gotModel string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")

		reader, err := r.MultipartReader()
		if err != nil {
			t.Fatalf("MultipartReader() error: %v", err)
		}
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("NextPart() error: %v", err)
			}
			data, err := io.ReadAll(part)
			if err != nil {
				t.Fatalf("ReadAll() error: %v", err)
			}
			if part.FormName() == "model" {
				gotModel = string(data)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(TranscriptionResponse{Text: "oauth transcription"}); err != nil {
			t.Fatalf("Encode() error: %v", err)
		}
	}))
	defer server.Close()

	tr := NewWhisperTranscriber(&config.ModelConfig{
		Model:      "openai/gpt-4o-transcribe",
		APIBase:    server.URL,
		AuthMethod: "oauth",
	})
	tr.httpClient = server.Client()
	tr.tokenSource = func() (string, error) {
		return "oauth-token", nil
	}

	resp, err := tr.TranscribeData(context.Background(), []byte("audio"), "clip.mp3")
	if err != nil {
		t.Fatalf("TranscribeData() error: %v", err)
	}
	if resp.Text != "oauth transcription" {
		t.Errorf("Text = %q, want %q", resp.Text, "oauth transcription")
	}
	if gotAuth != "Bearer oauth-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer oauth-token")
	}
	if gotModel != "gpt-4o-transcribe" {
		t.Errorf("model field = %q, want %q", gotModel, "gpt-4o-transcribe")
	}
}
