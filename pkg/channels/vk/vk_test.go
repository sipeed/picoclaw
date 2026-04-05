package vk

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewVKChannel(t *testing.T) {
	msgBus := bus.NewMessageBus()

	t.Run("missing group_id", func(t *testing.T) {
		cfg := &config.Config{
			Channels: config.ChannelsConfig{
				VK: config.VKConfig{
					Enabled: true,
					Token:   *config.NewSecureString("test_token"),
				},
			},
		}
		ch, err := NewVKChannel(cfg, msgBus)
		if err != nil {
			t.Fatalf("unexpected error during creation: %v", err)
		}
		if ch.Name() != "vk" {
			t.Errorf("Name() = %q, want %q", ch.Name(), "vk")
		}
		if ch.IsRunning() {
			t.Error("new channel should not be running")
		}
	})

	t.Run("valid config with group_id", func(t *testing.T) {
		cfg := &config.Config{
			Channels: config.ChannelsConfig{
				VK: config.VKConfig{
					Enabled: true,
					Token:   *config.NewSecureString("test_token"),
					GroupID: 123456789,
				},
			},
		}
		ch, err := NewVKChannel(cfg, msgBus)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.Name() != "vk" {
			t.Errorf("Name() = %q, want %q", ch.Name(), "vk")
		}
		if ch.IsRunning() {
			t.Error("new channel should not be running")
		}
	})

	t.Run("with allow_from", func(t *testing.T) {
		cfg := &config.Config{
			Channels: config.ChannelsConfig{
				VK: config.VKConfig{
					Enabled:   true,
					Token:     *config.NewSecureString("test_token"),
					GroupID:   123456789,
					AllowFrom: []string{"123456789"},
				},
			},
		}
		ch, err := NewVKChannel(cfg, msgBus)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ch.IsAllowedSender(bus.SenderInfo{PlatformID: "123456789"}) {
			t.Error("user 123456789 should be allowed")
		}
		if ch.IsAllowedSender(bus.SenderInfo{PlatformID: "999999999"}) {
			t.Error("user 999999999 should not be allowed")
		}
	})

	t.Run("with group_trigger", func(t *testing.T) {
		cfg := &config.Config{
			Channels: config.ChannelsConfig{
				VK: config.VKConfig{
					Enabled: true,
					Token:   *config.NewSecureString("test_token"),
					GroupID: 123456789,
					GroupTrigger: config.GroupTriggerConfig{
						MentionOnly: false,
						Prefixes:    []string{"/bot", "!bot"},
					},
				},
			},
		}
		ch, err := NewVKChannel(cfg, msgBus)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.Name() != "vk" {
			t.Errorf("Name() = %q, want %q", ch.Name(), "vk")
		}
	})
}

func TestVKChannel_MaxMessageLength(t *testing.T) {
	msgBus := bus.NewMessageBus()
	cfg := &config.Config{
		Channels: config.ChannelsConfig{
			VK: config.VKConfig{
				Enabled: true,
				Token:   *config.NewSecureString("test_token"),
				GroupID: 123456789,
			},
		},
	}
	ch, err := NewVKChannel(cfg, msgBus)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	maxLen := ch.MaxMessageLength()
	if maxLen != 4000 {
		t.Errorf("MaxMessageLength() = %d, want 4000", maxLen)
	}
}

func TestVKChannel_SplitMessage(t *testing.T) {
	tests := []struct {
		name    string
		content string
		maxLen  int
		want    int
	}{
		{
			name:    "short message",
			content: "hello",
			maxLen:  4000,
			want:    1,
		},
		{
			name:    "exact length",
			content: string(make([]byte, 4000)),
			maxLen:  4000,
			want:    1,
		},
		{
			name:    "needs split",
			content: string(make([]byte, 5000)),
			maxLen:  4000,
			want:    2,
		},
		{
			name:    "empty message",
			content: "",
			maxLen:  4000,
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := channels.SplitMessage(tt.content, tt.maxLen)
			if len(got) != tt.want {
				t.Errorf("SplitMessage() got %d parts, want %d parts", len(got), tt.want)
			}
		})
	}
}

func TestVKChannel_ProcessAttachments(t *testing.T) {
	tests := []struct {
		name        string
		attachments []string
		want        string
	}{
		{
			name:        "empty attachments",
			attachments: []string{},
			want:        "",
		},
		{
			name:        "photo attachment",
			attachments: []string{"photo"},
			want:        "[photo]",
		},
		{
			name:        "video attachment",
			attachments: []string{"video"},
			want:        "[video]",
		},
		{
			name:        "audio attachment",
			attachments: []string{"audio"},
			want:        "[audio]",
		},
		{
			name:        "document attachment",
			attachments: []string{"doc"},
			want:        "[doc]",
		},
		{
			name:        "sticker attachment",
			attachments: []string{"sticker"},
			want:        "[sticker]",
		},
		{
			name:        "audio_message attachment",
			attachments: []string{"audio_message"},
			want:        "[voice]",
		},
		{
			name:        "multiple attachments",
			attachments: []string{"photo", "video", "audio"},
			want:        "[photo] [video] [audio]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			for i, att := range tt.attachments {
				if i > 0 {
					result += " "
				}
				if att == "audio_message" {
					result += "[voice]"
				} else {
					result += "[" + att + "]"
				}
			}
			if result != tt.want {
				t.Errorf("processAttachments() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestVKChannel_VoiceCapabilities(t *testing.T) {
	msgBus := bus.NewMessageBus()
	cfg := &config.Config{
		Channels: config.ChannelsConfig{
			VK: config.VKConfig{
				Enabled: true,
				Token:   *config.NewSecureString("test_token"),
				GroupID: 123456789,
			},
		},
	}
	ch, err := NewVKChannel(cfg, msgBus)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	caps := ch.VoiceCapabilities()
	if !caps.ASR {
		t.Error("VoiceCapabilities().ASR should be true")
	}
	if !caps.TTS {
		t.Error("VoiceCapabilities().TTS should be true")
	}
}
