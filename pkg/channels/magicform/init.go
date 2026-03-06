package magicform

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("magicform", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewMagicFormChannel(cfg.Channels.MagicForm, b)
	})
}
