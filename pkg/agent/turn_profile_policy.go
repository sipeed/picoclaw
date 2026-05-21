package agent

import (
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func resolveTurnProfileOptions(cfg *config.Config, opts processOptions) (processOptions, error) {
	name := strings.TrimSpace(opts.TurnProfileName)
	if name == "" {
		return opts, nil
	}
	if cfg == nil {
		return opts, fmt.Errorf("unknown turn profile %q", name)
	}
	profile, _, err := cfg.Agents.Defaults.ResolveTurnProfile(name)
	if err != nil {
		return opts, err
	}
	opts.TurnProfileName = profile.Name
	opts.TurnProfile = profile
	if profile.HistoryMode == config.TurnProfileModeOff {
		opts.NoHistory = true
		opts.EnableSummary = false
	}
	return opts, nil
}

func turnProfileSystemPromptOff(profile config.EffectiveTurnProfile) bool {
	return profile.Enabled && profile.SystemPromptMode == config.TurnProfileModeOff
}

func turnProfileSkillsOff(profile config.EffectiveTurnProfile) bool {
	return profile.Enabled && profile.SkillsMode == config.TurnProfileModeOff
}

func turnProfileToolsOff(profile config.EffectiveTurnProfile) bool {
	return profile.Enabled && profile.ToolsMode == config.TurnProfileModeOff
}

func turnProfileCustomSkills(profile config.EffectiveTurnProfile) bool {
	return profile.Enabled && profile.SkillsMode == config.TurnProfileModeCustom
}

func turnProfileCustomTools(profile config.EffectiveTurnProfile) bool {
	return profile.Enabled && profile.ToolsMode == config.TurnProfileModeCustom
}

func turnProfileAllowsTools(profile config.EffectiveTurnProfile) bool {
	if !profile.Enabled {
		return true
	}
	switch profile.ToolsMode {
	case config.TurnProfileModeOff:
		return false
	case config.TurnProfileModeCustom:
		return len(profile.AllowedTools) > 0
	default:
		return true
	}
}

func turnProfileToolAllowed(profile config.EffectiveTurnProfile, name string) bool {
	if !profile.Enabled {
		return true
	}
	switch profile.ToolsMode {
	case config.TurnProfileModeOff:
		return false
	case config.TurnProfileModeCustom:
		allowed := cleanAllowedSet(profile.AllowedTools)
		if len(allowed) == 0 {
			return false
		}
		_, ok := allowed[strings.ToLower(strings.TrimSpace(name))]
		return ok
	default:
		return true
	}
}

func toolUseSystemPromptRule() string {
	return "**ALWAYS use tools** - When you need to perform an action (schedule reminders, send messages, execute commands, etc.), you MUST call the appropriate tool. Do NOT just say you'll do it or pretend to do it."
}

func numberedToolUseSystemPromptRule() string {
	return "1. " + toolUseSystemPromptRule()
}

func filterNamesByTurnProfile(names []string, allowed []string) []string {
	if len(names) == 0 {
		return nil
	}
	allowedSet := cleanAllowedSet(allowed)
	if len(allowedSet) == 0 {
		return nil
	}
	out := make([]string, 0, len(names))
	for _, name := range names {
		if _, ok := allowedSet[strings.ToLower(strings.TrimSpace(name))]; ok {
			out = append(out, name)
		}
	}
	return out
}

func filterToolsByTurnProfile(
	defs []providers.ToolDefinition,
	profile config.EffectiveTurnProfile,
) []providers.ToolDefinition {
	if !profile.Enabled {
		return defs
	}
	switch profile.ToolsMode {
	case config.TurnProfileModeOff:
		return nil
	case config.TurnProfileModeCustom:
		allowed := cleanAllowedSet(profile.AllowedTools)
		if len(allowed) == 0 {
			return nil
		}
		filtered := make([]providers.ToolDefinition, 0, len(defs))
		for _, def := range defs {
			if _, ok := allowed[strings.ToLower(strings.TrimSpace(def.Function.Name))]; ok {
				filtered = append(filtered, def)
			}
		}
		return filtered
	default:
		return defs
	}
}

func cleanAllowedSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		out[value] = struct{}{}
	}
	return out
}

func turnProfileNameFromInbound(ctx *bus.InboundContext) string {
	if ctx == nil || ctx.Raw == nil {
		return ""
	}
	return strings.TrimSpace(ctx.Raw["turn_profile"])
}
