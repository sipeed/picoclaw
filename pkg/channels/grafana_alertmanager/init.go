package grafana_alertmanager

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory(
		"grafana_alertmanager",
		func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
			return NewGrafanaAlertmanagerChannel(cfg.Channels.GrafanaAlertmanager, b)
		},
	)
}
