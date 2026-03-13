package matrix

import (
	"jane/pkg/bus"
	"jane/pkg/channels"
	"jane/pkg/config"
)

func init() {
	channels.RegisterFactory("matrix", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewMatrixChannel(cfg.Channels.Matrix, b)
	})
}
