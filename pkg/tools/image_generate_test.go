package tools

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/media"
)

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
