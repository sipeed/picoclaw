package imessage

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("imessage", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewiMessageChannel(cfg.Channels.Imessage, b)
	})
}