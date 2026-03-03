package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

func BuiltinDefinitions(cfg *config.Config) []Definition {
	return builtinDefinitions(cfg, nil)
}

// BuiltinDefinitionsWithRuntime returns builtin command definitions with runtime-backed
// session command handlers enabled only when runtime is usable.
func BuiltinDefinitionsWithRuntime(cfg *config.Config, runtime Runtime) []Definition {
	return builtinDefinitions(cfg, runtime)
}

func builtinDefinitions(cfg *config.Config, runtime Runtime) []Definition {
	// Runtime-backed handlers keep session-aware commands decoupled from concrete
	// agent/channel implementations. Commands are enabled only when runtime is valid.
	sessionRuntime := runtimeIfUsable(runtime)

	var newHandler Handler
	var sessionHandler Handler
	if sessionRuntime != nil {
		newHandler = func(_ context.Context, req Request) error {
			return handleNewCommand(req, sessionRuntime, cfg)
		}
		sessionHandler = func(_ context.Context, req Request) error {
			return handleSessionCommand(req, sessionRuntime)
		}
	}

	return []Definition{
		{
			Name:        "start",
			Description: "Start the bot",
			Usage:       "/start",
			Handler:     replyText("Hello! I am PicoClaw 🦞"),
		},
		{
			Name:        "help",
			Description: "Show this help message",
			Usage:       "/help",
			Handler: func(_ context.Context, req Request) error {
				if req.Reply == nil {
					return nil
				}
				defs := NewRegistry(BuiltinDefinitions(cfg)).Definitions()
				return req.Reply(FormatHelpMessage(defs))
			},
		},
		{
			Name:        "new",
			Aliases:     []string{"reset"},
			Description: "Start a new chat session",
			Usage:       "/new",
			Handler:     newHandler,
		},
		{
			Name:        "session",
			Description: "Manage chat sessions",
			Usage:       "/session [list|resume <index>]",
			Handler:     sessionHandler,
		},
		{
			Name:        "show",
			Description: "Show current configuration",
			Usage:       "/show [model|channel]",
			Handler: func(_ context.Context, req Request) error {
				return handleShowCommand(req, cfg)
			},
		},
		{
			Name:        "list",
			Description: "List available options",
			Usage:       "/list [models|channels]",
			Handler: func(_ context.Context, req Request) error {
				return handleListCommand(req, cfg)
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
		return reply(req, text)
	}
}

func handleNewCommand(req Request, runtime Runtime, fallbackCfg *config.Config) error {
	// /new rotates the active session within one runtime scope, then prunes
	// older sessions according to configured backlog policy.
	scopeKey := runtime.ScopeKey()
	newSessionKey, err := runtime.SessionOps().StartNew(scopeKey)
	if err != nil {
		return reply(req, fmt.Sprintf("Failed to start new session: %v", err))
	}

	backlogLimit := config.DefaultSessionBacklogLimit
	cfg := fallbackCfg
	if runtime.Config() != nil {
		cfg = runtime.Config()
	}
	if cfg != nil {
		backlogLimit = cfg.Session.EffectiveBacklogLimit()
	}

	pruned, err := runtime.SessionOps().Prune(scopeKey, backlogLimit)
	if err != nil {
		return reply(req, fmt.Sprintf(
			"Started new session (%s), but pruning old sessions failed: %v",
			newSessionKey,
			err,
		))
	}

	if len(pruned) == 0 {
		return reply(req, fmt.Sprintf("Started new session: %s", newSessionKey))
	}
	return reply(req, fmt.Sprintf("Started new session: %s (pruned %d old session(s))", newSessionKey, len(pruned)))
}

func handleSessionCommand(req Request, runtime Runtime) error {
	// /session subcommands always operate on runtime scope to prevent cross-chat
	// session pointer leakage.
	args := strings.Fields(commandArgs(req.Text))
	if len(args) < 1 {
		return reply(req, "Usage: /session [list|resume <index>]")
	}

	scopeKey := runtime.ScopeKey()
	switch args[0] {
	case "list":
		list, err := runtime.SessionOps().List(scopeKey)
		if err != nil {
			return reply(req, fmt.Sprintf("Failed to list sessions: %v", err))
		}
		if len(list) == 0 {
			return reply(req, "No sessions found for current chat.")
		}

		lines := make([]string, 0, len(list)+1)
		lines = append(lines, "Sessions for current chat:")
		for _, item := range list {
			activeMarker := " "
			if item.Active {
				activeMarker = "*"
			}
			updated := "-"
			if !item.UpdatedAt.IsZero() {
				updated = item.UpdatedAt.Format("2006-01-02 15:04")
			}
			lines = append(lines, fmt.Sprintf(
				"%d. [%s] %s (%d msgs, updated %s)",
				item.Ordinal,
				activeMarker,
				item.SessionKey,
				item.MessageCnt,
				updated,
			))
		}
		return reply(req, strings.Join(lines, "\n"))

	case "resume":
		if len(args) != 2 {
			return reply(req, "Usage: /session resume <index>")
		}
		index, err := strconv.Atoi(args[1])
		if err != nil || index < 1 {
			return reply(req, "Usage: /session resume <index>")
		}
		sessionKey, err := runtime.SessionOps().Resume(scopeKey, index)
		if err != nil {
			return reply(req, fmt.Sprintf("Failed to resume session %d: %v", index, err))
		}
		return reply(req, fmt.Sprintf("Resumed session %d: %s", index, sessionKey))

	default:
		return reply(req, "Usage: /session [list|resume <index>]")
	}
}

func handleShowCommand(req Request, cfg *config.Config) error {
	if cfg == nil {
		return reply(req, "Command unavailable in current context.")
	}

	args := commandArgs(req.Text)
	if args == "" {
		return reply(req, "Usage: /show [model|channel]")
	}

	switch args {
	case "model":
		return reply(req, fmt.Sprintf(
			"Current Model: %s (Provider: %s)",
			cfg.Agents.Defaults.GetModelName(),
			cfg.Agents.Defaults.Provider,
		))
	case "channel":
		return reply(req, fmt.Sprintf("Current Channel: %s", req.Channel))
	default:
		return reply(req, fmt.Sprintf("Unknown parameter: %s. Try 'model' or 'channel'.", args))
	}
}

func handleListCommand(req Request, cfg *config.Config) error {
	if cfg == nil {
		return reply(req, "Command unavailable in current context.")
	}

	args := commandArgs(req.Text)
	if args == "" {
		return reply(req, "Usage: /list [models|channels]")
	}

	switch args {
	case "models":
		provider := cfg.Agents.Defaults.Provider
		if provider == "" {
			provider = "configured default"
		}
		return reply(req, fmt.Sprintf(
			"Configured Model: %s\nProvider: %s\n\nTo change models, update config.json",
			cfg.Agents.Defaults.GetModelName(),
			provider,
		))
	case "channels":
		enabled := enabledChannels(cfg)
		return reply(req, fmt.Sprintf("Enabled Channels:\n- %s", strings.Join(enabled, "\n- ")))
	default:
		return reply(req, fmt.Sprintf("Unknown parameter: %s. Try 'models' or 'channels'.", args))
	}
}

func reply(req Request, text string) error {
	if req.Reply == nil {
		return nil
	}
	return req.Reply(text)
}

func runtimeIfUsable(runtime Runtime) Runtime {
	// Guardrails: runtime-backed handlers are disabled unless scope and session ops
	// are both present, so command registration can still expose metadata safely.
	if runtime == nil {
		return nil
	}
	if runtime.SessionOps() == nil {
		return nil
	}
	if strings.TrimSpace(runtime.ScopeKey()) == "" {
		return nil
	}
	return runtime
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
