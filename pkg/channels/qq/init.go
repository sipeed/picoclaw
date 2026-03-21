package qq

import (
	"github.com/sipeed/piconomous/pkg/bus"
	"github.com/sipeed/piconomous/pkg/channels"
	"github.com/sipeed/piconomous/pkg/config"
)

func init() {
	channels.RegisterFactory("qq", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewQQChannel(cfg.Channels.QQ, b)
	})
}
