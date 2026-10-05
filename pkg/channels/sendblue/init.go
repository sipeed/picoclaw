package sendblue

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory(config.ChannelSendblue,
		func(name, channelType string, cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
			bc := cfg.Channels[name]
			decoded, err := bc.GetDecoded()
			if err != nil {
				return nil, err
			}
			settings, ok := decoded.(*config.SendblueSettings)
			if !ok {
				return nil, channels.ErrSendFailed
			}
			ch, err := NewSendblueChannel(bc, settings, b)
			if err == nil {
				ch.SetName(name)
			}
			return ch, err
		})
}
