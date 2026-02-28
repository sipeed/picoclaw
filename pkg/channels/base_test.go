package channels

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestBaseChannelIsAllowed(t *testing.T) {
	tests := []struct {
		name      string
		allowList []string
		senderID  string
		want      bool
	}{
		{
			name:      "empty allowlist allows all",
			allowList: nil,
			senderID:  "anyone",
			want:      true,
		},
		{
			name:      "compound sender matches numeric allowlist",
			allowList: []string{"123456"},
			senderID:  "123456|alice",
			want:      true,
		},
		{
			name:      "compound sender matches username allowlist",
			allowList: []string{"@alice"},
			senderID:  "123456|alice",
			want:      true,
		},
		{
			name:      "numeric sender matches legacy compound allowlist",
			allowList: []string{"123456|alice"},
			senderID:  "123456",
			want:      true,
		},
		{
			name:      "non matching sender is denied",
			allowList: []string{"123456"},
			senderID:  "654321|bob",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := NewBaseChannel("test", nil, nil, tt.allowList)
			if got := ch.IsAllowed(tt.senderID); got != tt.want {
				t.Fatalf("IsAllowed(%q) = %v, want %v", tt.senderID, got, tt.want)
			}
		})
	}
}

func TestShouldRespondInGroup(t *testing.T) {
	tests := []struct {
		name        string
		gt          config.GroupTriggerConfig
		isMentioned bool
		content     string
		wantRespond bool
		wantContent string
	}{
		{
			name:        "no config - permissive default",
			gt:          config.GroupTriggerConfig{},
			isMentioned: false,
			content:     "hello world",
			wantRespond: true,
			wantContent: "hello world",
		},
		{
			name:        "no config - mentioned",
			gt:          config.GroupTriggerConfig{},
			isMentioned: true,
			content:     "hello world",
			wantRespond: true,
			wantContent: "hello world",
		},
		{
			name:        "mention_only - not mentioned",
			gt:          config.GroupTriggerConfig{MentionOnly: true},
			isMentioned: false,
			content:     "hello world",
			wantRespond: false,
			wantContent: "hello world",
		},
		{
			name:        "mention_only - mentioned",
			gt:          config.GroupTriggerConfig{MentionOnly: true},
			isMentioned: true,
			content:     "hello world",
			wantRespond: true,
			wantContent: "hello world",
		},
		{
			name:        "prefix match",
			gt:          config.GroupTriggerConfig{Prefixes: []string{"/ask"}},
			isMentioned: false,
			content:     "/ask hello",
			wantRespond: true,
			wantContent: "hello",
		},
		{
			name:        "prefix no match - not mentioned",
			gt:          config.GroupTriggerConfig{Prefixes: []string{"/ask"}},
			isMentioned: false,
			content:     "hello world",
			wantRespond: false,
			wantContent: "hello world",
		},
		{
			name:        "prefix no match - but mentioned",
			gt:          config.GroupTriggerConfig{Prefixes: []string{"/ask"}},
			isMentioned: true,
			content:     "hello world",
			wantRespond: true,
			wantContent: "hello world",
		},
		{
			name:        "multiple prefixes - second matches",
			gt:          config.GroupTriggerConfig{Prefixes: []string{"/ask", "/bot"}},
			isMentioned: false,
			content:     "/bot help me",
			wantRespond: true,
			wantContent: "help me",
		},
		{
			name:        "mention_only with prefixes - mentioned overrides",
			gt:          config.GroupTriggerConfig{MentionOnly: true, Prefixes: []string{"/ask"}},
			isMentioned: true,
			content:     "hello",
			wantRespond: true,
			wantContent: "hello",
		},
		{
			name:        "mention_only with prefixes - not mentioned, no prefix",
			gt:          config.GroupTriggerConfig{MentionOnly: true, Prefixes: []string{"/ask"}},
			isMentioned: false,
			content:     "hello",
			wantRespond: false,
			wantContent: "hello",
		},
		{
			name:        "empty prefix in list is skipped",
			gt:          config.GroupTriggerConfig{Prefixes: []string{"", "/ask"}},
			isMentioned: false,
			content:     "/ask test",
			wantRespond: true,
			wantContent: "test",
		},
		{
			name:        "prefix strips leading whitespace after prefix",
			gt:          config.GroupTriggerConfig{Prefixes: []string{"/ask "}},
			isMentioned: false,
			content:     "/ask hello",
			wantRespond: true,
			wantContent: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := NewBaseChannel("test", nil, nil, nil, WithGroupTrigger(tt.gt))
			gotRespond, gotContent := ch.ShouldRespondInGroup(tt.isMentioned, tt.content)
			if gotRespond != tt.wantRespond {
				t.Errorf("ShouldRespondInGroup() respond = %v, want %v", gotRespond, tt.wantRespond)
			}
			if gotContent != tt.wantContent {
				t.Errorf("ShouldRespondInGroup() content = %q, want %q", gotContent, tt.wantContent)
			}
		})
	}
}

func TestIsAllowedSender(t *testing.T) {
	tests := []struct {
		name      string
		allowList []string
		sender    bus.SenderInfo
		want      bool
	}{
		{
			name:      "empty allowlist allows all",
			allowList: nil,
			sender:    bus.SenderInfo{PlatformID: "anyone"},
			want:      true,
		},
		{
			name:      "numeric ID matches PlatformID",
			allowList: []string{"123456"},
			sender: bus.SenderInfo{
				Platform:    "telegram",
				PlatformID:  "123456",
				CanonicalID: "telegram:123456",
			},
			want: true,
		},
		{
			name:      "canonical format matches",
			allowList: []string{"telegram:123456"},
			sender: bus.SenderInfo{
				Platform:    "telegram",
				PlatformID:  "123456",
				CanonicalID: "telegram:123456",
			},
			want: true,
		},
		{
			name:      "canonical format wrong platform",
			allowList: []string{"discord:123456"},
			sender: bus.SenderInfo{
				Platform:    "telegram",
				PlatformID:  "123456",
				CanonicalID: "telegram:123456",
			},
			want: false,
		},
		{
			name:      "@username matches",
			allowList: []string{"@alice"},
			sender: bus.SenderInfo{
				Platform:    "telegram",
				PlatformID:  "123456",
				CanonicalID: "telegram:123456",
				Username:    "alice",
			},
			want: true,
		},
		{
			name:      "compound id|username matches by ID",
			allowList: []string{"123456|alice"},
			sender: bus.SenderInfo{
				Platform:    "telegram",
				PlatformID:  "123456",
				CanonicalID: "telegram:123456",
				Username:    "alice",
			},
			want: true,
		},
		{
			name:      "non matching sender denied",
			allowList: []string{"654321"},
			sender: bus.SenderInfo{
				Platform:    "telegram",
				PlatformID:  "123456",
				CanonicalID: "telegram:123456",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := NewBaseChannel("test", nil, nil, tt.allowList)
			if got := ch.IsAllowedSender(tt.sender); got != tt.want {
				t.Fatalf("IsAllowedSender(%+v) = %v, want %v", tt.sender, got, tt.want)
			}
		})
	}
}
