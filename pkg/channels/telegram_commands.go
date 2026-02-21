package channels

import (
	"context"
	"fmt"
	"strings"

	"github.com/mymmrac/telego"

	"github.com/sipeed/picoclaw/pkg/config"
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
}

func NewTelegramCommands(bot *telego.Bot, cfg *config.Config) TelegramCommander {
	return &cmd{
		bot:    bot,
		config: cfg,
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
	msg := `Available commands:
  /help                     Show this help message
  /new                      Start a new conversation
  /status                   Show current session info
  /show model               Show current model
  /show channel             Show current channel
  /show agents              Show registered agents
  /list models              List available models
  /list channels            List enabled channels
  /list agents              List registered agents
  /switch model to <name>   Switch to a different model
  /switch channel to <name> Switch target channel`
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
			Text:   "Usage: /show [model|channel|agents]",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	var response string
	switch args {
	case "model":
		response = fmt.Sprintf("Current model: %s", c.config.Agents.Defaults.Model)
	case "channel":
		response = "Current channel: telegram"
	case "agents":
		agentIDs := c.listAgentIDs()
		response = fmt.Sprintf("Registered agents: %s", strings.Join(agentIDs, ", "))
	default:
		response = fmt.Sprintf("Unknown show target: %s", args)
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
			Text:   "Usage: /list [models|channels|agents]",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	var response string
	switch args {
	case "models":
		var lines []string
		// List per-agent models to match agent loop output
		for _, agent := range c.config.Agents.List {
			model := c.config.Agents.Defaults.Model
			if agent.Model != nil && agent.Model.Primary != "" {
				model = agent.Model.Primary
			}
			entry := fmt.Sprintf("  %s: %s", agent.ID, model)
			if agent.Model != nil && len(agent.Model.Fallbacks) > 0 {
				entry += fmt.Sprintf(" (fallbacks: %s)", strings.Join(agent.Model.Fallbacks, ", "))
			}
			lines = append(lines, entry)
		}
		if len(lines) == 0 {
			// Fallback: show default model if no agents are configured
			response = fmt.Sprintf("Configured models:\n  default: %s", c.config.Agents.Defaults.Model)
		} else {
			response = fmt.Sprintf("Configured models:\n%s", strings.Join(lines, "\n"))
		}

	case "channels":
		enabled := c.listEnabledChannels()
		if len(enabled) == 0 {
			response = "No channels enabled"
		} else {
			response = fmt.Sprintf("Enabled channels: %s", strings.Join(enabled, ", "))
		}

	case "agents":
		agentIDs := c.listAgentIDs()
		response = fmt.Sprintf("Registered agents: %s", strings.Join(agentIDs, ", "))

	default:
		response = fmt.Sprintf("Unknown list target: %s", args)
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
	args := commandArgs(message.Text)

	// Parse: "model to <name>" or "channel to <name>"
	parts := strings.Fields(args)
	if len(parts) < 3 || parts[1] != "to" {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "Usage: /switch [model|channel] to <name>",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	target := parts[0]
	value := parts[2]

	var response string
	switch target {
	case "model":
		oldModel := c.config.Agents.Defaults.Model
		c.config.Agents.Defaults.Model = value
		response = fmt.Sprintf("Switched model from %s to %s", oldModel, value)
	case "channel":
		response = fmt.Sprintf("Switched target channel to %s", value)
	default:
		response = fmt.Sprintf("Unknown switch target: %s", target)
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

// listEnabledChannels returns all enabled channel names from config.
func (c *cmd) listEnabledChannels() []string {
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
	if c.config.Channels.MaixCam.Enabled {
		enabled = append(enabled, "maixcam")
	}
	if c.config.Channels.QQ.Enabled {
		enabled = append(enabled, "qq")
	}
	if c.config.Channels.DingTalk.Enabled {
		enabled = append(enabled, "dingtalk")
	}
	if c.config.Channels.LINE.Enabled {
		enabled = append(enabled, "line")
	}
	if c.config.Channels.OneBot.Enabled {
		enabled = append(enabled, "onebot")
	}
	if c.config.Channels.WeCom.Enabled {
		enabled = append(enabled, "wecom")
	}
	if c.config.Channels.WeComApp.Enabled {
		enabled = append(enabled, "wecom_app")
	}
	return enabled
}

// listAgentIDs returns all configured agent IDs.
func (c *cmd) listAgentIDs() []string {
	ids := make([]string, 0, len(c.config.Agents.List))
	for _, agent := range c.config.Agents.List {
		ids = append(ids, agent.ID)
	}
	if len(ids) == 0 {
		ids = append(ids, "default")
	}
	return ids
}
