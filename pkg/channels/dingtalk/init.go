package dingtalk

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("dingtalk", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		if !cfg.Channels.DingTalk.Enabled {
			return nil, nil
		}
		return NewDingTalkChannel(cfg.Channels.DingTalk, b)
	})
}
