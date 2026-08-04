package config

import (
	"fmt"
	"strings"
)

// DynamicContextTime controls the precision of the "## Current Time" line in
// the per-request dynamic context block.
type DynamicContextTime string

const (
	// DynamicContextTimeMinute emits minute precision (default).
	DynamicContextTimeMinute DynamicContextTime = "minute"
	// DynamicContextTimeHour rounds the clock down to the hour, widening the
	// window over which an identical prompt prefix can be reused.
	DynamicContextTimeHour DynamicContextTime = "hour"
	// DynamicContextTimeOff omits the current time entirely.
	DynamicContextTimeOff DynamicContextTime = "off"
)

// DynamicContextPosition controls where the per-request dynamic context block
// (time / runtime / session / sender) is placed in the message array.
type DynamicContextPosition string

const (
	// DynamicContextPositionTail places the block after conversation history,
	// immediately before the current user message (default).
	//
	// Prefix caching is positional: any change before the history invalidates
	// every token after it. Keeping the volatile block at the tail means the
	// static system prompt plus the whole history stay byte-identical from turn
	// to turn, so backends that only do byte-prefix matching (llama.cpp, Ollama)
	// hit the cache on every turn instead of missing once per minute.
	DynamicContextPositionTail DynamicContextPosition = "tail"
	// DynamicContextPositionSystem keeps the block inside the system message,
	// the layout used before this option existed. Retained as an escape hatch.
	DynamicContextPositionSystem DynamicContextPosition = "system"
)

// DynamicContextConfig is the user-facing agents.defaults.dynamic_context block.
// Zero values mean "unset" and resolve to the defaults via Effective().
type DynamicContextConfig struct {
	Time     DynamicContextTime     `json:"time,omitempty"     env:"PICOCLAW_AGENTS_DEFAULTS_DYNAMIC_CONTEXT_TIME"`
	Position DynamicContextPosition `json:"position,omitempty" env:"PICOCLAW_AGENTS_DEFAULTS_DYNAMIC_CONTEXT_POSITION"`
}

// EffectiveDynamicContext is the resolved, runtime-facing form.
type EffectiveDynamicContext struct {
	Time     DynamicContextTime
	Position DynamicContextPosition
}

// Effective normalizes the time precision, defaulting to minute precision.
// Unknown values are passed through (lowercased) so the validator can report them.
func (t DynamicContextTime) Effective() DynamicContextTime {
	normalized := DynamicContextTime(strings.ToLower(strings.TrimSpace(string(t))))
	switch normalized {
	case "":
		return DynamicContextTimeMinute
	default:
		return normalized
	}
}

// Effective normalizes the position, defaulting to tail placement.
// Unknown values are passed through (lowercased) so the validator can report them.
func (p DynamicContextPosition) Effective() DynamicContextPosition {
	normalized := DynamicContextPosition(strings.ToLower(strings.TrimSpace(string(p))))
	switch normalized {
	case "":
		return DynamicContextPositionTail
	default:
		return normalized
	}
}

// DefaultDynamicContext returns the resolved defaults used when no agent
// defaults are available (for example in tests that build a ContextBuilder
// directly).
func DefaultDynamicContext() EffectiveDynamicContext {
	return EffectiveDynamicContext{
		Time:     DynamicContextTimeMinute,
		Position: DynamicContextPositionTail,
	}
}

// ResolveDynamicContext returns the effective dynamic context settings.
// Invalid values fall back to the defaults; LoadConfig rejects them up front
// via ValidateDynamicContext, so this path only matters for programmatically
// constructed configs.
func (d *AgentDefaults) ResolveDynamicContext() EffectiveDynamicContext {
	if d == nil {
		return DefaultDynamicContext()
	}

	resolved := EffectiveDynamicContext{
		Time:     d.DynamicContext.Time.Effective(),
		Position: d.DynamicContext.Position.Effective(),
	}
	if err := validateDynamicContext(d.DynamicContext); err != nil {
		return DefaultDynamicContext()
	}
	return resolved
}

// ValidateDynamicContext validates the agents.defaults.dynamic_context block.
func (c *Config) ValidateDynamicContext() error {
	if c == nil {
		return nil
	}
	return validateDynamicContext(c.Agents.Defaults.DynamicContext)
}

func validateDynamicContext(block DynamicContextConfig) error {
	switch block.Time.Effective() {
	case DynamicContextTimeMinute, DynamicContextTimeHour, DynamicContextTimeOff:
	default:
		return fmt.Errorf(
			"dynamic_context.time has unsupported value %q (want \"minute\", \"hour\" or \"off\")",
			block.Time,
		)
	}

	switch block.Position.Effective() {
	case DynamicContextPositionTail, DynamicContextPositionSystem:
	default:
		return fmt.Errorf(
			"dynamic_context.position has unsupported value %q (want \"tail\" or \"system\")",
			block.Position,
		)
	}

	return nil
}
