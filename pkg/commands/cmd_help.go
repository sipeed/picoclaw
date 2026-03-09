package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

func helpCommand() Definition {
	return Definition{
		Name:        "help",
		Description: "Show this help message",
		Usage:       "/help",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			var defs []Definition
			if rt != nil && rt.ListDefinitions != nil {
				defs = rt.ListDefinitions()
			} else {
				defs = BuiltinDefinitions()
			}
			return req.Reply(formatHelpMessage(defs))
		},
	}
}

func formatHelpMessage(defs []Definition) string {
	if len(defs) == 0 {
		return "No commands available."
	}

	sorted := make([]Definition, len(defs))
	copy(sorted, defs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	lines := make([]string, 0, len(sorted))
	for _, def := range sorted {
		usage := def.EffectiveUsage()
		if usage == "" {
			usage = "/" + def.Name
		}
		desc := def.Description
		if desc == "" {
			desc = "No description"
		}
		lines = append(lines, fmt.Sprintf("%s - %s", usage, desc))
	}
	return strings.Join(lines, "\n")
}
