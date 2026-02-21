package providers

import (
	"encoding/json"
	"strings"
)

// extractToolCallsFromText parses multiple tool call JSON blocks from response text.
// Both ClaudeCliProvider and CodexCliProvider use this to extract
// tool calls that the model outputs in its response text.
func extractToolCallsFromText(text string) []ToolCall {
	var result []ToolCall
	pos := 0

	for {
		_, _, jsonStart, jsonEnd, found := nextToolCallBlock(text, pos)
		if !found {
			break
		}

		jsonStr := text[jsonStart:jsonEnd]
		pos = jsonEnd

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
			continue
		}

		for _, tc := range wrapper.ToolCalls {
			var args map[string]any
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)

			result = append(result, ToolCall{
				ID:        tc.ID,
				Type:      tc.Type,
				Name:      tc.Function.Name,
				Arguments: args,
				Function: &FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
	}

	return result
}

// stripToolCallsFromText removes all tool call JSON blocks (and their markdown wrappers) from response text.
func stripToolCallsFromText(text string) string {
	res := text
	pos := 0
	for {
		blockStart, blockEnd, _, _, found := nextToolCallBlock(res, pos)
		if !found {
			break
		}

		// Remove the block and ensure exactly one double newline if it was in the middle of text
		prefix := strings.TrimRight(res[:blockStart], " \t\n\r")
		suffix := strings.TrimLeft(res[blockEnd:], " \t\n\r")

		if prefix == "" {
			res = suffix
		} else if suffix == "" {
			res = prefix
		} else {
			res = prefix + "\n\n" + suffix
		}
		pos = len(prefix)
	}
	return strings.TrimSpace(res)
}

// nextToolCallBlock finds the next tool_calls JSON block (and its markdown wrapper) in text starting from startFrom.
func nextToolCallBlock(text string, startFrom int) (blockStart, blockEnd, jsonStart, jsonEnd int, found bool) {
	idx := startFrom
	for {
		if idx >= len(text) {
			return 0, 0, 0, 0, false
		}

		// Find the start of a potential JSON object starting with "tool_calls"
		openingBrace := strings.Index(text[idx:], "{")
		if openingBrace == -1 {
			return 0, 0, 0, 0, false
		}
		jsonStart = idx + openingBrace

		// Check if it contains "tool_calls" after the brace
		afterBrace := text[jsonStart+1:]
		trimmed := strings.TrimLeft(afterBrace, " \t\n\r")
		if strings.HasPrefix(trimmed, `"tool_calls"`) {
			jsonEnd = findMatchingBrace(text, jsonStart)
			if jsonEnd != jsonStart {
				// Found a valid block
				break
			}
		}

		// Not a tool call block or no matching brace, continue search after this brace
		idx = jsonStart + 1
	}

	blockStart = jsonStart
	blockEnd = jsonEnd

	// Check for markdown code block wrapper
	// Look back for ```json or ``` ignoring intermediate whitespace/newlines
	prefix := text[:jsonStart]
	trimmedPrefix := strings.TrimRight(prefix, " \t\n\r")
	if strings.HasSuffix(trimmedPrefix, "```json") {
		blockStart = strings.LastIndex(trimmedPrefix, "```json")
	} else if strings.HasSuffix(trimmedPrefix, "```") {
		blockStart = strings.LastIndex(trimmedPrefix, "```")
	}

	// Look ahead for ``` ignoring intermediate whitespace/newlines
	suffix := text[jsonEnd:]
	trimmedSuffix := strings.TrimLeft(suffix, " \t\n\r")
	if strings.HasPrefix(trimmedSuffix, "```") {
		// blockEnd should include the opening whitespace of suffix + the 3 ticks
		wsLen := len(suffix) - len(trimmedSuffix)
		blockEnd = jsonEnd + wsLen + 3
	}

	return blockStart, blockEnd, jsonStart, jsonEnd, true
}

// findMatchingBrace finds the index after the closing brace matching the opening brace at pos.
// It accounts for braces inside strings and escaped characters.
func findMatchingBrace(text string, pos int) int {
	if pos < 0 || pos >= len(text) || text[pos] != '{' {
		return pos
	}

	depth := 0
	inString := false
	escaped := false

	for i := pos; i < len(text); i++ {
		char := text[i]

		if inString {
			if escaped {
				escaped = false
			} else if char == '\\' {
				escaped = true
			} else if char == '"' {
				inString = false
			}
			continue
		}

		if char == '"' {
			inString = true
			continue
		}

		if char == '{' {
			depth++
		} else if char == '}' {
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return pos
}