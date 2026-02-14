package providers

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildOpenAICompatMessages_WithImageMedia(t *testing.T) {
	messages := []Message{
		{
			Role:    "user",
			Content: "Describe this image",
			Media: []MediaItem{
				{Type: "image_url", URL: "data:image/png;base64,abc"},
			},
		},
	}

	out := buildOpenAICompatMessages(messages)
	if len(out) != 1 {
		t.Fatalf("expected 1 message, got %d", len(out))
	}

	content, ok := out[0]["content"].([]map[string]interface{})
	if !ok {
		t.Fatalf("expected content parts array")
	}
	if len(content) != 2 {
		t.Fatalf("expected 2 content parts, got %d", len(content))
	}
	if content[1]["type"] != "image_url" {
		t.Fatalf("expected image_url part")
	}
}

func TestParseOpenAIContent_TextArray(t *testing.T) {
	raw := json.RawMessage(`[
		{"type":"text","text":"line1"},
		{"type":"text","text":"line2"}
	]`)
	got := parseOpenAIContent(raw)
	if !strings.Contains(got, "line1") || !strings.Contains(got, "line2") {
		t.Fatalf("unexpected parsed content: %q", got)
	}
}

func TestParseOpenAIContent_String(t *testing.T) {
	raw := json.RawMessage(`"hello"`)
	got := parseOpenAIContent(raw)
	if got != "hello" {
		t.Fatalf("expected hello, got %q", got)
	}
}
