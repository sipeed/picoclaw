package channels

import (
	"strings"
	"testing"
)

func TestSanitizeTelegramOutgoingContent_RemovesThinkBlock(t *testing.T) {
	in := "<think>\nsecret reasoning\n</think>\n\nユーザー向け本文"
	got := sanitizeTelegramOutgoingContent(in)
	want := "ユーザー向け本文"
	if got != want {
		t.Fatalf("sanitizeTelegramOutgoingContent() = %q, want %q", got, want)
	}
}

func TestSanitizeTelegramOutgoingContent_EmptyAfterThink(t *testing.T) {
	in := "<think>only reasoning</think>"
	got := sanitizeTelegramOutgoingContent(in)
	want := "(empty response)"
	if got != want {
		t.Fatalf("sanitizeTelegramOutgoingContent() = %q, want %q", got, want)
	}
}

func TestMarkdownToTelegramHTML_Heading(t *testing.T) {
	in := "### メッセージ\n- A\n- B"
	got := markdownToTelegramHTML(in)
	if !strings.Contains(got, "<b>メッセージ</b>") {
		t.Fatalf("expected heading to be bold, got: %q", got)
	}
}

func TestMarkdownToTelegramHTML_Table(t *testing.T) {
	in := "| col1 | col2 |\n| --- | --- |\n| a | b |"
	got := markdownToTelegramHTML(in)
	if !strings.Contains(got, "<pre><code>") {
		t.Fatalf("expected table to render as code block, got: %q", got)
	}
	if !strings.Contains(got, "col1") || !strings.Contains(got, "a") {
		t.Fatalf("expected table content to remain, got: %q", got)
	}
}

func TestMarkdownToTelegramHTML_TableCJKAlignment(t *testing.T) {
	in := "| 項目 | value |\n| --- | --- |\n| 温度 | 23.5 |\n| 状態 | 正常 |"
	got := markdownToTelegramHTML(in)

	mustContain := []string{
		"<pre><code>",
		"| 項目",
		"| value",
		"| ----",
		"| 温度",
		"| 状態",
	}
	for _, s := range mustContain {
		if !strings.Contains(got, s) {
			t.Fatalf("expected output to contain %q, got: %q", s, got)
		}
	}
}

func TestMarkdownToTelegramHTML_TableWrapsLongCell(t *testing.T) {
	in := "| col1 | col2 |\n| --- | --- |\n| short | this is a very very very very long cell that should wrap |"
	got := markdownToTelegramHTML(in)

	if !strings.Contains(got, "<pre><code>") {
		t.Fatalf("expected table to render as code block, got: %q", got)
	}
	if !strings.Contains(got, "| col1") || !strings.Contains(got, "| col2") {
		t.Fatalf("expected header row, got: %q", got)
	}
	if !strings.Contains(got, "| short") {
		t.Fatalf("expected data row with short cell, got: %q", got)
	}
	// Wrapped content should include a prefix from the long sentence.
	if !strings.Contains(got, "this is a very") {
		t.Fatalf("expected wrapped long cell content, got: %q", got)
	}
}

func TestDisplayWidth_EmojiIsThree(t *testing.T) {
	got := displayWidth("⭐⭐⭐⭐⭐")
	if got != 15 {
		t.Fatalf("displayWidth(stars) = %d, want 15", got)
	}
}

func TestParseChatID_Plain(t *testing.T) {
	id, err := parseChatID("123456789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 123456789 {
		t.Errorf("expected 123456789, got %d", id)
	}
}
