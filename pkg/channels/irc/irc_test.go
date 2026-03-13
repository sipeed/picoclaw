package irc

import (
	"testing"

	"jane/pkg/bus"
	"jane/pkg/config"
)

func TestNewIRCChannel(t *testing.T) {
	msgBus := bus.NewMessageBus()

	t.Run("missing server", func(t *testing.T) {
		cfg := config.IRCConfig{Nick: "bot"}
		_, err := NewIRCChannel(cfg, msgBus)
		if err == nil {
			t.Error("expected error for missing server, got nil")
		}
	})

	t.Run("missing nick", func(t *testing.T) {
		cfg := config.IRCConfig{Server: "irc.example.com:6667"}
		_, err := NewIRCChannel(cfg, msgBus)
		if err == nil {
			t.Error("expected error for missing nick, got nil")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		cfg := config.IRCConfig{
			Server:   "irc.example.com:6667",
			Nick:     "testbot",
			Channels: []string{"#test"},
		}
		ch, err := NewIRCChannel(cfg, msgBus)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.Name() != "irc" {
			t.Errorf("Name() = %q, want %q", ch.Name(), "irc")
		}
		if ch.IsRunning() {
			t.Error("new channel should not be running")
		}
	})
}

func TestExtractHost(t *testing.T) {
	tests := []struct {
		server string
		want   string
	}{
		{"irc.libera.chat:6697", "irc.libera.chat"},
		{"localhost:6667", "localhost"},
		{"irc.example.com", "irc.example.com"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.server, func(t *testing.T) {
			got := extractHost(tt.server)
			if got != tt.want {
				t.Errorf("extractHost(%q) = %q, want %q", tt.server, got, tt.want)
			}
		})
	}
}

func TestNickMentionedAt(t *testing.T) {
	tests := []struct {
		name    string
		content string
		nick    string
		want    int
	}{
		{"colon prefix", "bot: hello", "bot", 0},
		{"comma prefix", "bot, hello", "bot", 0},
		{"case insensitive", "BOT: hello", "bot", 0},
		{"word boundary mid", "hey bot what's up", "bot", 4},
		{"no mention", "hello world", "bot", -1},
		{"substring mismatch", "robotics are cool", "bot", -1},
		{"nick at end", "hello bot", "bot", 6},
		{"empty content", "", "bot", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nickMentionedAt(tt.content, tt.nick)
			if got != tt.want {
				t.Errorf("nickMentionedAt(%q, %q) = %d, want %d", tt.content, tt.nick, got, tt.want)
			}
		})
	}
}

func TestIsBotMentioned(t *testing.T) {
	tests := []struct {
		name    string
		content string
		nick    string
		want    bool
	}{
		{"colon prefix", "bot: hello", "bot", true},
		{"comma prefix", "bot, hello", "bot", true},
		{"case insensitive", "BOT: hello", "bot", true},
		{"word boundary mid", "hey bot what's up", "bot", true},
		{"no mention", "hello world", "bot", false},
		{"substring mismatch", "robotics are cool", "bot", false},
		{"nick at end", "hello bot", "bot", true},
		{"empty content", "", "bot", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBotMentioned(tt.content, tt.nick)
			if got != tt.want {
				t.Errorf("isBotMentioned(%q, %q) = %v, want %v", tt.content, tt.nick, got, tt.want)
			}
		})
	}
}

func TestStripBotMention(t *testing.T) {
	tests := []struct {
		name    string
		content string
		nick    string
		want    string
	}{
		{"colon prefix", "bot: hello there", "bot", "hello there"},
		{"comma prefix", "bot, help me", "bot", "help me"},
		{"case insensitive", "BOT: hello", "bot", "hello"},
		{"no prefix match", "hello bot", "bot", "hello bot"},
		{"only prefix", "bot:", "bot", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripBotMention(tt.content, tt.nick)
			if got != tt.want {
				t.Errorf("stripBotMention(%q, %q) = %q, want %q", tt.content, tt.nick, got, tt.want)
			}
		})
	}
}
