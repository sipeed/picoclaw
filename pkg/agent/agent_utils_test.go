package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
)

func TestInferMediaType(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		contentType string
		want        string
	}{
		{
			name:        "png content type",
			filename:    "diagram",
			contentType: "image/png",
			want:        "image",
		},
		{
			name:        "jpeg extension fallback",
			filename:    "photo.JPG",
			contentType: "",
			want:        "image",
		},
		{
			name:        "svg content type is file",
			filename:    "diagram",
			contentType: "image/svg+xml",
			want:        "file",
		},
		{
			name:        "svg content type with parameters is file",
			filename:    "diagram",
			contentType: "image/svg+xml; charset=utf-8",
			want:        "file",
		},
		{
			name:        "svg extension fallback is file",
			filename:    "diagram.SVG",
			contentType: "",
			want:        "file",
		},
		{
			name:        "audio content type",
			filename:    "voice",
			contentType: "audio/ogg",
			want:        "audio",
		},
		{
			name:        "ogg application content type",
			filename:    "voice.ogg",
			contentType: "application/ogg",
			want:        "audio",
		},
		{
			name:        "video extension fallback",
			filename:    "clip.MP4",
			contentType: "",
			want:        "video",
		},
		{
			name:        "unknown type",
			filename:    "archive.bin",
			contentType: "application/octet-stream",
			want:        "file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferMediaType(tt.filename, tt.contentType)
			if got != tt.want {
				t.Fatalf("inferMediaType(%q, %q) = %q, want %q", tt.filename, tt.contentType, got, tt.want)
			}
		})
	}
}

func TestOutboundContextFromInbound_FallsBackToOriginatingMessage(t *testing.T) {
	msgID := "4428"
	inbound := &bus.InboundContext{
		Channel:   "telegram",
		ChatID:    "-1001725217387",
		MessageID: msgID,
	}

	ctx := outboundContextFromInbound(inbound, "telegram", "-1001725217387", "")
	if ctx.ReplyToMessageID != msgID {
		t.Fatalf("ReplyToMessageID = %q, want originating message %q", ctx.ReplyToMessageID, msgID)
	}
}

func TestOutboundContextFromInbound_KeepsExplicitReplyTarget(t *testing.T) {
	inbound := &bus.InboundContext{
		Channel:          "telegram",
		ChatID:           "-1001725217387",
		MessageID:        "4428",
		ReplyToMessageID: "4422",
	}

	ctx := outboundContextFromInbound(inbound, "telegram", "-1001725217387", "")
	if ctx.ReplyToMessageID != "4422" {
		t.Fatalf("ReplyToMessageID = %q, want explicit target 4422", ctx.ReplyToMessageID)
	}
}

func TestOutboundContextFromInbound_NilInboundUnchanged(t *testing.T) {
	ctx := outboundContextFromInbound(nil, "telegram", "-1001725217387", "")
	if ctx.ReplyToMessageID != "" {
		t.Fatalf("ReplyToMessageID = %q, want empty", ctx.ReplyToMessageID)
	}
}
