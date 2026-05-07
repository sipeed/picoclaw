package skills

import (
	"sync"

	"github.com/sipeed/picoclaw/pkg/config"
)

type RegistryProviderBuilder func(name string, cfg config.SkillRegistryConfig) RegistryProvider

var (
	registryProviderBuildersMu sync.RWMutex
	registryProviderBuilders   = map[string]RegistryProviderBuilder{}
)

func RegisterRegistryProviderBuilder(name string, builder RegistryProviderBuilder) {
	if name == "" || builder == nil {
		return
	}
	registryProviderBuildersMu.Lock()
	defer registryProviderBuildersMu.Unlock()
	registryProviderBuilders[name] = builder
}

func buildRegistryProvider(name string, cfg config.SkillRegistryConfig) RegistryProvider {
	registryProviderBuildersMu.RLock()
	defer registryProviderBuildersMu.RUnlock()
	builder := registryProviderBuilders[name]
	if builder == nil {
		return nil
	}
	return builder(name, cfg)
}
