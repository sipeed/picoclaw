//go:build amd64 || arm64 || riscv64 || mips64 || ppc64

package feishu

import (
	"testing"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

func TestExtractContent(t *testing.T) {
	tests := []struct {
		name        string
		messageType string
		rawContent  string
		want        string
	}{
		{
			name:        "text message",
			messageType: "text",
			rawContent:  `{"text": "hello world"}`,
			want:        "hello world",
		},
		{
			name:        "text message invalid JSON",
			messageType: "text",
			rawContent:  `not json`,
			want:        "not json",
		},
		{
			name:        "post message returns raw JSON",
			messageType: "post",
			rawContent:  `{"title": "test post"}`,
			want:        `{"title": "test post"}`,
		},
		{
			name:        "image message returns empty",
			messageType: "image",
			rawContent:  `{"image_key": "img_xxx"}`,
			want:        "",
		},
		{
			name:        "file message with filename",
			messageType: "file",
			rawContent:  `{"file_key": "file_xxx", "file_name": "report.pdf"}`,
			want:        "report.pdf",
		},
		{
			name:        "file message without filename",
			messageType: "file",
			rawContent:  `{"file_key": "file_xxx"}`,
			want:        "",
		},
		{
			name:        "audio message with filename",
			messageType: "audio",
			rawContent:  `{"file_key": "file_xxx", "file_name": "recording.ogg"}`,
			want:        "recording.ogg",
		},
		{
			name:        "media message with filename",
			messageType: "media",
			rawContent:  `{"file_key": "file_xxx", "file_name": "video.mp4"}`,
			want:        "video.mp4",
		},
		{
			name:        "unknown message type returns raw",
			messageType: "sticker",
			rawContent:  `{"sticker_id": "sticker_xxx"}`,
			want:        `{"sticker_id": "sticker_xxx"}`,
		},
		{
			name:        "empty raw content",
			messageType: "text",
			rawContent:  "",
			want:        "",
		},
		{
			name:        "interactive card returns raw JSON",
			messageType: "interactive",
			rawContent:  `{"schema":"2.0","body":{"elements":[{"tag":"markdown","content":"Hello from card"}]}}`,
			want:        `{"schema":"2.0","body":{"elements":[{"tag":"markdown","content":"Hello from card"}]}}`,
		},
		{
			name:        "interactive card with complex structure returns raw JSON",
			messageType: "interactive",
			rawContent:  `{"header":{"title":{"tag":"plain_text","content":"Title"}},"elements":[{"tag":"div","text":{"tag":"lark_md","content":"Card content"}}]}`,
			want:        `{"header":{"title":{"tag":"plain_text","content":"Title"}},"elements":[{"tag":"div","text":{"tag":"lark_md","content":"Card content"}}]}`,
		},
		{
			name:        "interactive card invalid JSON returns as-is",
			messageType: "interactive",
			rawContent:  `not valid json`,
			want:        `not valid json`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractContent(tt.messageType, tt.rawContent)
			if got != tt.want {
				t.Errorf("extractContent(%q, %q) = %q, want %q", tt.messageType, tt.rawContent, got, tt.want)
			}
		})
	}
}

func TestAppendMediaTags(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		messageType string
		mediaRefs   []string
		want        string
	}{
		{
			name:        "no refs returns content unchanged",
			content:     "hello",
			messageType: "image",
			mediaRefs:   nil,
			want:        "hello",
		},
		{
			name:        "empty refs returns content unchanged",
			content:     "hello",
			messageType: "image",
			mediaRefs:   []string{},
			want:        "hello",
		},
		{
			name:        "image with content",
			content:     "check this",
			messageType: "image",
			mediaRefs:   []string{"ref1"},
			want:        "check this [image: photo]",
		},
		{
			name:        "image empty content",
			content:     "",
			messageType: "image",
			mediaRefs:   []string{"ref1"},
			want:        "[image: photo]",
		},
		{
			name:        "audio",
			content:     "listen",
			messageType: "audio",
			mediaRefs:   []string{"ref1"},
			want:        "listen [audio]",
		},
		{
			name:        "media/video",
			content:     "watch",
			messageType: "media",
			mediaRefs:   []string{"ref1"},
			want:        "watch [video]",
		},
		{
			name:        "file",
			content:     "report.pdf",
			messageType: "file",
			mediaRefs:   []string{"ref1"},
			want:        "report.pdf [file]",
		},
		{
			name:        "unknown type",
			content:     "something",
			messageType: "sticker",
			mediaRefs:   []string{"ref1"},
			want:        "something [attachment]",
		},
		{
			name:        "interactive card with images returns content unchanged",
			content:     `{"schema":"2.0","body":{"elements":[{"tag":"img","img_key":"img_123"}]}}`,
			messageType: "interactive",
			mediaRefs:   []string{"ref1"},
			want:        `{"schema":"2.0","body":{"elements":[{"tag":"img","img_key":"img_123"}]}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendMediaTags(tt.content, tt.messageType, tt.mediaRefs)
			if got != tt.want {
				t.Errorf(
					"appendMediaTags(%q, %q, %v) = %q, want %q",
					tt.content,
					tt.messageType,
					tt.mediaRefs,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestExtractFeishuSenderID(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name   string
		sender *larkim.EventSender
		want   string
	}{
		{
			name:   "nil sender",
			sender: nil,
			want:   "",
		},
		{
			name:   "nil sender ID",
			sender: &larkim.EventSender{SenderId: nil},
			want:   "",
		},
		{
			name: "userId preferred",
			sender: &larkim.EventSender{
				SenderId: &larkim.UserId{
					UserId:  strPtr("u_abc123"),
					OpenId:  strPtr("ou_def456"),
					UnionId: strPtr("on_ghi789"),
				},
			},
			want: "u_abc123",
		},
		{
			name: "openId fallback",
			sender: &larkim.EventSender{
				SenderId: &larkim.UserId{
					UserId:  strPtr(""),
					OpenId:  strPtr("ou_def456"),
					UnionId: strPtr("on_ghi789"),
				},
			},
			want: "ou_def456",
		},
		{
			name: "unionId fallback",
			sender: &larkim.EventSender{
				SenderId: &larkim.UserId{
					UserId:  strPtr(""),
					OpenId:  strPtr(""),
					UnionId: strPtr("on_ghi789"),
				},
			},
			want: "on_ghi789",
		},
		{
			name: "all empty strings",
			sender: &larkim.EventSender{
				SenderId: &larkim.UserId{
					UserId:  strPtr(""),
					OpenId:  strPtr(""),
					UnionId: strPtr(""),
				},
			},
			want: "",
		},
		{
			name: "nil userId pointer falls through",
			sender: &larkim.EventSender{
				SenderId: &larkim.UserId{
					UserId:  nil,
					OpenId:  strPtr("ou_def456"),
					UnionId: nil,
				},
			},
			want: "ou_def456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFeishuSenderID(tt.sender)
			if got != tt.want {
				t.Errorf("extractFeishuSenderID() = %q, want %q", got, tt.want)
			}
		})
	}
}
