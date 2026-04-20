// PicoClaw - Ultra-lightweight personal AI agent

package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/commands"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func (al *AgentLoop) handleCommand(
	ctx context.Context,
	msg bus.InboundMessage,
	agent *AgentInstance,
	opts *processOptions,
) (string, bool) {
	normalizeProcessOptionsInPlace(opts)

	if !commands.HasCommandPrefix(msg.Content) {
		return "", false
	}

	if matched, handled, reply := al.applyExplicitSkillCommand(msg.Content, agent, opts); matched {
		return reply, handled
	}

	if al.cmdRegistry == nil {
		return "", false
	}

	rt := al.buildCommandsRuntime(ctx, agent, opts)
	executor := commands.NewExecutor(al.cmdRegistry, rt)

	var commandReply string
	result := executor.Execute(ctx, commands.Request{
		Channel:  msg.Channel,
		ChatID:   msg.ChatID,
		SenderID: msg.SenderID,
		Text:     msg.Content,
		Reply: func(text string) error {
			commandReply = text
			return nil
		},
	})

	switch result.Outcome {
	case commands.OutcomeHandled:
		if result.Err != nil {
			return mapCommandError(result), true
		}
		if commandReply != "" {
			return commandReply, true
		}
		return "", true
	default: // OutcomePassthrough — let the message fall through to LLM
		return "", false
	}
}

func (al *AgentLoop) applyExplicitSkillCommand(
	raw string,
	agent *AgentInstance,
	opts *processOptions,
) (matched bool, handled bool, reply string) {
	normalizeProcessOptionsInPlace(opts)

	cmdName, ok := commands.CommandName(raw)
	if !ok || cmdName != "use" {
		return false, false, ""
	}

	if agent == nil || agent.ContextBuilder == nil {
		return true, true, commandsUnavailableSkillMessage()
	}

	parts := strings.Fields(strings.TrimSpace(raw))
	if len(parts) < 2 {
		return true, true, buildUseCommandHelp(agent)
	}

	arg := strings.TrimSpace(parts[1])
	if strings.EqualFold(arg, "clear") || strings.EqualFold(arg, "off") {
		if opts != nil {
			al.clearPendingSkills(opts.Dispatch.SessionKey)
		}
		return true, true, "Cleared pending skill override."
	}

	skillName, ok := agent.ContextBuilder.ResolveSkillName(arg)
	if !ok {
		return true, true, fmt.Sprintf("Unknown skill: %s\nUse /list skills to see installed skills.", arg)
	}

	if len(parts) < 3 {
		if opts == nil || strings.TrimSpace(opts.Dispatch.SessionKey) == "" {
			return true, true, commandsUnavailableSkillMessage()
		}
		al.setPendingSkills(opts.Dispatch.SessionKey, []string{skillName})
		return true, true, fmt.Sprintf(
			"Skill %q is armed for your next message. Send your next prompt normally, or use /use clear to cancel.",
			skillName,
		)
	}

	message := strings.TrimSpace(strings.Join(parts[2:], " "))
	if message == "" {
		return true, true, buildUseCommandHelp(agent)
	}

	if opts != nil {
		opts.ForcedSkills = append(opts.ForcedSkills, skillName)
		opts.Dispatch.UserMessage = message
		opts.UserMessage = message
	}

	return true, false, ""
}

func (al *AgentLoop) buildCommandsRuntime(
	ctx context.Context,
	agent *AgentInstance,
	opts *processOptions,
) *commands.Runtime {
	normalizeProcessOptionsInPlace(opts)

	registry := al.GetRegistry()
	cfg := al.GetConfig()
	rt := &commands.Runtime{
		Config:          cfg,
		ListAgentIDs:    registry.ListAgentIDs,
		ListDefinitions: al.cmdRegistry.Definitions,
		GetEnabledChannels: func() []string {
			if al.channelManager == nil {
				return nil
			}
			return al.channelManager.GetEnabledChannels()
		},
		GetActiveTurn: func() any {
			info := al.GetActiveTurn()
			if info == nil {
				return nil
			}
			return info
		},
		SwitchChannel: func(value string) error {
			if al.channelManager == nil {
				return fmt.Errorf("channel manager not initialized")
			}
			if _, exists := al.channelManager.GetChannel(value); !exists && value != "cli" {
				return fmt.Errorf("channel '%s' not found or not enabled", value)
			}
			return nil
		},
	}
	if agent != nil && agent.ContextBuilder != nil {
		rt.ListSkillNames = agent.ContextBuilder.ListSkillNames
	}
	rt.ReloadConfig = func() error {
		if al.reloadFunc == nil {
			return fmt.Errorf("reload not configured")
		}
		return al.reloadFunc()
	}
	if agent != nil {
		if agent.ContextBuilder != nil {
			rt.ListSkillNames = agent.ContextBuilder.ListSkillNames
		}
		rt.GetModelInfo = func() (string, string) {
			return agent.Model, resolvedCandidateProvider(agent.Candidates, cfg.Agents.Defaults.Provider)
		}
		rt.SwitchModel = func(value string) (string, error) {
			value = strings.TrimSpace(value)
			modelCfg, err := resolvedModelConfig(cfg, value, agent.Workspace)
			if err != nil {
				return "", err
			}

			nextProvider, _, err := providers.CreateProviderFromConfig(modelCfg)
			if err != nil {
				return "", fmt.Errorf("failed to initialize model %q: %w", value, err)
			}

			nextCandidates := resolveModelCandidates(cfg, cfg.Agents.Defaults.Provider, value, agent.Fallbacks)
			if len(nextCandidates) == 0 {
				return "", fmt.Errorf("model %q did not resolve to any provider candidates", value)
			}

			oldModel := agent.Model
			oldProvider := agent.Provider
			agent.Model = value
			agent.Provider = nextProvider
			agent.Candidates = nextCandidates
			agent.ThinkingLevel = parseThinkingLevel(modelCfg.ThinkingLevel)

			if oldProvider != nil && oldProvider != nextProvider {
				if stateful, ok := oldProvider.(providers.StatefulProvider); ok {
					stateful.Close()
				}
			}
			return oldModel, nil
		}

		rt.ClearHistory = func() error {
			if opts == nil {
				return fmt.Errorf("process options not available")
			}
			return al.contextManager.Clear(ctx, opts.SessionKey)
		}

		rt.AskSideQuestion = func(ctx context.Context, question string) (string, error) {
			return al.askSideQuestion(ctx, agent, opts, question)
		}
	}
	return rt
}

func (al *AgentLoop) setPendingSkills(sessionKey string, skillNames []string) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" || len(skillNames) == 0 {
		return
	}

	filtered := make([]string, 0, len(skillNames))
	for _, name := range skillNames {
		name = strings.TrimSpace(name)
		if name != "" {
			filtered = append(filtered, name)
		}
	}
	if len(filtered) == 0 {
		return
	}

	al.pendingSkills.Store(sessionKey, filtered)
}

func (al *AgentLoop) takePendingSkills(sessionKey string) []string {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return nil
	}

	value, ok := al.pendingSkills.LoadAndDelete(sessionKey)
	if !ok {
		return nil
	}

	skills, ok := value.([]string)
	if !ok {
		return nil
	}

	return append([]string(nil), skills...)
}

func (al *AgentLoop) clearPendingSkills(sessionKey string) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return
	}
	al.pendingSkills.Delete(sessionKey)
}
