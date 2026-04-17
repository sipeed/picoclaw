package utils

import "testing"

func TestFormatToolFeedbackMessage(t *testing.T) {
	got := FormatToolFeedbackMessage("read_file", "I will read README.md first to confirm the current project structure.")
	want := "\U0001f527 `read_file`\nI will read README.md first to confirm the current project structure."
	if got != want {
		t.Fatalf("FormatToolFeedbackMessage() = %q, want %q", got, want)
	}
}

func TestFormatToolFeedbackMessage_EmptyExplanationKeepsOnlyToolLine(t *testing.T) {
	got := FormatToolFeedbackMessage("read_file", "")
	want := "\U0001f527 `read_file`"
	if got != want {
		t.Fatalf("FormatToolFeedbackMessage() = %q, want %q", got, want)
	}
}

func TestFormatToolFeedbackMessage_EmptyToolNameOmitsToolLine(t *testing.T) {
	got := FormatToolFeedbackMessage("", "Continue drafting the final response.")
	want := "Continue drafting the final response."
	if got != want {
		t.Fatalf("FormatToolFeedbackMessage() = %q, want %q", got, want)
	}
}
