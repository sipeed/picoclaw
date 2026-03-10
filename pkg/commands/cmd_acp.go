package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/acp"
)

func acpCommand() Definition {
	return Definition{
		Name:        "acp",
		Description: "Agent Client Protocol capabilities (e.g. acp spawn, status, close)",
		Usage:       "/acp <action> [args...]",
		Handler: func(ctx context.Context, req Request, rt *Runtime) error {
			parts := strings.Fields(req.Text)
			if len(parts) < 2 {
				return req.Reply("Usage: /acp <action> [args...]")
			}

			action := parts[1]
			switch action {
			case "spawn":
				if len(parts) < 3 {
					return req.Reply("usage: /acp spawn <harness_id>")
				}
				harness := parts[2]
				session, err := acp.GetManager().Spawn(harness, "session", harness, "", "cli-spawned", []string{})
				if err != nil {
					return req.Reply(fmt.Sprintf("failed to spawn ACP session: %v", err))
				}
				return req.Reply(fmt.Sprintf("✓ Spawned ACP harness '%s'. Session Key: %s", harness, session.Key))

			case "status":
				sessions := acp.GetManager().ListSessions()
				if len(sessions) == 0 {
					return req.Reply("No active ACP sessions.")
				}
				var b strings.Builder
				b.WriteString("Active ACP Sessions:\n")
				for _, s := range sessions {
					b.WriteString(fmt.Sprintf("- Key: %s | Harness: %s | Status: %s\n", s.Key, s.AgentID, s.Status()))
				}
				return req.Reply(b.String())

			case "close":
				if len(parts) < 3 {
					return req.Reply("usage: /acp close <session_key>")
				}
				key := parts[2]
				err := acp.GetManager().CloseSession(key)
				if err != nil {
					return req.Reply(fmt.Sprintf("failed to close session: %v", err))
				}
				return req.Reply(fmt.Sprintf("✓ Closed ACP session %s", key))

			default:
				return req.Reply(fmt.Sprintf("unknown acp action: %s", action))
			}
		},
	}
}
