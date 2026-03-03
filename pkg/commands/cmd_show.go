package commands

import (
	"context"
	"fmt"
)

func showCommand(deps *Deps) Definition {
	return Definition{
		Name:        "show",
		Description: "Show current configuration",
		SubCommands: []SubCommand{
			{
				Name:        "model",
				Description: "Current model and provider",
				Handler: func(_ context.Context, req Request) error {
					if deps.GetModelInfo == nil {
						return req.Reply(unavailableMsg)
					}
					name, provider := deps.GetModelInfo()
					return req.Reply(fmt.Sprintf("Current Model: %s (Provider: %s)", name, provider))
				},
			},
			{
				Name:        "channel",
				Description: "Current channel",
				Handler: func(_ context.Context, req Request) error {
					return req.Reply(fmt.Sprintf("Current Channel: %s", req.Channel))
				},
			},
			{
				Name:        "agents",
				Description: "Registered agents",
				Handler:     agentsHandler(deps),
			},
		},
	}
}
