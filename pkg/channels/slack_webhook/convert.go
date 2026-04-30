package slackwebhook

import (
	"regexp"
	"strings"
)

const maxTableRowWidth = 60

var (
	boldRe          = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	strikeRe        = regexp.MustCompile(`~~([^~]+)~~`)
	linkRe          = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	headerRe        = regexp.MustCompile(`(?m)^#{1,6}\s+(.+)$`)
	bulletRe        = regexp.MustCompile(`(?m)^- (.+)$`)
	markdownTableRe = regexp.MustCompile(`(?m)^(\|[^\n]+\|)\n(\|[-:\|\s]+\|)\n((?:\|[^\n]+\|\n?)+)`)
	codeBlockRe     = regexp.MustCompile("(?s)```.*?```")
	inlineCodeRe    = regexp.MustCompile("`[^`]+`")
	italicRe        = regexp.MustCompile(`(?:^|[^*])\*([^*]+)\*(?:[^*]|$)`)
)

type contentSegment struct {
	content string
	isTable bool
}

func convertMarkdownToMrkdwn(text string) string {
	// Protect code blocks from conversion
	var codeBlocks []string
	text = codeBlockRe.ReplaceAllStringFunc(text, func(match string) string {
		codeBlocks = append(codeBlocks, match)
		return "\x00CODEBLOCK\x00"
	})

	// Protect inline code
	var inlineCode []string
	text = inlineCodeRe.ReplaceAllStringFunc(text, func(match string) string {
		inlineCode = append(inlineCode, match)
		return "\x00INLINE\x00"
	})

	// Convert italic *text* → _text_ BEFORE bold conversion
	text = italicRe.ReplaceAllStringFunc(text, func(match string) string {
		// Find the asterisk positions
		firstAsterisk := strings.Index(match, "*")
		lastAsterisk := strings.LastIndex(match, "*")
		if firstAsterisk == lastAsterisk {
			return match // Only one asterisk, not italic
		}

		// Extract content between asterisks
		content := match[firstAsterisk+1 : lastAsterisk]

		// Replace with underscores, preserving any prefix/suffix
		return match[:firstAsterisk] + "_" + content + "_" + match[lastAsterisk+1:]
	})

	// Convert bold **text** → *text*
	text = boldRe.ReplaceAllString(text, "*$1*")

	// Convert strikethrough ~~text~~ → ~text~
	text = strikeRe.ReplaceAllString(text, "~$1~")

	// Convert links [text](url) → <url|text>
	text = linkRe.ReplaceAllString(text, "<$2|$1>")

	// Convert headers # text → *text*
	text = headerRe.ReplaceAllString(text, "*$1*")

	// Convert bullet lists - item → • item
	text = bulletRe.ReplaceAllString(text, "• $1")

	// Restore inline code
	for _, code := range inlineCode {
		text = strings.Replace(text, "\x00INLINE\x00", code, 1)
	}

	// Restore code blocks
	for _, block := range codeBlocks {
		text = strings.Replace(text, "\x00CODEBLOCK\x00", block, 1)
	}

	return text
}

func splitContentWithTables(content string) []contentSegment {
	var segments []contentSegment

	matches := markdownTableRe.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return []contentSegment{{content: content, isTable: false}}
	}

	lastEnd := 0
	for _, match := range matches {
		if match[0] > lastEnd {
			segments = append(segments, contentSegment{
				content: content[lastEnd:match[0]],
				isTable: false,
			})
		}
		segments = append(segments, contentSegment{
			content: content[match[0]:match[1]],
			isTable: true,
		})
		lastEnd = match[1]
	}

	if lastEnd < len(content) {
		segments = append(segments, contentSegment{
			content: content[lastEnd:],
			isTable: false,
		})
	}

	return segments
}

func renderTable(tableStr string) string {
	lines := strings.Split(strings.TrimSpace(tableStr), "\n")
	if len(lines) < 2 {
		return "```\n" + tableStr + "\n```"
	}

	// Parse all rows to get column widths
	var allRows [][]string
	for i, line := range lines {
		if i == 1 && isSeparatorRow(line) {
			continue
		}
		cells := parseTableRow(line)
		if len(cells) > 0 {
			allRows = append(allRows, cells)
		}
	}

	if len(allRows) == 0 {
		return "```\n" + tableStr + "\n```"
	}

	// Calculate max width for each column
	colWidths := make([]int, len(allRows[0]))
	for _, row := range allRows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Check if table is narrow enough for mrkdwn format
	totalWidth := 0
	for _, w := range colWidths {
		totalWidth += w + 3 // " | " separator
	}
	if totalWidth <= maxTableRowWidth {
		// Render as formatted text with bold headers
		var result strings.Builder
		for i, row := range allRows {
			if i == 0 {
				var boldCells []string
				for _, cell := range row {
					boldCells = append(boldCells, "*"+cell+"*")
				}
				result.WriteString(strings.Join(boldCells, " | "))
			} else {
				result.WriteString(strings.Join(row, " | "))
			}
			result.WriteString("\n")
		}
		return strings.TrimSuffix(result.String(), "\n")
	}

	// Render as aligned code block
	var result strings.Builder
	result.WriteString("```\n")
	for i, row := range allRows {
		var paddedCells []string
		for j, cell := range row {
			if j < len(colWidths) {
				paddedCells = append(paddedCells, padRight(cell, colWidths[j]))
			} else {
				paddedCells = append(paddedCells, cell)
			}
		}
		result.WriteString("| ")
		result.WriteString(strings.Join(paddedCells, " | "))
		result.WriteString(" |\n")

		// Add separator after header
		if i == 0 {
			var sepParts []string
			for _, w := range colWidths {
				sepParts = append(sepParts, strings.Repeat("-", w))
			}
			result.WriteString("|-")
			result.WriteString(strings.Join(sepParts, "-|-"))
			result.WriteString("-|\n")
		}
	}
	result.WriteString("```")
	return result.String()
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func isSeparatorRow(line string) bool {
	cleaned := strings.ReplaceAll(line, "|", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, ":", "")
	return cleaned == ""
}

func parseTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	if line == "" {
		return nil
	}
	parts := strings.Split(line, "|")
	var cells []string
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}
