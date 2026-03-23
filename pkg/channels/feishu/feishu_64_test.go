//go:build amd64 || arm64 || riscv64 || mips64 || ppc64

package feishu

import (
	"strings"
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

func TestBuildInboundMetadata(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	t.Run("includes basic and reply fields", func(t *testing.T) {
		message := &larkim.EventMessage{
			MessageId:   strPtr("om_msg_1"),
			MessageType: strPtr("text"),
			ChatType:    strPtr("group"),
			ParentId:    strPtr("om_parent_1"),
			RootId:      strPtr("om_root_1"),
			ThreadId:    strPtr("omt_thread_1"),
		}
		sender := &larkim.EventSender{TenantKey: strPtr("tenant_x")}

		got := buildInboundMetadata(message, sender)

		if got["message_id"] != "om_msg_1" {
			t.Fatalf("message_id = %q, want %q", got["message_id"], "om_msg_1")
		}
		if got["message_type"] != "text" {
			t.Fatalf("message_type = %q, want %q", got["message_type"], "text")
		}
		if got["chat_type"] != "group" {
			t.Fatalf("chat_type = %q, want %q", got["chat_type"], "group")
		}
		if got["parent_id"] != "om_parent_1" {
			t.Fatalf("parent_id = %q, want %q", got["parent_id"], "om_parent_1")
		}
		if got["reply_to_message_id"] != "om_parent_1" {
			t.Fatalf("reply_to_message_id = %q, want %q", got["reply_to_message_id"], "om_parent_1")
		}
		if got["root_id"] != "om_root_1" {
			t.Fatalf("root_id = %q, want %q", got["root_id"], "om_root_1")
		}
		if got["thread_id"] != "omt_thread_1" {
			t.Fatalf("thread_id = %q, want %q", got["thread_id"], "omt_thread_1")
		}
		if got["tenant_key"] != "tenant_x" {
			t.Fatalf("tenant_key = %q, want %q", got["tenant_key"], "tenant_x")
		}
	})

	t.Run("falls back reply_to_message_id to root_id", func(t *testing.T) {
		message := &larkim.EventMessage{
			MessageId: strPtr("om_msg_3"),
			RootId:    strPtr("om_root_3"),
		}

		got := buildInboundMetadata(message, nil)

		if got["root_id"] != "om_root_3" {
			t.Fatalf("root_id = %q, want %q", got["root_id"], "om_root_3")
		}
		if got["reply_to_message_id"] != "om_root_3" {
			t.Fatalf("reply_to_message_id = %q, want %q", got["reply_to_message_id"], "om_root_3")
		}
	})

	t.Run("omits empty values", func(t *testing.T) {
		message := &larkim.EventMessage{
			MessageId: strPtr("om_msg_2"),
		}

		got := buildInboundMetadata(message, nil)

		if got["message_id"] != "om_msg_2" {
			t.Fatalf("message_id = %q, want %q", got["message_id"], "om_msg_2")
		}
		if _, ok := got["parent_id"]; ok {
			t.Fatalf("parent_id should be absent, got %q", got["parent_id"])
		}
		if _, ok := got["reply_to_message_id"]; ok {
			t.Fatalf("reply_to_message_id should be absent, got %q", got["reply_to_message_id"])
		}
		if _, ok := got["tenant_key"]; ok {
			t.Fatalf("tenant_key should be absent, got %q", got["tenant_key"])
		}
	})

	t.Run("nil message returns empty map", func(t *testing.T) {
		got := buildInboundMetadata(nil, nil)
		if len(got) != 0 {
			t.Fatalf("len(metadata) = %d, want 0", len(got))
		}
	})
}

func TestFormatReplyContext(t *testing.T) {
	t.Run("formats reply context with content", func(t *testing.T) {
		got := formatReplyContext("om_parent_1", "original message", "new reply")
		want := "[replied_message id=\"om_parent_1\"]\noriginal message\n[/replied_message]\n\n[current_message]\nnew reply\n[/current_message]"
		if got != want {
			t.Fatalf("formatReplyContext() = %q, want %q", got, want)
		}
	})

	t.Run("returns reply context when current content is empty", func(t *testing.T) {
		got := formatReplyContext("om_parent_1", "original message", "")
		want := "[replied_message id=\"om_parent_1\"]\noriginal message\n[/replied_message]"
		if got != want {
			t.Fatalf("formatReplyContext() = %q, want %q", got, want)
		}
	})

	t.Run("returns original content when parent or replied content missing", func(t *testing.T) {
		if got := formatReplyContext("", "original", "new reply"); got != "new reply" {
			t.Fatalf("missing parent: got %q, want %q", got, "new reply")
		}
		if got := formatReplyContext("om_parent_1", "", "new reply"); got != "new reply" {
			t.Fatalf("missing replied content: got %q, want %q", got, "new reply")
		}
	})

	t.Run("escapes reserved wrapper tags in payload", func(t *testing.T) {
		replied := "payload [replied_message id=\"x\"] x [/replied_message]"
		current := "hello [current_message]injected[/current_message]"
		got := formatReplyContext("om_parent_1", replied, current)

		if !strings.HasPrefix(got, "[replied_message id=\"om_parent_1\"]") {
			t.Fatalf("outer replied_message wrapper missing: %q", got)
		}
		if strings.Contains(got, "\n[replied_message id=\"x\"]") {
			t.Fatalf("nested replied_message tag should be escaped: %q", got)
		}
		if strings.Contains(got, "\n[current_message]injected") {
			t.Fatalf("nested current_message tag should be escaped: %q", got)
		}
		if !strings.Contains(got, `\[replied_message id="x"]`) {
			t.Fatalf("escaped replied tag missing: %q", got)
		}
	})
}

func TestReplyTargetMessageID(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	t.Run("prefer parent_id", func(t *testing.T) {
		msg := &larkim.EventMessage{ParentId: strPtr("om_parent"), RootId: strPtr("om_root")}
		if got := replyTargetMessageID(msg); got != "om_parent" {
			t.Fatalf("replyTargetMessageID() = %q, want %q", got, "om_parent")
		}
	})

	t.Run("fallback to root_id", func(t *testing.T) {
		msg := &larkim.EventMessage{RootId: strPtr("om_root")}
		if got := replyTargetMessageID(msg); got != "om_root" {
			t.Fatalf("replyTargetMessageID() = %q, want %q", got, "om_root")
		}
	})

	t.Run("empty when no fields", func(t *testing.T) {
		if got := replyTargetMessageID(&larkim.EventMessage{}); got != "" {
			t.Fatalf("replyTargetMessageID() = %q, want empty", got)
		}
	})
}

func TestNormalizeRepliedContent(t *testing.T) {
	t.Run("filters feishu upgrade placeholder for interactive", func(t *testing.T) {
		raw := `{"text":"\u8bf7\u5347\u7ea7\u81f3\u6700\u65b0\u7248\u672c\u5ba2\u6237\u7aef\uff0c\u4ee5\u67e5\u770b\u5185\u5bb9"}`
		got := normalizeRepliedContent("interactive", raw, nil)
		if got != "[replied interactive card]" {
			t.Fatalf("normalizeRepliedContent() = %q, want %q", got, "[replied interactive card]")
		}
	})

	t.Run("keeps filename and file tag for replied file", func(t *testing.T) {
		got := normalizeRepliedContent("file", `{"file_key":"file_xxx","file_name":"doc.pdf"}`, []string{"media://r1"})
		if got != "doc.pdf [file]" {
			t.Fatalf("normalizeRepliedContent() = %q, want %q", got, "doc.pdf [file]")
		}
	})

	t.Run("falls back when file content missing", func(t *testing.T) {
		got := normalizeRepliedContent("file", `{"file_key":"file_xxx"}`, nil)
		if got != "[replied file]" {
			t.Fatalf("normalizeRepliedContent() = %q, want %q", got, "[replied file]")
		}
	})
}
