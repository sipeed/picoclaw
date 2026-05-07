package magicform

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory(
		config.ChannelMagicForm,
		func(channelName, channelType string, cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
			bc := cfg.Channels[channelName]
			decoded, err := bc.GetDecoded()
			if err != nil {
				return nil, err
			}
			settings, ok := decoded.(*config.MagicFormSettings)
			if !ok {
				return nil, channels.ErrSendFailed
			}
			ch, err := NewMagicFormChannel(bc, settings, cfg.Agents.Defaults.WorkspaceRoot, b)
			if err != nil {
				return nil, err
			}
			if channelName != config.ChannelMagicForm {
				ch.SetName(channelName)
			}
			return ch, nil
		},
	)
}
