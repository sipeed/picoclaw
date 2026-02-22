package providers

import (
	"encoding/json"
	"fmt"
	"regexp"
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
		var args map[string]any
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

// --- Shared helpers for XML tool call extraction ---

// normalizeAlpha keeps only lowercase ASCII letters.
// "tool_call" → "toolcall", "Tool-Call" → "toolcall", "ReadFile" → "readfile".
func normalizeAlpha(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			b.WriteRune(r + 32)
		} else if r >= 'a' && r <= 'z' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

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

// Known tool call tag patterns (already alpha-normalized).
// Providers may use different names: tool_call, function_call, tool_use, etc.
var toolCallPatterns = []string{"toolcall", "functioncall", "tooluse"}

// isToolCallTag returns true if the tag name is close to any known tool call
// pattern after alpha normalization + edit distance (threshold ≤ 2).
func isToolCallTag(name string) bool {
	const threshold = 2
	norm := normalizeAlpha(name)
	for _, pat := range toolCallPatterns {
		if levenshtein(norm, pat) <= threshold {
			return true
		}
	}
	return false
}

// tagSuffix returns the part after the last ':' (namespace separator),
// or the whole string if there is no ':'.
func tagSuffix(tag string) string {
	if i := strings.LastIndex(tag, ":"); i >= 0 {
		return tag[i+1:]
	}
	return tag
}

// --- XML block detection via regex ---
//
// Strategy: find <TAG>…</TAG> pairs using regex, then check if the tag
// suffix normalizes to something close to "toolcall" (edit distance ≤ 2).
// Uses greedy (longest) match for the closing tag to capture the full block.

var (
	reOpenTag       = regexp.MustCompile(`<([a-zA-Z][\w:.-]*)>`)
	reCloseTag      = regexp.MustCompile(`</([a-zA-Z][\w:.-]*)>`)
	reBracketMarker = regexp.MustCompile(`\[TOOLCALL\]`)
)

// findToolCallBlock finds the first XML block whose tag suffix matches
// "toolcall" by edit distance. Returns the block boundaries and the inner
// content, or found=false.
func findToolCallBlock(text string) (blockStart, blockEnd int, content string, found bool) {
	for _, om := range reOpenTag.FindAllStringSubmatchIndex(text, -1) {
		tagName := text[om[2]:om[3]]
		if !isToolCallTag(tagSuffix(tagName)) {
			continue
		}
		// Found a toolcall opening tag. Search for the last matching close tag (greedy).
		afterOpen := text[om[1]:]
		closes := reCloseTag.FindAllStringSubmatchIndex(afterOpen, -1)
		for i := len(closes) - 1; i >= 0; i-- {
			closeTagName := afterOpen[closes[i][2]:closes[i][3]]
			if isToolCallTag(tagSuffix(closeTagName)) {
				return om[0], om[1] + closes[i][1], afterOpen[:closes[i][0]], true
			}
		}
	}

	// Fallback: look for orphaned closing tags (missing opening tag).
	// Some LLMs emit the closing </ns:tool_call> without a matching opener.
	// Reconstruct the block start from the first <invoke preceding the closer.
	for _, cm := range reCloseTag.FindAllStringSubmatchIndex(text, -1) {
		closeTagName := text[cm[2]:cm[3]]
		if !isToolCallTag(tagSuffix(closeTagName)) {
			continue
		}
		// Found an orphaned toolcall closing tag. Scan backwards for <invoke.
		before := text[:cm[0]]
		invokePos := strings.LastIndex(before, "<invoke")
		if invokePos == -1 {
			continue
		}
		// Also consume a preceding [TOOLCALL] marker if present.
		start := invokePos
		if loc := reBracketMarker.FindStringIndex(before[:start]); loc != nil && strings.TrimSpace(before[loc[1]:start]) == "" {
			start = loc[0]
		}
		return start, cm[1], text[invokePos:cm[0]], true
	}

	return 0, 0, "", false
}

// --- XML tool call extraction ---
//
// Expected format:
//
//	<ns:toolcall>
//	<invoke name="tool_name">
//	<parameter name="param">value</parameter>
//	</invoke>
//	</ns:toolcall>
// ExtractXMLToolCalls is the exported version for use by the agent loop.
func ExtractXMLToolCalls(text string) []ToolCall {
	return extractXMLToolCalls(text)
}

func extractXMLToolCalls(text string) []ToolCall {
	var result []ToolCall
	remaining := text
	callIdx := 0

	for {
		_, blockEnd, block, found := findToolCallBlock(remaining)
		if !found {
			break
		}
		remaining = remaining[blockEnd:]
		result = append(result, parseInvokeElements(block, &callIdx)...)
	}

	return result
}

// parseInvokeElements extracts ToolCall entries from <invoke>...</invoke> blocks.
func parseInvokeElements(text string, callIdx *int) []ToolCall {
	var result []ToolCall
	invokeRemaining := text
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

		argsJSON, _ := json.Marshal(args)
		*callIdx++
		result = append(result, ToolCall{
			ID:        fmt.Sprintf("xmltc_%d", *callIdx),
			Type:      "function",
			Name:      toolName,
			Arguments: args,
			Function: &FunctionCall{
				Name:      toolName,
				Arguments: string(argsJSON),
			},
		})
	}
	return result
}

// StripXMLToolCalls is the exported version for use by the agent loop.
func StripXMLToolCalls(text string) string {
	return stripXMLToolCalls(text)
}

// stripXMLToolCalls removes XML tool call blocks from response text.
// Prevents raw XML tool calls from leaking to users.
func stripXMLToolCalls(text string) string {
	blockStart, blockEnd, _, found := findToolCallBlock(text)
	if found {
		cleaned := text[:blockStart] + text[blockEnd:]
		if _, _, _, more := findToolCallBlock(cleaned); more {
			cleaned = stripXMLToolCalls(cleaned)
		}
		return strings.TrimSpace(cleaned)
	}

	return strings.TrimSpace(text)
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
