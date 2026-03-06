package builtin

import (
	"sort"

	"github.com/sipeed/picoclaw/pkg/plugin"
	"github.com/sipeed/picoclaw/pkg/plugin/demoplugin"
)

// Factory creates one builtin plugin instance.
type Factory func() plugin.Plugin

// Catalog returns compile-time builtin plugin factories by name.
func Catalog() map[string]Factory {
	return map[string]Factory{
		"policy-demo": func() plugin.Plugin {
			return demoplugin.NewPolicyDemoPlugin(demoplugin.PolicyDemoConfig{})
		},
	}
}

// Names returns sorted builtin plugin names.
func Names() []string {
	catalog := Catalog()
	names := make([]string, 0, len(catalog))
	for name := range catalog {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
