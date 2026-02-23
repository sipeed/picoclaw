package utils

import (
	"strings"
)

// SplitMessage splits long messages into chunks, preserving code block integrity.
// The function reserves a buffer (10% of maxLen, min 50) to leave room for closing code blocks,
// but may extend to maxLen when needed. It respects rune counts to ensure multi-byte
// characters (like emojis or CJK) are not split in half.
func SplitMessage(content string, maxLen int) []string {
	if content == "" {
		return nil
	}
	if maxLen <= 0 {
		return []string{content}
	}

	var messages []string

	// Dynamic buffer: 10% of maxLen, but at least 50 chars if possible
	codeBlockBuffer := maxLen / 10
	if codeBlockBuffer < 50 {
		codeBlockBuffer = 50
	}
	if codeBlockBuffer > maxLen/2 {
		codeBlockBuffer = maxLen / 2
	}

	runes := []rune(content)
	startIndex := 0

	for startIndex < len(runes) {
		remainingRunes := runes[startIndex:]
		if len(remainingRunes) <= maxLen {
			messages = append(messages, string(remainingRunes))
			break
		}

		// Effective split point: maxLen minus buffer, to leave room for code blocks
		effectiveLimit := maxLen - codeBlockBuffer
		if effectiveLimit < maxLen/2 {
			effectiveLimit = maxLen / 2
		}

		// Find natural split point within the effective limit from the current startIndex
		// We pass the full slice so findLastSentenceBoundaryRunes can look ahead past the window
		msgEndOffset := findLastSentenceBoundaryRunes(runes, startIndex+effectiveLimit, 300)
		if msgEndOffset <= startIndex {
			msgEndOffset = findLastNewlineRunes(remainingRunes[:effectiveLimit], 200)
			if msgEndOffset >= 0 {
				msgEndOffset += startIndex
			}
		}
		if msgEndOffset <= startIndex {
			msgEndOffset = findLastSpaceRunes(remainingRunes[:effectiveLimit], 100)
			if msgEndOffset >= 0 {
				msgEndOffset += startIndex
			}
		}
		if msgEndOffset <= startIndex {
			msgEndOffset = startIndex + effectiveLimit
		}

		// Check if this would end with an incomplete code block
		candidateRunes := runes[startIndex:msgEndOffset]
		unclosedIdx := findLastUnclosedCodeBlockRunes(candidateRunes)

		if unclosedIdx >= 0 {
			// Absolute index of the unclosed fence
			absUnclosedIdx := startIndex + unclosedIdx

			// Try to extend up to maxLen to include the closing ```
			closingIdx := findNextClosingCodeBlockRunes(runes, msgEndOffset)
			if closingIdx > 0 && closingIdx <= startIndex+maxLen {
				msgEndOffset = closingIdx
			} else {
				// Find first newline after opening fence to extract header
				headerEnd := -1
				for i := absUnclosedIdx; i < len(runes); i++ {
					if runes[i] == '\n' {
						headerEnd = i
						break
					}
				}
				if headerEnd == -1 {
					headerEnd = absUnclosedIdx + 3
				} else {
					headerEnd++ // include newline
				}
				header := strings.TrimSpace(string(runes[absUnclosedIdx:headerEnd]))

				if msgEndOffset > headerEnd+20 {
					innerLimit := maxLen - 5
					betterEnd := findLastSentenceBoundaryRunes(runes, startIndex+innerLimit, 300)
					if betterEnd <= headerEnd {
						betterEnd = findLastNewlineRunes(runes[startIndex:startIndex+innerLimit], 200)
						if betterEnd >= 0 {
							betterEnd += startIndex
						}
					}

					if betterEnd > headerEnd {
						msgEndOffset = betterEnd
					} else {
						msgEndOffset = startIndex + innerLimit
					}

					chunkStr := string(runes[startIndex:msgEndOffset])
					messages = append(messages, strings.TrimRight(chunkStr, " \t\n\r")+"\n```")

					// Move startIndex to msgEndOffset but "inject" the header for the next iteration.
					// We prepend the header and a newline.
					injectedHeader := header + "\n"
					nextRunes := append([]rune(injectedHeader), runes[msgEndOffset:]...)
					runes = append(runes[:0], nextRunes...) // Reuse capacity
					startIndex = 0
					continue
				}

				// Try to split before the code block
				newEnd := findLastSentenceBoundaryRunes(runes, absUnclosedIdx, 300)
				if newEnd <= startIndex {
					newEnd = findLastNewlineRunes(runes[startIndex:absUnclosedIdx], 200)
					if newEnd >= 0 {
						newEnd += startIndex
					}
				}
				if newEnd <= startIndex {
					newEnd = findLastSpaceRunes(runes[startIndex:absUnclosedIdx], 100)
					if newEnd >= 0 {
						newEnd += startIndex
					}
				}

				if newEnd > startIndex {
					msgEndOffset = newEnd
				} else {
					// Hard split inside (last resort)
					msgEndOffset = startIndex + maxLen - 5
					chunkStr := string(runes[startIndex:msgEndOffset])
					messages = append(messages, strings.TrimRight(chunkStr, " \t\n\r")+"\n```")

					injectedHeader := header + "\n"
					nextRunes := append([]rune(injectedHeader), runes[msgEndOffset:]...)
					runes = append(runes[:0], nextRunes...)
					startIndex = 0
					continue
				}
			}
		}

		if msgEndOffset <= startIndex {
			msgEndOffset = startIndex + effectiveLimit
		}

		messages = append(messages, string(runes[startIndex:msgEndOffset]))

		// Advance startIndex and skip leading whitespace for next chunk
		startIndex = msgEndOffset
		for startIndex < len(runes) && (runes[startIndex] == ' ' || runes[startIndex] == '\n' || runes[startIndex] == '\t' || runes[startIndex] == '\r') {
			startIndex++
		}
	}

	return messages
}

// findLastUnclosedCodeBlockRunes finds the last opening ``` that doesn't have a closing ```
// Returns the position of the opening ``` or -1 if all code blocks are complete
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

// findNextClosingCodeBlockRunes finds the next closing ``` starting from a position
// Returns the position after the closing ``` or -1 if not found
func findNextClosingCodeBlockRunes(runes []rune, startIdx int) int {
	for i := startIdx; i < len(runes); i++ {
		if i+2 < len(runes) && runes[i] == '`' && runes[i+1] == '`' && runes[i+2] == '`' {
			return i + 3
		}
	}
	return -1
}

// findLastNewlineRunes finds the last newline character within the last N characters
// Returns the position of the newline or -1 if not found
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

// findLastSpaceRunes finds the last space character within the last N characters
// Returns the position of the space or -1 if not found
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

// findLastSentenceBoundaryRunes finds the last sentence-ending punctuation within a limit.
// It looks ahead to verify the boundary is real (followed by space, newline, or end of string).
func findLastSentenceBoundaryRunes(runes []rune, limit int, searchWindow int) int {
	if limit > len(runes) {
		limit = len(runes)
	}
	searchStart := limit - searchWindow
	if searchStart < 0 {
		searchStart = 0
	}
	for i := limit - 1; i >= searchStart; i-- {
		switch runes[i] {
		case '.', '!', '?', '。', '！', '？':
			// Ensure it's a true boundary: 
			// either it's the very end of the full message, 
			// or the NEXT rune (lookahead) is a space, newline, or tab.
			if i == len(runes)-1 || (i+1 < len(runes) && (runes[i+1] == ' ' || runes[i+1] == '\n' || runes[i+1] == '\t')) {
				return i + 1
			}
		}
	}
	return -1
}

