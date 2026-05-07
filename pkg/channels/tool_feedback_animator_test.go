package channels

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
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
		if content != "🔧 `write_file`\nUpdating config" {
			t.Fatalf("content = %q, want updated animated content", content)
		}
		return nil
	})
	defer animator.StopAll()

	animator.Record("chat-1", "msg-1", "🔧 `read_file`\nChecking config")

	msgID, handled, err := animator.Update(context.Background(), "chat-1", "🔧 `write_file`\nUpdating config")
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

func TestToolFeedbackAnimator_UpdateFailureRestoresTracking(t *testing.T) {
	editErr := errors.New("edit failed")
	animator := NewToolFeedbackAnimator(func(context.Context, string, string, string) error {
		return editErr
	})
	defer animator.StopAll()

	animator.Record("chat-1", "msg-1", "🔧 `read_file`\nChecking config")

	msgID, handled, err := animator.Update(context.Background(), "chat-1", "🔧 `write_file`\nUpdating config")
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

func TestToolFeedbackAnimator_UpdateDoesNotDebounceByDefault(t *testing.T) {
	editCalls := 0
	animator := NewToolFeedbackAnimator(func(context.Context, string, string, string) error {
		editCalls++
		return nil
	})
	defer animator.StopAll()

	animator.Record("chat-1", "msg-1", "🔧 `read_file`")
	animator.markEdit("chat-1")

	msgID, handled, err := animator.Update(context.Background(), "chat-1", "🔧 `write_file`")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !handled || msgID != "msg-1" {
		t.Fatalf("Update() = (%q, %v), want (msg-1, true)", msgID, handled)
	}
	if editCalls != 1 {
		t.Fatalf("edit calls = %d, want 1 when min edit interval is unset", editCalls)
	}
}

func TestToolFeedbackAnimator_UpdateDebouncesRecentEditWhenConfigured(t *testing.T) {
	editCalls := 0
	animator := NewToolFeedbackAnimator(func(context.Context, string, string, string) error {
		editCalls++
		return nil
	})
	defer animator.StopAll()
	animator.Configure(ToolFeedbackAnimatorConfig{MinEditInterval: 10 * time.Second})

	animator.Record("chat-1", "msg-1", "🔧 `read_file`")
	animator.markEdit("chat-1")

	msgID, handled, err := animator.Update(context.Background(), "chat-1", "🔧 `write_file`")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !handled || msgID != "msg-1" {
		t.Fatalf("Update() = (%q, %v), want (msg-1, true)", msgID, handled)
	}
	if editCalls != 0 {
		t.Fatalf("edit calls = %d, want 0 while debounced", editCalls)
	}
	_, baseContent, ok := animator.Take("chat-1")
	if !ok {
		t.Fatal("expected debounced update to keep tracking")
	}
	if baseContent != "🔧 `write_file`" {
		t.Fatalf("tracked content = %q, want latest content", baseContent)
	}
}

func TestToolFeedbackAnimator_RetryAfterDelayParsesTelegramErrorWhenConfigured(t *testing.T) {
	animator := NewToolFeedbackAnimator(nil)
	animator.Configure(ToolFeedbackAnimatorConfig{MinEditInterval: 10 * time.Second})

	err := fmt.Errorf("telego: editMessageText: api: 429 %q: %w", "Too Many Requests: retry after 14", ErrRateLimit)
	delay, ok := animator.retryAfterDelay(err)
	if !ok {
		t.Fatal("retryAfterDelay() ok = false, want true")
	}
	if delay != 14*time.Second {
		t.Fatalf("retryAfterDelay() = %v, want 14s", delay)
	}
}

func TestToolFeedbackAnimator_RetryAfterDisabledByDefault(t *testing.T) {
	animator := NewToolFeedbackAnimator(nil)
	err := fmt.Errorf("telego: editMessageText: api: 429 %q: %w", "Too Many Requests: retry after 14", ErrRateLimit)
	if delay, ok := animator.retryAfterDelay(err); ok {
		t.Fatalf("retryAfterDelay() = (%v, true), want disabled by default", delay)
	}
}
