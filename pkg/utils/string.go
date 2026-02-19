package utils

import "strings"

// Truncate returns a truncated version of s with at most maxLen runes.
// Handles multi-byte Unicode characters properly.
// If the string is truncated, "..." is appended to indicate truncation.
func Truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	// Reserve 3 chars for "..."
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

// SplitMessage splits long messages into chunks, preserving code block integrity where possible.
// Logic is inspired by the Discord channel implementation and is channel-agnostic.
// This allocates a slice to hold all chunks; for streaming use SplitMessageIter.
func SplitMessage(content string, limit int) []string {
	var messages []string

	_ = SplitMessageIter(content, limit, func(chunk string) error {
		messages = append(messages, chunk)
		return nil
	})

	return messages
}

// SplitMessageIter splits content into chunks and calls cb for each chunk.
// This avoids allocating a slice to hold all chunks and is more memory-efficient for very large messages.
func SplitMessageIter(content string, limit int, cb func(chunk string) error) error {
	content = strings.TrimSpace(content)
	for len(content) > 0 {
		if len(content) <= limit {
			if content != "" {
				if err := cb(content); err != nil {
					return err
				}
			}
			break
		}

		msgEnd := limit

		// Find natural split point within the limit
		msgEnd = findLastNewline(content[:limit], 200)
		if msgEnd <= 0 {
			msgEnd = findLastSpace(content[:limit], 100)
		}
		if msgEnd <= 0 {
			msgEnd = limit
		}

		// Check if this would end with an incomplete code block
		candidate := content[:msgEnd]
		unclosedIdx := findLastUnclosedCodeBlock(candidate)

		if unclosedIdx >= 0 {
			// Message would end with incomplete code block
			// Try to extend to include the closing ``` (with some buffer)
			extendedLimit := limit + 500 // Allow buffer for code blocks
			if len(content) > extendedLimit {
				closingIdx := findNextClosingCodeBlock(content, msgEnd)
				if closingIdx > 0 && closingIdx <= extendedLimit {
					// Extend to include the closing ```
					msgEnd = closingIdx
				} else {
					// Can't find closing, split before the code block
					msgEnd = findLastNewline(content[:unclosedIdx], 200)
					if msgEnd <= 0 {
						msgEnd = findLastSpace(content[:unclosedIdx], 100)
					}
					if msgEnd <= 0 {
						msgEnd = unclosedIdx
					}
				}
			} else {
				// Remaining content fits within extended limit
				msgEnd = len(content)
			}
		}

		if msgEnd <= 0 {
			msgEnd = limit
		}

		chunk := strings.TrimSpace(content[:msgEnd])
		if chunk != "" {
			if err := cb(chunk); err != nil {
				return err
			}
		}
		content = strings.TrimSpace(content[msgEnd:])
	}

	return nil
}

// findLastUnclosedCodeBlock finds the last opening ``` that doesn't have a closing ```.
// Returns the position of the opening ``` or -1 if all code blocks are complete.
func findLastUnclosedCodeBlock(text string) int {
	count := 0
	lastOpenIdx := -1

	for i := 0; i < len(text); i++ {
		if i+2 < len(text) && text[i] == '`' && text[i+1] == '`' && text[i+2] == '`' {
			if count == 0 {
				lastOpenIdx = i
			}
			count++
			i += 2
		}
	}

	// If odd number of ``` markers, last one is unclosed
	if count%2 == 1 {
		return lastOpenIdx
	}
	return -1
}

// findNextClosingCodeBlock finds the next closing ``` starting from a position.
// Returns the position after the closing ``` or -1 if not found.
func findNextClosingCodeBlock(text string, startIdx int) int {
	for i := startIdx; i < len(text); i++ {
		if i+2 < len(text) && text[i] == '`' && text[i+1] == '`' && text[i+2] == '`' {
			return i + 3
		}
	}
	return -1
}

// findLastNewline finds the last newline character within the last N characters.
// Returns the position of the newline or -1 if not found.
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

// findLastSpace finds the last space character within the last N characters.
// Returns the position of the space or -1 if not found.
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
