package mqtt

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("mqtt", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		if !cfg.Channels.MQTT.Enabled {
			return nil, nil
		}
		return NewMQTTChannel(cfg.Channels.MQTT, b)
	})
}
