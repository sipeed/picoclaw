package commands

import "context"

func clearCommand() Definition {
	return Definition{
		Name:        "clear",
		Description: "Clear the chat history",
		Usage:       "/clear",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if err := rt.ClearHistory(); err != nil {
				return req.Reply("Failed to clear chat history: " + err.Error())
			}
			return req.Reply("Chat history cleared!")
		},
	}
}
