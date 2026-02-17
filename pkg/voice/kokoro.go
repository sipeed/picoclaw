package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// KokoroSynthesizer uses a Kokoro TTS server (OpenAI-compatible /v1/audio/speech API).
type KokoroSynthesizer struct {
	apiBase    string
	voice      string
	model      string
	httpClient *http.Client
}

type kokoroRequest struct {
	Model  string `json:"model"`
	Input  string `json:"input"`
	Voice  string `json:"voice"`
	Format string `json:"response_format,omitempty"`
}

// NewKokoroSynthesizer creates a Kokoro TTS client.
// apiBase defaults to "http://localhost:8102".
// voice defaults to "af_nova".
func NewKokoroSynthesizer(apiBase, voice string) *KokoroSynthesizer {
	if apiBase == "" {
		apiBase = "http://localhost:8102"
	}
	if voice == "" {
		voice = "af_nova"
	}

	logger.InfoCF("voice", "Creating Kokoro TTS synthesizer", map[string]interface{}{
		"api_base": apiBase,
		"voice":    voice,
	})

	return &KokoroSynthesizer{
		apiBase: apiBase,
		voice:   voice,
		model:   "kokoro",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Synthesize converts text to audio, writes it to a temp file, and returns the path.
// The caller must delete the file when done.
func (s *KokoroSynthesizer) Synthesize(ctx context.Context, text string) (string, error) {
	logger.InfoCF("voice", "Synthesizing speech", map[string]interface{}{
		"text_length": len(text),
		"voice":       s.voice,
	})

	reqBody := kokoroRequest{
		Model:  s.model,
		Input:  text,
		Voice:  s.voice,
		Format: "mp3",
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal TTS request: %w", err)
	}

	url := s.apiBase + "/v1/audio/speech"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create TTS request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("TTS request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Kokoro TTS error (status %d): %s", resp.StatusCode, string(body))
	}

	// Write audio to temp file
	tmpFile, err := os.CreateTemp("", "picoclaw-tts-*.mp3")
	if err != nil {
		return "", fmt.Errorf("failed to create temp audio file: %w", err)
	}
	defer tmpFile.Close()

	written, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write TTS audio: %w", err)
	}

	logger.InfoCF("voice", "Speech synthesized successfully", map[string]interface{}{
		"path":       tmpFile.Name(),
		"size_bytes": written,
		"voice":      s.voice,
	})

	return tmpFile.Name(), nil
}

// IsAvailable checks if the Kokoro TTS server is reachable.
func (s *KokoroSynthesizer) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", s.apiBase+"/v1/models", nil)
	if err != nil {
		return false
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		logger.DebugCF("voice", "Kokoro TTS health check failed", map[string]interface{}{
			"error": err.Error(),
		})
		return false
	}
	defer resp.Body.Close()

	available := resp.StatusCode == http.StatusOK
	logger.DebugCF("voice", "Kokoro TTS availability", map[string]interface{}{
		"available":   available,
		"status_code": resp.StatusCode,
	})
	return available
}
