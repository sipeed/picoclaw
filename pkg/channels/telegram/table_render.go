package telegram

import (
	"strings"
	"unicode"
)

const tableMaxMonoWidth = 40

// runeWidth returns the display width of a rune (CJK = 2, others = 1).
func runeWidth(r rune) int {
	if unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hangul, r) ||
		// Hiragana, Katakana, CJK symbols & punctuation (includes ー U+30FC)
		(r >= 0x3000 && r <= 0x30FF) ||
		// Fullwidth forms
		(r >= 0xFF01 && r <= 0xFF60) ||
		(r >= 0xFFE0 && r <= 0xFFE6) {
		return 2
	}
	return 1
}

// stringWidth returns the total display width of s in half-width units.
func stringWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}

// padRight pads s with spaces to reach targetWidth display units.
func padRight(s string, targetWidth int) string {
	gap := targetWidth - stringWidth(s)
	if gap <= 0 {
		return s
	}
	return s + strings.Repeat(" ", gap)
}

// tableData holds parsed table information for rendering.
type tableData struct {
	headers []string
	rows    [][]string
}

// renderTableMono renders a table in monospace format.
// Output is plain text (no HTML/MDV2 escaping); the caller wraps it.
func renderTableMono(td tableData) string {
	numCols := len(td.headers)
	if numCols == 0 {
		return ""
	}

	// Compute column widths.
	colWidths := make([]int, numCols)
	for i, h := range td.headers {
		colWidths[i] = stringWidth(h)
	}
	for _, row := range td.rows {
		for i := 0; i < numCols && i < len(row); i++ {
			w := stringWidth(row[i])
			if w > colWidths[i] {
				colWidths[i] = w
			}
		}
	}

	var b strings.Builder

	// Header row.
	for i, h := range td.headers {
		if i > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(padRight(h, colWidths[i]))
	}
	b.WriteByte('\n')

	// Separator row.
	for i, w := range colWidths {
		if i > 0 {
			b.WriteString("-+-")
		}
		b.WriteString(strings.Repeat("-", w))
	}

	// Data rows.
	for _, row := range td.rows {
		b.WriteByte('\n')
		for i := 0; i < numCols; i++ {
			if i > 0 {
				b.WriteString(" | ")
			}
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			b.WriteString(padRight(cell, colWidths[i]))
		}
	}

	return b.String()
}

// renderTableAsListHTML renders a wide table as a bulleted list in HTML.
func renderTableAsListHTML(td tableData) string {
	var b strings.Builder
	for i, row := range td.rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("<b>Row ")
		b.WriteString(itoa(i + 1))
		b.WriteString(":</b>\n")
		for j, h := range td.headers {
			cell := ""
			if j < len(row) {
				cell = row[j]
			}
			b.WriteString("• ")
			b.WriteString(escapeHTML(h))
			b.WriteString(": ")
			b.WriteString(escapeHTML(cell))
			b.WriteByte('\n')
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// renderTableAsListMDV2 renders a wide table as a bulleted list in MarkdownV2.
func renderTableAsListMDV2(td tableData) string {
	var b strings.Builder
	for i, row := range td.rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("*Row ")
		b.WriteString(escapeMarkdownV2(itoa(i + 1)))
		b.WriteString(":*\n")
		for j, h := range td.headers {
			cell := ""
			if j < len(row) {
				cell = row[j]
			}
			b.WriteString("• ")
			b.WriteString(escapeMarkdownV2(h))
			b.WriteString(": ")
			b.WriteString(escapeMarkdownV2(cell))
			b.WriteByte('\n')
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// tableWidth returns the total display width of a mono-rendered table.
func tableWidth(td tableData) int {
	numCols := len(td.headers)
	if numCols == 0 {
		return 0
	}

	colWidths := make([]int, numCols)
	for i, h := range td.headers {
		colWidths[i] = stringWidth(h)
	}
	for _, row := range td.rows {
		for i := 0; i < numCols && i < len(row); i++ {
			w := stringWidth(row[i])
			if w > colWidths[i] {
				colWidths[i] = w
			}
		}
	}

	total := 0
	for _, w := range colWidths {
		total += w
	}
	// Add separators: " | " = 3 chars per gap
	total += 3 * (numCols - 1)
	return total
}

// itoa is a small helper to avoid importing strconv for int→string.
func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	// Simple recursive for small numbers (row counts are always small).
	return itoa(n/10) + string(rune('0'+n%10))
}
