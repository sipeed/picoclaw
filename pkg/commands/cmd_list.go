package commands

import (
	"context"
	"fmt"
	"strings"
)

func listCommand() Definition {
	return Definition{
		Name:        "list",
		Description: "List available options",
		SubCommands: []SubCommand{
			{
				Name:        "models",
				Description: "Configured models",
				Handler: func(_ context.Context, req Request, rt *Runtime) error {
					if rt == nil || rt.GetModelInfo == nil {
						return req.Reply(unavailableMsg)
					}
					name, provider := rt.GetModelInfo()
					if provider == "" {
						provider = "configured default"
					}
					return req.Reply(fmt.Sprintf(
						"Configured Model: %s\nProvider: %s\n\nTo change models, update config.json",
						name, provider,
					))
				},
			},
			{
				Name:        "channels",
				Description: "Enabled channels",
				Handler: func(_ context.Context, req Request, rt *Runtime) error {
					if rt == nil || rt.GetEnabledChannels == nil {
						return req.Reply(unavailableMsg)
					}
					enabled := rt.GetEnabledChannels()
					if len(enabled) == 0 {
						return req.Reply("No channels enabled")
					}
					return req.Reply(fmt.Sprintf("Enabled Channels:\n- %s", strings.Join(enabled, "\n- ")))
				},
			},
			{
				Name:        "agents",
				Description: "Registered agents",
				Handler:     agentsHandler(),
			},
		},
	}
}
