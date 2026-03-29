package slack

import (
	"fmt"
	"regexp"
	"strings"

	goslack "github.com/slack-go/slack"
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

// slackMessage holds the converted content, optionally including a Block Kit
// table for native Slack table rendering.
type slackMessage struct {
	text  string             // mrkdwn-formatted text (without the first table)
	table *goslack.TableBlock // nil when the message has no table
}

// convertMessage converts standard Markdown into Slack's mrkdwn format and
// extracts the first Markdown table as a Block Kit TableBlock. Slack allows
// only one table per message, so any additional tables fall back to
// preformatted code blocks.
func convertMessage(text string) slackMessage {
	if text == "" {
		return slackMessage{}
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

	// 2. Extract tables — first table becomes a Block Kit TableBlock,
	//    any subsequent tables become preformatted code blocks.
	var table *goslack.TableBlock
	text, table = extractTables(text)

	// 3. Apply inline mrkdwn conversions.
	text = convertInlineMarkdown(text)

	// 4. Restore inline code, then code blocks.
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

	return slackMessage{text: text, table: table}
}

// convertInlineMarkdown applies all non-table mrkdwn transformations.
func convertInlineMarkdown(text string) string {
	// Headings → bold. Strip any bold markers from heading content first
	// since the heading itself is rendered as bold.
	text = reMrkdwnHeading.ReplaceAllStringFunc(text, func(match string) string {
		sub := reMrkdwnHeading.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		content := reMrkdwnBoldStar.ReplaceAllString(sub[1], "$1")
		content = reMrkdwnBoldUnder.ReplaceAllString(content, "$1")
		return "*" + content + "*"
	})

	// Bold: **text** and __text__ → *text*.
	text = reMrkdwnBoldStar.ReplaceAllString(text, "*$1*")
	text = reMrkdwnBoldUnder.ReplaceAllString(text, "*$1*")

	// Strikethrough: ~~text~~ → ~text~.
	text = reMrkdwnStrike.ReplaceAllString(text, "~$1~")

	// Images before links (images have the ! prefix).
	text = reMrkdwnImage.ReplaceAllString(text, "<$2|$1>")

	// Links: [text](url) → <url|text>.
	text = reMrkdwnLink.ReplaceAllString(text, "<$2|$1>")

	// Horizontal rules → remove.
	text = reMrkdwnHR.ReplaceAllString(text, "")

	// Unordered list markers → bullet character.
	text = reMrkdwnULItem.ReplaceAllStringFunc(text, func(match string) string {
		sub := reMrkdwnULItem.FindStringSubmatch(match)
		indent := ""
		if len(sub) > 1 {
			indent = sub[1]
		}
		return indent + "\u2022 "
	})

	return text
}

// extractTables scans the text for Markdown tables. The first table found is
// converted to a Block Kit TableBlock and replaced with a placeholder that is
// later removed. Any additional tables are converted to preformatted code blocks.
func extractTables(text string) (string, *goslack.TableBlock) {
	lines := strings.Split(text, "\n")
	var result []string
	var tableLines []string
	var firstTable *goslack.TableBlock
	inTable := false
	tableCount := 0

	flush := func() {
		if len(tableLines) == 0 {
			return
		}
		rows := parseTableRows(tableLines)
		tableCount++

		if tableCount == 1 && len(rows) > 0 {
			// First table → Block Kit TableBlock.
			firstTable = buildTableBlock(rows)
			// Leave a blank line where the table was (table renders at bottom).
		} else {
			// Subsequent tables → preformatted code block fallback.
			result = append(result, "```")
			for _, row := range rows {
				result = append(result, strings.Join(row, " | "))
			}
			result = append(result, "```")
		}
		tableLines = nil
	}

	for _, line := range lines {
		if reMrkdwnTableRow.MatchString(line) {
			if !inTable {
				inTable = true
			}
			tableLines = append(tableLines, line)
		} else {
			if inTable {
				inTable = false
				flush()
			}
			result = append(result, line)
		}
	}
	if inTable {
		flush()
	}

	return strings.Join(result, "\n"), firstTable
}

// parseTableRows extracts cell contents from Markdown table lines, skipping
// separator rows and stripping inline markdown formatting.
func parseTableRows(lines []string) [][]string {
	var rows [][]string
	for _, line := range lines {
		if reMrkdwnTableSep.MatchString(line) {
			continue
		}
		row := splitTableRow(line)
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}
	return rows
}

// splitTableRow parses a single | ... | line into trimmed, markdown-stripped cells.
func splitTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")

	cells := strings.Split(line, "|")
	for i, cell := range cells {
		cell = strings.TrimSpace(cell)
		cell = reMrkdwnBoldStar.ReplaceAllString(cell, "$1")
		cell = reMrkdwnBoldUnder.ReplaceAllString(cell, "$1")
		cell = reMrkdwnStrike.ReplaceAllString(cell, "$1")
		cell = reMrkdwnInlineCode.ReplaceAllString(cell, "$1")
		cells[i] = cell
	}
	return cells
}

// buildTableBlock constructs a Slack Block Kit TableBlock from parsed rows.
// The first row is automatically rendered as a header by Slack.
func buildTableBlock(rows [][]string) *goslack.TableBlock {
	table := goslack.NewTableBlock("")

	// Set all columns to left-aligned and wrapped.
	if len(rows) > 0 {
		settings := make([]goslack.ColumnSetting, len(rows[0]))
		for i := range settings {
			settings[i] = goslack.ColumnSetting{
				Align:     goslack.ColumnAlignmentLeft,
				IsWrapped: true,
			}
		}
		table.WithColumnSettings(settings...)
	}

	for _, row := range rows {
		cells := make([]*goslack.RichTextBlock, len(row))
		for i, cellText := range row {
			cells[i] = goslack.NewRichTextBlock("",
				goslack.NewRichTextSection(
					goslack.NewRichTextSectionTextElement(cellText, nil),
				),
			)
		}
		table.AddRow(cells...)
	}

	return table
}

func codePlaceholder(idx int) string {
	return fmt.Sprintf("\x00CODEBLOCK_%d\x00", idx)
}

func inlineCodePlaceholder(idx int) string {
	return fmt.Sprintf("\x00INLINECODE_%d\x00", idx)
}
