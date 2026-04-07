package teamswebhook

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("teams_webhook", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewTeamsWebhookChannel(cfg.Channels.TeamsWebhook, b)
	})
}
