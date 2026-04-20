package utils

import "testing"

func TestFormatToolFeedbackMessage(t *testing.T) {
	got := FormatToolFeedbackMessage("read_file", "{\"path\":\"README.md\"}")
	want := "\U0001f527 `read_file`\n```\n{\"path\":\"README.md\"}\n```"
	if got != want {
		t.Fatalf("FormatToolFeedbackMessage() = %q, want %q", got, want)
	}
}
