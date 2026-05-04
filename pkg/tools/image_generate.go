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
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sipeed/picoclaw/pkg/media"
	oauthprovider "github.com/sipeed/picoclaw/pkg/providers/oauth"
)

const (
	defaultImageGenerationModel   = "gpt-image-2"
	defaultImageGenerationBaseURL = "https://chatgpt.com/backend-api/codex"
	defaultImageGenerationSize    = "1024x1024"
	defaultImageGenerationTimeout = 180 * time.Second
	maxImageGenerationResults     = 4
	maxImageGenerationSSEBytes    = 64 * 1024 * 1024
	maxImageGenerationEvents      = 512
)

// ImageGenerateTool generates images through OpenAI/Codex OAuth and returns
// generated files through the MediaStore outbound media pipeline.
type ImageGenerateTool struct {
	workspace  string
	model      string
	provider   imageGenerationProvider
	mediaStore media.MediaStore
}

type ImageGenerateToolOption func(*ImageGenerateTool)

type imageGenerationProvider interface {
	ID() string
	DefaultModel() string
	GenerateImages(ctx context.Context, req imageGenerationRequest) ([]generatedImage, error)
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

func WithImageGenerationProvider(provider imageGenerationProvider) ImageGenerateToolOption {
	return func(t *ImageGenerateTool) {
		if provider != nil {
			t.provider = provider
		}
	}
}

func NewImageGenerateTool(
	workspace string,
	model string,
	store media.MediaStore,
	options ...ImageGenerateToolOption,
) *ImageGenerateTool {
	if strings.TrimSpace(model) == "" {
		model = defaultImageGenerationModel
	}
	provider := newOpenAICodexImageGenerationProvider()
	tool := &ImageGenerateTool{
		workspace:  workspace,
		model:      stripProviderPrefix(model),
		provider:   provider,
		mediaStore: store,
	}
	for _, option := range options {
		option(tool)
	}
	return tool
}

func (t *ImageGenerateTool) SetMediaStore(store media.MediaStore) {
	t.mediaStore = store
}

func (t *ImageGenerateTool) Name() string { return "image_generate" }

func (t *ImageGenerateTool) Description() string {
	return `Generate an image from a prompt and send it to the current chat.

Uses OpenAI/Codex OAuth with gpt-image-2 by default. Use this when the user asks to create an image, infographic, diagram, poster, visual summary, or other generated raster artwork.`
}

func (t *ImageGenerateTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"prompt": map[string]any{
				"type":        "string",
				"description": "Image generation prompt.",
			},
			"size": map[string]any{
				"type":        "string",
				"description": "Output size. Defaults to 1024x1024. Supported examples: 1024x1024, 1536x1024, 1024x1536, 2048x2048, 3840x2160.",
			},
			"quality": map[string]any{
				"type":        "string",
				"enum":        []string{"low", "medium", "high", "auto"},
				"description": "Optional quality hint.",
			},
			"output_format": map[string]any{
				"type":        "string",
				"enum":        []string{"png", "jpeg", "webp"},
				"description": "Output image format. Defaults to png.",
			},
			"count": map[string]any{
				"type":        "integer",
				"description": "Number of images to generate, 1-4. Defaults to 1.",
			},
		},
		"required": []string{"prompt"},
	}
}

func (t *ImageGenerateTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	prompt, _ := args["prompt"].(string)
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return ErrorResult("prompt is required")
	}
	if t.mediaStore == nil {
		return ErrorResult("media store not configured")
	}
	if t.provider == nil {
		return ErrorResult("image generation provider not configured")
	}

	req := imageGenerationRequest{
		Prompt:       prompt,
		Model:        t.model,
		Size:         readStringDefault(args, "size", defaultImageGenerationSize),
		Quality:      readStringDefault(args, "quality", ""),
		OutputFormat: readStringDefault(args, "output_format", "png"),
		Count:        readImageCount(args["count"]),
	}
	if strings.TrimSpace(req.Model) == "" {
		req.Model = t.provider.DefaultModel()
	}
	images, err := t.provider.GenerateImages(ctx, req)
	if err != nil {
		return ErrorResult(fmt.Sprintf("image generation failed: %v", err)).WithError(err)
	}
	if len(images) == 0 {
		return ErrorResult("image generation returned no images")
	}

	refs := make([]string, 0, len(images))
	paths := make([]string, 0, len(images))
	scope := t.mediaScope(ctx)
	for i, image := range images {
		path, err := writeGeneratedImage(image, i)
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to write generated image: %v", err)).WithError(err)
		}
		ref, err := t.mediaStore.Store(path, media.MediaMeta{
			Filename:      filepath.Base(path),
			ContentType:   image.MimeType,
			Source:        "tool:image_generate",
			CleanupPolicy: media.CleanupPolicyDeleteOnCleanup,
		}, scope)
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to register generated image: %v", err)).WithError(err)
		}
		refs = append(refs, ref)
		paths = append(paths, path)
	}

	message := fmt.Sprintf("Generated %d image(s) with %s via %s.", len(refs), req.Model, t.provider.ID())
	result := MediaResult(message, refs).WithResponseHandled()
	result.ArtifactTags = make([]string, 0, len(paths))
	for _, path := range paths {
		result.ArtifactTags = append(result.ArtifactTags, "[file:"+path+"]")
	}
	return result
}

type imageGenerationRequest struct {
	Prompt       string
	Model        string
	Size         string
	Quality      string
	OutputFormat string
	Count        int
}

type generatedImage struct {
	Data     []byte
	MimeType string
	Ext      string
}

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

func writeGeneratedImage(image generatedImage, index int) (string, error) {
	dir, err := os.MkdirTemp("", "picoclaw-image-generate-*")
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("image-%d-%s.%s", index+1, uuid.NewString(), image.Ext)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, image.Data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func (t *ImageGenerateTool) mediaScope(ctx context.Context) string {
	parts := []string{"tool:image_generate"}
	if channel := ToolChannel(ctx); channel != "" {
		parts = append(parts, channel)
	}
	if chatID := ToolChatID(ctx); chatID != "" {
		parts = append(parts, chatID)
	}
	if sessionKey := ToolSessionKey(ctx); sessionKey != "" {
		parts = append(parts, sessionKey)
	}
	return strings.Join(parts, ":")
}

func readStringDefault(args map[string]any, key string, fallback string) string {
	value, _ := args[key].(string)
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func readImageCount(raw any) int {
	count := 1
	switch v := raw.(type) {
	case int:
		count = v
	case float64:
		count = int(v)
	case json.Number:
		if parsed, err := v.Int64(); err == nil {
			count = int(parsed)
		}
	}
	if count < 1 {
		return 1
	}
	if count > maxImageGenerationResults {
		return maxImageGenerationResults
	}
	return count
}

func stripProviderPrefix(model string) string {
	model = strings.TrimSpace(model)
	if strings.HasPrefix(model, "openai/") {
		return strings.TrimPrefix(model, "openai/")
	}
	if strings.HasPrefix(model, "openai-codex/") {
		return strings.TrimPrefix(model, "openai-codex/")
	}
	return model
}

func imageMimeAndExtension(outputFormat string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(outputFormat)) {
	case "jpeg", "jpg":
		return "image/jpeg", "jpg"
	case "webp":
		return "image/webp", "webp"
	default:
		return "image/png", "png"
	}
}
