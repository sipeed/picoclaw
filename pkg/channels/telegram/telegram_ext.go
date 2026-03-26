package telegram

import (
	"context"
	"fmt"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/sipeed/picoclaw/pkg/channels"
)

// SendWithID implements channels.MessageSenderWithID.
// It sends a message and returns the platform message ID.
func (c *TelegramChannel) SendWithID(ctx context.Context, chatID string, content string) (string, error) {
	if !c.IsRunning() {
		return "", channels.ErrNotRunning
	}

	cid, tid, err := parseTelegramChatID(chatID)
	if err != nil {
		return "", fmt.Errorf("invalid chat ID %s: %w", chatID, channels.ErrSendFailed)
	}

	htmlContent := parseContent(content, false)
	tgMsg := tu.Message(tu.ID(cid), htmlContent)
	tgMsg.ParseMode = telego.ModeHTML
	tgMsg.MessageThreadID = tid

	sent, err := c.bot.SendMessage(ctx, tgMsg)
	if err != nil {
		// Fallback to plain text
		tgMsg.ParseMode = ""
		sent, err = c.bot.SendMessage(ctx, tgMsg)
		if err != nil {
			return "", fmt.Errorf("telegram send: %w", channels.ErrTemporary)
		}
	}

	return fmt.Sprintf("%d", sent.MessageID), nil
}

// SendDraft implements channels.DraftSender.
// It uses Telegram Bot API's sendMessageDraft for progressive message streaming
// without the "edited" indicator. In groups, draft is used for dedicated topics only.
func (c *TelegramChannel) SendDraft(ctx context.Context, chatID string, draftID int, content string) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}
	cid, tid, err := parseTelegramChatID(chatID)
	if err != nil {
		return fmt.Errorf("invalid chat ID %s: %w", chatID, channels.ErrSendFailed)
	}
	if !isLikelyPrivateChatID(cid) && tid == 0 {
		return fmt.Errorf("telegram draft unsupported for non-threaded group chat: %w", channels.ErrSendFailed)
	}
	htmlContent := parseContent(content, false)
	params := &telego.SendMessageDraftParams{
		ChatID:          cid,
		MessageThreadID: tid,
		DraftID:         draftID,
		Text:            htmlContent,
		ParseMode:       telego.ModeHTML,
	}
	if err = c.bot.SendMessageDraft(ctx, params); err != nil {
		// HTML parse failure — retry as plain text
		params.ParseMode = ""
		params.Text = content
		return c.bot.SendMessageDraft(ctx, params)
	}
	return nil
}

// formatChatID formats a chat ID with optional thread ID as "chatID/threadID".
func formatChatID(chatID int64, threadID int) string {
	if threadID != 0 {
		return fmt.Sprintf("%d/%d", chatID, threadID)
	}
	return fmt.Sprintf("%d", chatID)
}

// isLikelyPrivateChatID returns true for positive chat IDs (private chats).
func isLikelyPrivateChatID(chatID int64) bool {
	return chatID > 0
}
