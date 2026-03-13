// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"encoding/json"
	"fmt"
	"strings"
)

// buildCLIToolsPrompt creates the tool definitions section for a CLI provider system prompt.
func buildCLIToolsPrompt(tools []ToolDefinition) string {
	var sb strings.Builder

	sb.WriteString("## Available Tools\n\n")
	sb.WriteString("When you need to use a tool, respond with ONLY a JSON object:\n\n")
	sb.WriteString("```json\n")
	sb.WriteString(
		`{"tool_calls":[{"id":"call_xxx","type":"function","function":{"name":"tool_name","arguments":"{...}"}}]}`,
	)
	sb.WriteString("\n```\n\n")
	sb.WriteString("CRITICAL: The 'arguments' field MUST be a JSON-encoded STRING.\n\n")
	sb.WriteString("### Tool Definitions:\n\n")

	for _, tool := range tools {
		if tool.Type != "function" {
			continue
		}
		sb.WriteString(fmt.Sprintf("#### %s\n", tool.Function.Name))
		if tool.Function.Description != "" {
			sb.WriteString(fmt.Sprintf("Description: %s\n", tool.Function.Description))
		}
		if len(tool.Function.Parameters) > 0 {
			paramsJSON, _ := json.Marshal(tool.Function.Parameters)
			sb.WriteString(fmt.Sprintf("Parameters:\n```json\n%s\n```\n", string(paramsJSON)))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// NormalizeToolCall normalizes a ToolCall to ensure all fields are properly populated.
// It handles cases where Name/Arguments might be in different locations (top-level vs Function)
// and ensures both are populated consistently.
func NormalizeToolCall(tc ToolCall) ToolCall {
	normalized := tc

	// Ensure Name is populated from Function if not set
	if normalized.Name == "" && normalized.Function != nil {
		normalized.Name = normalized.Function.Name
	}

	// Ensure Arguments is not nil
	if normalized.Arguments == nil {
		normalized.Arguments = map[string]any{}
	}

	// Parse Arguments from Function.Arguments if not already set
	if len(normalized.Arguments) == 0 && normalized.Function != nil && len(normalized.Function.Arguments) > 0 {
		normalized.Arguments = cloneToolArgs(normalized.Function.Arguments)
	}

	// Ensure Function is populated with consistent values
	if normalized.Function == nil {
		normalized.Function = &FunctionCall{
			Name:      normalized.Name,
			Arguments: cloneToolArgs(normalized.Arguments),
		}
	} else {
		if normalized.Function.Name == "" {
			normalized.Function.Name = normalized.Name
		}
		if normalized.Name == "" {
			normalized.Name = normalized.Function.Name
		}
		if len(normalized.Function.Arguments) == 0 {
			normalized.Function.Arguments = cloneToolArgs(normalized.Arguments)
		}
	}

	return normalized
}
