package zalo

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("zalo", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewZaloChannel(cfg, b)
	})
}
