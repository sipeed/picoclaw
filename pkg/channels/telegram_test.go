package channels

import (
	"strings"
	"testing"
)

func TestMarkdownToTelegramHTML_MultilineFormatting(t *testing.T) {
	input := "## Titulo\n- item um\n- item dois\n1. item tres\n> citação"
	got := markdownToTelegramHTML(input)

	if strings.Contains(got, "##") {
		t.Fatalf("heading marker should be removed, got: %q", got)
	}
	if !strings.Contains(got, "• item um") || !strings.Contains(got, "• item dois") || !strings.Contains(got, "• item tres") {
		t.Fatalf("list markers should be normalized, got: %q", got)
	}
	if strings.Contains(got, "> citação") {
		t.Fatalf("blockquote marker should be removed, got: %q", got)
	}
}

func TestMarkdownToTelegramHTML_MultipleCodeBlocks(t *testing.T) {
	input := "```go\nfmt.Println(1)\n```\ntexto\n```js\nconsole.log(2)\n```"
	got := markdownToTelegramHTML(input)

	if strings.Count(got, "<pre><code>") != 2 {
		t.Fatalf("expected 2 code blocks, got: %q", got)
	}
	if !strings.Contains(got, "fmt.Println(1)") || !strings.Contains(got, "console.log(2)") {
		t.Fatalf("missing code block contents, got: %q", got)
	}
}

func TestMarkdownToTelegramHTML_MultipleInlineCodes(t *testing.T) {
	input := "use `foo` and `bar`"
	got := markdownToTelegramHTML(input)

	if strings.Count(got, "<code>") != 2 {
		t.Fatalf("expected 2 inline code tags, got: %q", got)
	}
	if !strings.Contains(got, "<code>foo</code>") || !strings.Contains(got, "<code>bar</code>") {
		t.Fatalf("missing inline code contents, got: %q", got)
	}
}
