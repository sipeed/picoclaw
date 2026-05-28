package commands

import (
	"context"
	"fmt"
)

func checkCommand() Definition {
	return Definition{
		Name:        "check",
		Description: "Check channel availability",
		SubCommands: []SubCommand{
			{
				Name:        "channel",
				Description: "Check if a channel is available",
				ArgsUsage:   "<name>",
				Handler: func(_ context.Context, req Request, rt *Runtime) error {
					if rt == nil || rt.SwitchChannel == nil {
						return req.Reply(unavailableMsg)
					}
					value := nthToken(req.Text, 2)
					if value == "" {
						return req.Reply("Usage: /check channel <name>")
					}
					if err := rt.SwitchChannel(value); err != nil {
						return req.Reply(err.Error())
					}
					return req.Reply(fmt.Sprintf("Channel '%s' is available and enabled", value))
				},
			},
		},
	}
}
