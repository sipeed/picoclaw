package telegram

import (
	"context"
	"strconv"

	"github.com/mymmrac/telego"

	"github.com/sipeed/picoclaw/pkg/commands"
	"github.com/sipeed/picoclaw/pkg/logger"
)

func (c *TelegramChannel) DispatchCommand(ctx context.Context, req commands.Request) commands.Result {
	if c.dispatcher == nil {
		return commands.Result{Matched: false}
	}
	return c.dispatcher.Dispatch(ctx, req)
}

func (c *TelegramChannel) dispatchCommand(ctx context.Context, message telego.Message) bool {
	senderID := ""
	if message.From != nil {
		senderID = strconv.FormatInt(message.From.ID, 10)
	}

	res := c.DispatchCommand(ctx, commands.Request{
		Channel:   "telegram",
		ChatID:    strconv.FormatInt(message.Chat.ID, 10),
		SenderID:  senderID,
		Text:      message.Text,
		MessageID: strconv.Itoa(message.MessageID),
		Reply: func(text string) error {
			_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
				ChatID: telego.ChatID{ID: message.Chat.ID},
				Text:   text,
				ReplyParameters: &telego.ReplyParameters{
					MessageID: message.MessageID,
				},
			})
			return err
		},
	})
	if !res.Matched {
		return false
	}

	if res.Err != nil {
		logger.ErrorCF("telegram", "Command execution failed", map[string]any{
			"command": res.Command,
			"error":   res.Err.Error(),
		})
	}

	return true
}
