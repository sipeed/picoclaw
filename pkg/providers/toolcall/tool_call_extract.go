package toolcall

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

// parseStructuredToolCalls converts API response tool calls to the internal ToolCall format.
func ParseStructuredToolCalls(apiToolCalls []struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function *struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}) []protocoltypes.ToolCall {
	if len(apiToolCalls) == 0 {
		return nil
	}

	toolCalls := make([]protocoltypes.ToolCall, 0, len(apiToolCalls))
	for _, tc := range apiToolCalls {
		if tc.Function == nil {
			continue
		}

		arguments := ParseToolCallArguments(tc.Function.Arguments)

		toolCalls = append(toolCalls, protocoltypes.ToolCall{
			ID:        tc.ID,
			Type:      tc.Type,
			Name:      tc.Function.Name,
			Arguments: arguments,
			Function: &protocoltypes.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}

	return toolCalls
}

// parseToolCallArguments parses JSON arguments string into a map.
func ParseToolCallArguments(argsStr string) map[string]interface{} {
	if argsStr == "" {
		return make(map[string]interface{})
	}

	var arguments map[string]interface{}
	if err := json.Unmarshal([]byte(argsStr), &arguments); err != nil {
		log.Printf("openai_compat: failed to decode tool call arguments: %v", err)
		return map[string]interface{}{"raw": argsStr}
	}

	return arguments
}

// extractToolCallsFromText parses tool call JSON from response text.
// This handles cases where models embed tool calls in the content field.
func ExtractToolCallsFromText(text string) []protocoltypes.ToolCall {
	start := strings.Index(text, `{"tool_calls"`)
	if start == -1 {
		return nil
	}

	end := FindMatchingBrace(text, start)
	if end == start {
		return nil
	}

	jsonStr := text[start:end]

	var wrapper struct {
		ToolCalls []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		return nil
	}

	var result []protocoltypes.ToolCall
	for _, tc := range wrapper.ToolCalls {
		args := ParseToolCallArguments(tc.Function.Arguments)

		result = append(result, protocoltypes.ToolCall{
			ID:        tc.ID,
			Type:      tc.Type,
			Name:      tc.Function.Name,
			Arguments: args,
			Function: &protocoltypes.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}

	return result
}

// FindMatchingBrace finds the index after the closing brace matching the opening brace at pos.
func FindMatchingBrace(text string, pos int) int {
	depth := 0
	for i := pos; i < len(text); i++ {
		if text[i] == '{' {
			depth++
		} else if text[i] == '}' {
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return pos
}

// stripJSONObject removes a JSON object starting with the specified pattern from text.
func StripJSONObject(text, pattern string) string {
	start := strings.Index(text, pattern)
	if start == -1 {
		return text
	}

	end := FindMatchingBrace(text, start)
	if end == start {
		return text
	}

	before := strings.TrimRight(text[:start], " \t\n\r")
	after := strings.TrimLeft(text[end:], " \t\n\r")
	if before != "" && after != "" {
		return before + " " + after
	}
	return before + after
}

// StripToolCallsFromText removes tool call JSON from response text.
func StripToolCallsFromText(text string) string {
	return StripJSONObject(text, `{"tool_calls"`)
}
