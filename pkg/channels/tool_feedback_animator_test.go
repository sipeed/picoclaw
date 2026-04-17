package channels

import "testing"

func TestFormatAnimatedToolFeedbackContent(t *testing.T) {
	got := formatAnimatedToolFeedbackContent("🔧 `read_file`\nReading config file", "running..")
	want := "🔧 `read_filerunning..`\nReading config file"
	if got != want {
		t.Fatalf("formatAnimatedToolFeedbackContent() = %q, want %q", got, want)
	}
}

func TestInitialAnimatedToolFeedbackContent(t *testing.T) {
	got := InitialAnimatedToolFeedbackContent("🔧 `exec`\nRunning command")
	want := "🔧 `exec`\nRunning command"
	if got != want {
		t.Fatalf("InitialAnimatedToolFeedbackContent() = %q, want %q", got, want)
	}
}

func TestFormatAnimatedToolFeedbackContent_WithoutCodeSpan(t *testing.T) {
	got := formatAnimatedToolFeedbackContent("hello", "running..")
	want := "hellorunning.."
	if got != want {
		t.Fatalf("formatAnimatedToolFeedbackContent() without code span = %q, want %q", got, want)
	}
}

func TestToolFeedbackAnimator_RecordCurrentAndClear(t *testing.T) {
	animator := NewToolFeedbackAnimator(nil)
	animator.Record("chat-1", "msg-1", "🔧 `read_file`")

	msgID, ok := animator.Current("chat-1")
	if !ok || msgID != "msg-1" {
		t.Fatalf("Current() = (%q, %v), want (msg-1, true)", msgID, ok)
	}

	animator.Clear("chat-1")

	msgID, ok = animator.Current("chat-1")
	if ok || msgID != "" {
		t.Fatalf("Current() after Clear = (%q, %v), want (\"\", false)", msgID, ok)
	}
}
