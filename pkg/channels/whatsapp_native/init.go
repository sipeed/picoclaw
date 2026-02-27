package whatsapp

import (
	"path/filepath"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
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
