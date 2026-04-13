package vk

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("vk", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		if !cfg.Channels.VK.Enabled {
			return nil, nil
		}
		return NewVKChannel(cfg, b)
	})
}
