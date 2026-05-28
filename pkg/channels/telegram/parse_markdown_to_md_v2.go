package telegram

import (
	"regexp"
	"strings"
)

// mdV2SpecialChars are all characters that must be escaped in Telegram MarkdownV2
var mdV2SpecialChars = map[rune]bool{
	'*':  true,
	'_':  true,
	'[':  true,
	']':  true,
	'(':  true,
	')':  true,
	'~':  true,
	'`':  true,
	'>':  true,
	'<':  true,
	'#':  true,
	'+':  true,
	'-':  true,
	'=':  true,
	'|':  true,
	'{':  true,
	'}':  true,
	'.':  true,
	'!':  true,
	'\\': true,
}

// entityPattern describes one Telegram MarkdownV2 inline entity type.
type entityPattern struct {
	re    *regexp.Regexp
	open  string
	close string
}

// allEntityPatterns lists every recognized entity in priority order
// (longer / more-specific delimiters first so they win over shorter ones).
// Each entry's regex is anchored to find the first occurrence in a string.
var allEntityPatterns = []entityPattern{
	// fenced code block — content is completely verbatim
	{re: regexp.MustCompile("(?s)```(?:[\\w]*\\n)?[\\s\\S]*?```"), open: "```", close: "```"},
	// inline code — content is completely verbatim
	{re: regexp.MustCompile("`(?:[^`\\\n]|\\\\.)*`"), open: "`", close: "`"},
	// expandable block-quote opener  **>…
	{re: regexp.MustCompile(`(?m)\*\*>(?:[^\n]*)`), open: "**>", close: ""},
	// block-quote line  >…
	{re: regexp.MustCompile(`(?m)^>(?:[^\n]*)`), open: ">", close: ""},
	// custom emoji / timestamp  ![…](…)   — must come before plain link
	{re: regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`), open: "!", close: ""},
	// inline URL / user mention  […](…)
	{re: regexp.MustCompile(`\[[^\]]*\]\([^)]*\)`), open: "[", close: ""},
	// spoiler  ||…||  — before single | so it wins
	{re: regexp.MustCompile(`\|\|(?:[^|\\\n]|\\.)*\|\|`), open: "||", close: "||"},
	// underline  __…__  — before single _ so it wins
	{re: regexp.MustCompile(`__(?:[^_\\\n]|\\.)*__`), open: "__", close: "__"},
	// bold  *…*
	{re: regexp.MustCompile(`\*(?:[^*\\\n]|\\.)*\*`), open: "*", close: "*"},
	// italic  _…_
	{re: regexp.MustCompile(`_(?:[^_\\\n]|\\.)*_`), open: "_", close: "_"},
	// strikethrough  ~…~
	{re: regexp.MustCompile(`~(?:[^~\\\n]|\\.)*~`), open: "~", close: "~"},
}

// verbatimEntities are entity types whose inner content must never be
// touched (code blocks, URLs, quotes, custom emoji).
// Their content is passed through completely unchanged.
var verbatimEntities = map[string]bool{
	"```": true,
	"`":   true,
	"**>": true,
	">":   true,
	"!":   true,
	"[":   true,
}

// markdownToTelegramMarkdownV2 converts a Markdown string into a string safe
// for sending with Telegram's MarkdownV2 parse mode.
//
// Rules:
//   - Markdown headings (# … ######) are converted to *bold*.
//   - **bold** Markdown syntax is converted to *bold*.
//   - Recognized Telegram MarkdownV2 entity spans are preserved; their inner
//     content is processed recursively so that nested valid entities are kept
//     intact while stray special characters are escaped.
//   - All plain-text segments have their MarkdownV2 special characters escaped.
//
// Reference: https://core.telegram.org/bots/api#formatting-options
func markdownToTelegramMarkdownV2(text string) string {
	// 1. Convert Markdown headings → *escaped heading text*
	text = reHeading.ReplaceAllStringFunc(text, func(match string) string {
		sub := reHeading.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		// The heading content is fresh plain text — escape everything
		// including * so the resulting *…* bold span stays valid.
		return "*" + escapeMarkdownV2(sub[1]) + "*"
	})

	// 2. Convert **bold** → *bold*
	text = reBoldStar.ReplaceAllString(text, "*$1*")

	// 3. Recursively escape the full string.
	return processText(text)
}

// processText walks `text`, finds the leftmost / longest matching entity,
// escapes the gap before it, processes the entity (recursing into its inner
// content when appropriate), then continues with the remainder.
func processText(text string) string {
	if text == "" {
		return ""
	}

	// Find the leftmost match among all entity patterns.
	bestStart := -1
	bestEnd := -1
	var bestPat *entityPattern

	for i := range allEntityPatterns {
		p := &allEntityPatterns[i]
		loc := p.re.FindStringIndex(text)
		if loc == nil {
			continue
		}
		if bestStart == -1 || loc[0] < bestStart ||
			(loc[0] == bestStart && (loc[1]-loc[0]) > (bestEnd-bestStart)) {
			bestStart = loc[0]
			bestEnd = loc[1]
			bestPat = p
		}
	}

	if bestPat == nil {
		// No entity found — escape everything.
		return escapeMarkdownV2(text)
	}

	var b strings.Builder

	// Plain text before the entity.
	if bestStart > 0 {
		b.WriteString(escapeMarkdownV2(text[:bestStart]))
	}

	// The matched entity span.
	matched := text[bestStart:bestEnd]

	if verbatimEntities[bestPat.open] {
		// Code blocks, URLs, quotes: pass through completely untouched.
		b.WriteString(matched)
	} else {
		// Inline formatting (bold, italic, underline, strikethrough, spoiler):
		// keep the delimiters and recursively process the inner content so that
		// nested entities survive but stray specials get escaped.
		openLen := len(bestPat.open)
		closeLen := len(bestPat.close)
		inner := matched[openLen : len(matched)-closeLen]

		b.WriteString(bestPat.open)
		b.WriteString(processText(inner))
		b.WriteString(bestPat.close)
	}

	// Continue with the remainder of the string.
	b.WriteString(processText(text[bestEnd:]))

	return b.String()
}

// escapeMarkdownV2 escapes every MarkdownV2 special character in a plain-text
// segment (i.e. a segment that is not part of any recognized entity).
// Already-escaped sequences (backslash + char) are forwarded verbatim to avoid
// double-escaping.
func escapeMarkdownV2(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 8)
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		// Forward an existing escape sequence verbatim.
		if ch == '\\' && i+1 < len(runes) {
			b.WriteRune(ch)
			b.WriteRune(runes[i+1])
			i++
			continue
		}
		if mdV2SpecialChars[ch] {
			b.WriteByte('\\')
		}
		b.WriteRune(ch)
	}
	return b.String()
}
