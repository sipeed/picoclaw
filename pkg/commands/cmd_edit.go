package commands

import "context"

func editCommand() Definition {
	return Definition{
		Name:        "edit",
		Description: "View or edit files (works in both chat and command mode)",
		Usage:       "/edit <file> [operation]",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.EditFile == nil {
				return req.Reply(unavailableMsg)
			}
			// Pass the full text; EditFile strips the /edit prefix internally
			return req.Reply(rt.EditFile(req.Text))
		},
	}
}
