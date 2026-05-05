package oauthprovider

import (
	"encoding/base64"
	"testing"

	"github.com/openai/openai-go/v3/responses"
)

type mockCodexImageStream struct {
	events []responses.ResponseStreamEventUnion
	index  int
	err    error
}

func (s *mockCodexImageStream) Next() bool {
	if s.index >= len(s.events) {
		return false
	}
	s.index++
	return true
}

func (s *mockCodexImageStream) Current() responses.ResponseStreamEventUnion {
	return s.events[s.index-1]
}

func (s *mockCodexImageStream) Err() error { return s.err }

func TestCodexProviderSupportsImageGeneration(t *testing.T) {
	provider := NewCodexProvider("test-token", "acct-123")
	if !provider.SupportsImageGeneration() {
		t.Fatal("SupportsImageGeneration = false, want true")
	}
	if provider.ImageGenerationProviderID() != "openai-codex" {
		t.Fatalf("provider id = %q, want openai-codex", provider.ImageGenerationProviderID())
	}
	if provider.DefaultImageGenerationModel() != "gpt-image-2" {
		t.Fatalf("default image model = %q, want gpt-image-2", provider.DefaultImageGenerationModel())
	}
}

func TestBuildCodexImageParams(t *testing.T) {
	params := buildCodexImageParams(ImageGenerationRequest{
		Prompt:       "make a tiny icon",
		Model:        "gpt-image-2",
		Size:         "1536x1024",
		Quality:      "medium",
		OutputFormat: "png",
	})
	if params.Model != "gpt-5.4" {
		t.Fatalf("request model = %q, want gpt-5.4", params.Model)
	}
	if len(params.Tools) != 1 || params.Tools[0].OfImageGeneration == nil {
		t.Fatalf("expected one image_generation tool, got %#v", params.Tools)
	}
	tool := params.Tools[0].OfImageGeneration
	if tool.Model != "gpt-image-2" {
		t.Fatalf("image model = %q, want gpt-image-2", tool.Model)
	}
	if tool.Size != "1536x1024" {
		t.Fatalf("size = %q, want 1536x1024", tool.Size)
	}
	if tool.Quality != "medium" {
		t.Fatalf("quality = %q, want medium", tool.Quality)
	}
}

func TestParseCodexImageSSECompletedResponseFallback(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString([]byte("fake-png"))
	stream := &mockCodexImageStream{
		events: []responses.ResponseStreamEventUnion{{
			Type: "response.completed",
			Response: responses.Response{
				Output: []responses.ResponseOutputItemUnion{{
					Type:   "image_generation_call",
					Result: payload,
				}},
			},
		}},
	}

	images, err := parseCodexImageSSE(stream, "png")
	if err != nil {
		t.Fatalf("parseCodexImageSSE: %v", err)
	}
	if len(images) != 1 {
		t.Fatalf("images = %d, want 1", len(images))
	}
	if string(images[0].Data) != "fake-png" {
		t.Fatalf("image data = %q, want fake-png", string(images[0].Data))
	}
}
