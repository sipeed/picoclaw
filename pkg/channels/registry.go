package channels

import (
	"sync"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

// ChannelFactory is a constructor function that creates a Channel from config and message bus.
// Each channel subpackage registers one or more factories via init().
type ChannelFactory func(cfg *config.Config, bus *bus.MessageBus) (Channel, error)

// TelegramBotFactory is a constructor function that creates a TelegramChannel from a
// TelegramBotConfig and an explicit channel name. Registered by the telegram subpackage
// to allow the manager to initialize named bots without a direct import (avoiding cycles).
type TelegramBotFactory func(botCfg config.TelegramBotConfig, channelName string, bus *bus.MessageBus) (Channel, error)

var (
	factoriesMu sync.RWMutex
	factories   = map[string]ChannelFactory{}

	telegramBotFactoryMu sync.RWMutex
	telegramBotFactory   TelegramBotFactory
)

// RegisterFactory registers a named channel factory. Called from subpackage init() functions.
func RegisterFactory(name string, f ChannelFactory) {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	factories[name] = f
}

// getFactory looks up a channel factory by name.
func getFactory(name string) (ChannelFactory, bool) {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()
	f, ok := factories[name]
	return f, ok
}

// RegisterTelegramBotFactory registers the factory used to create named Telegram bots.
// Called from the telegram subpackage init() to avoid a direct import cycle.
func RegisterTelegramBotFactory(f TelegramBotFactory) {
	telegramBotFactoryMu.Lock()
	defer telegramBotFactoryMu.Unlock()
	telegramBotFactory = f
}

// getTelegramBotFactory returns the registered TelegramBotFactory, if any.
func getTelegramBotFactory() (TelegramBotFactory, bool) {
	telegramBotFactoryMu.RLock()
	defer telegramBotFactoryMu.RUnlock()
	return telegramBotFactory, telegramBotFactory != nil
}
