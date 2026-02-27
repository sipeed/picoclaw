package identity

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
)

func TestBuildCanonicalID(t *testing.T) {
	tests := []struct {
		platform   string
		platformID string
		want       string
	}{
		{"telegram", "123456", "telegram:123456"},
		{"Discord", "98765432", "discord:98765432"},
		{"SLACK", "U123ABC", "slack:U123ABC"},
		{"", "123", ""},
		{"telegram", "", ""},
		{"  telegram  ", "  123  ", "telegram:123"},
	}

	for _, tt := range tests {
		got := BuildCanonicalID(tt.platform, tt.platformID)
		if got != tt.want {
			t.Errorf("BuildCanonicalID(%q, %q) = %q, want %q",
				tt.platform, tt.platformID, got, tt.want)
		}
	}
}

func TestParseCanonicalID(t *testing.T) {
	tests := []struct {
		input        string
		wantPlatform string
		wantID       string
		wantOk       bool
	}{
		{"telegram:123456", "telegram", "123456", true},
		{"discord:98765432", "discord", "98765432", true},
		{"slack:U123ABC", "slack", "U123ABC", true},
		{"nocolon", "", "", false},
		{"", "", "", false},
		{":missing", "", "", false},
		{"missing:", "", "", false},
	}

	for _, tt := range tests {
		platform, id, ok := ParseCanonicalID(tt.input)
		if ok != tt.wantOk || platform != tt.wantPlatform || id != tt.wantID {
			t.Errorf("ParseCanonicalID(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tt.input, platform, id, ok,
				tt.wantPlatform, tt.wantID, tt.wantOk)
		}
	}
}

func TestMatchAllowed(t *testing.T) {
	telegramSender := bus.SenderInfo{
		Platform:    "telegram",
		PlatformID:  "123456",
		CanonicalID: "telegram:123456",
		Username:    "alice",
		DisplayName: "Alice Smith",
	}

	discordSender := bus.SenderInfo{
		Platform:    "discord",
		PlatformID:  "98765432",
		CanonicalID: "discord:98765432",
		Username:    "bob",
		DisplayName: "bob#1234",
	}

	noCanonicalSender := bus.SenderInfo{
		Platform:   "telegram",
		PlatformID: "999",
		Username:   "carol",
	}

	tests := []struct {
		name    string
		sender  bus.SenderInfo
		allowed string
		want    bool
	}{
		// Pure numeric ID matching
		{
			name:    "numeric ID matches PlatformID",
			sender:  telegramSender,
			allowed: "123456",
			want:    true,
		},
		{
			name:    "numeric ID does not match",
			sender:  telegramSender,
			allowed: "654321",
			want:    false,
		},
		// Username matching
		{
			name:    "@username matches Username",
			sender:  telegramSender,
			allowed: "@alice",
			want:    true,
		},
		{
			name:    "@username does not match",
			sender:  telegramSender,
			allowed: "@bob",
			want:    false,
		},
		// Compound format "id|username"
		{
			name:    "compound matches by ID",
			sender:  telegramSender,
			allowed: "123456|alice",
			want:    true,
		},
		{
			name:    "compound matches by username",
			sender:  telegramSender,
			allowed: "999|alice",
			want:    true,
		},
		{
			name:    "compound does not match",
			sender:  telegramSender,
			allowed: "654321|bob",
			want:    false,
		},
		// Canonical format "platform:id"
		{
			name:    "canonical matches exactly",
			sender:  telegramSender,
			allowed: "telegram:123456",
			want:    true,
		},
		{
			name:    "canonical case-insensitive platform",
			sender:  telegramSender,
			allowed: "Telegram:123456",
			want:    true,
		},
		{
			name:    "canonical wrong platform",
			sender:  telegramSender,
			allowed: "discord:123456",
			want:    false,
		},
		{
			name:    "canonical wrong ID",
			sender:  telegramSender,
			allowed: "telegram:654321",
			want:    false,
		},
		// Cross-platform canonical
		{
			name:    "discord canonical match",
			sender:  discordSender,
			allowed: "discord:98765432",
			want:    true,
		},
		{
			name:    "telegram canonical does not match discord sender",
			sender:  discordSender,
			allowed: "telegram:98765432",
			want:    false,
		},
		// Sender without canonical ID
		{
			name:    "canonical match falls back to platform+platformID",
			sender:  noCanonicalSender,
			allowed: "telegram:999",
			want:    true,
		},
		{
			name:    "platform mismatch on fallback",
			sender:  noCanonicalSender,
			allowed: "discord:999",
			want:    false,
		},
		// Empty allowed string
		{
			name:    "empty allowed never matches",
			sender:  telegramSender,
			allowed: "",
			want:    false,
		},
		// Whitespace handling
		{
			name:    "trimmed allowed matches",
			sender:  telegramSender,
			allowed: "  123456  ",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchAllowed(tt.sender, tt.allowed)
			if got != tt.want {
				t.Errorf("MatchAllowed(%+v, %q) = %v, want %v",
					tt.sender, tt.allowed, got, tt.want)
			}
		})
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"123456", true},
		{"0", true},
		{"", false},
		{"abc", false},
		{"12a34", false},
		{"telegram", false},
	}

	for _, tt := range tests {
		got := isNumeric(tt.input)
		if got != tt.want {
			t.Errorf("isNumeric(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
