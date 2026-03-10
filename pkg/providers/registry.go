package providers

import (
	"github.com/sipeed/picoclaw/pkg/config"
)

// ProviderResolver is a function that attempts to resolve a provider and its configuration.
// It returns the selection, a boolean indicating if it was successfully resolved, and any error.
type ProviderResolver func(cfg *config.Config, providerName string, model string) (providerSelection, bool, error)

var (
	registry = make(map[string]ProviderResolver)
)

// RegisterProvider registers a new provider resolver.
func RegisterProvider(name string, resolver ProviderResolver) {
	registry[name] = resolver
}

func init() {
	// Built-in providers will register themselves here or via init() in their respective files.
}
