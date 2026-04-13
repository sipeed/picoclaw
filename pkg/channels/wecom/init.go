package wecom

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("wecom", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		if !cfg.Channels.WeCom.Enabled {
			return nil, nil
		}
		return NewChannel(cfg.Channels.WeCom, b)
	})
}
