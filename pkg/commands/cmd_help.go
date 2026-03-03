package commands

import (
	"context"
	"fmt"
	"strings"
)

func helpCommand(deps *Deps) Definition {
	return Definition{
		Name:        "help",
		Description: "Show this help message",
		Usage:       "/help",
		Handler: func(_ context.Context, req Request) error {
			if req.Reply == nil {
				return nil
			}
			defs := BuiltinDefinitions(deps)
			return req.Reply(formatHelpMessage(defs))
		},
	}
}

func formatHelpMessage(defs []Definition) string {
	if len(defs) == 0 {
		return "No commands available."
	}

	lines := make([]string, 0, len(defs))
	for _, def := range defs {
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
