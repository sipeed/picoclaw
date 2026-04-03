package imsg

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("imsg", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		if !cfg.Channels.IMsg.Enabled {
			return nil, nil
		}
		return NewIMsgChannel(cfg.Channels.IMsg, b)
	})
}
