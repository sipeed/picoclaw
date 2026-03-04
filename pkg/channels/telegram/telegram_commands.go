package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/mymmrac/telego"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
)

type TelegramCommander interface {
	Help(ctx context.Context, message telego.Message) error
	Start(ctx context.Context, message telego.Message) error
	Show(ctx context.Context, message telego.Message) error
	List(ctx context.Context, message telego.Message) error
	Switch(ctx context.Context, message telego.Message) error
}

type cmd struct {
	bot    *telego.Bot
	config *config.Config
	bus    *bus.MessageBus
}

func NewTelegramCommands(bot *telego.Bot, cfg *config.Config, bus *bus.MessageBus) TelegramCommander {
	return &cmd{
		bot:    bot,
		config: cfg,
		bus:    bus,
	}
}

func commandArgs(text string) string {
	parts := strings.SplitN(text, " ", 2)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func isTelegramSwitchAllowed(allowFrom config.FlexibleStringSlice, sender bus.SenderInfo) bool {
	if len(allowFrom) == 0 {
		return true
	}
	for _, allowedEntry := range allowFrom {
		if identity.MatchAllowed(sender, allowedEntry) {
			return true
		}
	}
	return false
}

func (c *cmd) Help(ctx context.Context, message telego.Message) error {
	msg := `/start - Start the bot
/help - Show this help message
/show [model|channel] - Show current configuration
/list [models|channels] - List available options
/switch model <name> - Switch to a different model

**Examples:**
/switch model gpt-4
/switch model claude-sonnet-4.6

Use /list models to see all available models.
`
	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: message.Chat.ID},
		Text:   msg,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	return err
}

func (c *cmd) Start(ctx context.Context, message telego.Message) error {
	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: message.Chat.ID},
		Text:   "Hello! I am PicoClaw 🦞",
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	return err
}

func (c *cmd) Show(ctx context.Context, message telego.Message) error {
	args := commandArgs(message.Text)
	if args == "" {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "Usage: /show [model|channel]",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	var response string
	switch args {
	case "model":
		currentModel := c.config.Agents.Defaults.GetModelName()
		provider := c.config.Agents.Defaults.Provider
		response = fmt.Sprintf("Current Model: %s (Provider: %s)",
			currentModel, provider)
	case "channel":
		response = "Current Channel: telegram"
	default:
		response = fmt.Sprintf("Unknown parameter: %s. Try 'model' or 'channel'.", args)
	}

	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: message.Chat.ID},
		Text:   response,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	return err
}

func (c *cmd) List(ctx context.Context, message telego.Message) error {
	args := commandArgs(message.Text)
	if args == "" {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "Usage: /list [models|channels]",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	var response string
	switch args {
	case "models":
		response = c.formatModelsList()

	case "channels":
		var enabled []string
		if c.config.Channels.Telegram.Enabled {
			enabled = append(enabled, "telegram")
		}
		if c.config.Channels.WhatsApp.Enabled {
			enabled = append(enabled, "whatsapp")
		}
		if c.config.Channels.Feishu.Enabled {
			enabled = append(enabled, "feishu")
		}
		if c.config.Channels.Discord.Enabled {
			enabled = append(enabled, "discord")
		}
		if c.config.Channels.Slack.Enabled {
			enabled = append(enabled, "slack")
		}
		response = fmt.Sprintf("Enabled Channels:\n- %s", strings.Join(enabled, "\n- "))

	default:
		response = fmt.Sprintf("Unknown parameter: %s. Try 'models' or 'channels'.", args)
	}

	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: message.Chat.ID},
		Text:   response,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	return err
}

func (c *cmd) Switch(ctx context.Context, message telego.Message) error {
	if message.From == nil {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "❌ Cannot determine sender",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	platformID := fmt.Sprintf("%d", message.From.ID)
	sender := bus.SenderInfo{
		Platform:    "telegram",
		PlatformID:  platformID,
		CanonicalID: identity.BuildCanonicalID("telegram", platformID),
		Username:    message.From.Username,
		DisplayName: message.From.FirstName,
	}
	if !isTelegramSwitchAllowed(c.config.Channels.Telegram.AllowFrom, sender) {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "❌ You are not allowed to use this command.",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	content := strings.TrimSpace(message.Text)
	content = strings.TrimPrefix(content, "/switch")
	content = strings.TrimSpace(content)

	parts := strings.SplitN(content, " ", 2)
	if len(parts) < 2 || parts[0] != "model" {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "Usage: /switch model <name>\nUse /list models to see available models.",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	modelName := strings.TrimSpace(parts[1])
	if modelName == "" {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "Usage: /switch model <name>\nUse /list models to see available models.",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	// Optional: validate model exists to provide immediate feedback
	if _, err := c.config.GetModelConfig(modelName); err != nil {
		_, sendErr := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   fmt.Sprintf("❌ Model not found: %s\n\n%s", modelName, c.formatModelsList()),
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return sendErr
	}

	if c.bus == nil {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "❌ Internal error: message bus not initialized",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	inbound := bus.InboundMessage{
		Channel:   "telegram",
		SenderID:  platformID,
		Sender:    sender,
		ChatID:    fmt.Sprintf("%d", message.Chat.ID),
		Content:   fmt.Sprintf("/switch model to %s", modelName),
		MessageID: fmt.Sprintf("%d", message.MessageID),
		Metadata: map[string]string{
			"is_group": fmt.Sprintf("%t", message.Chat.Type != "private"),
		},
	}

	if err := c.bus.PublishInbound(ctx, inbound); err != nil {
		_, sendErr := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   fmt.Sprintf("❌ Failed to switch model: %v", err),
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return sendErr
	}

	// The agent will respond via the normal outbound flow; no immediate reply here.
	return nil
}

func (c *cmd) formatModelsList() string {
	if len(c.config.ModelList) == 0 {
		return "No models configured. Please check your configuration."
	}

	currentModel := c.config.Agents.Defaults.GetModelName()

	var sb strings.Builder
	sb.WriteString("*Available Models:*\n\n")

	for _, mc := range c.config.ModelList {
		if mc.ModelName == "" {
			continue
		}

		prefix := "  "
		if mc.ModelName == currentModel {
			prefix = "✓ "
		}

		providerStr := "openai"
		if strings.Contains(mc.Model, "/") {
			protocolParts := strings.SplitN(mc.Model, "/", 2)
			if len(protocolParts) > 0 {
				providerStr = protocolParts[0]
			}
		}

		sb.WriteString(fmt.Sprintf("%s%s - %s (%s)\n", prefix, mc.ModelName, mc.Model, providerStr))
	}

	return sb.String()
}
