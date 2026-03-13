package telegram

import (
	"jane/pkg/bus"
	"jane/pkg/channels"
	"jane/pkg/config"
)

func init() {
	channels.RegisterFactory("telegram", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewTelegramChannel(cfg, b)
	})
}
