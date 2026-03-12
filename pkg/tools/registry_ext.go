package tools

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// ExecuteWithContext executes a tool with channel/chatID context and optional async callback.

// If the tool implements AsyncTool and a non-nil callback is provided,

// the callback will be set on the tool before execution.

func (r *ToolRegistry) ExecuteWithContext(
	ctx context.Context,

	name string,

	args map[string]any,

	channel, chatID string,

	asyncCallback AsyncCallback,
) *ToolResult {
	logger.InfoCF("tool", "Tool execution started",

		map[string]any{
			"tool": name,

			"args": args,
		})

	tool, ok := r.Get(name)

	if !ok {
		available := strings.Join(r.List(), ", ")

		logger.ErrorCF("tool", "Tool not found",

			map[string]any{
				"tool": name,
			})

		return ErrorResult(fmt.Sprintf(

			"tool %q not found. Available tools: %s", name, available,
		)).WithError(fmt.Errorf("tool not found"))
	}

	// If tool implements ContextualTool, set context

	if contextualTool, ok := tool.(ContextualTool); ok && channel != "" && chatID != "" {
		contextualTool.SetContext(channel, chatID)
	}

	// If tool implements AsyncTool and callback is provided, set callback

	if asyncTool, ok := tool.(AsyncTool); ok && asyncCallback != nil {
		asyncTool.SetCallback(asyncCallback)

		logger.DebugCF("tool", "Async callback injected",

			map[string]any{
				"tool": name,
			})
	}

	start := time.Now()

	result := tool.Execute(ctx, args)

	duration := time.Since(start)

	// Log based on result type

	if result.IsError {
		logger.ErrorCF("tool", "Tool execution failed",

			map[string]any{
				"tool": name,

				"duration": duration.Milliseconds(),

				"error": result.ForLLM,
			})
	} else if result.Async {
		logger.InfoCF("tool", "Tool started (async)",

			map[string]any{
				"tool": name,

				"duration": duration.Milliseconds(),
			})
	} else {
		logger.InfoCF("tool", "Tool execution completed",

			map[string]any{
				"tool": name,

				"duration_ms": duration.Milliseconds(),

				"result_length": len(result.ForLLM),
			})
	}

	return result
}

// ToProviderDefs converts tool definitions to provider-compatible format.

// This is the format expected by LLM provider APIs.

func (r *ToolRegistry) ToProviderDefs() []providers.ToolDefinition {
	r.mu.RLock()

	defer r.mu.RUnlock()

	sorted := r.sortedToolNames()

	definitions := make([]providers.ToolDefinition, 0, len(sorted))

	for _, name := range sorted {
		tool := r.tools[name]

		schema := ToolToSchema(tool)

		// Safely extract nested values with type checks

		fn, ok := schema["function"].(map[string]any)

		if !ok {
			continue
		}

		name, _ := fn["name"].(string)

		desc, _ := fn["description"].(string)

		params, _ := fn["parameters"].(map[string]any)

		paramsRaw := json.RawMessage(`{}`)

		if len(params) > 0 {
			if payload, err := json.Marshal(params); err == nil {
				paramsRaw = json.RawMessage(payload)
			}
		}

		definitions = append(definitions, providers.ToolDefinition{
			Type: "function",

			Function: providers.ToolFunctionDefinition{
				Name: name,

				Description: desc,

				Parameters: paramsRaw,
			},
		})
	}

	return definitions
}

// GetRuntimeStatus aggregates runtime status from all tools that implement StatusProvider.

// Returns empty string if no tool has status to report.

func (r *ToolRegistry) GetRuntimeStatus() string {
	r.mu.RLock()

	defer r.mu.RUnlock()

	var parts []string

	for _, tool := range r.tools {
		if sp, ok := tool.(StatusProvider); ok {
			if s := sp.RuntimeStatus(); s != "" {
				parts = append(parts, s)
			}
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "\n\n")
}

// buildParamHint extracts parameter names from a JSON schema and returns

// a hint string like "(task, label?, preset?)". Required params are bare,

// optional params have a trailing "?".

func buildParamHint(schema map[string]any) string {
	props, _ := schema["properties"].(map[string]any)

	if len(props) == 0 {
		return ""
	}

	reqSlice, _ := schema["required"].([]string)

	reqSet := make(map[string]bool, len(reqSlice))

	for _, r := range reqSlice {
		reqSet[r] = true
	}

	names := make([]string, 0, len(props))

	for name := range props {
		names = append(names, name)
	}

	sort.Strings(names)

	parts := make([]string, 0, len(names))

	// Required params first, then optional

	for _, name := range names {
		if reqSet[name] {
			parts = append(parts, name)
		}
	}

	for _, name := range names {
		if !reqSet[name] {
			parts = append(parts, name+"?")
		}
	}

	return "(" + strings.Join(parts, ", ") + ")"
}

// GetSummaries returns human-readable summaries of all registered tools.

// Returns a slice of "- `name`(params) - description" strings.

func (r *ToolRegistry) GetSummaries() []string {
	r.mu.RLock()

	defer r.mu.RUnlock()

	sorted := r.sortedToolNames()

	summaries := make([]string, 0, len(sorted))

	for _, name := range sorted {
		tool := r.tools[name]

		hint := buildParamHint(tool.Parameters())

		summaries = append(summaries, fmt.Sprintf("- `%s`%s - %s", tool.Name(), hint, tool.Description()))
	}

	return summaries
}
