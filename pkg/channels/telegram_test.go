package channels

import (
	"strings"
	"testing"
)

func TestSplitTelegramMessageContentShortMessage(t *testing.T) {
	input := "hello world"
	chunks := splitTelegramMessageContent(input, telegramMaxMessageLength)

	if len(chunks) != 1 {
		t.Fatalf("len(chunks) = %d, want 1", len(chunks))
	}
	if chunks[0] != input {
		t.Fatalf("chunk[0] = %q, want %q", chunks[0], input)
	}
}

func TestSplitTelegramMessageContentLongMessage(t *testing.T) {
	input := strings.Repeat("This is a long telegram message chunk. ", 300)
	chunks := splitTelegramMessageContent(input, telegramMaxMessageLength)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	for i, chunk := range chunks {
		if strings.TrimSpace(chunk) == "" {
			t.Fatalf("chunk %d is empty", i)
		}
		html := markdownToTelegramHTML(chunk)
		if runeLen(html) > telegramMaxMessageLength {
			t.Fatalf("chunk %d HTML length = %d, want <= %d", i, runeLen(html), telegramMaxMessageLength)
		}
	}
}

func TestSplitTelegramMessageContentEscapingExpansion(t *testing.T) {
	// '&' expands to '&amp;' in HTML, so this validates recursive splitting safety.
	input := strings.Repeat("&", 5000)
	chunks := splitTelegramMessageContent(input, telegramMaxMessageLength)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	for i, chunk := range chunks {
		html := markdownToTelegramHTML(chunk)
		if runeLen(html) > telegramMaxMessageLength {
			t.Fatalf("chunk %d escaped HTML length = %d, want <= %d", i, runeLen(html), telegramMaxMessageLength)
		}
	}
}
