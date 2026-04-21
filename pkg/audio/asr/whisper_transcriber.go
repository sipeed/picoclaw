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
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type WhisperTranscriber struct {
	apiKey       string
	apiBase      string
	modelID      string
	providerName string
	httpClient   *http.Client
}

func NewWhisperTranscriber(modelCfg *config.ModelConfig) *WhisperTranscriber {
	if modelCfg == nil {
		return nil
	}

	protocol, modelID := providers.ExtractProtocol(modelCfg)
	if modelID == "" {
		modelID = strings.TrimSpace(modelCfg.Model)
	}

	tr := newWhisperTranscriber(
		modelCfg.APIKey(),
		providers.ResolveAPIBase(modelCfg),
		modelID,
		protocol,
	)
	if tr == nil {
		return nil
	}

	logger.DebugCF("voice", "Creating whisper transcriber", map[string]any{
		"api_base": tr.apiBase,
		"has_key":  tr.apiKey != "",
		"model":    tr.modelID,
		"provider": tr.providerName,
	})
	return tr
}

func NewGroqTranscriber(apiKey, modelID string) *WhisperTranscriber {
	return newWhisperTranscriber(apiKey, "https://api.groq.com/openai/v1", modelID, "groq")
}

func newWhisperTranscriber(apiKey, apiBase, modelID, providerName string) *WhisperTranscriber {
	if modelID == "" {
		return nil
	}
	if providerName == "" {
		providerName = "whisper"
	}
	return &WhisperTranscriber{
		apiKey:       apiKey,
		apiBase:      strings.TrimRight(apiBase, "/"),
		modelID:      modelID,
		providerName: providerName,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (t *WhisperTranscriber) transcriptionURL() string {
	base := strings.TrimRight(t.apiBase, "/")
	if strings.HasSuffix(base, "/audio/transcriptions") {
		return base
	}
	return base + "/audio/transcriptions"
}

func (t *WhisperTranscriber) TranscribeData(
	ctx context.Context,
	data []byte,
	filename string,
) (*TranscriptionResponse, error) {
	logger.InfoCF("voice", "Starting whisper transcription from memory", map[string]any{
		"bytes":    len(data),
		"filename": filename,
		"model":    t.modelID,
		"provider": t.providerName,
	})

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		logger.ErrorCF("voice", "Failed to create whisper form file", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, copyErr := io.Copy(part, bytes.NewReader(data)); copyErr != nil {
		logger.ErrorCF("voice", "Failed to copy whisper file content", map[string]any{"error": copyErr})
		return nil, fmt.Errorf("failed to copy file content: %w", copyErr)
	}

	if err = writer.WriteField("model", t.modelID); err != nil {
		logger.ErrorCF("voice", "Failed to write whisper model field", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to write model field: %w", err)
	}

	if err = writer.WriteField("response_format", "json"); err != nil {
		logger.ErrorCF("voice", "Failed to write whisper response_format field", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to write response_format field: %w", err)
	}

	if err = writer.Close(); err != nil {
		logger.ErrorCF("voice", "Failed to close whisper multipart writer", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return t.doRequest(ctx, &requestBody, writer.FormDataContentType(), int64(len(data)))
}

func (t *WhisperTranscriber) Transcribe(ctx context.Context, audioFilePath string) (*TranscriptionResponse, error) {
	logger.InfoCF("voice", "Starting whisper transcription", map[string]any{
		"audio_file": audioFilePath,
		"model":      t.modelID,
		"provider":   t.providerName,
	})

	audioFile, err := os.Open(audioFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open audio file %s: %w", audioFilePath, err)
	}
	defer audioFile.Close()

	fileInfo, err := audioFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat audio file %s: %w", audioFilePath, err)
	}

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("file", filepath.Base(audioFilePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, copyErr := io.Copy(part, audioFile); copyErr != nil {
		return nil, fmt.Errorf("failed to copy audio data: %w", copyErr)
	}

	if err = writer.WriteField("model", t.modelID); err != nil {
		return nil, fmt.Errorf("failed to write model field: %w", err)
	}

	if err = writer.WriteField("response_format", "json"); err != nil {
		return nil, fmt.Errorf("failed to write response_format field: %w", err)
	}

	if err = writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return t.doRequest(ctx, &requestBody, writer.FormDataContentType(), fileInfo.Size())
}

func (t *WhisperTranscriber) doRequest(
	ctx context.Context,
	requestBody *bytes.Buffer,
	contentType string,
	fileSize int64,
) (*TranscriptionResponse, error) {
	url := t.transcriptionURL()
	req, err := http.NewRequestWithContext(ctx, "POST", url, requestBody)
	if err != nil {
		logger.ErrorCF("voice", "Failed to create whisper request", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	if t.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+t.apiKey)
	}

	logger.DebugCF("voice", "Sending whisper transcription request", map[string]any{
		"file_size_bytes":    fileSize,
		"model":              t.modelID,
		"provider":           t.providerName,
		"request_size_bytes": requestBody.Len(),
		"url":                url,
	})

	resp, err := t.httpClient.Do(req)
	if err != nil {
		logger.ErrorCF("voice", "Failed to send whisper request", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorCF("voice", "Failed to read whisper response", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.ErrorCF("voice", "Whisper API error", map[string]any{
			"provider":    t.providerName,
			"response":    string(body),
			"status_code": resp.StatusCode,
		})
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result TranscriptionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		logger.ErrorCF("voice", "Failed to unmarshal whisper response", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	logger.InfoCF("voice", "Whisper transcription completed successfully", map[string]any{
		"duration_seconds":      result.Duration,
		"language":              result.Language,
		"provider":              t.providerName,
		"text_length":           len(result.Text),
		"transcription_preview": utils.Truncate(result.Text, 50),
	})

	return &result, nil
}

func (t *WhisperTranscriber) Name() string {
	return "whisper"
}
