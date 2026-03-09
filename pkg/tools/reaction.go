package tools

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/sipeed/picoclaw/pkg/config"
)

type ReactionCallback func(ctx context.Context, channel, chatID, messageID, emoji string) error

const (
	reactionTargetCurrent = "current"
	reactionTargetParent  = "parent"
	reactionTargetMessage = "message_id"
)

type ReactionTool struct {
	allowedEmoji   []string
	reactCallback  ReactionCallback
	handledInRound atomic.Bool
}

func NewReactionTool(allowedEmoji []string) *ReactionTool {
	emoji := normalizeAllowedEmoji(allowedEmoji)
	if len(emoji) == 0 {
		emoji = normalizeAllowedEmoji([]string(config.DefaultTelegramReactionEmoji()))
	}
	return &ReactionTool{allowedEmoji: emoji}
}

func normalizeAllowedEmoji(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func (t *ReactionTool) Name() string {
	return "reaction"
}

func (t *ReactionTool) Description() string {
	if len(t.allowedEmoji) == 0 {
		return "Add an emoji reaction to the current Telegram message instead of sending a text reply. Use this for short acknowledgements like thanks, ok, or got it."
	}
	return fmt.Sprintf(
		"Add an emoji reaction to the current Telegram message instead of sending a text reply. Use this for short acknowledgements like thanks, ok, or got it. You MUST choose one of these configured emojis: %s.",
		strings.Join(t.allowedEmoji, " "),
	)
}

func (t *ReactionTool) Available(ctx context.Context) bool {
	return t.reactCallback != nil && ToolChannel(ctx) == "telegram"
}

func (t *ReactionTool) Parameters() map[string]any {
	emojiSchema := map[string]any{
		"type":        "string",
		"description": "Emoji reaction to add to the target Telegram message",
	}
	if len(t.allowedEmoji) > 0 {
		emojiSchema["enum"] = append([]string(nil), t.allowedEmoji...)
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"emoji": emojiSchema,
			"target": map[string]any{
				"type":        "string",
				"description": "Which Telegram message to react to. Defaults to current.",
				"enum":        []string{reactionTargetCurrent, reactionTargetParent, reactionTargetMessage},
			},
			"message_id": map[string]any{
				"type":        "string",
				"description": "Explicit Telegram message ID when target=message_id",
			},
		},
		"required": []string{"emoji"},
	}
}

func (t *ReactionTool) ExecuteSequentially() bool {
	return true
}

func (t *ReactionTool) SetReactionCallback(callback ReactionCallback) {
	t.reactCallback = callback
}

func (t *ReactionTool) ResetHandledInRound() {
	t.handledInRound.Store(false)
}

func (t *ReactionTool) HasHandledInRound() bool {
	return t.handledInRound.Load()
}

func (t *ReactionTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	emoji, _ := args["emoji"].(string)
	emoji = strings.TrimSpace(emoji)
	if emoji == "" {
		return ErrorResult("emoji is required")
	}
	if len(t.allowedEmoji) > 0 {
		allowed := false
		for _, candidate := range t.allowedEmoji {
			if candidate == emoji {
				allowed = true
				break
			}
		}
		if !allowed {
			return ErrorResult(fmt.Sprintf("emoji %q is not allowed; use one of: %s", emoji, strings.Join(t.allowedEmoji, " ")))
		}
	}

	channel := ToolChannel(ctx)
	chatID := ToolChatID(ctx)
	if channel == "" || chatID == "" {
		return ErrorResult("reaction tool requires a current channel/chat context")
	}
	if channel != "telegram" {
		return ErrorResult("reaction tool currently supports Telegram only")
	}

	messageID, err := resolveReactionTarget(ctx, args)
	if err != nil {
		return ErrorResult(err.Error()).WithError(err)
	}
	if t.reactCallback == nil {
		return ErrorResult("reaction sending not configured")
	}

	if err := t.reactCallback(ctx, channel, chatID, messageID, emoji); err != nil {
		return ErrorResult(fmt.Sprintf("adding reaction: %v", err)).WithError(err)
	}

	t.handledInRound.Store(true)
	return SilentResult(fmt.Sprintf("Reaction %s added to telegram:%s message %s", emoji, chatID, messageID))
}

func resolveReactionTarget(ctx context.Context, args map[string]any) (string, error) {
	target, _ := args["target"].(string)
	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" {
		target = reactionTargetCurrent
	}

	switch target {
	case reactionTargetCurrent:
		if id := strings.TrimSpace(ToolCurrentMessageID(ctx)); id != "" {
			return id, nil
		}
		return "", fmt.Errorf("target=current requested but current message id is unavailable")
	case reactionTargetParent:
		if id := strings.TrimSpace(ToolParentMessageID(ctx)); id != "" {
			return id, nil
		}
		return "", fmt.Errorf("target=parent requested but parent message id is unavailable")
	case reactionTargetMessage:
		messageID, _ := args["message_id"].(string)
		messageID = strings.TrimSpace(messageID)
		if messageID == "" {
			return "", fmt.Errorf("target=message_id requires message_id")
		}
		return messageID, nil
	default:
		return "", fmt.Errorf("unsupported reaction target %q", target)
	}
}
