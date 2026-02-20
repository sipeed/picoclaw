package providers

import (
	"encoding/json"
	"fmt"
	"strings"
)

// extractToolCallsFromText parses tool call JSON from response text.
// Both ClaudeCliProvider and CodexCliProvider use this to extract
// tool calls that the model outputs in its response text.
func extractToolCallsFromText(text string) []ToolCall {
	start := strings.Index(text, `{"tool_calls"`)
	if start == -1 {
		return nil
	}

	end := findMatchingBrace(text, start)
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

	var result []ToolCall
	for _, tc := range wrapper.ToolCalls {
		var args map[string]interface{}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)

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

	return result
}

// extractXMLToolCalls parses XML tool call blocks (e.g. <ns:toolcall>)
// into structured ToolCall objects. Used as a fallback when the provider returns
// tool calls as XML in Content but not in the structured tool_calls field.
//
// Expected format:
//
//	<ns:toolcall>
//	<invoke name="tool_name">
//	<parameter name="param">value</parameter>
//	</invoke>
//	</ns:toolcall>
// levenshtein computes the edit distance between two strings.
// O(n*m) where n,m are string lengths — negligible for short tag names.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr := make([]int, lb+1)
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev = curr
	}
	return prev[lb]
}

// isToolCallTag returns true if name is close enough to "toolcall" by edit
// distance (threshold ≤ 2). Case-insensitive. Catches variants like
// "tool_call", "Tool-Call", "toolCall", "ToolCall", etc.
func isToolCallTag(name string) bool {
	const threshold = 2
	return levenshtein(strings.ToLower(name), "toolcall") <= threshold
}

// findToolCallOpenTag finds the next opening toolcall tag using fuzzy
// matching (edit distance). Scans for <ns:name> patterns where name is
// close to "toolcall". Returns the index of '<', the namespace, and the
// full tag length, or idx=-1 if not found.
func findToolCallOpenTag(text string) (idx int, ns string, tagLen int) {
	search := text
	offset := 0
	for {
		lt := strings.Index(search, "<")
		if lt == -1 {
			return -1, "", 0
		}
		// Skip closing tags and comments
		if lt+1 < len(search) && (search[lt+1] == '/' || search[lt+1] == '!') {
			offset += lt + 2
			search = search[lt+2:]
			continue
		}
		gt := strings.Index(search[lt:], ">")
		if gt == -1 {
			return -1, "", 0
		}
		tagContent := search[lt+1 : lt+gt] // e.g. "minimax:tool_call"
		colon := strings.Index(tagContent, ":")
		if colon != -1 {
			nsCandidate := tagContent[:colon]
			nameCandidate := tagContent[colon+1:]
			if isToolCallTag(nameCandidate) {
				fullTag := "<" + tagContent + ">"
				return offset + lt, nsCandidate, len(fullTag)
			}
		}
		offset += lt + gt + 1
		search = search[lt+gt+1:]
	}
}

// findToolCallCloseTag finds the close tag for a toolcall block using
// fuzzy matching (edit distance). Returns the index and length of the close tag, or -1.
func findToolCallCloseTag(text, ns string) (idx int, tagLen int) {
	// Scan for all </ns: patterns and check if the tag name normalizes to "toolcall"
	prefix := "</" + ns + ":"
	search := text
	offset := 0
	for {
		i := strings.Index(search, prefix)
		if i == -1 {
			return -1, 0
		}
		afterPrefix := search[i+len(prefix):]
		end := strings.Index(afterPrefix, ">")
		if end == -1 {
			return -1, 0
		}
		tagName := afterPrefix[:end]
		if isToolCallTag(tagName) {
			fullTag := prefix + tagName + ">"
			return offset + i, len(fullTag)
		}
		offset += i + len(prefix)
		search = search[i+len(prefix):]
	}
}

func extractXMLToolCalls(text string) []ToolCall {
	var result []ToolCall
	remaining := text
	callIdx := 0

	for {
		// Find next opening toolcall tag using normalized matching
		openIdx, ns, openLen := findToolCallOpenTag(remaining)
		if openIdx == -1 {
			break
		}
		afterOpen := remaining[openIdx+openLen:]
		closeIdx, closeLen := findToolCallCloseTag(afterOpen, ns)
		if closeIdx == -1 {
			break
		}

		block := afterOpen[:closeIdx]
		remaining = afterOpen[closeIdx+closeLen:]

		// Parse <invoke> elements within the block
		invokeRemaining := block
		for {
			invokeStart := strings.Index(invokeRemaining, "<invoke")
			if invokeStart == -1 {
				break
			}
			invokeEnd := strings.Index(invokeRemaining[invokeStart:], "</invoke>")
			if invokeEnd == -1 {
				break
			}
			invokeBody := invokeRemaining[invokeStart : invokeStart+invokeEnd+len("</invoke>")]
			invokeRemaining = invokeRemaining[invokeStart+invokeEnd+len("</invoke>"):]

			// Extract tool name from <invoke name="...">
			nameStart := strings.Index(invokeBody, `name="`)
			if nameStart == -1 {
				continue
			}
			nameStart += len(`name="`)
			nameEnd := strings.Index(invokeBody[nameStart:], `"`)
			if nameEnd == -1 {
				continue
			}
			toolName := invokeBody[nameStart : nameStart+nameEnd]

			// Extract parameters
			args := make(map[string]interface{})
			paramRemaining := invokeBody
			for {
				pStart := strings.Index(paramRemaining, "<parameter")
				if pStart == -1 {
					break
				}
				pNameStart := strings.Index(paramRemaining[pStart:], `name="`)
				if pNameStart == -1 {
					break
				}
				pNameStart += pStart + len(`name="`)
				pNameEnd := strings.Index(paramRemaining[pNameStart:], `"`)
				if pNameEnd == -1 {
					break
				}
				paramName := paramRemaining[pNameStart : pNameStart+pNameEnd]

				// Find closing > of the <parameter ...> tag
				tagClose := strings.Index(paramRemaining[pNameStart:], ">")
				if tagClose == -1 {
					break
				}
				valueStart := pNameStart + tagClose + 1
				valueEnd := strings.Index(paramRemaining[valueStart:], "</parameter>")
				if valueEnd == -1 {
					break
				}
				paramValue := paramRemaining[valueStart : valueStart+valueEnd]
				args[paramName] = paramValue
				paramRemaining = paramRemaining[valueStart+valueEnd+len("</parameter>"):]
			}

			// Build Arguments JSON string
			argsJSON, _ := json.Marshal(args)

			callIdx++
			result = append(result, ToolCall{
				ID:        fmt.Sprintf("xmltc_%d", callIdx),
				Type:      "function",
				Name:      toolName,
				Arguments: args,
				Function: &FunctionCall{
					Name:      toolName,
					Arguments: string(argsJSON),
				},
			})
		}
	}

	return result
}

// stripXMLToolCalls removes XML tool call blocks (e.g. <ns:toolcall>...</ns:toolcall>)
// from response text. Some providers embed raw XML tool calls in Content alongside
// structured tool_calls; this prevents them from leaking to users.
// Uses normalized tag matching so that <ns:toolcall>, <ns:tool_call>, <ns:Tool-Call>
// etc. are all recognized and stripped.
func stripXMLToolCalls(text string) string {
	openIdx, ns, openLen := findToolCallOpenTag(text)
	if openIdx == -1 {
		return text
	}
	afterOpen := text[openIdx+openLen:]
	closeIdx, closeLen := findToolCallCloseTag(afterOpen, ns)
	if closeIdx == -1 {
		return text
	}
	cleaned := text[:openIdx] + afterOpen[closeIdx+closeLen:]
	// Recursively strip if there are more blocks
	if openIdx2, _, _ := findToolCallOpenTag(cleaned); openIdx2 != -1 {
		cleaned = stripXMLToolCalls(cleaned)
	}
	return strings.TrimSpace(cleaned)
}

// stripToolCallsFromText removes tool call JSON from response text.
func stripToolCallsFromText(text string) string {
	start := strings.Index(text, `{"tool_calls"`)
	if start == -1 {
		return text
	}

	end := findMatchingBrace(text, start)
	if end == start {
		return text
	}

	return strings.TrimSpace(text[:start] + text[end:])
}
