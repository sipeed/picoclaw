package pico

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("pico", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		if !cfg.Channels.Pico.Enabled {
			return nil, nil
		}
		return NewPicoChannel(cfg.Channels.Pico, b)
	})
	channels.RegisterFactory("pico_client", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		if !cfg.Channels.PicoClient.Enabled {
			return nil, nil
		}
		return NewPicoClientChannel(cfg.Channels.PicoClient, b)
	})
}
