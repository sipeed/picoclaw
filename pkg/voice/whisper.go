package voice

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
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type WhisperTranscriber struct {
	apiBase    string
	httpClient *http.Client
}

func NewWhisperTranscriber(apiBase string) *WhisperTranscriber {
	if apiBase == "" {
		apiBase = "http://localhost:8200"
	}
	
	logger.InfoCF("voice", "Creating Whisper transcriber", map[string]interface{}{
		"api_base": apiBase,
	})

	return &WhisperTranscriber{
		apiBase: apiBase,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (t *WhisperTranscriber) Transcribe(ctx context.Context, audioFilePath string) (*TranscriptionResponse, error) {
	logger.InfoCF("voice", "Starting Whisper transcription", map[string]interface{}{
		"audio_file": audioFilePath,
	})

	audioFile, err := os.Open(audioFilePath)
	if err != nil {
		logger.ErrorCF("voice", "Failed to open audio file", map[string]interface{}{
			"path":  audioFilePath,
			"error": err,
		})
		return nil, fmt.Errorf("failed to open audio file: %w", err)
	}
	defer audioFile.Close()

	fileInfo, err := audioFile.Stat()
	if err != nil {
		logger.ErrorCF("voice", "Failed to get file info", map[string]interface{}{
			"path":  audioFilePath,
			"error": err,
		})
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	logger.DebugCF("voice", "Audio file details", map[string]interface{}{
		"size_bytes": fileInfo.Size(),
		"file_name":  filepath.Base(audioFilePath),
	})

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("file", filepath.Base(audioFilePath))
	if err != nil {
		logger.ErrorCF("voice", "Failed to create form file", map[string]interface{}{
			"error": err,
		})
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	copied, err := io.Copy(part, audioFile)
	if err != nil {
		logger.ErrorCF("voice", "Failed to copy file content", map[string]interface{}{
			"error": err,
		})
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	logger.DebugCF("voice", "File copied to request", map[string]interface{}{
		"bytes_copied": copied,
	})

	if err := writer.Close(); err != nil {
		logger.ErrorCF("voice", "Failed to close multipart writer", map[string]interface{}{
			"error": err,
		})
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := t.apiBase + "/transcribe"
	req, err := http.NewRequestWithContext(ctx, "POST", url, &requestBody)
	if err != nil {
		logger.ErrorCF("voice", "Failed to create request", map[string]interface{}{
			"error": err,
		})
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	logger.DebugCF("voice", "Sending transcription request to Whisper API", map[string]interface{}{
		"url":                url,
		"request_size_bytes": requestBody.Len(),
		"file_size_bytes":    fileInfo.Size(),
	})

	resp, err := t.httpClient.Do(req)
	if err != nil {
		logger.ErrorCF("voice", "Failed to send request", map[string]interface{}{
			"error": err,
		})
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorCF("voice", "Failed to read response", map[string]interface{}{
			"error": err,
		})
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.ErrorCF("voice", "Whisper API error", map[string]interface{}{
			"status_code": resp.StatusCode,
			"response":    string(body),
		})
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	logger.DebugCF("voice", "Received response from Whisper API", map[string]interface{}{
		"status_code":         resp.StatusCode,
		"response_size_bytes": len(body),
	})

	var result TranscriptionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		logger.ErrorCF("voice", "Failed to unmarshal response", map[string]interface{}{
			"error": err,
		})
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	logger.InfoCF("voice", "Whisper transcription completed successfully", map[string]interface{}{
		"text_length":           len(result.Text),
		"language":              result.Language,
		"duration_seconds":      result.Duration,
		"transcription_preview": utils.Truncate(result.Text, 50),
	})

	return &result, nil
}

func (t *WhisperTranscriber) IsAvailable() bool {
	// Check if Whisper API is reachable
	resp, err := t.httpClient.Get(t.apiBase + "/health")
	if err != nil {
		logger.DebugCF("voice", "Whisper API health check failed", map[string]interface{}{
			"error": err.Error(),
		})
		return false
	}
	defer resp.Body.Close()
	
	available := resp.StatusCode == http.StatusOK
	logger.DebugCF("voice", "Whisper transcriber availability", map[string]interface{}{
		"available":   available,
		"status_code": resp.StatusCode,
	})
	return available
}
