package telegram

import (
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
)

// parseMarkdownAST parses standard Markdown text into a gomarkdown AST.
func parseMarkdownAST(text string) ast.Node {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs |
		parser.Strikethrough | parser.Tables | parser.FencedCode
	p := parser.NewWithExtensions(extensions)
	return markdown.Parse([]byte(text), p)
}

// telegramRenderer walks a gomarkdown AST and produces Telegram-compatible
// HTML or MarkdownV2 output.
type telegramRenderer struct {
	buf  strings.Builder
	mdv2 bool // true → MarkdownV2, false → HTML
}

// render produces the final formatted string from an AST root node.
func (r *telegramRenderer) render(doc ast.Node) string {
	r.buf.Reset()
	r.walkChildren(doc)
	return strings.TrimRight(r.buf.String(), "\n")
}

// walkChildren iterates over direct children of a node.
func (r *telegramRenderer) walkChildren(node ast.Node) {
	for _, child := range node.GetChildren() {
		r.renderNode(child)
	}
}

// renderNode dispatches rendering for a single AST node.
func (r *telegramRenderer) renderNode(node ast.Node) {
	switch n := node.(type) {
	case *ast.Document:
		r.walkChildren(n)

	case *ast.Paragraph:
		r.renderParagraph(n)

	case *ast.Heading:
		r.renderHeading(n)

	case *ast.CodeBlock:
		r.renderCodeBlock(n)

	case *ast.BlockQuote:
		r.renderBlockQuote(n)

	case *ast.List:
		r.renderList(n)

	case *ast.ListItem:
		r.walkChildren(n)

	case *ast.Table:
		r.renderTable(n)

	case *ast.HorizontalRule:
		r.buf.WriteString("———\n")

	case *ast.Text:
		r.renderText(n)

	case *ast.Strong:
		r.renderStrong(n)

	case *ast.Emph:
		r.renderEmph(n)

	case *ast.Del:
		r.renderDel(n)

	case *ast.Code:
		r.renderInlineCode(n)

	case *ast.Link:
		r.renderLink(n)

	case *ast.Image:
		r.renderImage(n)

	case *ast.Softbreak:
		r.buf.WriteByte('\n')

	case *ast.Hardbreak:
		r.buf.WriteByte('\n')

	case *ast.HTMLSpan:
		// Pass through raw HTML spans (e.g. <br>)
		r.writeEscaped(string(n.Literal))

	case *ast.HTMLBlock:
		r.writeEscaped(string(n.Literal))

	default:
		// Fallback: render children if any
		r.walkChildren(node)
	}
}

// --- Block-level renderers ---

func (r *telegramRenderer) renderParagraph(n *ast.Paragraph) {
	// Don't add newline before the very first output.
	if r.buf.Len() > 0 {
		r.buf.WriteByte('\n')
	}
	r.walkChildren(n)
	r.buf.WriteByte('\n')
}

func (r *telegramRenderer) renderHeading(n *ast.Heading) {
	if r.buf.Len() > 0 {
		r.buf.WriteByte('\n')
	}
	if r.mdv2 {
		r.buf.WriteString("*")
		r.walkChildren(n)
		r.buf.WriteString("*")
	} else {
		r.buf.WriteString("<b>")
		r.walkChildren(n)
		r.buf.WriteString("</b>")
	}
	r.buf.WriteByte('\n')
}

func (r *telegramRenderer) renderCodeBlock(n *ast.CodeBlock) {
	if r.buf.Len() > 0 {
		r.buf.WriteByte('\n')
	}
	lang := strings.TrimSpace(string(n.Info))
	content := string(n.Literal)
	// Remove trailing newline from code content (gomarkdown includes it).
	content = strings.TrimRight(content, "\n")

	if r.mdv2 {
		r.buf.WriteString("```")
		if lang != "" {
			r.buf.WriteString(lang)
		}
		r.buf.WriteByte('\n')
		r.buf.WriteString(content)
		r.buf.WriteString("\n```\n")
	} else {
		if lang != "" {
			r.buf.WriteString("<pre><code class=\"language-")
			r.buf.WriteString(escapeHTMLFull(lang))
			r.buf.WriteString("\">")
		} else {
			r.buf.WriteString("<pre><code>")
		}
		r.buf.WriteString(escapeHTMLFull(content))
		r.buf.WriteString("</code></pre>\n")
	}
}

func (r *telegramRenderer) renderBlockQuote(n *ast.BlockQuote) {
	if r.buf.Len() > 0 {
		r.buf.WriteByte('\n')
	}

	// Render children to a sub-renderer to get the text, then prefix lines.
	sub := &telegramRenderer{mdv2: r.mdv2}
	sub.walkChildren(n)
	text := strings.TrimRight(sub.buf.String(), "\n")

	if r.mdv2 {
		for i, line := range strings.Split(text, "\n") {
			if i > 0 {
				r.buf.WriteByte('\n')
			}
			r.buf.WriteString(">")
			r.buf.WriteString(line)
		}
	} else {
		r.buf.WriteString("<blockquote>")
		r.buf.WriteString(text)
		r.buf.WriteString("</blockquote>")
	}
	r.buf.WriteByte('\n')
}

func (r *telegramRenderer) renderList(n *ast.List) {
	if r.buf.Len() > 0 {
		r.buf.WriteByte('\n')
	}

	ordered := (n.ListFlags & ast.ListTypeOrdered) != 0
	counter := 1

	for _, item := range n.GetChildren() {
		li, ok := item.(*ast.ListItem)
		if !ok {
			continue
		}

		if ordered {
			num := itoa(counter)
			if r.mdv2 {
				r.buf.WriteString(escapeMarkdownV2(num))
				r.buf.WriteString("\\. ")
			} else {
				r.buf.WriteString(num)
				r.buf.WriteString(". ")
			}
			counter++
		} else {
			r.buf.WriteString("• ")
		}

		// Render list item children inline (strip paragraph wrapping).
		r.renderListItemChildren(li)
		r.buf.WriteByte('\n')
	}
}

// renderListItemChildren renders the children of a list item, flattening
// single-paragraph items to avoid extra newlines.
func (r *telegramRenderer) renderListItemChildren(li *ast.ListItem) {
	children := li.GetChildren()
	if len(children) == 1 {
		if p, ok := children[0].(*ast.Paragraph); ok {
			// Single paragraph — render its children directly.
			r.walkChildren(p)
			return
		}
	}
	// Multiple children or non-paragraph: render normally.
	for _, child := range children {
		if p, ok := child.(*ast.Paragraph); ok {
			r.walkChildren(p)
		} else {
			r.renderNode(child)
		}
	}
}

func (r *telegramRenderer) renderTable(n *ast.Table) {
	if r.buf.Len() > 0 {
		r.buf.WriteByte('\n')
	}

	td := r.extractTableData(n)

	if tableWidth(td) <= tableMaxMonoWidth {
		mono := renderTableMono(td)
		if r.mdv2 {
			r.buf.WriteString("```\n")
			r.buf.WriteString(mono)
			r.buf.WriteString("\n```\n")
		} else {
			r.buf.WriteString("<pre>")
			r.buf.WriteString(escapeHTMLFull(mono))
			r.buf.WriteString("</pre>\n")
		}
	} else {
		if r.mdv2 {
			r.buf.WriteString(renderTableAsListMDV2(td))
		} else {
			r.buf.WriteString(renderTableAsListHTML(td))
		}
		r.buf.WriteByte('\n')
	}
}

// extractTableData converts a gomarkdown ast.Table into our tableData struct.
func (r *telegramRenderer) extractTableData(n *ast.Table) tableData {
	var td tableData

	for _, child := range n.GetChildren() {
		switch section := child.(type) {
		case *ast.TableHeader:
			for _, row := range section.GetChildren() {
				if tr, ok := row.(*ast.TableRow); ok {
					for _, cell := range tr.GetChildren() {
						if tc, ok := cell.(*ast.TableCell); ok {
							td.headers = append(td.headers, r.cellText(tc))
						}
					}
				}
			}
		case *ast.TableBody:
			for _, row := range section.GetChildren() {
				if tr, ok := row.(*ast.TableRow); ok {
					var rowCells []string
					for _, cell := range tr.GetChildren() {
						if tc, ok := cell.(*ast.TableCell); ok {
							rowCells = append(rowCells, r.cellText(tc))
						}
					}
					td.rows = append(td.rows, rowCells)
				}
			}
		}
	}

	return td
}

// cellText extracts plain text from a table cell, stripping inline markup.
func (r *telegramRenderer) cellText(cell *ast.TableCell) string {
	var b strings.Builder
	plainTextWalk(&b, cell)
	return strings.TrimSpace(b.String())
}

// plainTextWalk recursively extracts plain text from an AST subtree.
func plainTextWalk(b *strings.Builder, node ast.Node) {
	if t, ok := node.(*ast.Text); ok {
		b.Write(t.Literal)
		return
	}
	if c, ok := node.(*ast.Code); ok {
		b.Write(c.Literal)
		return
	}
	for _, child := range node.GetChildren() {
		plainTextWalk(b, child)
	}
}

// --- Inline renderers ---

func (r *telegramRenderer) renderText(n *ast.Text) {
	text := string(n.Literal)
	r.writeEscaped(text)
}

func (r *telegramRenderer) renderStrong(n *ast.Strong) {
	if r.mdv2 {
		r.buf.WriteString("*")
		r.walkChildren(n)
		r.buf.WriteString("*")
	} else {
		r.buf.WriteString("<b>")
		r.walkChildren(n)
		r.buf.WriteString("</b>")
	}
}

func (r *telegramRenderer) renderEmph(n *ast.Emph) {
	if r.mdv2 {
		r.buf.WriteString("_")
		r.walkChildren(n)
		r.buf.WriteString("_")
	} else {
		r.buf.WriteString("<i>")
		r.walkChildren(n)
		r.buf.WriteString("</i>")
	}
}

func (r *telegramRenderer) renderDel(n *ast.Del) {
	if r.mdv2 {
		r.buf.WriteString("~")
		r.walkChildren(n)
		r.buf.WriteString("~")
	} else {
		r.buf.WriteString("<s>")
		r.walkChildren(n)
		r.buf.WriteString("</s>")
	}
}

func (r *telegramRenderer) renderInlineCode(n *ast.Code) {
	content := string(n.Literal)
	if r.mdv2 {
		r.buf.WriteString("`")
		r.buf.WriteString(content)
		r.buf.WriteString("`")
	} else {
		r.buf.WriteString("<code>")
		r.buf.WriteString(escapeHTMLFull(content))
		r.buf.WriteString("</code>")
	}
}

func (r *telegramRenderer) renderLink(n *ast.Link) {
	url := string(n.Destination)
	if r.mdv2 {
		r.buf.WriteString("[")
		r.walkChildren(n)
		r.buf.WriteString("](")
		r.buf.WriteString(url)
		r.buf.WriteString(")")
	} else {
		r.buf.WriteString(`<a href="`)
		r.buf.WriteString(escapeHTMLFull(url))
		r.buf.WriteString(`">`)
		r.walkChildren(n)
		r.buf.WriteString("</a>")
	}
}

func (r *telegramRenderer) renderImage(n *ast.Image) {
	// Telegram doesn't support inline images.
	// For tg:// URLs, pass through as-is for MarkdownV2 compatibility.
	url := string(n.Destination)
	if r.mdv2 && strings.HasPrefix(url, "tg://") {
		r.buf.WriteString("![")
		// Render alt text children
		for _, child := range n.GetChildren() {
			if t, ok := child.(*ast.Text); ok {
				r.buf.Write(t.Literal)
			}
		}
		r.buf.WriteString("](")
		r.buf.WriteString(url)
		r.buf.WriteString(")")
	} else {
		// For non-Telegram images, just output the alt text.
		for _, child := range n.GetChildren() {
			if t, ok := child.(*ast.Text); ok {
				r.writeEscaped(string(t.Literal))
			}
		}
	}
}

// writeEscaped writes text with the appropriate escaping for the current mode.
func (r *telegramRenderer) writeEscaped(text string) {
	if r.mdv2 {
		r.buf.WriteString(escapeMarkdownV2(text))
	} else {
		r.buf.WriteString(escapeHTMLFull(text))
	}
}

// mdV2SpecialChars are all characters that must be escaped in Telegram MarkdownV2.
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

// escapeMarkdownV2 escapes every MarkdownV2 special character in a plain-text
// segment. Already-escaped sequences (backslash + char) are forwarded verbatim.
// markdownToTelegramMarkdownV2 converts standard Markdown to Telegram MarkdownV2
// using the gomarkdown AST parser and the dual-mode telegramRenderer.
func markdownToTelegramMarkdownV2(text string) string {
	if text == "" {
		return ""
	}
	doc := parseMarkdownAST(text)
	r := &telegramRenderer{mdv2: true}
	return r.render(doc)
}

func escapeMarkdownV2(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 8)
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
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

// escapeHTMLFull escapes &, <, > for Telegram HTML mode.
func escapeHTMLFull(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	text = strings.ReplaceAll(text, "'", "&#39;")
	return text
}
