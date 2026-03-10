package telegram

import (
	"strings"
)

// markdownToTelegramMarkdownV2 takes a standardized markdown string and
// strictly escapes or transforms it to fit Telegram's MarkdownV2 requirements.
// https://core.telegram.org/bots/api#formatting-options
func markdownToTelegramMarkdownV2(text string) string {
	// replace Heading to bolding
	text = reHeading.ReplaceAllString(text, "*$1*")
	text = reBoldStar.ReplaceAllString(text, "*$1*")

	var result strings.Builder
	runes := []rune(text)
	length := len(runes)

	// List of characters that must be escaped in standard text contexts
	needsNormalEscape := func(r rune) bool {
		switch r {
		case '_', '*', '[', ']', '(', ')', '~', '`', '>', '#', '+', '-', '=', '|', '{', '}', '.', '!':
			return true
		}
		return false
	}

	i := 0
	for i < length {
		// 1. Check for Pre-formatted Code Block (```...```)
		if i+2 < length && runes[i] == '`' && runes[i+1] == '`' && runes[i+2] == '`' {
			result.WriteString("```")
			i += 3
			// Find closing ```
			for i < length {
				if i+2 < length && runes[i] == '`' && runes[i+1] == '`' && runes[i+2] == '`' {
					result.WriteString("```")
					i += 3
					break
				}
				// Inside code blocks, escape `\` and `\`
				if runes[i] == '\\' || runes[i] == '`' {
					result.WriteRune('\\')
				}
				result.WriteRune(runes[i])
				i++
			}
			continue
		}

		// 2. Check for Inline Code (`...`)
		if runes[i] == '`' {
			result.WriteRune('`')
			i++
			for i < length {
				if runes[i] == '`' {
					result.WriteRune('`')
					i++
					break
				}
				if runes[i] == '\\' || runes[i] == '`' {
					result.WriteRune('\\')
				}
				result.WriteRune(runes[i])
				i++
			}
			continue
		}

		// 3. Link or Custom Emoji definition: URL part (...)
		// We detect this by checking if the previous non-space character closed a bracket ']',
		// and we are currently on '('. To keep logic linear, we handle it as we traverse.
		// NOTE: A true deep-parser would link `[` to `](...)`. For safety, whenever we see `(`,
		// if it looks like a URL part, we escape it via URL rules. Let's do a basic lookbehind.
		if i != 0 && runes[i] == '(' && i > 0 && runes[i-1] == ']' {
			result.WriteRune('(')
			i++
			for i < length {
				if runes[i] == ')' {
					// Unescaped closing bracket ends the URL
					result.WriteRune(')')
					i++
					break
				}
				// In URL part, escape `\` and `)`
				if runes[i] == '\\' || runes[i] == ')' {
					result.WriteRune('\\')
				}
				result.WriteRune(runes[i])
				i++
			}
			continue
		}

		// 4. Handle blockquotes starts
		if i != 0 && runes[i] == '>' && (i == 0 || runes[i-1] == '\n') {
			result.WriteRune('>')
			i++
			continue
		}

		// 5. Handle expandable block quotation starts
		if i+3 < length && runes[i] == '>' && runes[i-1] == '*' && runes[i-2] == '*' && (i == 0 || runes[i-3] == '\n') {
			result.WriteRune(runes[i])
			i++
			continue
		}

		// 6. Handle standard Markdown Entities Boundaries
		// If they are part of valid markdown boundaries, we write them as-is.
		// We trust the syntax rules: * _ ~ || [ ]
		// (Assuming the text is a valid markdown, we don't escape these if formatting is intended)

		// Note on Ambiguity (__ vs _):
		// Telegram parses `__` from left to right greedily.
		if i+1 < length && runes[i] == '_' && runes[i+1] == '_' {
			result.WriteString("__")
			i += 2
			continue
		}

		if i+1 < length && runes[i] == '|' && runes[i+1] == '|' {
			result.WriteString("||")
			i += 2
			continue
		}

		// Standard single-char boundaries
		if runes[i] == '*' || runes[i] == '_' || runes[i] == '[' || runes[i] == ']' {
			result.WriteRune(runes[i])
			i++
			continue
		}

		// Custom emoji boundary check `![`
		if i+1 < length && runes[i] == '!' && runes[i+1] == '[' {
			result.WriteString("![")
			i += 2
			continue
		}

		// 7. Handle plain text characters
		// Escape remaining special characters if they aren't forming intended valid markup
		if needsNormalEscape(runes[i]) {
			// Check if it's already escaped; if an escape character exists, consume it legitimately
			if runes[i] == '\\' && i+1 < length && needsNormalEscape(runes[i+1]) {
				// Keep the backslash and the escaped char as is, avoiding double escaping
				result.WriteRune('\\')
				result.WriteRune(runes[i+1])
				i += 2
				continue
			}

			// Auto-escape the character
			result.WriteRune('\\')
		}

		result.WriteRune(runes[i])
		i++
	}

	return result.String()
}
