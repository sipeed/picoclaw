package channels

import (
	"context"
	"errors"
	"testing"
)

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

func TestToolFeedbackAnimator_TakeStopsTrackingAndReturnsState(t *testing.T) {
	animator := NewToolFeedbackAnimator(nil)
	animator.Record("chat-1", "msg-1", "🔧 `read_file`\nChecking config")

	msgID, baseContent, ok := animator.Take("chat-1")
	if !ok {
		t.Fatal("Take() = not found, want tracked message")
	}
	if msgID != "msg-1" {
		t.Fatalf("Take() msgID = %q, want msg-1", msgID)
	}
	if baseContent != "🔧 `read_file`\nChecking config" {
		t.Fatalf("Take() baseContent = %q", baseContent)
	}
	if _, ok := animator.Current("chat-1"); ok {
		t.Fatal("expected tracked message to be removed after Take()")
	}
}

func TestToolFeedbackAnimator_UpdateStopsTrackingBeforeEdit(t *testing.T) {
	var animator *ToolFeedbackAnimator
	animator = NewToolFeedbackAnimator(func(_ context.Context, chatID, messageID, content string) error {
		if _, ok := animator.Current(chatID); ok {
			t.Fatal("expected tracked tool feedback to be stopped before edit")
		}
		if messageID != "msg-1" {
			t.Fatalf("messageID = %q, want msg-1", messageID)
		}
		want := "Working...\n• tool: `read_file`\n• tool: `write_file`"
		if content != want {
			t.Fatalf("content = %q, want updated animated content", content)
		}
		return nil
	})
	defer animator.StopAll()

	animator.Record("chat-1", "msg-1", "Working...\n• tool: `read_file`")

	msgID, handled, err := animator.Update(context.Background(), "chat-1", "Working...\n• tool: `write_file`")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !handled {
		t.Fatal("Update() handled = false, want true")
	}
	if msgID != "msg-1" {
		t.Fatalf("Update() msgID = %q, want msg-1", msgID)
	}
}

func TestToolFeedbackAnimator_UpdateRawFeedbackReplacesContent(t *testing.T) {
	var animator *ToolFeedbackAnimator
	animator = NewToolFeedbackAnimator(func(_ context.Context, chatID, messageID, content string) error {
		if _, ok := animator.Current(chatID); ok {
			t.Fatal("expected tracked tool feedback to be stopped before edit")
		}
		if messageID != "msg-1" {
			t.Fatalf("messageID = %q, want msg-1", messageID)
		}
		want := "🔧 `write_file`\nWriting config"
		if content != want {
			t.Fatalf("content = %q, want replacement content", content)
		}
		return nil
	})
	defer animator.StopAll()

	animator.Record("chat-1", "msg-1", "🔧 `read_file`\nReading config")

	msgID, handled, err := animator.Update(context.Background(), "chat-1", "🔧 `write_file`\nWriting config")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !handled {
		t.Fatal("Update() handled = false, want true")
	}
	if msgID != "msg-1" {
		t.Fatalf("Update() msgID = %q, want msg-1", msgID)
	}
}

func TestToolFeedbackAnimationIntervalForWorkingSummary(t *testing.T) {
	got := toolFeedbackAnimationIntervalFor("Working...\n• tool: `read_file`")
	if got != workingSummaryToolFeedbackAnimationInterval {
		t.Fatalf("toolFeedbackAnimationIntervalFor() = %v, want %v", got, workingSummaryToolFeedbackAnimationInterval)
	}
}

func TestToolFeedbackAnimationIntervalForRawFeedback(t *testing.T) {
	got := toolFeedbackAnimationIntervalFor("🔧 `read_file`\nReading config")
	if got != toolFeedbackAnimationInterval {
		t.Fatalf("toolFeedbackAnimationIntervalFor() = %v, want %v", got, toolFeedbackAnimationInterval)
	}
}

func TestToolFeedbackAnimator_UpdateFailureRestoresTracking(t *testing.T) {
	editErr := errors.New("edit failed")
	animator := NewToolFeedbackAnimator(func(context.Context, string, string, string) error {
		return editErr
	})
	defer animator.StopAll()

	animator.Record("chat-1", "msg-1", "Working...\n• tool: `read_file`")

	msgID, handled, err := animator.Update(context.Background(), "chat-1", "Working...\n• tool: `write_file`")
	if !handled {
		t.Fatal("Update() handled = false, want true")
	}
	if !errors.Is(err, editErr) {
		t.Fatalf("Update() error = %v, want editErr", err)
	}
	if msgID != "" {
		t.Fatalf("Update() msgID = %q, want empty on failed edit", msgID)
	}
	if currentID, ok := animator.Current("chat-1"); !ok || currentID != "msg-1" {
		t.Fatalf("Current() after failed Update = (%q, %v), want (msg-1, true)", currentID, ok)
	}
}

func TestMergeToolFeedbackContent_PreservesNamedWorkingSummaryHeader(t *testing.T) {
	got := mergeToolFeedbackContent(
		"Deep Research working...\n• tool: `read_file`",
		"Deep Research working...\n• tool: `web_fetch`",
	)
	want := "Deep Research working...\n• tool: `read_file`\n• tool: `web_fetch`"
	if got != want {
		t.Fatalf("mergeToolFeedbackContent() = %q, want %q", got, want)
	}
}

func TestIsWorkingSummaryToolFeedback_AcceptsNamedHeader(t *testing.T) {
	if !isWorkingSummaryToolFeedback("Deep Research working...\n• tool: `read_file`") {
		t.Fatal("expected named working summary to be recognized")
	}
}
