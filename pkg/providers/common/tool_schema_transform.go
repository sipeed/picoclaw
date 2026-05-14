package common

import (
	"fmt"
	"strings"
)

const (
	ToolSchemaTransformOff    = ""
	ToolSchemaTransformSimple = "simple"
)

// NormalizeToolSchemaTransform resolves user-facing aliases to a canonical
// transform mode. Empty values and explicit "off"-style values disable schema
// transformation.
func NormalizeToolSchemaTransform(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "off", "none", "native":
		return ToolSchemaTransformOff, nil
	case "simple", "basic", "strict", "flat":
		return ToolSchemaTransformSimple, nil
	default:
		return "", fmt.Errorf("unsupported tool_schema_transform %q (supported: off, simple)", raw)
	}
}

// TransformToolDefinitions clones tool definitions and applies the configured
// schema transform to function parameter schemas. When the transform is off, the
// original slice is returned unchanged.
func TransformToolDefinitions(tools []ToolDefinition, transform string) ([]ToolDefinition, error) {
	transform, err := NormalizeToolSchemaTransform(transform)
	if err != nil {
		return nil, err
	}
	if transform == ToolSchemaTransformOff || len(tools) == 0 {
		return tools, nil
	}

	out := make([]ToolDefinition, len(tools))
	for i, tool := range tools {
		out[i] = tool
		if tool.Type != "function" {
			continue
		}
		out[i].Function = tool.Function
		out[i].Function.Parameters = transformToolSchema(tool.Function.Parameters, transform)
	}

	return out, nil
}

func transformToolSchema(schema map[string]any, transform string) map[string]any {
	switch transform {
	case ToolSchemaTransformSimple:
		return SanitizeSchemaForGoogle(schema)
	default:
		return cloneGeminiSchemaMap(schema)
	}
}
