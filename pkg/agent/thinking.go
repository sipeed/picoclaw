package agent

import (
	"regexp"
	"slices"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

// ThinkingLevel controls how the provider sends thinking parameters (main compatibility).
// Session-level resolution uses string internally; this type is used for agent default from model config.
type ThinkingLevel string

const (
	ThinkingOff      ThinkingLevel = "off"
	ThinkingLow      ThinkingLevel = "low"
	ThinkingMedium   ThinkingLevel = "medium"
	ThinkingHigh     ThinkingLevel = "high"
	ThinkingXHigh    ThinkingLevel = "xhigh"
	ThinkingAdaptive ThinkingLevel = "adaptive"
)

// parseThinkingLevel normalizes a config string to a ThinkingLevel (main compatibility).
// Case-insensitive and whitespace-tolerant. Returns ThinkingOff for unknown or empty.
func parseThinkingLevel(level string) ThinkingLevel {
	if normalized, ok := normalizeThinkingLevel(level); ok {
		return ThinkingLevel(normalized)
	}
	return ThinkingOff
}

// Unexported string constants for internal use (session, downgrade, aliases).
const (
	thinkingOff      = "off"
	thinkingMinimal  = "minimal"
	thinkingLow      = "low"
	thinkingMedium   = "medium"
	thinkingHigh     = "high"
	thinkingXHigh    = "xhigh"
	thinkingAdaptive = "adaptive"
)

var (
	supportedThinkingLevels = []string{
		thinkingOff, thinkingMinimal, thinkingLow, thinkingMedium, thinkingHigh, thinkingXHigh, thinkingAdaptive,
	}
	thinkingDowngradeOrder = []string{
		thinkingXHigh, thinkingHigh, thinkingMedium, thinkingLow, thinkingMinimal, thinkingOff,
	}
	supportedValuesRe = regexp.MustCompile(`(?i)supported values?\s*[:=]\s*([^\n]+)`)
)

func normalizeThinkingLevel(raw string) (string, bool) {
	level := strings.ToLower(strings.TrimSpace(raw))
	switch level {
	case "none":
		return thinkingOff, true
	case "on", "enable", "enabled":
		return thinkingLow, true
	case "off", "disable", "disabled":
		return thinkingOff, true
	case thinkingMinimal, thinkingLow, thinkingMedium, thinkingHigh, thinkingXHigh, thinkingAdaptive:
		return level, true
	default:
		return "", false
	}
}

func resolveThinkingLevel(cfg *config.Config, agent *AgentInstance, sessionKey string) string {
	if sessionLevel, ok := normalizeThinkingLevel(agent.Sessions.GetThinkingLevel(sessionKey)); ok {
		return sessionLevel
	}
	if modelLevel := modelThinkingLevel(cfg, agent.Model); modelLevel != "" {
		return modelLevel
	}
	// Main compatibility: fall back to agent default (from model config at creation).
	if agent.ThinkingLevel != ThinkingOff {
		return string(agent.ThinkingLevel)
	}
	if cfg != nil {
		if level, ok := normalizeThinkingLevel(cfg.Agents.Defaults.Thinking); ok {
			return level
		}
	}
	return ""
}

func modelThinkingLevel(cfg *config.Config, modelAlias string) string {
	if cfg == nil {
		return ""
	}
	for i := range cfg.ModelList {
		mc := cfg.ModelList[i]
		if mc.ModelName != modelAlias {
			continue
		}
		if level, ok := normalizeThinkingLevel(mc.ThinkingLevel); ok {
			return level
		}
	}
	return ""
}

// reasoningEffortForLevel maps an internal thinking level to the
// reasoning_effort value sent to providers.  "off" and "adaptive" return ""
// so that the field is omitted entirely — this avoids errors on providers
// that do not support the reasoning_effort parameter.
func reasoningEffortForLevel(level string) string {
	switch level {
	case thinkingMinimal, thinkingLow, thinkingMedium, thinkingHigh, thinkingXHigh:
		return level
	default:
		return ""
	}
}

func supportsXHigh(modelAlias string, cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	for i := range cfg.ModelList {
		mc := cfg.ModelList[i]
		if mc.ModelName != modelAlias {
			continue
		}
		model := strings.ToLower(strings.TrimSpace(mc.Model))
		if strings.Contains(model, "gpt-5.2") || strings.Contains(model, "codex") {
			return true
		}
	}
	return false
}

func parseSupportedThinkingLevels(errMsg string) []string {
	errMsg = strings.ToLower(errMsg)
	match := supportedValuesRe.FindStringSubmatch(errMsg)
	if len(match) == 0 {
		return nil
	}
	segment := match[1]
	candidates := []string{
		"none", "minimal", "low", "medium", "high", "xhigh", "adaptive", "off",
	}
	levels := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if !strings.Contains(segment, c) {
			continue
		}
		if normalized, ok := normalizeThinkingLevel(c); ok && !slices.Contains(levels, normalized) {
			levels = append(levels, normalized)
		}
	}
	return levels
}

func isThinkingUnsupportedError(errMsg string) bool {
	msg := strings.ToLower(errMsg)
	if strings.Contains(msg, "reasoning_effort") || strings.Contains(msg, "reasoning.effort") {
		return true
	}
	return strings.Contains(msg, "thinking") && strings.Contains(msg, "not supported")
}

func nextDowngradedThinkingLevel(current string, supported []string) (string, bool) {
	if current == "" || current == thinkingAdaptive {
		return thinkingOff, true
	}
	idx := slices.Index(thinkingDowngradeOrder, current)
	if idx < 0 {
		return thinkingOff, true
	}
	for i := idx + 1; i < len(thinkingDowngradeOrder); i++ {
		candidate := thinkingDowngradeOrder[i]
		if len(supported) == 0 || slices.Contains(supported, candidate) {
			return candidate, true
		}
	}
	return "", false
}
