package discord

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/voice"
)

func init() {
	channels.RegisterFactory("discord", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		ch, err := NewDiscordChannel(cfg.Channels.Discord, b)
		if err == nil {
			ch.tts = voice.DetectTTS(cfg)
		}
		return ch, err
	})
}
