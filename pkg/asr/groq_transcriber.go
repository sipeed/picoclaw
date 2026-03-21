package asr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type Transcriber interface {
	Name() string
	Transcribe(ctx context.Context, audioFilePath string) (*TranscriptionResponse, error)
	TranscribeData(ctx context.Context, data []byte, filename string) (*TranscriptionResponse, error)
}

type GroqTranscriber struct {
	apiKey     string
	apiBase    string
	httpClient *http.Client
}

func NewGroqTranscriber(apiKey string) *GroqTranscriber {
	logger.DebugCF("voice", "Creating Groq transcriber", map[string]any{"has_api_key": apiKey != ""})

	apiBase := "https://api.groq.com/openai/v1"
	return &GroqTranscriber{
		apiKey:  apiKey,
		apiBase: apiBase,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (t *GroqTranscriber) TranscribeData(ctx context.Context, data []byte, filename string) (*TranscriptionResponse, error) {
	logger.InfoCF("voice", "Starting memory transcription", map[string]any{"filename": filename, "bytes": len(data)})

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		logger.ErrorCF("voice", "Failed to create form file", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, bytes.NewReader(data)); err != nil {
		logger.ErrorCF("voice", "Failed to copy file content", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	if err = writer.WriteField("model", "whisper-large-v3-turbo"); err != nil {
		logger.ErrorCF("voice", "Failed to write model field", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to write model field: %w", err)
	}

	if err = writer.WriteField("response_format", "json"); err != nil {
		logger.ErrorCF("voice", "Failed to write response_format field", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to write response_format field: %w", err)
	}

	if err = writer.Close(); err != nil {
		logger.ErrorCF("voice", "Failed to close multipart writer", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return t.doRequest(ctx, &requestBody, writer.FormDataContentType(), int64(len(data)))
}

func (t *GroqTranscriber) Transcribe(ctx context.Context, audioFilePath string) (*TranscriptionResponse, error) {
	logger.InfoCF("voice", "Starting transcription", map[string]any{"audio_file": audioFilePath})

	audioFile, err := os.Open(audioFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open audio file: %w", err)
	}
	defer audioFile.Close()

	fileInfo, err := audioFile.Stat()
	if err != nil {
		return nil, err
	}

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	part, err := writer.CreateFormFile("file", filepath.Base(audioFilePath))
	if err != nil {
		return nil, err
	}
	io.Copy(part, audioFile)
	writer.WriteField("model", "whisper-large-v3")
	writer.WriteField("response_format", "json")
	writer.Close()

	return t.doRequest(ctx, &requestBody, writer.FormDataContentType(), fileInfo.Size())
}

func (t *GroqTranscriber) doRequest(ctx context.Context, requestBody *bytes.Buffer, contentType string, fileSize int64) (*TranscriptionResponse, error) {
	url := t.apiBase + "/audio/transcriptions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, requestBody)
	if err != nil {
		logger.ErrorCF("voice", "Failed to create request", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	logger.DebugCF("voice", "Sending transcription request to Groq API", map[string]any{
		"url":                url,
		"request_size_bytes": requestBody.Len(),
		"file_size_bytes":    fileSize,
	})

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
		logger.ErrorCF("voice", "API error", map[string]any{
			"status_code": resp.StatusCode,
			"response":    string(body),
		})
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	logger.DebugCF("voice", "Received response from Groq API", map[string]any{
		"status_code":         resp.StatusCode,
		"response_size_bytes": len(body),
	})

	var result TranscriptionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		logger.ErrorCF("voice", "Failed to unmarshal response", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	logger.InfoCF("voice", "Transcription completed successfully", map[string]any{
		"text_length":           len(result.Text),
		"language":              result.Language,
		"duration_seconds":      result.Duration,
		"transcription_preview": utils.Truncate(result.Text, 50),
	})

	return &result, nil
}

func (t *GroqTranscriber) Name() string {
	return "groq"
}

// DetectTranscriber inspects cfg and returns the appropriate Transcriber, or
// nil if no supported transcription provider is configured.
func DetectTranscriber(cfg *config.Config) Transcriber {
	// Direct Groq provider config takes priority.
	if key := cfg.Providers.Groq.APIKey; key != "" {
		return NewGroqTranscriber(key)
	}
	// Fall back to any model-list entry that uses the groq/ protocol or is explicitly named groq.
	for _, mc := range cfg.ModelList {
		if (strings.HasPrefix(mc.Model, "groq/") || mc.ModelName == "groq" || mc.Model == "whisper-large-v3-turbo") && mc.APIKey != "" {
			return NewGroqTranscriber(mc.APIKey)
		}
	}
	return nil
}
