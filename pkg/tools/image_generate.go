package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/sipeed/picoclaw/pkg/media"
)

const (
	defaultImageGenerationSize = "1024x1024"
	maxImageGenerationResults  = 4
)

// ImageGenerateTool generates images through a provider adapter and returns
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

type imageGenerationProviderFactory func() imageGenerationProvider

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
	spec := parseImageGenerationModel(model)
	factory := imageGenerationProviderFactories[spec.Provider]
	if factory == nil {
		factory = imageGenerationProviderFactories[defaultImageGenerationProvider]
	}
	provider := factory()
	if spec.Model == "" && provider != nil {
		spec.Model = provider.DefaultModel()
	}
	tool := &ImageGenerateTool{
		workspace:  workspace,
		model:      spec.Model,
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

Use this when the user asks to create an image, infographic, diagram, poster, visual summary, or other generated raster artwork. The active image backend is selected from the configured image model provider prefix.`
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
