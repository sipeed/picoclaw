package channels_test

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	_ "github.com/sipeed/picoclaw/pkg/channels/email"
	"github.com/sipeed/picoclaw/pkg/config"
)

// TestInitChannels_EmailPickedUp verifies that a channel with Enabled=true is
// actually registered in the manager after initialization.  This guards against
// the class of bug where a factory is registered in init() but the manager's
// dispatch logic is never wired up.
func TestInitChannels_EmailPickedUp(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Channels.Email = config.EmailConfig{
		Enabled:  true,
		SMTPHost: "smtp.example.com",
		SMTPFrom: *config.NewSecureString("bot@example.com"),
		IMAPHost: "imap.example.com",
		IMAPUser: *config.NewSecureString("bot@example.com"),
	}

	m, err := channels.NewManager(cfg, bus.NewMessageBus(), nil)
	if err != nil {
		t.Fatalf("NewManager returned error: %v", err)
	}

	ch, ok := m.GetChannel("email")
	if !ok || ch == nil {
		t.Error("email channel should be present in manager after NewManager with Email.Enabled=true")
	}
}

// TestInitChannels_DisabledChannelAbsent verifies that a factory returning
// (nil, nil) — i.e. the channel is disabled — does not populate the manager.
func TestInitChannels_DisabledChannelAbsent(t *testing.T) {
	cfg := config.DefaultConfig()
	// Email disabled (default)

	m, err := channels.NewManager(cfg, bus.NewMessageBus(), nil)
	if err != nil {
		t.Fatalf("NewManager returned error: %v", err)
	}

	ch, ok := m.GetChannel("email")
	if ok && ch != nil {
		t.Error("email channel should not be present in manager when Email.Enabled=false")
	}
}
