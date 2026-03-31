package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers/common"
)

type OpenAITTSProvider struct {
	apiKey     string
	apiBase    string
	voice      string
	model      string
	httpClient *http.Client
}

func NewOpenAITTSProvider(apiKey string, apiBase string, proxyURL string, model string) *OpenAITTSProvider {
	// Normalize apiBase to avoid malformed endpoints like
	// "https://api.openai.com/audio/speech" when "/v1" is required.
	if apiBase == "" {
		apiBase = "https://api.openai.com/v1/audio/speech"
	} else {
		if u, err := url.Parse(apiBase); err == nil && u.Scheme != "" && u.Host != "" {
			path := u.Path
			if u.Host == "api.openai.com" {
				// For the official OpenAI host, ensure exactly one /v1 prefix and
				// that the path ends with /audio/speech.
				if path == "" || path == "/" || path == "/v1" {
					path = "/v1/audio/speech"
				} else {
					if !strings.HasPrefix(path, "/") {
						path = "/" + path
					}
					if !strings.HasPrefix(path, "/v1/") {
						path = "/v1" + strings.TrimSuffix(path, "/")
					}
					if !strings.HasSuffix(path, "/audio/speech") {
						path = strings.TrimSuffix(path, "/") + "/audio/speech"
					}
				}
			} else {
				// For non-OpenAI hosts (e.g., proxies), preserve the existing base
				// path and only ensure it ends with /audio/speech.
				if !strings.HasSuffix(path, "/audio/speech") {
					path = strings.TrimSuffix(path, "/") + "/audio/speech"
				}
			}
			u.Path = path
			apiBase = u.String()
		} else {
			// Fallback to the previous string-based behavior if parsing fails.
			if apiBase == "https://api.openai.com/v1" {
				apiBase = "https://api.openai.com/v1/audio/speech"
			} else if !strings.HasSuffix(apiBase, "/audio/speech") {
				// Just in case they provide openrouter base or standard base
				apiBase = strings.TrimSuffix(apiBase, "/") + "/audio/speech"
			}
		}
	}

	client := common.NewHTTPClient(proxyURL)
	client.Timeout = 60 * time.Second

	model = strings.TrimSpace(model)
	if model == "" {
		model = "tts-1"
	}

	return &OpenAITTSProvider{
		apiKey:     apiKey,
		apiBase:    apiBase,
		voice:      "alloy",
		model:      model,
		httpClient: client,
	}
}

func (t *OpenAITTSProvider) Name() string {
	return "openai-tts"
}

func (t *OpenAITTSProvider) Synthesize(ctx context.Context, text string) (io.ReadCloser, error) {
	logger.DebugCF("voice-tts", "Starting TTS synthesis", map[string]any{"text_len": len(text)})

	reqBody := map[string]any{
		"model":           t.model,
		"input":           text,
		"voice":           t.voice,
		"response_format": "opus",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.apiBase, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return resp.Body, nil
}
