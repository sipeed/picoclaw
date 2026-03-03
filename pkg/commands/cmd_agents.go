package commands

import (
	"context"
	"fmt"
	"strings"
)

// agentsHandler returns a shared handler for both /show agents and /list agents.
func agentsHandler(deps *Deps) Handler {
	return func(_ context.Context, req Request) error {
		if deps.ListAgentIDs == nil {
			return req.Reply(unavailableMsg)
		}
		ids := deps.ListAgentIDs()
		if len(ids) == 0 {
			return req.Reply("No agents registered")
		}
		return req.Reply(fmt.Sprintf("Registered agents: %s", strings.Join(ids, ", ")))
	}
}
