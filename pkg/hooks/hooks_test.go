// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package hooks

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewHookRegistry(t *testing.T) {
	r := NewHookRegistry()
	ctx := context.Background()

	// Triggering all hooks on an empty registry should not panic.
	r.TriggerMessageReceived(ctx, &MessageReceivedEvent{Content: "hello"})
	r.TriggerMessageSending(ctx, &MessageSendingEvent{Content: "hello"})
	r.TriggerBeforeToolCall(ctx, &BeforeToolCallEvent{ToolName: "t"})
	r.TriggerAfterToolCall(ctx, &AfterToolCallEvent{ToolName: "t"})
	r.TriggerLLMInput(ctx, &LLMInputEvent{AgentID: "a"})
	r.TriggerLLMOutput(ctx, &LLMOutputEvent{AgentID: "a"})
	r.TriggerSessionStart(ctx, &SessionEvent{AgentID: "a"})
	r.TriggerSessionEnd(ctx, &SessionEvent{AgentID: "a"})
}

func TestVoidHookExecution(t *testing.T) {
	r := NewHookRegistry()
	ctx := context.Background()

	var called atomic.Bool
	r.OnMessageReceived("test", 0, func(_ context.Context, e *MessageReceivedEvent) error {
		called.Store(true)
		if e.Content != "ping" {
			t.Errorf("Expected content 'ping', got '%s'", e.Content)
		}
		return nil
	})

	r.TriggerMessageReceived(ctx, &MessageReceivedEvent{Content: "ping"})

	if !called.Load() {
		t.Error("Expected handler to be called")
	}
}

func TestVoidHooksConcurrent(t *testing.T) {
	r := NewHookRegistry()
	ctx := context.Background()

	var count atomic.Int32
	started := make(chan struct{}, 5)
	release := make(chan struct{})
	done := make(chan struct{})

	for i := range 5 {
		r.OnMessageReceived("hook-"+string(rune('A'+i)), i, func(_ context.Context, _ *MessageReceivedEvent) error {
			started <- struct{}{}
			<-release
			count.Add(1)
			return nil
		})
	}

	go func() {
		r.TriggerMessageReceived(ctx, &MessageReceivedEvent{Content: "test"})
		close(done)
	}()

	// All 5 handlers must reach the barrier concurrently.
	for i := range 5 {
		select {
		case <-started:
		case <-time.After(1 * time.Second):
			t.Fatalf("timeout waiting for handler %d to start", i+1)
		}
	}

	// Release all handlers.
	close(release)

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for handlers to complete")
	}

	if count.Load() != 5 {
		t.Errorf("Expected 5 handlers called, got %d", count.Load())
	}
}

func TestModifyingHookPriority(t *testing.T) {
	r := NewHookRegistry()
	ctx := context.Background()

	var mu sync.Mutex
	var order []string

	// Register in reverse priority order to verify sorting.
	r.OnMessageSending("third", 30, func(_ context.Context, _ *MessageSendingEvent) error {
		mu.Lock()
		order = append(order, "third")
		mu.Unlock()
		return nil
	})
	r.OnMessageSending("first", 10, func(_ context.Context, _ *MessageSendingEvent) error {
		mu.Lock()
		order = append(order, "first")
		mu.Unlock()
		return nil
	})
	r.OnMessageSending("second", 20, func(_ context.Context, _ *MessageSendingEvent) error {
		mu.Lock()
		order = append(order, "second")
		mu.Unlock()
		return nil
	})

	r.TriggerMessageSending(ctx, &MessageSendingEvent{Content: "hi"})

	if len(order) != 3 {
		t.Fatalf("Expected 3 handlers, got %d", len(order))
	}
	if order[0] != "first" || order[1] != "second" || order[2] != "third" {
		t.Errorf("Expected [first second third], got %v", order)
	}
}

func TestModifyingHookCancel(t *testing.T) {
	r := NewHookRegistry()
	ctx := context.Background()

	var secondCalled bool

	r.OnMessageSending("canceler", 10, func(_ context.Context, e *MessageSendingEvent) error {
		e.Cancel = true
		e.CancelReason = "blocked"
		return nil
	})
	r.OnMessageSending("after-cancel", 20, func(_ context.Context, _ *MessageSendingEvent) error {
		secondCalled = true
		return nil
	})

	event := &MessageSendingEvent{Content: "hi"}
	r.TriggerMessageSending(ctx, event)

	if !event.Cancel {
		t.Error("Expected Cancel to be true")
	}
	if secondCalled {
		t.Error("Expected second handler NOT to be called after cancel")
	}
}

func TestBeforeToolCallModification(t *testing.T) {
	r := NewHookRegistry()
	ctx := context.Background()

	r.OnBeforeToolCall("modifier", 10, func(_ context.Context, e *BeforeToolCallEvent) error {
		e.Args["injected"] = "value"
		return nil
	})

	event := &BeforeToolCallEvent{
		ToolName: "search",
		Args:     map[string]any{"query": "test"},
	}
	r.TriggerBeforeToolCall(ctx, event)

	if event.Args["injected"] != "value" {
		t.Error("Expected injected arg to persist")
	}
	if event.Args["query"] != "test" {
		t.Error("Expected original arg to remain")
	}
}

func TestMessageSendingFilter(t *testing.T) {
	r := NewHookRegistry()
	ctx := context.Background()

	r.OnMessageSending("rewriter", 10, func(_ context.Context, e *MessageSendingEvent) error {
		e.Content = "[filtered] " + e.Content
		return nil
	})

	event := &MessageSendingEvent{Content: "hello world"}
	r.TriggerMessageSending(ctx, event)

	if event.Content != "[filtered] hello world" {
		t.Errorf("Expected '[filtered] hello world', got '%s'", event.Content)
	}
}

func TestZeroCostWhenEmpty(t *testing.T) {
	r := NewHookRegistry()
	ctx := context.Background()

	// This is primarily a safety/smoke test — no panics, no allocations of note.
	for range 100 {
		r.TriggerMessageReceived(ctx, &MessageReceivedEvent{})
		r.TriggerMessageSending(ctx, &MessageSendingEvent{})
		r.TriggerBeforeToolCall(ctx, &BeforeToolCallEvent{})
		r.TriggerAfterToolCall(ctx, &AfterToolCallEvent{})
		r.TriggerLLMInput(ctx, &LLMInputEvent{})
		r.TriggerLLMOutput(ctx, &LLMOutputEvent{})
		r.TriggerSessionStart(ctx, &SessionEvent{})
		r.TriggerSessionEnd(ctx, &SessionEvent{})
	}
}

func TestLLMInputOutput(t *testing.T) {
	r := NewHookRegistry()
	ctx := context.Background()

	var inputCalled, outputCalled atomic.Bool

	r.OnLLMInput("input-hook", 0, func(_ context.Context, e *LLMInputEvent) error {
		if e.Model != "gpt-4" {
			t.Errorf("Expected model 'gpt-4', got '%s'", e.Model)
		}
		inputCalled.Store(true)
		return nil
	})

	r.OnLLMOutput("output-hook", 0, func(_ context.Context, e *LLMOutputEvent) error {
		if e.Content != "response" {
			t.Errorf("Expected content 'response', got '%s'", e.Content)
		}
		outputCalled.Store(true)
		return nil
	})

	r.TriggerLLMInput(ctx, &LLMInputEvent{AgentID: "a1", Model: "gpt-4", Iteration: 1})
	r.TriggerLLMOutput(ctx, &LLMOutputEvent{AgentID: "a1", Model: "gpt-4", Content: "response", Iteration: 1})

	if !inputCalled.Load() {
		t.Error("Expected LLM input hook to be called")
	}
	if !outputCalled.Load() {
		t.Error("Expected LLM output hook to be called")
	}
}

func TestSessionStartEnd(t *testing.T) {
	r := NewHookRegistry()
	ctx := context.Background()

	var startCalled, endCalled atomic.Bool

	r.OnSessionStart("start-hook", 0, func(_ context.Context, e *SessionEvent) error {
		if e.SessionKey != "sess-1" {
			t.Errorf("Expected session key 'sess-1', got '%s'", e.SessionKey)
		}
		startCalled.Store(true)
		return nil
	})

	r.OnSessionEnd("end-hook", 0, func(_ context.Context, e *SessionEvent) error {
		if e.SessionKey != "sess-1" {
			t.Errorf("Expected session key 'sess-1', got '%s'", e.SessionKey)
		}
		endCalled.Store(true)
		return nil
	})

	event := &SessionEvent{AgentID: "a1", SessionKey: "sess-1", Channel: "test", ChatID: "c1"}
	r.TriggerSessionStart(ctx, event)
	r.TriggerSessionEnd(ctx, event)

	if !startCalled.Load() {
		t.Error("Expected session start hook to be called")
	}
	if !endCalled.Load() {
		t.Error("Expected session end hook to be called")
	}
}

func TestConcurrentRegistrationAndTrigger(t *testing.T) {
	r := NewHookRegistry()
	ctx := context.Background()

	var wg sync.WaitGroup

	// Goroutines registering hooks.
	for i := range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.OnMessageReceived("reg-hook", i, func(_ context.Context, _ *MessageReceivedEvent) error {
				return nil
			})
		}()
	}

	// Goroutines triggering hooks concurrently.
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.TriggerMessageReceived(ctx, &MessageReceivedEvent{Content: "race"})
		}()
	}

	wg.Wait()
}

func TestInsertSorted(t *testing.T) {
	r := NewHookRegistry()
	ctx := context.Background()

	var order []int

	// Register with priorities: 50, 10, 30, 20, 40
	priorities := []int{50, 10, 30, 20, 40}
	for _, p := range priorities {
		r.OnBeforeToolCall("p-"+string(rune('0'+p)), p, func(_ context.Context, _ *BeforeToolCallEvent) error {
			order = append(order, p)
			return nil
		})
	}

	r.TriggerBeforeToolCall(ctx, &BeforeToolCallEvent{ToolName: "test", Args: map[string]any{}})

	expected := []int{10, 20, 30, 40, 50}
	if len(order) != len(expected) {
		t.Fatalf("Expected %d handlers, got %d", len(expected), len(order))
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("Position %d: expected priority %d, got %d", i, v, order[i])
		}
	}
}

func TestAfterToolCallExecution(t *testing.T) {
	r := NewHookRegistry()
	ctx := context.Background()

	var called bool
	var capturedName string
	r.OnAfterToolCall("logger", 0, func(_ context.Context, event *AfterToolCallEvent) error {
		called = true
		capturedName = event.ToolName
		return nil
	})

	r.TriggerAfterToolCall(ctx, &AfterToolCallEvent{
		ToolName: "shell",
		Args:     map[string]any{"cmd": "ls"},
		Channel:  "telegram",
		ChatID:   "123",
	})

	if !called {
		t.Error("Expected after_tool_call handler to be called")
	}
	if capturedName != "shell" {
		t.Errorf("Expected ToolName 'shell', got '%s'", capturedName)
	}
}

func TestHandlerErrorsSwallowed(t *testing.T) {
	r := NewHookRegistry()
	ctx := context.Background()

	// Test void hooks: error in one handler doesn't prevent others from running
	var secondCalled bool
	r.OnMessageReceived("erroring", 10, func(_ context.Context, _ *MessageReceivedEvent) error {
		return fmt.Errorf("handler error")
	})
	r.OnMessageReceived("observer", 20, func(_ context.Context, _ *MessageReceivedEvent) error {
		secondCalled = true
		return nil
	})

	r.TriggerMessageReceived(ctx, &MessageReceivedEvent{Content: "test"})
	if !secondCalled {
		t.Error("Expected second void handler to run despite first handler's error")
	}

	// Test modifying hooks: error doesn't stop chain (only Cancel does)
	var modifySecondCalled bool
	r.OnMessageSending("erroring", 10, func(_ context.Context, _ *MessageSendingEvent) error {
		return fmt.Errorf("handler error")
	})
	r.OnMessageSending("modifier", 20, func(_ context.Context, _ *MessageSendingEvent) error {
		modifySecondCalled = true
		return nil
	})

	r.TriggerMessageSending(ctx, &MessageSendingEvent{Content: "test"})
	if !modifySecondCalled {
		t.Error("Expected second modifying handler to run despite first handler's error")
	}
}

func TestPanicRecovery(t *testing.T) {
	r := NewHookRegistry()
	ctx := context.Background()

	// Void hook: panic in one handler shouldn't crash, other handlers should still run
	var safeHandlerCalled bool
	r.OnLLMInput("panicker", 10, func(_ context.Context, _ *LLMInputEvent) error {
		panic("boom")
	})
	r.OnLLMInput("safe", 10, func(_ context.Context, _ *LLMInputEvent) error {
		safeHandlerCalled = true
		return nil
	})

	// Should not panic
	r.TriggerLLMInput(ctx, &LLMInputEvent{AgentID: "test"})
	if !safeHandlerCalled {
		t.Error("Expected safe handler to run despite panicking sibling")
	}

	// Modifying hook: panic in handler shouldn't crash
	r.OnBeforeToolCall("panicker", 10, func(_ context.Context, _ *BeforeToolCallEvent) error {
		panic("boom")
	})

	// Should not panic
	r.TriggerBeforeToolCall(ctx, &BeforeToolCallEvent{ToolName: "test"})
}
