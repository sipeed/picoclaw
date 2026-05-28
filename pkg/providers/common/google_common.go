package common

import (
	"encoding/json"
	"strings"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

// NormalizeStoredToolCall extracts the tool name, arguments, and thought signature
// from a stored ToolCall. It handles both the top-level fields and the nested
// Function struct used by different API formats.
func NormalizeStoredToolCall(tc protocoltypes.ToolCall) (string, map[string]any, string) {
	name := tc.Name
	args := tc.Arguments
	thoughtSignature := ""

	if name == "" && tc.Function != nil {
		name = tc.Function.Name
		thoughtSignature = tc.Function.ThoughtSignature
	} else if tc.Function != nil {
		thoughtSignature = tc.Function.ThoughtSignature
	}

	if args == nil {
		args = map[string]any{}
	}

	if len(args) == 0 && tc.Function != nil && tc.Function.Arguments != "" {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &parsed); err == nil && parsed != nil {
			args = parsed
		}
	}

	return name, args, thoughtSignature
}

// ResolveToolResponseName returns the tool name for a given tool call ID.
// It first checks the provided name map, then falls back to inferring the
// name from the call ID format.
func ResolveToolResponseName(toolCallID string, toolCallNames map[string]string) string {
	if toolCallID == "" {
		return ""
	}

	if name, ok := toolCallNames[toolCallID]; ok && name != "" {
		return name
	}

	return InferToolNameFromCallID(toolCallID)
}

// InferToolNameFromCallID extracts a tool name from a call ID in the format
// "call_<name>_<suffix>". Returns the original ID if it doesn't match.
func InferToolNameFromCallID(toolCallID string) string {
	if !strings.HasPrefix(toolCallID, "call_") {
		return toolCallID
	}

	rest := strings.TrimPrefix(toolCallID, "call_")
	if idx := strings.LastIndex(rest, "_"); idx > 0 {
		candidate := rest[:idx]
		if candidate != "" {
			return candidate
		}
	}

	return toolCallID
}
