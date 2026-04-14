package mattermost

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory(
		config.ChannelMattermost,
		func(channelName, channelType string, cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
			bc := cfg.Channels[channelName]
			decoded, err := bc.GetDecoded()
			if err != nil {
				return nil, err
			}
			c, ok := decoded.(*config.MattermostSettings)
			if !ok {
				return nil, channels.ErrSendFailed
			}
			return NewMattermostChannel(config.MattermostConfig{
				Enabled:            bc.Enabled,
				URL:                c.URL,
				BotToken:           c.BotToken,
				AllowFrom:          bc.AllowFrom,
				GroupTrigger:       bc.GroupTrigger,
				Typing:             bc.Typing,
				Placeholder:        bc.Placeholder,
				ReasoningChannelID: bc.ReasoningChannelID,
			}, b)
		},
	)
}
