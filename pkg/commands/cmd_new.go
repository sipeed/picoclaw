package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

func newCommand() Definition {
	return Definition{
		Name:        "new",
		Aliases:     []string{"reset"},
		Description: "Start a new chat session",
		Usage:       "/new",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.SessionOps == nil || strings.TrimSpace(req.ScopeKey) == "" {
				return req.Reply(unavailableMsg)
			}

			newSessionKey, err := rt.SessionOps.StartNew(req.ScopeKey)
			if err != nil {
				return req.Reply(fmt.Sprintf("Failed to start new session: %v", err))
			}

			backlogLimit := config.DefaultSessionBacklogLimit
			if rt.Config != nil {
				backlogLimit = rt.Config.Session.EffectiveBacklogLimit()
			}

			pruned, err := rt.SessionOps.Prune(req.ScopeKey, backlogLimit)
			if err != nil {
				return req.Reply(fmt.Sprintf(
					"Started new session (%s), but pruning old sessions failed: %v",
					newSessionKey, err,
				))
			}

			if len(pruned) == 0 {
				return req.Reply(fmt.Sprintf("Started new session: %s", newSessionKey))
			}
			return req.Reply(
				fmt.Sprintf("Started new session: %s (pruned %d old session(s))", newSessionKey, len(pruned)),
			)
		},
	}
}
