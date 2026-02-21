package channels

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// TelegramPermissionManager handles inline keyboard permission prompts for outside-workspace access.
type TelegramPermissionManager struct {
	bot     *telego.Bot
	pending sync.Map // callbackID -> chan bool
	counter atomic.Int64
}

func NewTelegramPermissionManager(bot *telego.Bot) *TelegramPermissionManager {
	return &TelegramPermissionManager{bot: bot}
}

// AskPermission sends an inline keyboard to the user and blocks until they respond.
func (pm *TelegramPermissionManager) AskPermission(ctx context.Context, chatID int64, path string) (bool, error) {
	callbackID := strconv.FormatInt(pm.counter.Add(1), 10)

	resultCh := make(chan bool, 1)
	pm.pending.Store(callbackID, resultCh)
	defer pm.pending.Delete(callbackID)

	keyboard := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			telego.InlineKeyboardButton{Text: "Allow", CallbackData: "perm_allow_" + callbackID},
			telego.InlineKeyboardButton{Text: "Deny", CallbackData: "perm_deny_" + callbackID},
		),
	)

	text := fmt.Sprintf("Agent wants to access:\n%s\n\nAllow access to this directory?", path)
	msg := tu.Message(tu.ID(chatID), text)
	msg.ReplyMarkup = keyboard

	if _, err := pm.bot.SendMessage(ctx, msg); err != nil {
		return false, fmt.Errorf("sending permission prompt: %w", err)
	}

	select {
	case approved := <-resultCh:
		return approved, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

// HandleCallback processes a callback query from an inline keyboard button press.
// Returns true if this was a permission callback, false otherwise.
func (pm *TelegramPermissionManager) HandleCallback(ctx context.Context, query telego.CallbackQuery) bool {
	data := query.Data

	var callbackID string
	var approved bool

	if strings.HasPrefix(data, "perm_allow_") {
		callbackID = strings.TrimPrefix(data, "perm_allow_")
		approved = true
	} else if strings.HasPrefix(data, "perm_deny_") {
		callbackID = strings.TrimPrefix(data, "perm_deny_")
		approved = false
	} else {
		return false
	}

	ch, ok := pm.pending.Load(callbackID)
	if !ok {
		// Expired or already handled
		if pm.bot != nil {
			_ = pm.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
				CallbackQueryID: query.ID,
				Text:            "Permission request expired",
			})
		}
		return true
	}

	ch.(chan bool) <- approved

	label := "Denied"
	if approved {
		label = "Allowed"
	}
	if pm.bot != nil {
		_ = pm.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            label,
		})
	}

	return true
}

// NewPermissionFunc creates a PermissionFunc that uses Telegram inline keyboards.
func (pm *TelegramPermissionManager) NewPermissionFunc(chatIDStr string) func(ctx context.Context, path string) (bool, error) {
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		logger.ErrorCF("telegram", "Invalid chat ID for permission func", map[string]interface{}{
			"chat_id": chatIDStr,
			"error":   err.Error(),
		})
		return nil
	}
	return func(ctx context.Context, path string) (bool, error) {
		return pm.AskPermission(ctx, chatID, path)
	}
}
