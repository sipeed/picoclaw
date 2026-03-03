//go:build amd64 || arm64 || riscv64 || mips64 || ppc64

package feishu

import (
	"testing"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

func TestExtractFeishuImageKey(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "valid image key",
			content: `{"image_key":"img_v2_abc123"}`,
			want:    "img_v2_abc123",
		},
		{
			name:    "empty content",
			content: "",
			want:    "",
		},
		{
			name:    "invalid json",
			content: "not json",
			want:    "",
		},
		{
			name:    "missing image_key field",
			content: `{"text":"hello"}`,
			want:    "",
		},
		{
			name:    "empty image_key",
			content: `{"image_key":""}`,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFeishuImageKey(tt.content)
			if got != tt.want {
				t.Errorf("extractFeishuImageKey(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestExtractFeishuMessageContent_TextUnchanged(t *testing.T) {
	msgType := larkim.MsgTypeText
	content := `{"text":"hello world"}`

	msg := &larkim.EventMessage{
		MessageType: &msgType,
		Content:     &content,
	}

	got := extractFeishuMessageContent(msg)
	if got != "hello world" {
		t.Errorf("extractFeishuMessageContent() = %q, want %q", got, "hello world")
	}
}

func TestExtractFeishuMessageContent_ImageReturnsRaw(t *testing.T) {
	msgType := larkim.MsgTypeImage
	content := `{"image_key":"img_v2_xxx"}`

	msg := &larkim.EventMessage{
		MessageType: &msgType,
		Content:     &content,
	}

	// For non-text messages, extractFeishuMessageContent returns raw content
	got := extractFeishuMessageContent(msg)
	if got != content {
		t.Errorf("extractFeishuMessageContent() = %q, want %q", got, content)
	}
}

func TestExtractFeishuMessageContent_NilMessage(t *testing.T) {
	got := extractFeishuMessageContent(nil)
	if got != "" {
		t.Errorf("extractFeishuMessageContent(nil) = %q, want empty", got)
	}
}
