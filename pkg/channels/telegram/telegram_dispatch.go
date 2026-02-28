package telegram

import (
	"context"
	"strconv"

	"github.com/mymmrac/telego"

	"github.com/sipeed/picoclaw/pkg/commands"
	"github.com/sipeed/picoclaw/pkg/logger"
)

func (c *TelegramChannel) dispatchCommand(ctx context.Context, message telego.Message) bool {
	if c.dispatcher == nil {
		return false
	}

	senderID := ""
	if message.From != nil {
		senderID = strconv.FormatInt(message.From.ID, 10)
	}

	res := c.dispatcher.Dispatch(ctx, commands.Request{
		Channel:   "telegram",
		ChatID:    strconv.FormatInt(message.Chat.ID, 10),
		SenderID:  senderID,
		Text:      message.Text,
		MessageID: strconv.Itoa(message.MessageID),
	})
	if !res.Matched {
		return false
	}

	switch res.Command {
	case "help":
		if err := c.commands.Help(ctx, message); err != nil {
			logger.ErrorCF("telegram", "Command execution failed", map[string]any{
				"command": "help",
				"error":   err.Error(),
			})
		}
	case "start":
		if err := c.commands.Start(ctx, message); err != nil {
			logger.ErrorCF("telegram", "Command execution failed", map[string]any{
				"command": "start",
				"error":   err.Error(),
			})
		}
	case "show":
		if err := c.commands.Show(ctx, message); err != nil {
			logger.ErrorCF("telegram", "Command execution failed", map[string]any{
				"command": "show",
				"error":   err.Error(),
			})
		}
	case "list":
		if err := c.commands.List(ctx, message); err != nil {
			logger.ErrorCF("telegram", "Command execution failed", map[string]any{
				"command": "list",
				"error":   err.Error(),
			})
		}
	}

	return true
}
