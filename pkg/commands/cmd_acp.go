package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/acp"
)

func acpCommand() Definition {
	return Definition{
		Prefix:      "acp",
		Description: "Agent Client Protocol capabilities (e.g. acp spawn, status, close)",
		Category:    "System",
		Usage:       "/acp <action> [args...]",
		Handler: func(ctx context.Context, args string, rt Runtime) error {
			parts := strings.Fields(args)
			if len(parts) == 0 {
				return fmt.Errorf("missing acp action (spawn, status, close)")
			}

			action := parts[0]
			switch action {
			case "spawn":
				if len(parts) < 2 {
					return fmt.Errorf("usage: /acp spawn <harness_id>")
				}
				harness := parts[1]
				session, err := acp.GetManager().Spawn(harness, "session", harness, "", "cli-spawned", []string{})
				if err != nil {
					return fmt.Errorf("failed to spawn ACP session: %v", err)
				}
				rt.PrintSysMessage(fmt.Sprintf("✓ Spawned ACP harness '%s'. Session Key: %s", harness, session.Key))

			case "status":
				sessions := acp.GetManager().ListSessions()
				if len(sessions) == 0 {
					rt.PrintSysMessage("No active ACP sessions.")
					return nil
				}
				var b strings.Builder
				b.WriteString("Active ACP Sessions:\n")
				for _, s := range sessions {
					b.WriteString(fmt.Sprintf("- Key: %s | Harness: %s | Status: %s\n", s.Key, s.AgentID, s.Status()))
				}
				rt.PrintSysMessage(b.String())

			case "close":
				if len(parts) < 2 {
					return fmt.Errorf("usage: /acp close <session_key>")
				}
				key := parts[1]
				err := acp.GetManager().CloseSession(key)
				if err != nil {
					return fmt.Errorf("failed to close session: %v", err)
				}
				rt.PrintSysMessage(fmt.Sprintf("✓ Closed ACP session %s", key))

			default:
				return fmt.Errorf("unknown acp action: %s", action)
			}
			return nil
		},
	}
}
