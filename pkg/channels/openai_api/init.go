package openai_api

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("openai_api", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewOpenAIAPIChannel(cfg, b)
	})
}
