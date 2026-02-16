package voice

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type Transcriber interface {
	Transcribe(ctx context.Context, audioFilePath string) (*TranscriptionResponse, error)
	IsAvailable() bool
}

type whisperTranscriber struct {
	apiKey       string
	apiBase      string
	model        string
	providerName string
	httpClient   *http.Client
}

type TranscriptionResponse struct {
	Text     string  `json:"text"`
	Language string  `json:"language,omitempty"`
	Duration float64 `json:"duration,omitempty"`
}

func NewGroqTranscriber(apiKey string) Transcriber {
	logger.DebugCF("voice", "Creating Groq transcriber", map[string]interface{}{"has_api_key": apiKey != ""})

	return &whisperTranscriber{
		apiKey:       apiKey,
		apiBase:      "https://api.groq.com/openai/v1",
		model:        "whisper-large-v3",
		providerName: "Groq",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type openRouterTranscriber struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewOpenRouterTranscriber(apiKey, model string) Transcriber {
	if model == "" {
		model = "google/gemini-2.5-flash"
	}
	logger.DebugCF("voice", "Creating OpenRouter transcriber", map[string]interface{}{
		"has_api_key": apiKey != "",
		"model":       model,
	})

	return &openRouterTranscriber{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func audioFormatFromExt(filePath string) string {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	switch ext {
	case "ogg", "oga":
		return "ogg"
	case "mp3":
		return "mp3"
	case "wav":
		return "wav"
	case "flac":
		return "flac"
	case "m4a", "aac":
		return "m4a"
	case "webm":
		return "webm"
	default:
		return "ogg"
	}
}

func (t *openRouterTranscriber) Transcribe(ctx context.Context, audioFilePath string) (*TranscriptionResponse, error) {
	logger.InfoCF("voice", "Starting OpenRouter transcription", map[string]interface{}{
		"audio_file": audioFilePath,
		"model":      t.model,
	})

	audioData, err := os.ReadFile(audioFilePath)
	if err != nil {
		logger.ErrorCF("voice", "Failed to read audio file", map[string]interface{}{"path": audioFilePath, "error": err})
		return nil, fmt.Errorf("failed to read audio file: %w", err)
	}

	b64Data := base64.StdEncoding.EncodeToString(audioData)
	audioFormat := audioFormatFromExt(audioFilePath)

	logger.DebugCF("voice", "Audio file details", map[string]interface{}{
		"size_bytes": len(audioData),
		"format":     audioFormat,
		"file_name":  filepath.Base(audioFilePath),
	})

	reqBody := map[string]interface{}{
		"model": t.model,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{"type": "text", "text": "Transcribe this audio. Return only the transcription text, nothing else."},
					{"type": "input_audio", "input_audio": map[string]string{
						"data":   b64Data,
						"format": audioFormat,
					}},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := "https://openrouter.ai/api/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	logger.DebugCF("voice", "Sending transcription request to OpenRouter", map[string]interface{}{
		"url":                url,
		"model":              t.model,
		"request_size_bytes": len(jsonBody),
	})

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.ErrorCF("voice", "OpenRouter API error", map[string]interface{}{
			"status_code": resp.StatusCode,
			"response":    string(body),
		})
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	text := strings.TrimSpace(chatResp.Choices[0].Message.Content)

	logger.InfoCF("voice", "Transcription completed successfully", map[string]interface{}{
		"text_length":           len(text),
		"transcription_preview": utils.Truncate(text, 50),
	})

	return &TranscriptionResponse{Text: text}, nil
}

func (t *openRouterTranscriber) IsAvailable() bool {
	available := t.apiKey != ""
	logger.DebugCF("voice", "Checking OpenRouter transcriber availability", map[string]interface{}{"available": available})
	return available
}

func (t *whisperTranscriber) Transcribe(ctx context.Context, audioFilePath string) (*TranscriptionResponse, error) {
	logger.InfoCF("voice", "Starting transcription", map[string]interface{}{"audio_file": audioFilePath})

	audioFile, err := os.Open(audioFilePath)
	if err != nil {
		logger.ErrorCF("voice", "Failed to open audio file", map[string]interface{}{"path": audioFilePath, "error": err})
		return nil, fmt.Errorf("failed to open audio file: %w", err)
	}
	defer audioFile.Close()

	fileInfo, err := audioFile.Stat()
	if err != nil {
		logger.ErrorCF("voice", "Failed to get file info", map[string]interface{}{"path": audioFilePath, "error": err})
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
		logger.ErrorCF("voice", "Failed to create form file", map[string]interface{}{"error": err})
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	copied, err := io.Copy(part, audioFile)
	if err != nil {
		logger.ErrorCF("voice", "Failed to copy file content", map[string]interface{}{"error": err})
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	logger.DebugCF("voice", "File copied to request", map[string]interface{}{"bytes_copied": copied})

	if err := writer.WriteField("model", t.model); err != nil {
		logger.ErrorCF("voice", "Failed to write model field", map[string]interface{}{"error": err})
		return nil, fmt.Errorf("failed to write model field: %w", err)
	}

	if err := writer.WriteField("response_format", "json"); err != nil {
		logger.ErrorCF("voice", "Failed to write response_format field", map[string]interface{}{"error": err})
		return nil, fmt.Errorf("failed to write response_format field: %w", err)
	}

	if err := writer.Close(); err != nil {
		logger.ErrorCF("voice", "Failed to close multipart writer", map[string]interface{}{"error": err})
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := t.apiBase + "/audio/transcriptions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, &requestBody)
	if err != nil {
		logger.ErrorCF("voice", "Failed to create request", map[string]interface{}{"error": err})
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	logger.DebugCF("voice", fmt.Sprintf("Sending transcription request to %s API", t.providerName), map[string]interface{}{
		"url":                url,
		"request_size_bytes": requestBody.Len(),
		"file_size_bytes":    fileInfo.Size(),
	})

	resp, err := t.httpClient.Do(req)
	if err != nil {
		logger.ErrorCF("voice", "Failed to send request", map[string]interface{}{"error": err})
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorCF("voice", "Failed to read response", map[string]interface{}{"error": err})
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.ErrorCF("voice", "API error", map[string]interface{}{
			"status_code": resp.StatusCode,
			"response":    string(body),
		})
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	logger.DebugCF("voice", fmt.Sprintf("Received response from %s API", t.providerName), map[string]interface{}{
		"status_code":         resp.StatusCode,
		"response_size_bytes": len(body),
	})

	var result TranscriptionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		logger.ErrorCF("voice", "Failed to unmarshal response", map[string]interface{}{"error": err})
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	logger.InfoCF("voice", "Transcription completed successfully", map[string]interface{}{
		"text_length":           len(result.Text),
		"language":              result.Language,
		"duration_seconds":      result.Duration,
		"transcription_preview": utils.Truncate(result.Text, 50),
	})

	return &result, nil
}

func (t *whisperTranscriber) IsAvailable() bool {
	available := t.apiKey != ""
	logger.DebugCF("voice", "Checking transcriber availability", map[string]interface{}{"available": available})
	return available
}
