package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
)

func TestMessageHasBareFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"bare file only", "[file]", true},
		{"file with newline", "\n[file]", true},
		{"file with whitespace", "  [file]  ", true},
		{"file with caption", "please analyze this\n[file]", false},
		{"file with Japanese", "\u56f3\u7248\u4ed8\u304d\n[file]", false},
		{"no file", "hello world", false},
		{"image only", "[image: photo]", false},
		{"file and image bare", "[image: photo]\n[file]", true},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := bus.InboundMessage{Content: tt.content}
			if got := messageHasBareFile(msg); got != tt.want {
				t.Errorf("messageHasBareFile(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

func TestIsCancelKeyword(t *testing.T) {
	tests := []struct {
		content string
		want    bool
	}{
		{"cancel", true},
		{"Cancel this", true},
		{"ABORT", true},
		{"\u4e2d\u6b62\u3057\u3066", true},
		{"キャンセル", true},
		{"やめ", true},
		{"やめて", true},
		{"please continue", false},
		{"summarize this", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.content, func(t *testing.T) {
			if got := isCancelKeyword(tt.content); got != tt.want {
				t.Errorf("isCancelKeyword(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

func TestMergeBufferedMessages(t *testing.T) {
	buffered := []bus.InboundMessage{
		{ChatID: "chat1", Content: "summarize this"},
		{ChatID: "chat2", Content: "other chat"},
		{ChatID: "chat1", Content: "in Japanese"},
	}

	got := mergeBufferedMessages(buffered, "chat1")
	want := "summarize this\nin Japanese"
	if got != want {
		t.Errorf("mergeBufferedMessages = %q, want %q", got, want)
	}
}

func TestExtractNonChatMessages(t *testing.T) {
	buffered := []bus.InboundMessage{
		{ChatID: "chat1", Content: "same"},
		{ChatID: "chat2", Content: "other"},
		{ChatID: "chat1", Content: "same2"},
	}

	got := extractNonChatMessages(buffered, "chat1")
	if len(got) != 1 || got[0].Content != "other" {
		t.Errorf("extractNonChatMessages: got %d messages, want 1", len(got))
	}
}
