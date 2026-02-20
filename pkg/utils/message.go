package utils

import (
	"strings"
)

// SplitMessage splits long messages into chunks, preserving code block integrity.
// The function reserves a buffer (10% of maxLen, min 50) to leave room for closing code blocks,
// but may extend to maxLen when needed.
// Call SplitMessage with the full text content and the maximum allowed length of a single message;
// it returns a slice of message chunks that each respect maxLen and avoid splitting fenced code blocks.
func SplitMessage(content string, maxLen int) []string {
	var messages []string

	// Dynamic buffer: 10% of maxLen, but at least 50 chars if possible
	codeBlockBuffer := maxLen / 10
	if codeBlockBuffer < 50 {
		codeBlockBuffer = 50
	}
	if codeBlockBuffer > maxLen/2 {
		codeBlockBuffer = maxLen / 2
	}

	for len(content) > 0 {
		if len(content) <= maxLen {
			messages = append(messages, content)
			break
		}

		// Effective split point: maxLen minus buffer, to leave room for code blocks
		effectiveLimit := maxLen - codeBlockBuffer
		if effectiveLimit < maxLen/2 {
			effectiveLimit = maxLen / 2
		}

		// Find natural split point within the effective limit
		msgEnd := findLastNewline(content[:effectiveLimit], 200)
		if msgEnd <= 0 {
			msgEnd = findLastSpace(content[:effectiveLimit], 100)
		}
		if msgEnd <= 0 {
			msgEnd = effectiveLimit
		}

		// Check if this would end with an incomplete code block
		candidate := content[:msgEnd]
		unclosedIdx := findLastUnclosedCodeBlock(candidate)

		if unclosedIdx >= 0 {
			// Message would end with incomplete code block
			// Try to extend up to maxLen to include the closing ```
			if len(content) > msgEnd {
				closingIdx := findNextClosingCodeBlock(content, msgEnd)
				if closingIdx > 0 && closingIdx <= maxLen {
					// Extend to include the closing ```
					msgEnd = closingIdx
				} else {
					// Code block is too long to fit in one chunk or missing closing fence.
					// Try to split inside by injecting closing and reopening fences.
					headerEnd := strings.Index(content[unclosedIdx:], "\n")
					if headerEnd == -1 {
						headerEnd = unclosedIdx + 3
					} else {
						headerEnd += unclosedIdx
					}
					header := strings.TrimSpace(content[unclosedIdx:headerEnd])

					// If we have a reasonable amount of content after the header, split inside
					if msgEnd > headerEnd+20 {
						// Find a better split point closer to maxLen
						innerLimit := maxLen - 5 // Leave room for "\n```"
						betterEnd := findLastNewline(content[:innerLimit], 200)
						if betterEnd > headerEnd {
							msgEnd = betterEnd
						} else {
							msgEnd = innerLimit
						}
						messages = append(messages, strings.TrimRight(content[:msgEnd], " \t\n\r")+"\n```")
						content = strings.TrimSpace(header + "\n" + content[msgEnd:])
						continue
					}

					// Otherwise, try to split before the code block starts
					newEnd := findLastNewline(content[:unclosedIdx], 200)
					if newEnd <= 0 {
						newEnd = findLastSpace(content[:unclosedIdx], 100)
					}
					if newEnd > 0 {
						msgEnd = newEnd
					} else {
						// If we can't split before, we MUST split inside (last resort)
						if unclosedIdx > 20 {
							msgEnd = unclosedIdx
						} else {
							msgEnd = maxLen - 5
							messages = append(messages, strings.TrimRight(content[:msgEnd], " \t\n\r")+"\n```")
							content = strings.TrimSpace(header + "\n" + content[msgEnd:])
							continue
						}
					}
				}
			}
		}

		if msgEnd <= 0 {
			msgEnd = effectiveLimit
		}

		messages = append(messages, content[:msgEnd])
		content = strings.TrimSpace(content[msgEnd:])
	}

	return messages
}

// findLastUnclosedCodeBlock finds the last opening ``` that doesn't have a closing ```
// Returns the position of the opening ``` or -1 if all code blocks are complete
func findLastUnclosedCodeBlock(text string) int {
	inCodeBlock := false
	lastOpenIdx := -1

	for i := 0; i < len(text); i++ {
		if i+2 < len(text) && text[i] == '`' && text[i+1] == '`' && text[i+2] == '`' {
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

// findNextClosingCodeBlock finds the next closing ``` starting from a position
// Returns the position after the closing ``` or -1 if not found
func findNextClosingCodeBlock(text string, startIdx int) int {
	for i := startIdx; i < len(text); i++ {
		if i+2 < len(text) && text[i] == '`' && text[i+1] == '`' && text[i+2] == '`' {
			return i + 3
		}
	}
	return -1
}

// findLastNewline finds the last newline character within the last N characters
// Returns the position of the newline or -1 if not found
func findLastNewline(s string, searchWindow int) int {
	searchStart := len(s) - searchWindow
	if searchStart < 0 {
		searchStart = 0
	}
	for i := len(s) - 1; i >= searchStart; i-- {
		if s[i] == '\n' {
			return i
		}
	}
	return -1
}

// findLastSpace finds the last space character within the last N characters
// Returns the position of the space or -1 if not found
func findLastSpace(s string, searchWindow int) int {
	searchStart := len(s) - searchWindow
	if searchStart < 0 {
		searchStart = 0
	}
	for i := len(s) - 1; i >= searchStart; i-- {
		if s[i] == ' ' || s[i] == '\t' {
			return i
		}
	}
	return -1
}
