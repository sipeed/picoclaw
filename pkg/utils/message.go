package utils

import (
	"strings"
)

// SplitMessage splits long messages into chunks, preserving code block integrity.
// The maxLen parameter is measured in runes (Unicode characters), not bytes.
// The function reserves a buffer (10% of maxLen, min 50) to leave room for closing code blocks,
// but may extend to maxLen when needed.
// Call SplitMessage with the full text content and the maximum allowed length of a single message;
// it returns a slice of message chunks that each respect maxLen and avoid splitting fenced code blocks.
func SplitMessage(content string, maxLen int) []string {
	if maxLen <= 0 {
		if content == "" {
			return nil
		}
		return []string{content}
	}

	runes := []rune(content)
	var messages []string

	// Dynamic buffer: 10% of maxLen, but at least 50 chars if possible
	codeBlockBuffer := maxLen / 10
	if codeBlockBuffer < 50 {
		codeBlockBuffer = 50
	}
	if codeBlockBuffer > maxLen/2 {
		codeBlockBuffer = maxLen / 2
	}

	for len(runes) > 0 {
		if len(runes) <= maxLen {
			messages = append(messages, string(runes))
			break
		}

		// Effective split point: maxLen minus buffer, to leave room for code blocks
		effectiveLimit := maxLen - codeBlockBuffer
		if effectiveLimit < maxLen/2 {
			effectiveLimit = maxLen / 2
		}

		// Find natural split point within the effective limit
		msgEnd := findLastNewlineRunes(runes[:effectiveLimit], 200)
		if msgEnd <= 0 {
			msgEnd = findLastSpaceRunes(runes[:effectiveLimit], 100)
		}
		if msgEnd <= 0 {
			msgEnd = effectiveLimit
		}

		// Check if this would end with an incomplete code block
		candidate := runes[:msgEnd]
		unclosedIdx := findLastUnclosedCodeBlockRunes(candidate)

		if unclosedIdx >= 0 {
			// Message would end with incomplete code block
			// Try to extend up to maxLen to include the closing ```
			if len(runes) > msgEnd {
				closingIdx := findNextClosingCodeBlockRunes(runes, msgEnd)
				if closingIdx > 0 && closingIdx <= maxLen {
					// Extend to include the closing ```
					msgEnd = closingIdx
				} else {
					// Code block is too long to fit in one chunk or missing closing fence.
					// Try to split inside by injecting closing and reopening fences.
					candidateStr := string(candidate)
					unclosedStr := string(runes[unclosedIdx:])
					headerEnd := strings.Index(unclosedStr, "\n")
					var header string
					if headerEnd == -1 {
						header = strings.TrimSpace(string(runes[unclosedIdx : unclosedIdx+3]))
					} else {
						header = strings.TrimSpace(string(runes[unclosedIdx : unclosedIdx+headerEnd]))
					}
					headerEndIdx := unclosedIdx + len([]rune(header))
					if headerEnd != -1 {
						headerEndIdx = unclosedIdx + headerEnd
					}

					_ = candidateStr // used above for context

					// If we have a reasonable amount of content after the header, split inside
					if msgEnd > headerEndIdx+20 {
						// Find a better split point closer to maxLen
						innerLimit := maxLen - 5 // Leave room for "\n```"
						betterEnd := findLastNewlineRunes(runes[:innerLimit], 200)
						if betterEnd > headerEndIdx {
							msgEnd = betterEnd
						} else {
							msgEnd = innerLimit
						}
						chunk := strings.TrimRight(string(runes[:msgEnd]), " \t\n\r") + "\n```"
						messages = append(messages, chunk)
						remaining := strings.TrimSpace(header + "\n" + string(runes[msgEnd:]))
						runes = []rune(remaining)
						continue
					}

					// Otherwise, try to split before the code block starts
					newEnd := findLastNewlineRunes(runes[:unclosedIdx], 200)
					if newEnd <= 0 {
						newEnd = findLastSpaceRunes(runes[:unclosedIdx], 100)
					}
					if newEnd > 0 {
						msgEnd = newEnd
					} else {
						// If we can't split before, we MUST split inside (last resort)
						if unclosedIdx > 20 {
							msgEnd = unclosedIdx
						} else {
							msgEnd = maxLen - 5
							chunk := strings.TrimRight(string(runes[:msgEnd]), " \t\n\r") + "\n```"
							messages = append(messages, chunk)
							remaining := strings.TrimSpace(header + "\n" + string(runes[msgEnd:]))
							runes = []rune(remaining)
							continue
						}
					}
				}
			}
		}

		if msgEnd <= 0 {
			msgEnd = effectiveLimit
		}

		messages = append(messages, string(runes[:msgEnd]))
		remaining := strings.TrimSpace(string(runes[msgEnd:]))
		runes = []rune(remaining)
	}

	return messages
}

// findLastUnclosedCodeBlockRunes finds the last opening ``` that doesn't have a closing ```
// Returns the rune position of the opening ``` or -1 if all code blocks are complete
func findLastUnclosedCodeBlockRunes(runes []rune) int {
	inCodeBlock := false
	lastOpenIdx := -1

	for i := 0; i < len(runes); i++ {
		if i+2 < len(runes) && runes[i] == '`' && runes[i+1] == '`' && runes[i+2] == '`' {
			// Toggle code block state on each fence
			if !inCodeBlock {
				// Entering a code block: record this opening fence
				lastOpenIdx = i
			}
			inCodeBlock = !inCodeBlock
			i += 2
		}
	}

	if inCodeBlock {
		return lastOpenIdx
	}
	return -1
}

// findNextClosingCodeBlockRunes finds the next closing ``` starting from a rune position
// Returns the rune position after the closing ``` or -1 if not found
func findNextClosingCodeBlockRunes(runes []rune, startIdx int) int {
	for i := startIdx; i < len(runes); i++ {
		if i+2 < len(runes) && runes[i] == '`' && runes[i+1] == '`' && runes[i+2] == '`' {
			return i + 3
		}
	}
	return -1
}

// findLastNewlineRunes finds the last newline character within the last N runes
// Returns the rune position of the newline or -1 if not found
func findLastNewlineRunes(runes []rune, searchWindow int) int {
	searchStart := len(runes) - searchWindow
	if searchStart < 0 {
		searchStart = 0
	}
	for i := len(runes) - 1; i >= searchStart; i-- {
		if runes[i] == '\n' {
			return i
		}
	}
	return -1
}

// findLastSpaceRunes finds the last space character within the last N runes
// Returns the rune position of the space or -1 if not found
func findLastSpaceRunes(runes []rune, searchWindow int) int {
	searchStart := len(runes) - searchWindow
	if searchStart < 0 {
		searchStart = 0
	}
	for i := len(runes) - 1; i >= searchStart; i-- {
		if runes[i] == ' ' || runes[i] == '\t' {
			return i
		}
	}
	return -1
}
