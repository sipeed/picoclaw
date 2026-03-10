package commands

import (
	"context"
	"fmt"
	"strings"
)

func helpCommand() Definition {
	return Definition{
		Name:        "help",
		Description: "Show this help message",
		Usage:       "/help",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			var defs []Definition
			if rt != nil && rt.ListDefinitions != nil {
				defs = rt.ListDefinitions()
			} else {
				defs = BuiltinDefinitions()
			}
			return req.Reply(formatHelpMessage(defs))
		},
	}
}

func formatHelpMessage(_ []Definition) string {
	return `System:
  /help             Show this help
  /model [name]     Show or switch the active model
  /version          Show version info
  /tools            List available tools
  /debug            Toggle debug mode
  /ping             Connectivity check
  /vps login <pw>   Set VPS password securely
  /whatsapp qr      Get WhatsApp pairing QR code
  /acp <cmd> <args> Manage active ACP harness sessions

Jobs:
  /job <desc>       Create a new job
  /status <id>      Check job status
  /cancel <id>      Cancel a job
  /list             List all jobs

Session:
  /undo             Undo last turn
  /redo             Redo undone turn
  /compact          Compress context window
  /clear            Clear current thread
  /interrupt        Stop current operation
  /new              New conversation thread
  /thread <id>      Switch to thread
  /resume <id>      Resume from checkpoint

Skills:
  /skills             List installed skills
  /skills search <q>  Search ClawHub registry

Agent:
  /heartbeat        Run heartbeat check
  /summarize        Summarize current thread
  /suggest          Suggest next steps

  /quit             Exit`
}
