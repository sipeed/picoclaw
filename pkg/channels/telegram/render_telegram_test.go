package telegram

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func renderHTML(input string) string {
	doc := parseMarkdownAST(input)
	r := &telegramRenderer{mdv2: false}
	return r.render(doc)
}

func renderMDV2(input string) string {
	doc := parseMarkdownAST(input)
	r := &telegramRenderer{mdv2: true}
	return r.render(doc)
}

// --- HTML mode tests ---

func TestRenderHTML_PlainText(t *testing.T) {
	assert.Equal(t, "Hello, world!", renderHTML("Hello, world!"))
}

func TestRenderHTML_Bold(t *testing.T) {
	assert.Equal(t, "<b>bold</b>", renderHTML("**bold**"))
}

func TestRenderHTML_Italic(t *testing.T) {
	assert.Equal(t, "<i>italic</i>", renderHTML("*italic*"))
}

func TestRenderHTML_BoldItalic(t *testing.T) {
	got := renderHTML("***bold italic***")
	assert.Contains(t, got, "<b>")
	assert.Contains(t, got, "<i>")
}

func TestRenderHTML_Strikethrough(t *testing.T) {
	assert.Equal(t, "<s>strike</s>", renderHTML("~~strike~~"))
}

func TestRenderHTML_InlineCode(t *testing.T) {
	assert.Equal(t, "<code>code</code>", renderHTML("`code`"))
}

func TestRenderHTML_InlineCode_HTMLEscape(t *testing.T) {
	assert.Equal(t, "<code>&lt;div&gt;</code>", renderHTML("`<div>`"))
}

func TestRenderHTML_CodeBlock(t *testing.T) {
	input := "```\nfoo\nbar\n```"
	got := renderHTML(input)
	assert.Contains(t, got, "<pre><code>")
	assert.Contains(t, got, "foo\nbar")
	assert.Contains(t, got, "</code></pre>")
}

func TestRenderHTML_CodeBlock_WithLanguage(t *testing.T) {
	input := "```python\nprint('hello')\n```"
	got := renderHTML(input)
	assert.Contains(t, got, `class="language-python"`)
	assert.Contains(t, got, "print(&#39;hello&#39;)")
}

func TestRenderHTML_Heading(t *testing.T) {
	assert.Equal(t, "<b>Hello</b>", renderHTML("# Hello"))
}

func TestRenderHTML_HeadingH2(t *testing.T) {
	assert.Equal(t, "<b>Sub heading</b>", renderHTML("## Sub heading"))
}

func TestRenderHTML_Link(t *testing.T) {
	got := renderHTML("[click here](https://example.com)")
	assert.Equal(t, `<a href="https://example.com">click here</a>`, got)
}

func TestRenderHTML_UnorderedList(t *testing.T) {
	input := "- item one\n- item two\n- item three"
	got := renderHTML(input)
	assert.Contains(t, got, "• item one")
	assert.Contains(t, got, "• item two")
	assert.Contains(t, got, "• item three")
}

func TestRenderHTML_OrderedList(t *testing.T) {
	input := "1. first\n2. second\n3. third"
	got := renderHTML(input)
	assert.Contains(t, got, "1. first")
	assert.Contains(t, got, "2. second")
	assert.Contains(t, got, "3. third")
}

func TestRenderHTML_Blockquote(t *testing.T) {
	input := "> quoted text"
	got := renderHTML(input)
	assert.Contains(t, got, "<blockquote>")
	assert.Contains(t, got, "quoted text")
	assert.Contains(t, got, "</blockquote>")
}

func TestRenderHTML_HorizontalRule(t *testing.T) {
	input := "before\n\n---\n\nafter"
	got := renderHTML(input)
	assert.Contains(t, got, "———")
}

func TestRenderHTML_HTMLEscaping(t *testing.T) {
	got := renderHTML("a < b & c > d")
	assert.Contains(t, got, "&lt;")
	assert.Contains(t, got, "&amp;")
	assert.Contains(t, got, "&gt;")
}

func TestRenderHTML_Table_Narrow(t *testing.T) {
	input := "| A | B |\n|---|---|\n| 1 | 2 |"
	got := renderHTML(input)
	assert.Contains(t, got, "<pre>")
	assert.Contains(t, got, "A")
	assert.Contains(t, got, "B")
	assert.Contains(t, got, "1")
	assert.Contains(t, got, "2")
}

func TestRenderHTML_Table_Wide(t *testing.T) {
	input := "| Very Long Header Name | Another Long Header |\n|---|---|\n| Some long cell value here | Another long value here too |"
	got := renderHTML(input)
	// Wide table should fall back to list format
	assert.Contains(t, got, "<b>Row 1:</b>")
	assert.Contains(t, got, "• Very Long Header Name:")
}

func TestRenderHTML_Image_AltText(t *testing.T) {
	got := renderHTML("![alt text](https://example.com/img.png)")
	assert.Contains(t, got, "alt text")
	assert.NotContains(t, got, "<img")
}

func TestRenderHTML_Empty(t *testing.T) {
	assert.Equal(t, "", renderHTML(""))
}

func TestRenderHTML_MultipleParagraphs(t *testing.T) {
	input := "First paragraph.\n\nSecond paragraph."
	got := renderHTML(input)
	assert.Contains(t, got, "First paragraph.")
	assert.Contains(t, got, "Second paragraph.")
}

// --- MarkdownV2 mode tests ---

func TestRenderMDV2_PlainText(t *testing.T) {
	assert.Equal(t, "Hello, world\\!", renderMDV2("Hello, world!"))
}

func TestRenderMDV2_Bold(t *testing.T) {
	assert.Equal(t, "*bold*", renderMDV2("**bold**"))
}

func TestRenderMDV2_Italic(t *testing.T) {
	assert.Equal(t, "_italic_", renderMDV2("*italic*"))
}

func TestRenderMDV2_Strikethrough(t *testing.T) {
	assert.Equal(t, "~strike~", renderMDV2("~~strike~~"))
}

func TestRenderMDV2_InlineCode(t *testing.T) {
	assert.Equal(t, "`code`", renderMDV2("`code`"))
}

func TestRenderMDV2_CodeBlock(t *testing.T) {
	input := "```\nfoo\n```"
	got := renderMDV2(input)
	assert.Contains(t, got, "```\nfoo\n```")
}

func TestRenderMDV2_CodeBlock_WithLanguage(t *testing.T) {
	input := "```python\nprint('hello')\n```"
	got := renderMDV2(input)
	assert.Contains(t, got, "```python\n")
	assert.Contains(t, got, "print('hello')")
}

func TestRenderMDV2_Heading(t *testing.T) {
	assert.Equal(t, "*Hello*", renderMDV2("# Hello"))
}

func TestRenderMDV2_Link(t *testing.T) {
	got := renderMDV2("[click](https://example.com)")
	assert.Equal(t, "[click](https://example.com)", got)
}

func TestRenderMDV2_UnorderedList(t *testing.T) {
	input := "- item one\n- item two"
	got := renderMDV2(input)
	assert.Contains(t, got, "• item one")
	assert.Contains(t, got, "• item two")
}

func TestRenderMDV2_OrderedList(t *testing.T) {
	input := "1. first\n2. second"
	got := renderMDV2(input)
	assert.Contains(t, got, "1\\. first")
	assert.Contains(t, got, "2\\. second")
}

func TestRenderMDV2_Blockquote(t *testing.T) {
	input := "> quoted text"
	got := renderMDV2(input)
	assert.Contains(t, got, ">")
	assert.Contains(t, got, "quoted text")
}

func TestRenderMDV2_Escaping(t *testing.T) {
	got := renderMDV2("price is 10.99!")
	assert.Contains(t, got, "10\\.99\\!")
}

func TestRenderMDV2_Table_Narrow(t *testing.T) {
	input := "| A | B |\n|---|---|\n| 1 | 2 |"
	got := renderMDV2(input)
	assert.Contains(t, got, "```\n")
}

func TestRenderMDV2_Table_Wide(t *testing.T) {
	input := "| Very Long Header Name | Another Long Header |\n|---|---|\n| Some long cell value here | Another long value here too |"
	got := renderMDV2(input)
	assert.Contains(t, got, "*Row 1:*")
}

func TestRenderMDV2_Image_TgEmoji(t *testing.T) {
	// tg:// URLs should be passed through verbatim in MDV2
	input := "![👍](tg://emoji?id=5368324170671202286)"
	got := renderMDV2(input)
	assert.Equal(t, "![👍](tg://emoji?id=5368324170671202286)", got)
}

func TestRenderMDV2_HorizontalRule(t *testing.T) {
	input := "before\n\n---\n\nafter"
	got := renderMDV2(input)
	assert.Contains(t, got, "———")
}

// --- Integration / edge cases ---

func TestRender_NestedBoldItalic(t *testing.T) {
	input := "***bold and italic***"
	htmlGot := renderHTML(input)
	assert.Contains(t, htmlGot, "<b>")
	assert.Contains(t, htmlGot, "<i>")

	mdv2Got := renderMDV2(input)
	assert.Contains(t, mdv2Got, "*")
	assert.Contains(t, mdv2Got, "_")
}

func TestRender_CodeBlockPreservesContent(t *testing.T) {
	input := "```\n**not bold** <html>\n```"
	htmlGot := renderHTML(input)
	assert.Contains(t, htmlGot, "**not bold**")
	assert.Contains(t, htmlGot, "&lt;html&gt;")
}

func TestRender_Table_WithInlineMarkup(t *testing.T) {
	input := "| **Bold** | `code` |\n|---|---|\n| [link](url) | ~~strike~~ |"
	htmlGot := renderHTML(input)
	// Table cells should be plain text
	assert.Contains(t, htmlGot, "Bold")
	assert.Contains(t, htmlGot, "code")
}

func TestRender_Table_EmptyCells(t *testing.T) {
	input := "| A | B |\n|---|---|\n|   | 2 |"
	htmlGot := renderHTML(input)
	assert.Contains(t, htmlGot, "2")
}

func TestRender_Table_CJK(t *testing.T) {
	input := "| 名前 | バージョン |\n|---|---|\n| Go | 1.22 |"
	htmlGot := renderHTML(input)
	assert.Contains(t, htmlGot, "名前")
	assert.Contains(t, htmlGot, "Go")
}

func TestParseContent_HTML(t *testing.T) {
	got := parseContent("**hello** world", false)
	assert.Contains(t, got, "<b>hello</b>")
	assert.Contains(t, got, "world")
}

func TestParseContent_MDV2(t *testing.T) {
	got := parseContent("**hello** world", true)
	assert.Contains(t, got, "*hello*")
}

func TestParseContent_Empty(t *testing.T) {
	assert.Equal(t, "", parseContent("", false))
	assert.Equal(t, "", parseContent("", true))
}
