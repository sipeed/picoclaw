package tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	oauthprovider "github.com/sipeed/picoclaw/pkg/providers/oauth"
)

const (
	defaultImageGenerationBaseURL = "https://chatgpt.com/backend-api/codex"
	defaultImageGenerationTimeout = 180 * time.Second
	maxImageGenerationSSEBytes    = 64 * 1024 * 1024
	maxImageGenerationEvents      = 512
)

type openAICodexImageGenerationProvider struct {
	baseURL     string
	timeout     time.Duration
	httpClient  *http.Client
	tokenSource func() (accessToken, accountID string, err error)
}

func newOpenAICodexImageGenerationProvider() *openAICodexImageGenerationProvider {
	return &openAICodexImageGenerationProvider{
		baseURL:     defaultImageGenerationBaseURL,
		timeout:     defaultImageGenerationTimeout,
		httpClient:  http.DefaultClient,
		tokenSource: oauthprovider.CreateCodexTokenSource(),
	}
}

func WithImageGenerateBaseURL(baseURL string) ImageGenerateToolOption {
	return func(t *ImageGenerateTool) {
		if provider, ok := t.provider.(*openAICodexImageGenerationProvider); ok {
			provider.baseURL = baseURL
		}
	}
}

func WithImageGenerateHTTPClient(client *http.Client) ImageGenerateToolOption {
	return func(t *ImageGenerateTool) {
		if provider, ok := t.provider.(*openAICodexImageGenerationProvider); ok {
			provider.httpClient = client
		}
	}
}

func WithImageGenerateTokenSource(source func() (string, string, error)) ImageGenerateToolOption {
	return func(t *ImageGenerateTool) {
		if provider, ok := t.provider.(*openAICodexImageGenerationProvider); ok {
			provider.tokenSource = source
		}
	}
}

func (p *openAICodexImageGenerationProvider) ID() string { return "openai-codex" }

func (p *openAICodexImageGenerationProvider) DefaultModel() string {
	return defaultImageGenerationModel
}

func (p *openAICodexImageGenerationProvider) GenerateImages(
	ctx context.Context,
	req imageGenerationRequest,
) ([]generatedImage, error) {
	accessToken, accountID, err := p.tokenSource()
	if err != nil {
		return nil, fmt.Errorf("OpenAI/Codex OAuth not configured: %w", err)
	}

	timeout := p.timeout
	if timeout <= 0 {
		timeout = defaultImageGenerationTimeout
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	images := make([]generatedImage, 0, req.Count)
	for i := 0; i < req.Count; i++ {
		body, err := json.Marshal(buildCodexImageRequest(req))
		if err != nil {
			return nil, err
		}
		httpReq, err := http.NewRequestWithContext(
			callCtx,
			http.MethodPost,
			strings.TrimRight(p.baseURL, "/")+"/responses",
			bytes.NewReader(body),
		)
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Authorization", "Bearer "+accessToken)
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "text/event-stream")
		httpReq.Header.Set("originator", "codex_cli_rs")
		httpReq.Header.Set("OpenAI-Beta", "responses=experimental")
		if strings.TrimSpace(accountID) != "" {
			httpReq.Header.Set("Chatgpt-Account-Id", accountID)
		}

		resp, err := p.httpClient.Do(httpReq)
		if err != nil {
			return nil, err
		}
		eventImages, readErr := parseCodexImageSSE(resp.Body, req.OutputFormat)
		closeErr := resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("codex image request failed: HTTP %d", resp.StatusCode)
		}
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		images = append(images, eventImages...)
	}
	if len(images) > maxImageGenerationResults {
		images = images[:maxImageGenerationResults]
	}
	return images, nil
}

func buildCodexImageRequest(req imageGenerationRequest) map[string]any {
	tool := map[string]any{
		"type":  "image_generation",
		"model": req.Model,
		"size":  req.Size,
	}
	if req.Quality != "" {
		tool["quality"] = req.Quality
	}
	if req.OutputFormat != "" {
		tool["output_format"] = req.OutputFormat
	}
	return map[string]any{
		"model": "gpt-5.4",
		"input": []map[string]any{{
			"role": "user",
			"content": []map[string]any{{
				"type": "input_text",
				"text": req.Prompt,
			}},
		}},
		"instructions": "You are an image generation assistant.",
		"tools":        []map[string]any{tool},
		"tool_choice":  map[string]any{"type": "image_generation"},
		"stream":       true,
		"store":        false,
	}
}

func parseCodexImageSSE(r io.Reader, outputFormat string) ([]generatedImage, error) {
	reader := bufio.NewReader(r)
	var totalBytes int
	var events int
	var images []generatedImage
	var completedImages []generatedImage

	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			totalBytes += len(line)
			if totalBytes > maxImageGenerationSSEBytes {
				return nil, fmt.Errorf("codex image response exceeded size limit")
			}
			if strings.HasPrefix(line, "data: ") {
				events++
				if events > maxImageGenerationEvents {
					return nil, fmt.Errorf("codex image response exceeded event limit")
				}
				eventImages, eventCompletedImages, parseErr := parseCodexImageEvent(
					strings.TrimSpace(strings.TrimPrefix(line, "data: ")),
					outputFormat,
				)
				if parseErr != nil {
					return nil, parseErr
				}
				images = append(images, eventImages...)
				completedImages = append(completedImages, eventCompletedImages...)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	if len(images) > 0 {
		return images, nil
	}
	return completedImages, nil
}

func parseCodexImageEvent(data string, outputFormat string) ([]generatedImage, []generatedImage, error) {
	if data == "" || data == "[DONE]" {
		return nil, nil, nil
	}
	var event map[string]any
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return nil, nil, nil
	}
	eventType, _ := event["type"].(string)
	if eventType == "response.failed" || eventType == "error" {
		return nil, nil, fmt.Errorf("codex image generation failed")
	}

	var itemImages []generatedImage
	if eventType == "response.output_item.done" {
		if item, _ := event["item"].(map[string]any); item != nil {
			if image, ok, err := imageFromCodexItem(item, outputFormat); err != nil {
				return nil, nil, err
			} else if ok {
				itemImages = append(itemImages, image)
			}
		}
	}

	var completedImages []generatedImage
	if eventType == "response.completed" {
		if response, _ := event["response"].(map[string]any); response != nil {
			if output, _ := response["output"].([]any); output != nil {
				for _, raw := range output {
					item, _ := raw.(map[string]any)
					if image, ok, err := imageFromCodexItem(item, outputFormat); err != nil {
						return nil, nil, err
					} else if ok {
						completedImages = append(completedImages, image)
					}
				}
			}
		}
	}
	return itemImages, completedImages, nil
}

func imageFromCodexItem(item map[string]any, outputFormat string) (generatedImage, bool, error) {
	if item == nil || item["type"] != "image_generation_call" {
		return generatedImage{}, false, nil
	}
	payload, _ := item["result"].(string)
	if payload == "" {
		return generatedImage{}, false, nil
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return generatedImage{}, false, err
	}
	mime, ext := imageMimeAndExtension(outputFormat)
	return generatedImage{Data: data, MimeType: mime, Ext: ext}, true, nil
}
