package whatsapp

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("whatsapp", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		waCfg := cfg.Channels.WhatsApp
		if !waCfg.Enabled || waCfg.UseNative || waCfg.BridgeURL == "" {
			return nil, nil
		}
		return NewWhatsAppChannel(waCfg, b)
	})
}
