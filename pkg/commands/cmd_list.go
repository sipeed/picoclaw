package commands

import (
	"context"
	"fmt"
	"strings"
)

func listCommand(deps *Deps) Definition {
	return Definition{
		Name:        "list",
		Description: "List available options",
		SubCommands: []SubCommand{
			{
				Name:        "models",
				Description: "Configured models",
				Handler: func(_ context.Context, req Request) error {
					if req.Reply == nil {
						return nil
					}
					if deps.GetModelInfo == nil {
						return req.Reply("Command unavailable in current context.")
					}
					name, provider := deps.GetModelInfo()
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
				Handler: func(_ context.Context, req Request) error {
					if req.Reply == nil {
						return nil
					}
					if deps.GetEnabledChannels == nil {
						return req.Reply("Command unavailable in current context.")
					}
					enabled := deps.GetEnabledChannels()
					if len(enabled) == 0 {
						return req.Reply("No channels enabled")
					}
					return req.Reply(fmt.Sprintf("Enabled Channels:\n- %s", strings.Join(enabled, "\n- ")))
				},
			},
			{
				Name:        "agents",
				Description: "Registered agents",
				Handler: func(_ context.Context, req Request) error {
					if req.Reply == nil {
						return nil
					}
					if deps.ListAgentIDs == nil {
						return req.Reply("Command unavailable in current context.")
					}
					ids := deps.ListAgentIDs()
					if len(ids) == 0 {
						return req.Reply("No agents registered")
					}
					return req.Reply(fmt.Sprintf("Registered agents: %s", strings.Join(ids, ", ")))
				},
			},
		},
	}
}
