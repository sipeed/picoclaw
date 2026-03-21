package onebot

import (
	"github.com/sipeed/piconomous/pkg/bus"
	"github.com/sipeed/piconomous/pkg/channels"
	"github.com/sipeed/piconomous/pkg/config"
)

func init() {
	channels.RegisterFactory("onebot", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewOneBotChannel(cfg.Channels.OneBot, b)
	})
}
