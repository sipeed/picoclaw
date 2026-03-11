package commands

import (
	"context"
	"strings"
)

// execCommand registers /exec (alias: !) for running shell commands.
// Available in both pico and cmd mode; in cmd mode bare text is rewritten to /exec
// before dispatch so all shell execution flows through this single handler.
func execCommand() Definition {
	return Definition{
		Name:        "exec",
		Description: "Execute a shell command",
		Usage:       "/exec <command>",
		Handler: func(ctx context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.ExecCmd == nil {
				return req.Reply(unavailableMsg)
			}
			// Strip the leading command token ("/exec", "!exec", or "!" fallback)
			// and treat the remainder as the shell command to run.
			args := strings.TrimSpace(req.Text)
			if idx := strings.IndexAny(args, " \t"); idx >= 0 {
				args = strings.TrimSpace(args[idx:])
			} else {
				args = ""
			}
			if args == "" {
				return req.Reply("Usage: /exec <command>\nExample: /exec ls -la")
			}
			result, err := rt.ExecCmd(ctx, args)
			if err != nil {
				return req.Reply("Error: " + err.Error())
			}
			return req.Reply(result)
		},
	}
}
