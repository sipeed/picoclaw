package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/media"
)

type fakeImageGenerationProvider struct {
	id           string
	defaultModel string
	request      imageGenerationRequest
}

func (p *fakeImageGenerationProvider) ID() string { return p.id }

func (p *fakeImageGenerationProvider) DefaultModel() string { return p.defaultModel }

func (p *fakeImageGenerationProvider) GenerateImages(
	_ context.Context,
	req imageGenerationRequest,
) ([]generatedImage, error) {
	p.request = req
	return []generatedImage{{
		Data:     []byte("fake-image"),
		MimeType: "image/png",
		Ext:      "png",
	}}, nil
}

func TestImageGenerateToolCodexOAuthRequestAndMediaResult(t *testing.T) {
	var captured map[string]any
	var gotAuth string
	var gotAccount string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccount = r.Header.Get("Chatgpt-Account-Id")
		if r.URL.Path != "/responses" {
			t.Fatalf("path = %q, want /responses", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		payload := base64.StdEncoding.EncodeToString([]byte("fake-png"))
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"type":"response.output_item.done","item":{"type":"image_generation_call","result":"` + payload + `"}}` + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	store := media.NewFileMediaStore()
	tool := NewImageGenerateTool(
		t.TempDir(),
		"openai/gpt-image-2",
		store,
		WithImageGenerateBaseURL(server.URL),
		WithImageGenerateTokenSource(func() (string, string, error) {
			return "test-token", "acct-123", nil
		}),
	)

	result := tool.Execute(
		WithToolContext(t.Context(), "telegram", "chat-1"),
		map[string]any{
			"prompt":        "make a tiny icon",
			"size":          "1536x1024",
			"quality":       "medium",
			"output_format": "png",
		},
	)
	if result.IsError {
		t.Fatalf("Execute returned error: %s", result.ContentForLLM())
	}
	if !result.ResponseHandled {
		t.Fatal("ResponseHandled = false, want true")
	}
	if len(result.Media) != 1 {
		t.Fatalf("media refs = %d, want 1", len(result.Media))
	}
	path, err := store.Resolve(result.Media[0])
	if err != nil {
		t.Fatalf("resolve media: %v", err)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("Authorization = %q, want Bearer test-token", gotAuth)
	}
	if gotAccount != "acct-123" {
		t.Fatalf("Chatgpt-Account-Id = %q, want acct-123", gotAccount)
	}
	if captured["model"] != "gpt-5.4" {
		t.Fatalf("request model = %v, want gpt-5.4", captured["model"])
	}
	toolsRaw := captured["tools"].([]any)
	imageTool := toolsRaw[0].(map[string]any)
	if imageTool["model"] != "gpt-image-2" {
		t.Fatalf("image model = %v, want gpt-image-2", imageTool["model"])
	}
	if imageTool["size"] != "1536x1024" {
		t.Fatalf("size = %v, want 1536x1024", imageTool["size"])
	}
	if imageTool["quality"] != "medium" {
		t.Fatalf("quality = %v, want medium", imageTool["quality"])
	}
	if path == "" {
		t.Fatal("generated media path is empty")
	}
}

func TestImageGenerateToolCanUseInjectedProvider(t *testing.T) {
	store := media.NewFileMediaStore()
	provider := &fakeImageGenerationProvider{
		id:           "test-provider",
		defaultModel: "test-default-image-model",
	}
	tool := NewImageGenerateTool(
		t.TempDir(),
		"test-provider/custom-image-model",
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

func TestParseImageGenerationModel(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		wantProvider string
		wantModel    string
	}{
		{
			name:         "empty uses default provider and model",
			model:        "",
			wantProvider: "openai-codex",
			wantModel:    "gpt-image-2",
		},
		{
			name:         "bare model uses default provider",
			model:        "custom-image-model",
			wantProvider: "openai-codex",
			wantModel:    "custom-image-model",
		},
		{
			name:         "openai alias routes to codex oauth provider",
			model:        "openai/gpt-image-2",
			wantProvider: "openai-codex",
			wantModel:    "gpt-image-2",
		},
		{
			name:         "future provider prefix is preserved",
			model:        "gemini/imagen-4",
			wantProvider: "gemini",
			wantModel:    "imagen-4",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseImageGenerationModel(tt.model)
			if got.Provider != tt.wantProvider {
				t.Fatalf("provider = %q, want %q", got.Provider, tt.wantProvider)
			}
			if got.Model != tt.wantModel {
				t.Fatalf("model = %q, want %q", got.Model, tt.wantModel)
			}
		})
	}
}

func TestParseCodexImageSSECompletedResponseFallback(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString([]byte("fake-png"))
	body := `data: {"type":"response.completed","response":{"output":[{"type":"image_generation_call","result":"` + payload + `"}]}}` + "\n\n"

	images, err := parseCodexImageSSE(strings.NewReader(body), "png")
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
