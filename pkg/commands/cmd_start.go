package commands

import "context"

func startCommand() Definition {
	return Definition{
		Name:        "start",
		Description: "Start the bot",
		Usage:       "/start",
		Handler: func(_ context.Context, req Request) error {
			if req.Reply == nil {
				return nil
			}
			return req.Reply("Hello! I am PicoClaw 🦞")
		},
	}
}
