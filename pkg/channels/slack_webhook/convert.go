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

	// Check if any row exceeds max width
	for _, line := range lines {
		if len(line) > maxTableRowWidth {
			return "```\n" + tableStr + "\n```"
		}
	}

	// Render as formatted text with bold headers
	var result strings.Builder
	for i, line := range lines {
		if i == 1 && isSeparatorRow(line) {
			continue
		}
		cells := parseTableRow(line)
		if len(cells) == 0 {
			continue
		}
		if i == 0 {
			// Header row - bold each cell
			var boldCells []string
			for _, cell := range cells {
				boldCells = append(boldCells, "*"+strings.TrimSpace(cell)+"*")
			}
			result.WriteString(strings.Join(boldCells, " | "))
		} else {
			result.WriteString(strings.Join(cells, " | "))
		}
		result.WriteString("\n")
	}
	return strings.TrimSuffix(result.String(), "\n")
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