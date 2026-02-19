package channels

import "testing"

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

