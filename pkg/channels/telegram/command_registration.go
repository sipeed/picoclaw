package telegram

import (
	"context"
	"time"

	"github.com/mymmrac/telego"

	"github.com/sipeed/picoclaw/pkg/commands"
	"github.com/sipeed/picoclaw/pkg/logger"
)

var commandRegistrationBackoff = []time.Duration{
	5 * time.Second,
	15 * time.Second,
	60 * time.Second,
	5 * time.Minute,
	10 * time.Minute,
}

// RegisterCommands registers bot commands on Telegram platform.
func (c *TelegramChannel) RegisterCommands(ctx context.Context, defs []commands.Definition) error {
	botCommands := make([]telego.BotCommand, 0, len(defs))
	for _, def := range defs {
		if def.Name == "" || def.Description == "" {
			continue
		}
		botCommands = append(botCommands, telego.BotCommand{
			Command:     def.Name,
			Description: def.Description,
		})
	}

	return c.bot.SetMyCommands(ctx, &telego.SetMyCommandsParams{
		Commands: botCommands,
	})
}

func (c *TelegramChannel) startCommandRegistration(ctx context.Context, defs []commands.Definition) {
	if len(defs) == 0 {
		return
	}

	register := c.registerFunc
	if register == nil {
		register = c.RegisterCommands
	}

	regCtx, cancel := context.WithCancel(ctx)
	c.commandRegCancel = cancel

	go func() {
		attempt := 0
		for {
			err := register(regCtx, defs)
			if err == nil {
				logger.InfoCF("telegram", "Telegram commands registered", map[string]any{
					"count": len(defs),
				})
				return
			}

			delay := commandRegistrationBackoff[minInt(attempt, len(commandRegistrationBackoff)-1)]
			logger.WarnCF("telegram", "Telegram command registration failed; will retry", map[string]any{
				"error":      err.Error(),
				"retry_after": delay.String(),
			})
			attempt++

			select {
			case <-regCtx.Done():
				return
			case <-time.After(delay):
			}
		}
	}()
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
