package qqc2c

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

// init registers the channel factory under the `qq_c2c` channel type, so a
// config entry such as
//
//	"channel_list": { "qq_c2c": { "enabled": true, "type": "qq_c2c",
//	    "settings": { "app_id": "...", "app_secret": "..." } } }
//
// is enough to bring the channel up. The legacy `qq` channel stays untouched.
func init() {
	// Register the settings prototype so config decoding can build QQC2CSettings.
	config.RegisterChannelSettings(config.ChannelQQC2C, config.QQC2CSettings{})

	channels.RegisterFactory(
		config.ChannelQQC2C,
		func(channelName, channelType string, cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
			bc := cfg.Channels[channelName]
			decoded, err := bc.GetDecoded()
			if err != nil {
				return nil, err
			}
			c, ok := decoded.(*config.QQC2CSettings)
			if !ok {
				return nil, channels.ErrSendFailed
			}
			return NewChannel(bc, c, b)
		},
	)
}
