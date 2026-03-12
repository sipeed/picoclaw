package tools

import (
	"sort"
	"strings"
)

// --- Fork-only registry extensions ---

// GetRuntimeStatus aggregates runtime status from all tools that implement StatusProvider.
// Returns empty string if no tool has status to report.
func (r *ToolRegistry) GetRuntimeStatus() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var parts []string
	for _, entry := range r.tools {
		if sp, ok := entry.Tool.(StatusProvider); ok {
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
