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

	"github.com/sipeed/picoclaw/pkg/providers/common"
)

const dashScopeTTSEndpoint = "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation"

type DashScopeTTSProvider struct {
	apiKey     string
	model      string
	voice      string
	workspace  string
	httpClient *http.Client
}

type dashScopeTTSAudioStream struct {
	io.ReadCloser
	fileExt     string
	contentType string
}

func (s *dashScopeTTSAudioStream) AudioFileMeta() (string, string) {
	return s.fileExt, s.contentType
}

type dashScopeTTSRequest struct {
	Model  string                `json:"model"`
	Input  dashScopeTTSInput     `json:"input"`
	Params dashScopeTTSParams    `json:"parameters,omitempty"`
}

type dashScopeTTSInput struct {
	Text string `json:"text"`
}

type dashScopeTTSParams struct {
	Voice  string `json:"voice,omitempty"`
	Format string `json:"format,omitempty"`
}

type dashScopeTTSResponse struct {
	Output struct {
		Audio struct {
			Data      string `json:"data"`
			URL       string `json:"url"`
			ExpiresAt int64  `json:"expires_at"`
		} `json:"audio"`
	} `json:"output"`
	RequestID string `json:"request_id"`
}

func NewDashScopeTTSProvider(apiKey, workspace, model, voice string) *DashScopeTTSProvider {
	client := common.NewHTTPClient("")
	client.Timeout = 60 * time.Second

	return &DashScopeTTSProvider{
		apiKey:     apiKey,
		model:      model,
		voice:      voice,
		workspace:  workspace,
		httpClient: client,
	}
}

func (t *DashScopeTTSProvider) Name() string {
	return "dashscope-tts"
}

func (t *DashScopeTTSProvider) Synthesize(ctx context.Context, text string) (io.ReadCloser, error) {
	// Step 1: Call DashScope TTS API
	reqBody := dashScopeTTSRequest{
		Model: t.model,
		Input: dashScopeTTSInput{Text: text},
	}
	if t.voice != "" {
		reqBody.Params = dashScopeTTSParams{
			Voice:  t.voice,
			Format: "wav",
		}
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("dashscope tts: marshal failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", dashScopeTTSEndpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("dashscope tts: create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	if t.workspace != "" {
		req.Header.Set("X-DashScope-WorkSpace", t.workspace)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dashscope tts: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("dashscope tts: API error %d: %s", resp.StatusCode, string(body))
	}

	var ttsResp dashScopeTTSResponse
	if err := json.NewDecoder(resp.Body).Decode(&ttsResp); err != nil {
		return nil, fmt.Errorf("dashscope tts: parse response failed: %w", err)
	}

	audioURL := strings.TrimSpace(ttsResp.Output.Audio.URL)
	if audioURL == "" {
		return nil, fmt.Errorf("dashscope tts: no audio URL in response")
	}

	// Step 2: Download audio from OSS
	dlReq, err := http.NewRequestWithContext(ctx, "GET", audioURL, nil)
	if err != nil {
		return nil, fmt.Errorf("dashscope tts: create download request failed: %w", err)
	}

	dlResp, err := t.httpClient.Do(dlReq)
	if err != nil {
		return nil, fmt.Errorf("dashscope tts: download audio failed: %w", err)
	}

	if dlResp.StatusCode != http.StatusOK {
		defer dlResp.Body.Close()
		body, _ := io.ReadAll(dlResp.Body)
		return nil, fmt.Errorf("dashscope tts: download audio HTTP %d: %s", dlResp.StatusCode, string(body))
	}

	audioBytes, err := io.ReadAll(dlResp.Body)
	dlResp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("dashscope tts: read audio failed: %w", err)
	}

	if len(audioBytes) == 0 {
		return nil, fmt.Errorf("dashscope tts: empty audio response")
	}

	return &dashScopeTTSAudioStream{
		ReadCloser:  io.NopCloser(bytes.NewReader(audioBytes)),
		fileExt:     ".wav",
		contentType: "audio/wav",
	}, nil
}

// extractWorkspaceFromBaseURL extracts the workspace ID from a Bailian maas URL.
// e.g. "https://llm-hubsv5i2n0jjbjb6.cn-beijing.maas.aliyuncs.com/..." → "llm-hubsv5i2n0jjbjb6"
func extractWorkspaceFromBaseURL(apiBase string) string {
	if apiBase == "" {
		return ""
	}
	u, err := url.Parse(apiBase)
	if err != nil {
		return ""
	}
	host := u.Hostname()
	// Match maas host pattern: <workspace-id>.<region>.maas.aliyuncs.com
	parts := strings.Split(host, ".")
	if len(parts) >= 4 && strings.HasSuffix(host, ".maas.aliyuncs.com") {
		return parts[0]
	}
	return ""
}
