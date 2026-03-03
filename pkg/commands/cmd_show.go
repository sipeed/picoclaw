package commands

import (
	"context"
	"fmt"
	"strings"
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
					if req.Reply == nil {
						return nil
					}
					if deps.GetModelInfo == nil {
						return req.Reply("Command unavailable in current context.")
					}
					name, provider := deps.GetModelInfo()
					return req.Reply(fmt.Sprintf("Current Model: %s (Provider: %s)", name, provider))
				},
			},
			{
				Name:        "channel",
				Description: "Current channel",
				Handler: func(_ context.Context, req Request) error {
					if req.Reply == nil {
						return nil
					}
					return req.Reply(fmt.Sprintf("Current Channel: %s", req.Channel))
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
