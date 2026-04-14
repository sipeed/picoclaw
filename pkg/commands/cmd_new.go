package commands

import "context"

func newCommand() Definition {
	return Definition{
		Name:        "new",
		Description: "Start a new session (reset conversation history)",
		Usage:       "/new",
		Aliases:     []string{"reset"},
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil {
				return req.Reply(unavailableMsg)
			}

			if rt.NewSession == nil {
				if rt.ClearHistory != nil {
					if err := rt.ClearHistory(); err != nil {
						return req.Reply("Failed to start new session: " + err.Error())
					}
					return req.Reply(
						"New session started.\n\n" +
							"Previous conversation has been cleared.\n" +
							"You now have a fresh context window.",
					)
				}
				return req.Reply(unavailableMsg)
			}

			if err := rt.NewSession(); err != nil {
				return req.Reply("Failed to start new session: " + err.Error())
			}

			return req.Reply(
				"New session started.\n\n" +
					"Previous conversation has been cleared.\n" +
					"You now have a fresh context window.\n\n" +
					"Tip: Use /compact to summarize older history before resetting a session.",
			)
		},
	}
}
