package chatmail

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("chatmail", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		if !cfg.Channels.Chatmail.Enabled {
			return nil, nil
		}
		return NewChatmailChannel(cfg.Channels.Chatmail, b)
	})
}
