package commands

import (
	"context"
	"fmt"
	"strings"
)

func resetCommand() Definition {
	return Definition{
		Name:        "reset",
		Description: "Start a fresh session without deleting stored history",
		Usage:       "/reset [clear|off]",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.ResetSession == nil {
				return req.Reply(unavailableMsg)
			}

			arg := strings.ToLower(strings.TrimSpace(nthToken(req.Text, 1)))
			clearOverride := arg == "clear" || arg == "off"
			if arg != "" && !clearOverride {
				return req.Reply("Usage: /reset [clear|off]")
			}

			sessionKey, err := rt.ResetSession(clearOverride)
			if err != nil {
				return req.Reply("Failed to reset session: " + err.Error())
			}
			if clearOverride {
				return req.Reply("Soft reset cleared. Future messages will use the default routed session again.")
			}
			return req.Reply(fmt.Sprintf(
				"Started a fresh session. Previous history was preserved. New session key: %s",
				sessionKey,
			))
		},
	}
}
