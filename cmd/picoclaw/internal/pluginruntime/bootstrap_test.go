package pluginruntime

import (
	"slices"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/plugin"
	"github.com/sipeed/picoclaw/pkg/plugin/builtin"
)

func TestResolveConfiguredPlugins_UnknownEnabledReturnsError(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Plugins = config.PluginsConfig{
		DefaultEnabled: false,
		Enabled:        []string{"missing-plugin"},
		Disabled:       []string{},
	}

	instances, summary, err := ResolveConfiguredPlugins(cfg)
	if err == nil {
		t.Fatal("expected error for unknown enabled plugin")
	}
	if !strings.Contains(err.Error(), "missing-plugin") {
		t.Fatalf("expected error to mention missing plugin, got %v", err)
	}
	if len(instances) != 0 {
		t.Fatalf("expected no instances on error, got %d", len(instances))
	}
	if !slices.Equal(summary.UnknownEnabled, []string{"missing-plugin"}) {
		t.Fatalf("UnknownEnabled mismatch: got %v", summary.UnknownEnabled)
	}
}

func TestResolveConfiguredPlugins_ReturnsDeterministicInstances(t *testing.T) {
	available := builtin.Names()
	if len(available) == 0 {
		t.Fatal("expected at least one builtin plugin")
	}

	enabled := slices.Clone(available)
	slices.Reverse(enabled)

	cfg := config.DefaultConfig()
	cfg.Plugins = config.PluginsConfig{
		DefaultEnabled: false,
		Enabled:        enabled,
		Disabled:       []string{},
	}

	instances, summary, err := ResolveConfiguredPlugins(cfg)
	if err != nil {
		t.Fatalf("ResolveConfiguredPlugins() error = %v", err)
	}

	gotNames := pluginNames(instances)
	if !slices.Equal(gotNames, available) {
		t.Fatalf("plugin names mismatch: got %v, want %v", gotNames, available)
	}
	if !slices.Equal(summary.Enabled, available) {
		t.Fatalf("summary enabled mismatch: got %v, want %v", summary.Enabled, available)
	}
}

func TestResolveConfiguredPlugins_UnknownDisabledWarns(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Plugins = config.PluginsConfig{
		DefaultEnabled: true,
		Enabled:        []string{},
		Disabled:       []string{"missing-plugin"},
	}

	instances, summary, err := ResolveConfiguredPlugins(cfg)
	if err != nil {
		t.Fatalf("ResolveConfiguredPlugins() error = %v", err)
	}

	expectedEnabled := builtin.Names()
	if !slices.Equal(pluginNames(instances), expectedEnabled) {
		t.Fatalf("plugin names mismatch: got %v, want %v", pluginNames(instances), expectedEnabled)
	}
	if !slices.Equal(summary.UnknownDisabled, []string{"missing-plugin"}) {
		t.Fatalf("UnknownDisabled mismatch: got %v", summary.UnknownDisabled)
	}
	if !hasWarningSubstring(summary.Warnings, `unknown disabled plugin "missing-plugin" ignored`) {
		t.Fatalf("expected unknown disabled warning, got %v", summary.Warnings)
	}
}

func pluginNames(instances []plugin.Plugin) []string {
	names := make([]string, 0, len(instances))
	for _, instance := range instances {
		names = append(names, instance.Name())
	}
	return names
}

func hasWarningSubstring(warnings []string, sub string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, sub) {
			return true
		}
	}
	return false
}
