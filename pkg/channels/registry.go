package channels

import (
	"fmt"
	"sync"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

// ChannelFactory is a constructor function that creates a Channel from config and message bus.
// Each channel subpackage registers one or more factories via init().
// channelName is the config map key for this channel instance (may differ from the channel type).
// channelType is the channel type string used to look up the Channel config.
type ChannelFactory func(channelName, channelType string, cfg *config.Config, bus *bus.MessageBus) (Channel, error)

var (
	factoriesMu sync.RWMutex
	factories   = map[string]ChannelFactory{}
)

// RegisterFactory registers a named channel factory. Called from subpackage init() functions.
func RegisterFactory(name string, f ChannelFactory) {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	factories[name] = f
}

// RegisterSafeFactory is a convenience wrapper that handles GetDecoded() error checking
// and type assertion, reducing boilerplate in channel init() functions.
//
// Usage:
//
//	func init() {
//	    channels.RegisterSafeFactory(config.ChannelTelegram,
//	        func(bc *config.Channel, c *config.TelegramSettings, b *bus.MessageBus) (channels.Channel, error) {
//	            return NewTelegramChannel(bc, c, b)
//	        })
//	}
func RegisterSafeFactory[S any](
	channelType string,
	ctor func(bc *config.Channel, settings *S, bus *bus.MessageBus) (Channel, error),
) {
	RegisterFactory(channelType, func(channelName, _ string, cfg *config.Config, b *bus.MessageBus) (Channel, error) {
		bc := cfg.Channels[channelName]
		if bc == nil {
			return nil, fmt.Errorf("channel %q: config not found", channelName)
		}
		decoded, err := bc.GetDecoded()
		if err != nil {
			return nil, fmt.Errorf("channel %q: failed to decode settings: %w", channelName, err)
		}
		settings, ok := decoded.(*S)
		if !ok {
			return nil, fmt.Errorf("channel %q: expected %T settings, got %T", channelName, (*S)(nil), decoded)
		}
		return ctor(bc, settings, b)
	})
}

// getFactory looks up a channel factory by name.
func getFactory(name string) (ChannelFactory, bool) {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()
	f, ok := factories[name]
	return f, ok
}

// GetRegisteredFactoryNames returns a slice of all registered channel factory names.
func GetRegisteredFactoryNames() []string {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()
	names := make([]string, 0, len(factories))
	for name := range factories {
		names = append(names, name)
	}
	return names
}
