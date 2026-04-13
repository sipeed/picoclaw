package telegram

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("telegram", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		if !cfg.Channels.Telegram.Enabled {
			return nil, nil
		}
		return NewTelegramChannel(cfg, b)
	})
}
