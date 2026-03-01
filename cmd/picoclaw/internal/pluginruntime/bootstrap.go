package pluginruntime

import (
	"fmt"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/plugin"
	"github.com/sipeed/picoclaw/pkg/plugin/builtin"
)

type Summary struct {
	Enabled         []string
	Disabled        []string
	UnknownEnabled  []string
	UnknownDisabled []string
	Warnings        []string
}

func ResolveConfiguredPlugins(cfg *config.Config) ([]plugin.Plugin, Summary, error) {
	if cfg == nil {
		return nil, Summary{}, fmt.Errorf("config is nil")
	}

	resolved, err := plugin.ResolveSelection(
		builtin.Names(),
		plugin.SelectionInput{
			DefaultEnabled: cfg.Plugins.DefaultEnabled,
			Enabled:        cfg.Plugins.Enabled,
			Disabled:       cfg.Plugins.Disabled,
		},
	)

	summary := Summary{
		Enabled:         resolved.EnabledNames,
		Disabled:        resolved.DisabledNames,
		UnknownEnabled:  resolved.UnknownEnabled,
		UnknownDisabled: resolved.UnknownDisabled,
		Warnings:        resolved.Warnings,
	}
	if err != nil {
		return nil, summary, err
	}

	catalog := builtin.Catalog()
	normalizedCatalog := make(map[string]builtin.Factory, len(catalog))
	for name, factory := range catalog {
		normalizedCatalog[plugin.NormalizePluginName(name)] = factory
	}

	instances := make([]plugin.Plugin, 0, len(resolved.EnabledNames))
	for _, name := range resolved.EnabledNames {
		factory, ok := normalizedCatalog[name]
		if !ok {
			return nil, summary, fmt.Errorf("builtin plugin %q has no factory", name)
		}
		instance := factory()
		if instance == nil {
			return nil, summary, fmt.Errorf("builtin plugin %q factory returned nil", name)
		}
		instances = append(instances, instance)
	}

	return instances, summary, nil
}
