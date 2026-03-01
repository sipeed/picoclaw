package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/mymmrac/telego"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// AgentModelSwitcher is the minimal interface needed to get/set the active model.
type AgentModelSwitcher interface {
	GetDefaultAgentModel() string
	SwitchDefaultAgentModel(modelName string) (string, string, error)
}

type TelegramCommander interface {
	Help(ctx context.Context, message telego.Message) error
	Start(ctx context.Context, message telego.Message) error
	Show(ctx context.Context, message telego.Message) error
	List(ctx context.Context, message telego.Message) error
	Model(ctx context.Context, message telego.Message) error
}

type cmd struct {
	bot      *telego.Bot
	config   *config.Config
	switcher AgentModelSwitcher
}

func NewTelegramCommands(bot *telego.Bot, cfg *config.Config, switcher AgentModelSwitcher) TelegramCommander {
	return &cmd{
		bot:      bot,
		config:   cfg,
		switcher: switcher,
	}
}

func commandArgs(text string) string {
	parts := strings.SplitN(text, " ", 2)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func (c *cmd) Help(ctx context.Context, message telego.Message) error {
	msg := `/start - Start the bot
/help - Show this help message
/show [model|channel] - Show current configuration
/list [models|channels] - List available options
/model - Show or switch the active model
  /model           - show current model
  /model list      - list available models
  /model <name>    - switch to named model
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
		if c.switcher == nil {
			response = fmt.Sprintf("Current model: %s", c.config.Agents.Defaults.GetModelName())
		} else {
			response = fmt.Sprintf("Current model: %s", c.switcher.GetDefaultAgentModel())
		}
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
		provider := c.config.Agents.Defaults.Provider
		if provider == "" {
			provider = "configured default"
		}
		response = fmt.Sprintf("Configured Model: %s\nProvider: %s\n\nTo change models, update config.json",
			c.config.Agents.Defaults.GetModelName(), provider)

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

func (c *cmd) Model(ctx context.Context, message telego.Message) error {
	args := commandArgs(message.Text)

	var response string
	switch {
	case args == "":
		// Show current model
		if c.switcher == nil {
			response = "No agent configured."
		} else {
			response = fmt.Sprintf("Current model: %s", c.switcher.GetDefaultAgentModel())
		}

	case args == "list":
		// List models from config
		if len(c.config.ModelList) == 0 {
			response = "No models configured in model_list."
		} else {
			currentModel := ""
			if c.switcher != nil {
				currentModel = c.switcher.GetDefaultAgentModel()
			}
			lines := make([]string, 0, len(c.config.ModelList))
			for _, m := range c.config.ModelList {
				_, modelID := providers.ExtractProtocol(m.Model)
				line := "• " + m.ModelName
				if modelID == currentModel {
					line += " (active)"
				}
				line += " -> " + m.Model
				lines = append(lines, line)
			}
			response = "Available models:\n" + strings.Join(lines, "\n")
		}

	default:
		// Switch to the named model
		modelName := args
		if c.switcher == nil {
			response = "No agent configured."
		} else {
			oldModel, newModel, err := c.switcher.SwitchDefaultAgentModel(modelName)
			if err != nil {
				response = fmt.Sprintf("Failed to switch model: %v", err)
			} else if oldModel == newModel {
				response = fmt.Sprintf("Model already active: %s", newModel)
			} else {
				response = fmt.Sprintf("Switched model: %s -> %s", oldModel, newModel)
			}
		}
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
