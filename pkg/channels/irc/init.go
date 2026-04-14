package irc

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory(
		config.ChannelIRC,
		func(channelName, channelType string, cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
			bc := cfg.Channels[channelName]
			if bc == nil || !bc.Enabled {
				return nil, nil
			}
			decoded, err := bc.GetDecoded()
			if err != nil {
				return nil, err
			}
			c, ok := decoded.(*config.IRCSettings)
			if !ok {
				return nil, channels.ErrSendFailed
			}
			ch, err := NewIRCChannel(bc, c, b)
			if err != nil {
				return nil, err
			}
			if channelName != config.ChannelIRC {
				ch.SetName(channelName)
			}
			return ch, nil
		},
	)
}
