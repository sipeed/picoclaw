// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package hooks

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// Event represents a hook trigger point in the agent lifecycle.
type Event string

const (
	// PreMessage fires before the agent processes a user message.
	// inject_output appends hook stdout to user message context.
	PreMessage Event = "PreMessage"

	// PostMessage fires after the agent sends its final response.
	// Typically used for logging/analytics (fire-and-forget).
	PostMessage Event = "PostMessage"

	// PreToolUse fires before a tool's Execute method is called.
	// Typically used for validation/logging (fire-and-forget).
	PreToolUse Event = "PreToolUse"

	// PostToolUse fires after a tool's Execute method completes.
	// inject_output appends hook stdout to result.ForLLM.
	PostToolUse Event = "PostToolUse"
)

// HookRule defines a single hook configuration entry.
type HookRule struct {
	// Matcher filters by tool name (for tool hooks).
	// Empty string matches all tools.
	// Supports exact match ("exec"), wildcard ("*"), and prefix glob ("mcp_*").
	Matcher string `json:"matcher"`

	// Command is the shell command to execute.
	// Supports ~ expansion for home directory.
	Command string `json:"command"`

	// InjectOutput controls whether hook stdout is appended to the result.
	// For PostToolUse: appended to result.ForLLM.
	// For PreMessage: appended to user message context.
	InjectOutput bool `json:"inject_output"`
}

// HookPayload is the JSON payload passed to hook scripts via stdin.
type HookPayload struct {
	Event      Event          `json:"event"`
	ToolName   string         `json:"tool_name,omitempty"`
	ToolArgs   map[string]any `json:"tool_args,omitempty"`
	ToolOutput string         `json:"tool_output,omitempty"`
	ToolError  bool           `json:"tool_error,omitempty"`
	Channel    string         `json:"channel,omitempty"`
	ChatID     string         `json:"chat_id,omitempty"`
	Message    string         `json:"message,omitempty"`
}

// HookResult contains the output from a hook execution.
type HookResult struct {
	Output string // stdout from the hook script
	Err    error  // execution error, if any
}

// HookManager manages hook rules and triggers hook execution.
// It is safe for concurrent use.
type HookManager struct {
	rules    map[Event][]HookRule
	executor *Executor
	mu       sync.RWMutex
}

// NewHookManager creates a HookManager from configuration.
// If rules is nil or empty, the manager is effectively a no-op.
func NewHookManager(rules map[Event][]HookRule) *HookManager {
	if rules == nil {
		rules = make(map[Event][]HookRule)
	}
	return &HookManager{
		rules:    rules,
		executor: NewExecutor(),
	}
}

// HasHooks returns true if any hooks are configured for the given event.
// This is a fast-path check to avoid unnecessary work when no hooks exist.
func (hm *HookManager) HasHooks(event Event) bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return len(hm.rules[event]) > 0
}

// Trigger fires all matching hooks for the given event and payload.
// It returns a slice of HookResults (one per matching rule).
// Non-matching rules (by matcher) are skipped.
// Hooks are executed sequentially in configuration order to ensure
// deterministic output ordering.
func (hm *HookManager) Trigger(ctx context.Context, event Event, payload HookPayload) []HookResult {
	hm.mu.RLock()
	rules := hm.rules[event]
	hm.mu.RUnlock()

	if len(rules) == 0 {
		return nil
	}

	payload.Event = event

	// Build env vars from payload
	env := buildEnvVars(payload)

	// Serialize payload to JSON for stdin
	stdinData, err := json.Marshal(payload)
	if err != nil {
		logger.ErrorCF("hooks", "Failed to marshal hook payload",
			map[string]any{"event": string(event), "error": err.Error()})
		return nil
	}

	var results []HookResult
	for _, rule := range rules {
		// Matcher filtering: empty matcher matches everything;
		// non-empty must match the tool name.
		if rule.Matcher != "" && !matchesToolName(rule.Matcher, payload.ToolName) {
			continue
		}

		logger.InfoCF("hooks", "Triggering hook",
			map[string]any{
				"event":   string(event),
				"command": rule.Command,
				"matcher": rule.Matcher,
				"tool":    payload.ToolName,
			})

		result := hm.executor.Run(ctx, rule.Command, stdinData, env)
		if result.Err != nil {
			logger.WarnCF("hooks", "Hook execution failed",
				map[string]any{
					"event":   string(event),
					"command": rule.Command,
					"error":   result.Err.Error(),
				})
		} else {
			logger.DebugCF("hooks", "Hook completed",
				map[string]any{
					"event":      string(event),
					"command":    rule.Command,
					"output_len": len(result.Output),
				})
		}

		results = append(results, result)
	}

	return results
}

// CollectInjectedOutput triggers all hooks for the given event and returns
// concatenated stdout from hooks that have inject_output=true and succeeded.
// Hooks with inject_output=false are still executed for side effects.
func (hm *HookManager) CollectInjectedOutput(
	ctx context.Context,
	event Event,
	payload HookPayload,
) string {
	hm.mu.RLock()
	rules := hm.rules[event]
	hm.mu.RUnlock()

	if len(rules) == 0 {
		return ""
	}

	payload.Event = event
	env := buildEnvVars(payload)
	stdinData, err := json.Marshal(payload)
	if err != nil {
		return ""
	}

	var parts []string
	for _, rule := range rules {
		if rule.Matcher != "" && !matchesToolName(rule.Matcher, payload.ToolName) {
			continue
		}

		logger.InfoCF("hooks", "Triggering hook",
			map[string]any{
				"event":         string(event),
				"command":       rule.Command,
				"inject_output": rule.InjectOutput,
			})

		result := hm.executor.Run(ctx, rule.Command, stdinData, env)

		if result.Err != nil {
			logger.WarnCF("hooks", "Hook execution failed",
				map[string]any{
					"event":   string(event),
					"command": rule.Command,
					"error":   result.Err.Error(),
				})
			continue
		}

		if rule.InjectOutput && strings.TrimSpace(result.Output) != "" {
			parts = append(parts, strings.TrimSpace(result.Output))
		}
	}

	return strings.Join(parts, "\n")
}

// matchesToolName checks if the matcher matches the tool name.
// Supports exact match, wildcard "*" (matches everything),
// and prefix glob with trailing "*" (e.g. "mcp_*" matches "mcp_github").
func matchesToolName(matcher, toolName string) bool {
	if matcher == "*" {
		return true
	}
	if strings.HasSuffix(matcher, "*") {
		prefix := strings.TrimSuffix(matcher, "*")
		return strings.HasPrefix(toolName, prefix)
	}
	return matcher == toolName
}

// buildEnvVars constructs the environment variable slice for hook scripts.
// Large values (like tool output) are truncated to prevent E2BIG errors;
// the full payload is available via stdin JSON.
func buildEnvVars(payload HookPayload) []string {
	env := []string{
		"PICOCLAW_HOOK_EVENT=" + string(payload.Event),
	}
	if payload.ToolName != "" {
		env = append(env, "PICOCLAW_TOOL_NAME="+payload.ToolName)
	}
	if payload.ToolOutput != "" {
		// Truncate tool output in env var to prevent arg-list-too-long errors.
		// Full output is available via stdin JSON.
		output := payload.ToolOutput
		if len(output) > 8192 {
			output = output[:8192]
		}
		env = append(env, "PICOCLAW_TOOL_OUTPUT="+output)
	}
	if payload.ToolError {
		env = append(env, "PICOCLAW_TOOL_ERROR=true")
	} else {
		env = append(env, "PICOCLAW_TOOL_ERROR=false")
	}
	if payload.Channel != "" {
		env = append(env, "PICOCLAW_CHANNEL="+payload.Channel)
	}
	if payload.ChatID != "" {
		env = append(env, "PICOCLAW_CHAT_ID="+payload.ChatID)
	}
	return env
}
