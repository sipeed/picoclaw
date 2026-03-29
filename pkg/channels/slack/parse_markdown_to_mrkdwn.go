package slack

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// Patterns are applied in order; code blocks extracted first to protect verbatim content.
	reMrkdwnCodeBlock  = regexp.MustCompile("(?s)```[\\w]*\\n?([\\s\\S]*?)```")
	reMrkdwnInlineCode = regexp.MustCompile("`([^`\n]+)`")
	reMrkdwnHeading    = regexp.MustCompile(`(?m)^#{1,6}\s+(.+)$`)
	reMrkdwnBoldStar   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reMrkdwnBoldUnder  = regexp.MustCompile(`__(.+?)__`)
	reMrkdwnStrike     = regexp.MustCompile(`~~(.+?)~~`)
	reMrkdwnImage      = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	reMrkdwnLink       = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	reMrkdwnHR         = regexp.MustCompile(`(?m)^[-*_]{3,}\s*$`)
	reMrkdwnULItem     = regexp.MustCompile(`(?m)^(\s*)[-*+]\s+`)
	// Table: line starting with optional whitespace, pipe, content, pipe.
	reMrkdwnTableRow = regexp.MustCompile(`(?m)^\|(.+)\|$`)
	// Separator row: | --- | --- | or | :---: | etc.
	reMrkdwnTableSep = regexp.MustCompile(`(?m)^\|[\s|:-]+\|$`)
)

// markdownToSlackMrkdwn converts standard Markdown (as produced by LLMs) into
// Slack's mrkdwn format.
//
// Conversion rules:
//   - Fenced code blocks and inline code are preserved unchanged.
//   - Markdown tables are converted to preformatted code blocks.
//   - Headings (# … ######) become *bold* text.
//   - **bold** and __bold__ become *bold*.
//   - ~~strikethrough~~ becomes ~strikethrough~.
//   - [text](url) becomes <url|text>.
//   - ![alt](url) becomes <url|alt>.
//   - Horizontal rules (---) are removed.
//   - Unordered list markers (-, *, +) become bullet characters (•).
//   - Italic (_text_), blockquotes (> text), and code pass through unchanged.
func markdownToSlackMrkdwn(text string) string {
	if text == "" {
		return ""
	}

	// 1. Extract code blocks and inline code to protect them from conversion.
	var codeBlocks []string
	text = reMrkdwnCodeBlock.ReplaceAllStringFunc(text, func(match string) string {
		idx := len(codeBlocks)
		codeBlocks = append(codeBlocks, match)
		return codePlaceholder(idx)
	})

	var inlineCodes []string
	text = reMrkdwnInlineCode.ReplaceAllStringFunc(text, func(match string) string {
		idx := len(inlineCodes)
		inlineCodes = append(inlineCodes, match)
		return inlineCodePlaceholder(idx)
	})

	// 2. Convert tables to preformatted code blocks.
	text = convertTables(text)

	// 3. Headings → bold. Strip any bold markers from heading content first
	//    since the heading itself is rendered as bold.
	text = reMrkdwnHeading.ReplaceAllStringFunc(text, func(match string) string {
		sub := reMrkdwnHeading.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		content := reMrkdwnBoldStar.ReplaceAllString(sub[1], "$1")
		content = reMrkdwnBoldUnder.ReplaceAllString(content, "$1")
		return "*" + content + "*"
	})

	// 4. Bold: **text** and __text__ → *text*.
	//    Process **bold** before __bold__ so nested cases like **__text__** work.
	text = reMrkdwnBoldStar.ReplaceAllString(text, "*$1*")
	text = reMrkdwnBoldUnder.ReplaceAllString(text, "*$1*")

	// 5. Strikethrough: ~~text~~ → ~text~.
	text = reMrkdwnStrike.ReplaceAllString(text, "~$1~")

	// 6. Images before links (images have the ! prefix).
	text = reMrkdwnImage.ReplaceAllString(text, "<$2|$1>")

	// 7. Links: [text](url) → <url|text>.
	text = reMrkdwnLink.ReplaceAllString(text, "<$2|$1>")

	// 8. Horizontal rules → remove.
	text = reMrkdwnHR.ReplaceAllString(text, "")

	// 9. Unordered list markers → bullet character.
	text = reMrkdwnULItem.ReplaceAllStringFunc(text, func(match string) string {
		sub := reMrkdwnULItem.FindStringSubmatch(match)
		indent := ""
		if len(sub) > 1 {
			indent = sub[1]
		}
		return indent + "\u2022 "
	})

	// 10. Restore inline code, then code blocks.
	for i, code := range inlineCodes {
		text = strings.Replace(text, inlineCodePlaceholder(i), code, 1)
	}
	for i, block := range codeBlocks {
		text = strings.Replace(text, codePlaceholder(i), block, 1)
	}

	// Clean up excessive blank lines left by removed elements.
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}

	return text
}

// convertTables detects Markdown tables (consecutive lines matching |...|) and
// wraps them in fenced code blocks so Slack renders them as preformatted text.
func convertTables(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	inTable := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if reMrkdwnTableRow.MatchString(line) {
			if !inTable {
				inTable = true
				result = append(result, "```")
			}
			// Skip separator rows (| --- | --- |) — they add no value in plain text.
			if reMrkdwnTableSep.MatchString(line) {
				continue
			}
			// Strip leading/trailing pipes and clean up cell content.
			result = append(result, formatTableRow(line))
		} else {
			if inTable {
				inTable = false
				result = append(result, "```")
			}
			result = append(result, line)
		}
	}

	// Close any trailing table.
	if inTable {
		result = append(result, "```")
	}

	return strings.Join(result, "\n")
}

// formatTableRow strips the outer pipes from a table row and trims each cell.
func formatTableRow(line string) string {
	// Remove leading and trailing pipe.
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")

	cells := strings.Split(line, "|")
	for i, cell := range cells {
		cells[i] = strings.TrimSpace(cell)
	}
	return strings.Join(cells, " | ")
}

func codePlaceholder(idx int) string {
	return fmt.Sprintf("\x00CODEBLOCK_%d\x00", idx)
}

func inlineCodePlaceholder(idx int) string {
	return fmt.Sprintf("\x00INLINECODE_%d\x00", idx)
}
