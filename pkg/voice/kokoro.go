package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// KokoroSynthesizer uses any OpenAI-compatible /v1/audio/speech endpoint
// (Kokoro, Piper, OpenAI, etc.) and also supports Chatterbox's native
// /synthesize endpoint for exaggeration and cfg_weight control.
type KokoroSynthesizer struct {
	apiBase      string
	voice        string
	model        string
	format       string
	speed        float64
	exaggeration float64
	cfgWeight    float64
	httpClient   *http.Client
}

// openaiRequest is the body for the standard /v1/audio/speech endpoint.
type openaiRequest struct {
	Model  string  `json:"model"`
	Input  string  `json:"input"`
	Voice  string  `json:"voice"`
	Format string  `json:"response_format,omitempty"`
	Speed  float64 `json:"speed,omitempty"`
}

// chatterboxRequest is the body for Chatterbox's native /synthesize endpoint.
// Used when model starts with "chatterbox" — gives access to emotion controls.
type chatterboxRequest struct {
	Text         string  `json:"text"`
	Voice        string  `json:"voice,omitempty"`
	Exaggeration float64 `json:"exaggeration"`
	CFGWeight    float64 `json:"cfg_weight"`
	Format       string  `json:"format,omitempty"`
}

// TTSProfile holds the full voice profile for the synthesizer.
type TTSProfile struct {
	APIBase      string
	Voice        string
	Model        string
	Format       string
	Speed        float64
	Exaggeration float64 // Chatterbox only: emotion expressiveness 0.0–1.0
	CFGWeight    float64 // Chatterbox only: voice guidance weight 0.0–1.0
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
	if p.Exaggeration == 0 {
		p.Exaggeration = 0.5
	}
	if p.CFGWeight == 0 {
		p.CFGWeight = 0.5
	}

	logger.InfoCF("voice", "Creating TTS synthesizer", map[string]interface{}{
		"api_base":     p.APIBase,
		"voice":        p.Voice,
		"model":        p.Model,
		"format":       p.Format,
		"speed":        p.Speed,
		"exaggeration": p.Exaggeration,
		"cfg_weight":   p.CFGWeight,
	})

	return &KokoroSynthesizer{
		apiBase:      p.APIBase,
		voice:        p.Voice,
		model:        p.Model,
		format:       p.Format,
		speed:        p.Speed,
		exaggeration: p.Exaggeration,
		cfgWeight:    p.CFGWeight,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Synthesize converts text to audio, writes it to a temp file, and returns the path.
// The caller must delete the file when done.
// isChatterbox returns true when the configured model targets the Chatterbox
// server, which exposes a richer /synthesize endpoint alongside the standard
// /v1/audio/speech one.
func (s *KokoroSynthesizer) isChatterbox() bool {
	return strings.HasPrefix(strings.ToLower(s.model), "chatterbox")
}

func (s *KokoroSynthesizer) Synthesize(ctx context.Context, text string) (string, error) {
	logger.InfoCF("voice", "Synthesizing speech", map[string]interface{}{
		"text_length":  len(text),
		"voice":        s.voice,
		"model":        s.model,
		"chatterbox":   s.isChatterbox(),
	})

	var (
		bodyBytes []byte
		url       string
		err       error
	)

	if s.isChatterbox() {
		// Chatterbox native endpoint — supports exaggeration and cfg_weight.
		url = s.apiBase + "/synthesize"
		bodyBytes, err = json.Marshal(chatterboxRequest{
			Text:         text,
			Voice:        s.voice,
			Exaggeration: s.exaggeration,
			CFGWeight:    s.cfgWeight,
			Format:       s.format,
		})
	} else {
		// Standard OpenAI-compatible endpoint.
		url = s.apiBase + "/v1/audio/speech"
		bodyBytes, err = json.Marshal(openaiRequest{
			Model:  s.model,
			Input:  text,
			Voice:  s.voice,
			Format: s.format,
			Speed:  s.speed,
		})
	}
	if err != nil {
		return "", fmt.Errorf("failed to marshal TTS request: %w", err)
	}

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
		return "", fmt.Errorf("TTS error (status %d): %s", resp.StatusCode, string(body))
	}

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

// IsAvailable checks if the TTS server is reachable.
// Chatterbox uses /health; all others use /v1/models.
func (s *KokoroSynthesizer) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	endpoint := "/v1/models"
	if s.isChatterbox() {
		endpoint = "/health"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", s.apiBase+endpoint, nil)
	if err != nil {
		return false
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		logger.DebugCF("voice", "TTS health check failed", map[string]interface{}{
			"error": err.Error(),
		})
		return false
	}
	defer resp.Body.Close()

	available := resp.StatusCode == http.StatusOK
	logger.DebugCF("voice", "TTS availability", map[string]interface{}{
		"available":   available,
		"status_code": resp.StatusCode,
		"endpoint":    endpoint,
	})
	return available
}
