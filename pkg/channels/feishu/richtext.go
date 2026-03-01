//go:build amd64 || arm64 || riscv64 || mips64 || ppc64

package feishu

import (
	"encoding/json"
	"regexp"
	"strings"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// Feishu Interactive Card message format:
// {
//   "config": { "wide_screen_mode": true },
//   "elements": [
//     { "tag": "markdown", "content": "..." },
//     { "tag": "table", "columns": [...], "rows": [...] },
//     { "tag": "div", "text": { "tag": "lark_md", "content": "**Heading**" } }
//   ]
// }

// Feishu Card limits
const (
	maxTablesInCard = 1  // Feishu limit: only 1 table per card
	maxRowsInTable  = 30 // Feishu limit: max rows per table
)

var (
	// Match markdown tables (header + separator + data rows)
	reTable = regexp.MustCompile(`(?m)((?:^[ \t]*\|.+\|[ \t]*\n)(?:^[ \t]*\|[-:\s|]+\|[ \t]*\n)(?:^[ \t]*\|.+\|[ \t]*\n?)+)`)
	// Match headings
	reHeading = regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)
	// Match code blocks (to protect them from heading parsing)
	reCodeBlock = regexp.MustCompile("(```[\\s\\S]*?```)")
)

// cardElement represents an element in Feishu interactive card.
type cardElement map[string]any

// cardColumn represents a column definition in Feishu table.
type cardColumn struct {
	Tag         string `json:"tag"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Width       string `json:"width"`
}

// markdownToFeishuCard converts Markdown text to Feishu Interactive Card JSON content string.
func markdownToFeishuCard(text string) (string, error) {
	elements := buildCardElements(text)

	card := map[string]any{
		"config": map[string]any{
			"wide_screen_mode": true,
		},
		"elements": elements,
	}

	data, err := json.Marshal(card)
	return string(data), err
}

// buildCardElements splits content into div/markdown + table elements for Feishu card.
// If there are multiple tables, all tables are rendered as markdown (Feishu limit).
func buildCardElements(content string) []cardElement {
	// First, count all tables
	tableMatches := reTable.FindAllStringSubmatchIndex(content, -1)
	hasMultipleTables := len(tableMatches) > maxTablesInCard

	var elements []cardElement
	lastEnd := 0

	// Process all tables
	for _, m := range tableMatches {
		// Add content before table
		before := content[lastEnd:m[0]]
		if strings.TrimSpace(before) != "" {
			elements = append(elements, splitHeadings(before)...)
		}

		// Add table element
		tableText := content[m[0]:m[1]]
		if hasMultipleTables {
			// Multiple tables: render as markdown
			elements = append(elements, cardElement{"tag": "markdown", "content": tableText})
		} else {
			// Single table: try to parse as Feishu table
			tableEl := parseMarkdownTable(tableText)
			if tableEl != nil {
				elements = append(elements, tableEl)
			} else {
				elements = append(elements, cardElement{"tag": "markdown", "content": tableText})
			}
		}

		lastEnd = m[1]
	}

	// Add remaining content
	remaining := content[lastEnd:]
	if strings.TrimSpace(remaining) != "" {
		elements = append(elements, splitHeadings(remaining)...)
	}

	if len(elements) == 0 {
		return []cardElement{{"tag": "markdown", "content": content}}
	}

	return elements
}

// splitHeadings splits content by headings, converting headings to div elements.
func splitHeadings(content string) []cardElement {
	// Protect code blocks from heading parsing
	protected := content
	codeBlocks := make(map[string]string)
	for i, m := range reCodeBlock.FindAllString(content, -1) {
		placeholder := "\x00CODE" + string(rune('0'+i)) + "\x00"
		codeBlocks[placeholder] = m
		protected = strings.Replace(protected, m, placeholder, 1)
	}

	var elements []cardElement
	lastEnd := 0

	for _, m := range reHeading.FindAllStringSubmatchIndex(protected, -1) {
		// Add content before heading
		before := strings.TrimSpace(protected[lastEnd:m[0]])
		if before != "" {
			elements = append(elements, cardElement{"tag": "markdown", "content": before})
		}

		// Add heading as div with bold text
		headingText := strings.TrimSpace(protected[m[4]:m[5]])
		elements = append(elements, cardElement{
			"tag": "div",
			"text": map[string]any{
				"tag":     "lark_md",
				"content": "**" + headingText + "**",
			},
		})

		lastEnd = m[1]
	}

	// Add remaining content
	remaining := strings.TrimSpace(protected[lastEnd:])
	if remaining != "" {
		elements = append(elements, cardElement{"tag": "markdown", "content": remaining})
	}

	// Restore code blocks
	for placeholder, code := range codeBlocks {
		for i := range elements {
			if el, ok := elements[i]["content"].(string); ok {
				elements[i]["content"] = strings.Replace(el, placeholder, code, -1)
			}
		}
	}

	if len(elements) == 0 {
		return []cardElement{{"tag": "markdown", "content": content}}
	}

	return elements
}

// parseMarkdownTable parses a markdown table into a Feishu table element.
func parseMarkdownTable(tableText string) cardElement {
	lines := strings.Split(strings.TrimSpace(tableText), "\n")
	if len(lines) < 3 {
		return nil
	}

	// Parse header line
	headers := parseTableRow(lines[0])
	if len(headers) == 0 {
		return nil
	}

	// Skip separator line (line 1)

	// Parse data rows (limited)
	var rows []map[string]any
	for i := 2; i < len(lines) && len(rows) < maxRowsInTable; i++ {
		cells := parseTableRow(lines[i])
		if len(cells) == 0 {
			continue
		}
		row := make(map[string]any)
		for j, cell := range cells {
			if j < len(headers) {
				row[getColumnName(j)] = cell
			}
		}
		rows = append(rows, row)
	}

	// Build columns
	columns := make([]cardColumn, len(headers))
	for i, h := range headers {
		columns[i] = cardColumn{
			Tag:         "column",
			Name:        getColumnName(i),
			DisplayName: h,
			Width:       "auto",
		}
	}

	return cardElement{
		"tag":       "table",
		"page_size": len(rows) + 1,
		"columns":   columns,
		"rows":      rows,
	}
}

// parseTableRow parses a single markdown table row into cells.
func parseTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")

	parts := strings.Split(line, "|")
	cells := make([]string, 0, len(parts))
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

// getColumnName returns Feishu table column name by index (c0, c1, c2, ...).
func getColumnName(index int) string {
	if index < 10 {
		return "c" + string(rune('0'+index))
	}
	// For indices >= 10, use letter notation
	return "c" + string(rune('a'+index-10))
}

// buildFeishuContent converts content to Feishu message format.
// Returns (msgType, jsonPayload, error).
func buildFeishuContent(content string) (string, string, error) {
	cardPayload, err := markdownToFeishuCard(content)
	if err == nil {
		return larkim.MsgTypeInteractive, cardPayload, nil
	}
	// Fallback to plain text
	payload, err := json.Marshal(map[string]string{"text": content})
	if err != nil {
		return "", "", err
	}
	return larkim.MsgTypeText, string(payload), nil
}
