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
	// With markdownTableMaxWidth=42 the long cell MUST be wrapped into
	// multiple visual lines. Verify the continuation line exists.
	if !strings.Contains(got, "long cell that should wrap") {
		t.Fatalf("expected wrapped continuation line, got: %q", got)
	}
	// The continuation row must have an empty first column (padding only).
	if !strings.Contains(got, "|       | long cell") {
		t.Fatalf("expected continuation row with empty first col, got: %q", got)
	}
}

func TestFormatMarkdownTable_Width42(t *testing.T) {
	lines := []string{
		"| col1 | col2 |",
		"| --- | --- |",
		"| short | this is a very very very very long cell that should wrap |",
	}
	got := formatMarkdownTable(lines)

	// Every line must fit within markdownTableMaxWidth (42).
	for i, line := range strings.Split(got, "\n") {
		w := displayWidth(line)
		if w > markdownTableMaxWidth {
			t.Errorf("line %d width %d > %d: %q", i, w, markdownTableMaxWidth, line)
		}
	}

	// Must produce more lines than a non-wrapped table (header + sep + 1 data = 3).
	// With wrapping the data row becomes 2 visual lines → total 4.
	lineCount := len(strings.Split(got, "\n"))
	if lineCount < 4 {
		t.Errorf("expected at least 4 lines (wrap must occur), got %d:\n%s", lineCount, got)
	}

	// Verify continuation row has blank first column.
	if !strings.Contains(got, "|       | long cell that should wrap") {
		t.Errorf("expected continuation row, got:\n%s", got)
	}
}

// TestFormatMarkdownTable_MultiColWrap verifies that a multi-column,
// multi-row table correctly wraps one long cell while leaving other
// rows and columns unaffected.
// With markdownTableMaxWidth=42, col widths shrink to [7, 5, 20].
// Bob's "Needs more practice in writing" (30 chars) wraps at width 20.
func TestFormatMarkdownTable_MultiColWrap(t *testing.T) {
	lines := []string{
		"| Name | Score | Comment |",
		"| --- | --- | --- |",
		"| Alice | 95 | Great job |",
		"| Bob | 72 | Needs more practice in writing |",
		"| Charlie | 88 | Good |",
	}
	got := formatMarkdownTable(lines)
	rows := strings.Split(got, "\n")

	wantLines := []string{
		"| Name    | Score | Comment              |",
		"| ------- | ----- | -------------------- |",
		"| Alice   | 95    | Great job            |",
		"| Bob     | 72    | Needs more practice  |",
		"|         |       | in writing           |",
		"| Charlie | 88    | Good                 |",
	}

	if len(rows) != len(wantLines) {
		t.Fatalf("line count: got %d, want %d\nactual:\n%s", len(rows), len(wantLines), got)
	}

	for i, want := range wantLines {
		if rows[i] != want {
			t.Errorf("line %d:\n  got:  %q\n  want: %q", i, rows[i], want)
		}
	}

	// Every line must be exactly markdownTableMaxWidth.
	for i, line := range rows {
		w := displayWidth(line)
		if w != markdownTableMaxWidth {
			t.Errorf("line %d: displayWidth=%d, want %d: %q", i, w, markdownTableMaxWidth, line)
		}
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
