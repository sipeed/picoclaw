package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

func BuiltinDefinitions(cfg *config.Config) []Definition {
	return []Definition{
		{
			Name:        "start",
			Description: "Start the bot",
			Usage:       "/start",
			Channels:    []string{"telegram", "whatsapp", "whatsapp_native"},
			Handler:     replyText("Hello! I am PicoClaw 🦞"),
		},
		{
			Name:        "help",
			Description: "Show this help message",
			Usage:       "/help",
			Channels:    []string{"telegram", "whatsapp", "whatsapp_native"},
			Handler: func(_ context.Context, req Request) error {
				if req.Reply == nil {
					return nil
				}
				defs := NewRegistry(BuiltinDefinitions(cfg)).ForChannel(req.Channel)
				return req.Reply(FormatHelpMessage(defs))
			},
		},
		{
			Name:        "new",
			Aliases:     []string{"reset"},
			Description: "Start a new chat session",
			Usage:       "/new",
			Channels:    []string{"telegram", "whatsapp", "whatsapp_native"},
		},
		{
			Name:        "session",
			Description: "Manage chat sessions",
			Usage:       "/session [list|resume <index>]",
			Channels:    []string{"telegram", "whatsapp", "whatsapp_native"},
		},
		{
			Name:        "show",
			Description: "Show current configuration",
			Usage:       "/show [model|channel]",
			Channels:    []string{"telegram"},
			Handler: func(_ context.Context, req Request) error {
				if req.Reply == nil {
					return nil
				}
				if cfg == nil {
					return req.Reply("Command unavailable in current context.")
				}
				args := commandArgs(req.Text)
				if args == "" {
					return req.Reply("Usage: /show [model|channel]")
				}

				switch args {
				case "model":
					return req.Reply(fmt.Sprintf(
						"Current Model: %s (Provider: %s)",
						cfg.Agents.Defaults.GetModelName(),
						cfg.Agents.Defaults.Provider,
					))
				case "channel":
					return req.Reply(fmt.Sprintf("Current Channel: %s", req.Channel))
				default:
					return req.Reply(fmt.Sprintf("Unknown parameter: %s. Try 'model' or 'channel'.", args))
				}
			},
		},
		{
			Name:        "list",
			Description: "List available options",
			Usage:       "/list [models|channels]",
			Channels:    []string{"telegram"},
			Handler: func(_ context.Context, req Request) error {
				if req.Reply == nil {
					return nil
				}
				if cfg == nil {
					return req.Reply("Command unavailable in current context.")
				}
				args := commandArgs(req.Text)
				if args == "" {
					return req.Reply("Usage: /list [models|channels]")
				}

				switch args {
				case "models":
					provider := cfg.Agents.Defaults.Provider
					if provider == "" {
						provider = "configured default"
					}
					return req.Reply(fmt.Sprintf(
						"Configured Model: %s\nProvider: %s\n\nTo change models, update config.json",
						cfg.Agents.Defaults.GetModelName(),
						provider,
					))
				case "channels":
					enabled := enabledChannels(cfg)
					return req.Reply(fmt.Sprintf("Enabled Channels:\n- %s", strings.Join(enabled, "\n- ")))
				default:
					return req.Reply(fmt.Sprintf("Unknown parameter: %s. Try 'models' or 'channels'.", args))
				}
			},
		},
	}
}

func FormatHelpMessage(defs []Definition) string {
	if len(defs) == 0 {
		return "No commands available."
	}

	lines := make([]string, 0, len(defs))
	for _, def := range defs {
		usage := def.Usage
		if usage == "" {
			usage = "/" + def.Name
		}
		desc := def.Description
		if desc == "" {
			desc = "No description"
		}
		lines = append(lines, fmt.Sprintf("%s - %s", usage, desc))
	}
	return strings.Join(lines, "\n")
}

func commandArgs(text string) string {
	parts := strings.SplitN(text, " ", 2)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func replyText(text string) Handler {
	return func(_ context.Context, req Request) error {
		if req.Reply == nil {
			return nil
		}
		return req.Reply(text)
	}
}

func enabledChannels(cfg *config.Config) []string {
	enabled := make([]string, 0, 8)
	if cfg.Channels.Telegram.Enabled {
		enabled = append(enabled, "telegram")
	}
	if cfg.Channels.WhatsApp.Enabled {
		enabled = append(enabled, "whatsapp")
	}
	if cfg.Channels.Feishu.Enabled {
		enabled = append(enabled, "feishu")
	}
	if cfg.Channels.Discord.Enabled {
		enabled = append(enabled, "discord")
	}
	if cfg.Channels.Slack.Enabled {
		enabled = append(enabled, "slack")
	}
	if cfg.Channels.DingTalk.Enabled {
		enabled = append(enabled, "dingtalk")
	}
	if cfg.Channels.LINE.Enabled {
		enabled = append(enabled, "line")
	}
	if cfg.Channels.OneBot.Enabled {
		enabled = append(enabled, "onebot")
	}
	return enabled
}
