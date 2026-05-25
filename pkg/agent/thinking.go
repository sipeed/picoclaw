package agent

import (
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// ThinkingLevel controls how the provider sends thinking parameters.
//
//   - "adaptive": sends {thinking: {type: "adaptive"}} + output_config.effort (Claude 4.6+)
//   - "low"/"medium"/"high"/"xhigh": sends {thinking: {type: "enabled", budget_tokens: N}} (all models)
//   - "off": disables thinking
type ThinkingLevel string

const (
	ThinkingOff      ThinkingLevel = "off"
	ThinkingLow      ThinkingLevel = "low"
	ThinkingMedium   ThinkingLevel = "medium"
	ThinkingHigh     ThinkingLevel = "high"
	ThinkingXHigh    ThinkingLevel = "xhigh"
	ThinkingAdaptive ThinkingLevel = "adaptive"
)

// parseThinkingLevel normalizes a config string to a ThinkingLevel.
// Case-insensitive and whitespace-tolerant for user-facing config values.
// Returns ThinkingOff for unknown or empty values.
func parseThinkingLevel(level string) ThinkingLevel {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "adaptive":
		return ThinkingAdaptive
	case "low":
		return ThinkingLow
	case "medium":
		return ThinkingMedium
	case "high":
		return ThinkingHigh
	case "xhigh":
		return ThinkingXHigh
	default:
		return ThinkingOff
	}
}

func isConfiguredThinkingLevel(level string) bool {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "off", "low", "medium", "high", "xhigh", "adaptive":
		return true
	default:
		return false
	}
}

type thinkingSettings struct {
	level      ThinkingLevel
	configured bool
}

func thinkingSettingsFromModelConfig(mc *config.ModelConfig) thinkingSettings {
	if mc == nil || !isConfiguredThinkingLevel(mc.ThinkingLevel) {
		return thinkingSettings{}
	}
	return thinkingSettings{
		level:      parseThinkingLevel(mc.ThinkingLevel),
		configured: true,
	}
}

func activeThinkingSettings(agent *AgentInstance, modelCfg *config.ModelConfig) thinkingSettings {
	if settings := thinkingSettingsFromModelConfig(modelCfg); settings.configured {
		return settings
	}
	if modelCfg == nil && agent != nil {
		return thinkingSettings{
			level:      agent.ThinkingLevel,
			configured: agent.ThinkingLevelConfigured,
		}
	}
	return thinkingSettings{}
}

func applyThinkingOption(
	opts map[string]any,
	provider providers.LLMProvider,
	settings thinkingSettings,
	warnUnsupported bool,
	agentID string,
) {
	if opts == nil || !settings.configured {
		return
	}
	if settings.level == ThinkingOff {
		opts["thinking_level"] = string(settings.level)
		return
	}
	if tc, ok := provider.(providers.ThinkingCapable); ok && tc.SupportsThinking() {
		opts["thinking_level"] = string(settings.level)
		return
	}
	if warnUnsupported {
		logger.WarnCF("agent", "thinking_level is set but current provider does not support it, ignoring",
			map[string]any{"agent_id": agentID, "thinking_level": string(settings.level)})
	}
}

func applyTurnThinkingOptions(
	exec *turnExecution,
	agent *AgentInstance,
	provider providers.LLMProvider,
	warnUnsupported bool,
) {
	if exec == nil || exec.llmOpts == nil {
		return
	}
	delete(exec.llmOpts, "thinking_level")
	settings := activeThinkingSettings(agent, exec.activeModelConfig)
	agentID := ""
	if agent != nil {
		agentID = agent.ID
	}
	applyThinkingOption(exec.llmOpts, provider, settings, warnUnsupported, agentID)
	exec.suppressReasoning = shouldSuppressReasoningFor(settings)
}

func shouldSuppressReasoningFor(settings thinkingSettings) bool {
	return settings.configured && settings.level == ThinkingOff
}
