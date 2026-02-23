package voice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type DeepgramTranscriber struct {
	apiKey     string
	httpClient *http.Client
}

type deepgramResponse struct {
	Results struct {
		Channels []struct {
			Alternatives []struct {
				Transcript string `json:"transcript"`
			} `json:"alternatives"`
			DetectedLanguage string `json:"detected_language"`
		} `json:"channels"`
	} `json:"results"`
	Metadata struct {
		Duration float64 `json:"duration"`
	} `json:"metadata"`
}

func NewDeepgramTranscriber(apiKey string) *DeepgramTranscriber {
	logger.DebugCF("voice", "Creating Deepgram transcriber", map[string]any{"has_api_key": apiKey != ""})

	return &DeepgramTranscriber{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (t *DeepgramTranscriber) Transcribe(ctx context.Context, audioFilePath string) (*TranscriptionResponse, error) {
	logger.InfoCF("voice", "Starting Deepgram transcription", map[string]any{"audio_file": audioFilePath})

	audioFile, err := os.Open(audioFilePath)
	if err != nil {
		logger.ErrorCF("voice", "Failed to open audio file", map[string]any{"path": audioFilePath, "error": err})
		return nil, fmt.Errorf("failed to open audio file: %w", err)
	}
	defer audioFile.Close()

	url := "https://api.deepgram.com/v1/listen?model=nova-2&smart_format=true&detect_language=true"
	req, err := http.NewRequestWithContext(ctx, "POST", url, audioFile)
	if err != nil {
		logger.ErrorCF("voice", "Failed to create request", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Token "+t.apiKey)
	req.Header.Set("Content-Type", "audio/ogg")

	logger.DebugCF("voice", "Sending transcription request to Deepgram API", map[string]any{"url": url})

	resp, err := t.httpClient.Do(req)
	if err != nil {
		logger.ErrorCF("voice", "Failed to send request", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorCF("voice", "Failed to read response", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.ErrorCF("voice", "Deepgram API error", map[string]any{
			"status_code": resp.StatusCode,
			"response":    string(body),
		})
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var dgResp deepgramResponse
	if err := json.Unmarshal(body, &dgResp); err != nil {
		logger.ErrorCF("voice", "Failed to unmarshal Deepgram response", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	transcript := ""
	language := ""
	if len(dgResp.Results.Channels) > 0 {
		ch := dgResp.Results.Channels[0]
		if len(ch.Alternatives) > 0 {
			transcript = ch.Alternatives[0].Transcript
		}
		language = ch.DetectedLanguage
	}

	result := &TranscriptionResponse{
		Text:     transcript,
		Language: language,
		Duration: dgResp.Metadata.Duration,
	}

	logger.InfoCF("voice", "Deepgram transcription completed", map[string]any{
		"text_length":           len(result.Text),
		"language":              result.Language,
		"duration_seconds":      result.Duration,
		"transcription_preview": utils.Truncate(result.Text, 50),
	})

	return result, nil
}

func (t *DeepgramTranscriber) IsAvailable() bool {
	available := t.apiKey != ""
	logger.DebugCF("voice", "Checking Deepgram transcriber availability", map[string]any{"available": available})
	return available
}
