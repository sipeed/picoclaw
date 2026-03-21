package slack

import (
	"github.com/sipeed/piconomous/pkg/bus"
	"github.com/sipeed/piconomous/pkg/channels"
	"github.com/sipeed/piconomous/pkg/config"
)

func init() {
	channels.RegisterFactory("slack", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewSlackChannel(cfg.Channels.Slack, b)
	})
}
