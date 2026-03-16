package matrix

import (
	"context"
	"fmt"
	"strings"

	"github.com/gomarkdown/markdown"
	mdhtml "github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"jane/pkg/bus"
	"jane/pkg/channels"
	"jane/pkg/logger"
)

func markdownToHTML(md string) string {
	p := parser.NewWithExtensions(parser.CommonExtensions | parser.AutoHeadingIDs)
	renderer := mdhtml.NewRenderer(mdhtml.RendererOptions{Flags: mdhtml.CommonFlags})
	return strings.TrimSpace(string(markdown.ToHTML([]byte(md), p, renderer)))
}

func (c *MatrixChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	roomID := id.RoomID(strings.TrimSpace(msg.ChatID))
	if roomID == "" {
		return fmt.Errorf("matrix room ID is empty: %w", channels.ErrSendFailed)
	}

	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return nil
	}

	_, err := c.client.SendMessageEvent(ctx, roomID, event.EventMessage, c.messageContent(content))
	if err != nil {
		logger.ErrorCF("matrix", "Failed to send message", map[string]any{
			"room_id": roomID.String(),
			"error":   err.Error(),
		})
		return fmt.Errorf("matrix send: %w", channels.ErrTemporary)
	}

	logger.DebugCF("matrix", "Sent message", map[string]any{
		"room_id": roomID.String(),
	})
	return nil
}

func (c *MatrixChannel) messageContent(text string) *event.MessageEventContent {
	mc := &event.MessageEventContent{MsgType: event.MsgText, Body: text}
	if c.config.MessageFormat != "plain" {
		mc.Format = event.FormatHTML
		mc.FormattedBody = markdownToHTML(text)
	}
	return mc
}

// SendPlaceholder implements channels.PlaceholderCapable.
func (c *MatrixChannel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	if !c.config.Placeholder.Enabled {
		return "", nil
	}

	roomID := id.RoomID(strings.TrimSpace(chatID))
	if roomID == "" {
		return "", fmt.Errorf("matrix room ID is empty")
	}

	text := strings.TrimSpace(c.config.Placeholder.Text)
	if text == "" {
		text = "Thinking... 💭"
	}

	resp, err := c.client.SendMessageEvent(ctx, roomID, event.EventMessage, &event.MessageEventContent{
		MsgType: event.MsgNotice,
		Body:    text,
	})
	if err != nil {
		return "", err
	}

	return resp.EventID.String(), nil
}

// EditMessage implements channels.MessageEditor.
func (c *MatrixChannel) EditMessage(ctx context.Context, chatID string, messageID string, content string) error {
	roomID := id.RoomID(strings.TrimSpace(chatID))
	if roomID == "" {
		return fmt.Errorf("matrix room ID is empty")
	}
	if strings.TrimSpace(messageID) == "" {
		return fmt.Errorf("matrix message ID is empty")
	}

	editContent := c.messageContent(content)
	editContent.SetEdit(id.EventID(messageID))

	_, err := c.client.SendMessageEvent(ctx, roomID, event.EventMessage, editContent)
	if err != nil {
		logger.ErrorCF("matrix", "Failed to edit message", map[string]any{
			"room_id":    roomID.String(),
			"message_id": messageID,
			"error":      err.Error(),
		})
	} else {
		logger.DebugCF("matrix", "Edited message", map[string]any{
			"room_id":    roomID.String(),
			"message_id": messageID,
		})
	}
	return err
}
