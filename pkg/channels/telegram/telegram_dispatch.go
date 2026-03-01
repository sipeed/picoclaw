package telegram

import (
	"context"

	"github.com/mymmrac/telego"

	"github.com/sipeed/picoclaw/pkg/commands"
)

func (c *TelegramChannel) DispatchCommand(ctx context.Context, req commands.Request) commands.Result {
	if c.dispatcher == nil {
		return commands.Result{Matched: false}
	}
	return c.dispatcher.Dispatch(ctx, req)
}

func (c *TelegramChannel) dispatchCommand(ctx context.Context, message telego.Message) bool {
	// Generic slash commands are now executed in the agent-centric command path.
	// Channel adapters must not consume them locally.
	return false
}
