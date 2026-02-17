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

// KokoroSynthesizer uses any OpenAI-compatible /v1/audio/speech endpoint
// (Kokoro, Piper, Chatterbox, OpenAI, etc.).
type KokoroSynthesizer struct {
	apiBase    string
	voice      string
	model      string
	format     string
	speed      float64
	httpClient *http.Client
}

type kokoroRequest struct {
	Model  string  `json:"model"`
	Input  string  `json:"input"`
	Voice  string  `json:"voice"`
	Format string  `json:"response_format,omitempty"`
	Speed  float64 `json:"speed,omitempty"`
}

// TTSProfile holds the full voice profile for the synthesizer.
type TTSProfile struct {
	APIBase string
	Voice   string
	Model   string
	Format  string
	Speed   float64
}

// NewKokoroSynthesizer creates a TTS client from a voice profile.
// Sensible defaults are applied for any zero-value field.
func NewKokoroSynthesizer(apiBase, voice string) *KokoroSynthesizer {
	return NewKokoroSynthesizerFromProfile(TTSProfile{
		APIBase: apiBase,
		Voice:   voice,
	})
}

// NewKokoroSynthesizerFromProfile creates a TTS client with full profile control.
func NewKokoroSynthesizerFromProfile(p TTSProfile) *KokoroSynthesizer {
	if p.APIBase == "" {
		p.APIBase = "http://localhost:8100"
	}
	if p.Voice == "" {
		p.Voice = "en_us-lessac-medium"
	}
	if p.Model == "" {
		p.Model = "tts-1"
	}
	if p.Format == "" {
		p.Format = "mp3"
	}
	if p.Speed == 0 {
		p.Speed = 1.0
	}

	logger.InfoCF("voice", "Creating TTS synthesizer", map[string]interface{}{
		"api_base": p.APIBase,
		"voice":    p.Voice,
		"model":    p.Model,
		"format":   p.Format,
		"speed":    p.Speed,
	})

	return &KokoroSynthesizer{
		apiBase: p.APIBase,
		voice:   p.Voice,
		model:   p.Model,
		format:  p.Format,
		speed:   p.Speed,
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
		Format: s.format,
		Speed:  s.speed,
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
	tmpFile, err := os.CreateTemp("", "picoclaw-tts-*."+s.format)
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
