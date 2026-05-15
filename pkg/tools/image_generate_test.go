package tools

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type fakeImageGenerationProvider struct {
	id           string
	defaultModel string
	request      providers.ImageGenerationRequest
}

func (p *fakeImageGenerationProvider) SupportsImageGeneration() bool { return true }

func (p *fakeImageGenerationProvider) ImageGenerationProviderID() string { return p.id }

func (p *fakeImageGenerationProvider) DefaultImageGenerationModel() string { return p.defaultModel }

func (p *fakeImageGenerationProvider) GenerateImage(
	_ context.Context,
	req providers.ImageGenerationRequest,
) (*providers.ImageGenerationResponse, error) {
	p.request = req
	return &providers.ImageGenerationResponse{Images: []providers.GeneratedImage{{
		Data:     []byte("fake-image"),
		MimeType: "image/png",
		Ext:      "png",
	}}}, nil
}

func TestImageGenerateToolCanUseInjectedProvider(t *testing.T) {
	store := media.NewFileMediaStore()
	provider := &fakeImageGenerationProvider{
		id:           "test-provider",
		defaultModel: "test-default-image-model",
	}
	tool := NewImageGenerateTool(
		t.TempDir(),
		"custom-image-model",
		store,
		WithImageGenerationProvider(provider),
	)

	result := tool.Execute(
		WithToolContext(t.Context(), "telegram", "chat-1"),
		map[string]any{"prompt": "make a tiny icon"},
	)
	if result.IsError {
		t.Fatalf("Execute returned error: %s", result.ContentForLLM())
	}
	if provider.request.Model != "custom-image-model" {
		t.Fatalf("model = %q, want custom-image-model", provider.request.Model)
	}
	if len(result.Media) != 1 {
		t.Fatalf("media refs = %d, want 1", len(result.Media))
	}
}
