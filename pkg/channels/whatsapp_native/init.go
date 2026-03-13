package whatsapp

import (
	"path/filepath"

	"jane/pkg/bus"
	"jane/pkg/channels"
	"jane/pkg/config"
)

func init() {
	channels.RegisterFactory("whatsapp_native", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		waCfg := cfg.Channels.WhatsApp
		storePath := waCfg.SessionStorePath
		if storePath == "" {
			storePath = filepath.Join(cfg.WorkspacePath(), "whatsapp")
		}
		return NewWhatsAppNativeChannel(waCfg, b, storePath)
	})
}
