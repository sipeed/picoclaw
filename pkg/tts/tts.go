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

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type TTSProvider interface {
	Name() string
	Synthesize(ctx context.Context, text string) (io.ReadCloser, error)
}

type OpenAITTSProvider struct {
	apiKey     string
	apiBase    string
	voice      string
	model      string
	httpClient *http.Client
}

func NewOpenAITTSProvider(apiKey string, apiBase string, proxyURL string) *OpenAITTSProvider {
	if apiBase == "" || apiBase == "https://api.openai.com/v1" {
		apiBase = "https://api.openai.com/v1/audio/speech"
	} else if !strings.HasSuffix(apiBase, "/audio/speech") {
		// Just in case they provide openrouter base or standard base
		apiBase = strings.TrimSuffix(apiBase, "/") + "/audio/speech"
	}

	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	if proxyURL != "" {
		if pURL, err := url.Parse(proxyURL); err == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(pURL),
			}
		}
	}

	return &OpenAITTSProvider{
		apiKey:     apiKey,
		apiBase:    apiBase,
		voice:      "alloy",
		model:      "tts-1",
		httpClient: client,
	}
}

func (t *OpenAITTSProvider) Name() string {
	return "openai-tts"
}

func (t *OpenAITTSProvider) Synthesize(ctx context.Context, text string) (io.ReadCloser, error) {
	logger.InfoCF("voice-tts", "Starting TTS synthesis", map[string]any{"text_len": len(text)})

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

func DetectTTS(cfg *config.Config) TTSProvider {
	for _, mc := range cfg.ModelList {
		if strings.Contains(strings.ToLower(mc.ModelName), "tts") && mc.APIKey != "" {
			return NewOpenAITTSProvider(mc.APIKey, mc.APIBase, mc.Proxy)
		}
	}
	return nil
}
