package channels

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func registerTestFactory(t *testing.T, name string, f ChannelFactory) {
	t.Helper()

	factoriesMu.Lock()
	previous, hadPrevious := factories[name]
	factories[name] = f
	factoriesMu.Unlock()

	t.Cleanup(func() {
		factoriesMu.Lock()
		defer factoriesMu.Unlock()
		if hadPrevious {
			factories[name] = previous
		} else {
			delete(factories, name)
		}
	})
}

func newInitTestManager(cfg *config.Config) *Manager {
	return &Manager{
		channels: make(map[string]Channel),
		workers:  make(map[string]*channelWorker),
		bus:      bus.NewMessageBus(),
		config:   cfg,
	}
}

func TestInitChannels_MattermostEnabled(t *testing.T) {
	registerTestFactory(t, "mattermost", func(_ *config.Config, _ *bus.MessageBus) (Channel, error) {
		return &mockChannel{}, nil
	})

	cfg := &config.Config{}
	cfg.Channels.Mattermost.Enabled = true
	cfg.Channels.Mattermost.URL = "https://mm.example.com"
	cfg.Channels.Mattermost.BotToken = "token"

	m := newInitTestManager(cfg)
	if err := m.initChannels(); err != nil {
		t.Fatalf("initChannels() error: %v", err)
	}

	if _, ok := m.channels["mattermost"]; !ok {
		t.Fatal("expected mattermost channel to be initialized")
	}
}

func TestInitChannels_MattermostMissingRequiredFields(t *testing.T) {
	called := false
	registerTestFactory(t, "mattermost", func(_ *config.Config, _ *bus.MessageBus) (Channel, error) {
		called = true
		return &mockChannel{}, nil
	})

	cfg := &config.Config{}
	cfg.Channels.Mattermost.Enabled = true
	cfg.Channels.Mattermost.URL = "https://mm.example.com"
	// BotToken intentionally empty; channel should not initialize.

	m := newInitTestManager(cfg)
	if err := m.initChannels(); err != nil {
		t.Fatalf("initChannels() error: %v", err)
	}

	if called {
		t.Fatal("expected mattermost factory not to be called without bot_token")
	}
	if _, ok := m.channels["mattermost"]; ok {
		t.Fatal("expected mattermost channel not to be initialized without bot_token")
	}
}
