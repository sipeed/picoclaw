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

	"github.com/sipeed/picoclaw/pkg/logger"
)

// HookHandler is the callback signature for all hooks.
type HookHandler[T any] func(ctx context.Context, event *T) error

// HookRegistration tracks a handler with its priority and name.
type HookRegistration[T any] struct {
	Handler  HookHandler[T]
	Priority int // Lower = runs first
	Name     string
}

// HookRegistry manages all lifecycle hooks.
type HookRegistry struct {
	messageReceived []HookRegistration[MessageReceivedEvent]
	messageSending  []HookRegistration[MessageSendingEvent]
	beforeToolCall  []HookRegistration[BeforeToolCallEvent]
	afterToolCall   []HookRegistration[AfterToolCallEvent]
	llmInput        []HookRegistration[LLMInputEvent]
	llmOutput       []HookRegistration[LLMOutputEvent]
	sessionStart    []HookRegistration[SessionEvent]
	sessionEnd      []HookRegistration[SessionEvent]
	mu              sync.RWMutex
}

// NewHookRegistry creates an empty hook registry.
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{}
}

// insertSorted inserts a registration into a new slice sorted by priority.
// Always allocates a new backing array so concurrent readers of the old slice are safe.
func insertSorted[T any](slice []HookRegistration[T], reg HookRegistration[T]) []HookRegistration[T] {
	i := 0
	for i < len(slice) && slice[i].Priority <= reg.Priority {
		i++
	}
	result := make([]HookRegistration[T], len(slice)+1)
	copy(result, slice[:i])
	result[i] = reg
	copy(result[i+1:], slice[i:])
	return result
}

// Registration methods

func (r *HookRegistry) OnMessageReceived(name string, priority int, handler HookHandler[MessageReceivedEvent]) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messageReceived = insertSorted(r.messageReceived, HookRegistration[MessageReceivedEvent]{
		Handler: handler, Priority: priority, Name: name,
	})
}

func (r *HookRegistry) OnMessageSending(name string, priority int, handler HookHandler[MessageSendingEvent]) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messageSending = insertSorted(r.messageSending, HookRegistration[MessageSendingEvent]{
		Handler: handler, Priority: priority, Name: name,
	})
}

func (r *HookRegistry) OnBeforeToolCall(name string, priority int, handler HookHandler[BeforeToolCallEvent]) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.beforeToolCall = insertSorted(r.beforeToolCall, HookRegistration[BeforeToolCallEvent]{
		Handler: handler, Priority: priority, Name: name,
	})
}

func (r *HookRegistry) OnAfterToolCall(name string, priority int, handler HookHandler[AfterToolCallEvent]) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.afterToolCall = insertSorted(r.afterToolCall, HookRegistration[AfterToolCallEvent]{
		Handler: handler, Priority: priority, Name: name,
	})
}

func (r *HookRegistry) OnLLMInput(name string, priority int, handler HookHandler[LLMInputEvent]) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llmInput = insertSorted(r.llmInput, HookRegistration[LLMInputEvent]{
		Handler: handler, Priority: priority, Name: name,
	})
}

func (r *HookRegistry) OnLLMOutput(name string, priority int, handler HookHandler[LLMOutputEvent]) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llmOutput = insertSorted(r.llmOutput, HookRegistration[LLMOutputEvent]{
		Handler: handler, Priority: priority, Name: name,
	})
}

func (r *HookRegistry) OnSessionStart(name string, priority int, handler HookHandler[SessionEvent]) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionStart = insertSorted(r.sessionStart, HookRegistration[SessionEvent]{
		Handler: handler, Priority: priority, Name: name,
	})
}

func (r *HookRegistry) OnSessionEnd(name string, priority int, handler HookHandler[SessionEvent]) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionEnd = insertSorted(r.sessionEnd, HookRegistration[SessionEvent]{
		Handler: handler, Priority: priority, Name: name,
	})
}

// Trigger methods — void hooks

// triggerVoid runs all handlers concurrently and waits for completion.
// Handlers MUST NOT mutate the event — it is shared across goroutines.
// Errors are logged but do not propagate to the caller.
func triggerVoid[T any](ctx context.Context, hooks []HookRegistration[T], event *T, hookName string) {
	if len(hooks) == 0 {
		return
	}
	var wg sync.WaitGroup
	for _, h := range hooks {
		wg.Add(1)
		go func(reg HookRegistration[T]) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					logger.ErrorCF("hooks", "Hook panic",
						map[string]any{
							"hook":    hookName,
							"handler": reg.Name,
							"panic":   fmt.Sprintf("%v", r),
						})
				}
			}()
			if err := reg.Handler(ctx, event); err != nil {
				logger.WarnCF("hooks", "Hook error",
					map[string]any{
						"hook":    hookName,
						"handler": reg.Name,
						"error":   err.Error(),
					})
			}
		}(h)
	}
	wg.Wait()
}

// triggerModifying runs handlers sequentially by priority, stopping if Cancel is set.
// The cancelCheck function inspects the event to determine if Cancel was set.
func triggerModifying[T any](ctx context.Context, hooks []HookRegistration[T], event *T, hookName string, cancelCheck func(*T) bool) {
	if len(hooks) == 0 {
		return
	}
	for _, h := range hooks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					logger.ErrorCF("hooks", "Hook panic",
						map[string]any{
							"hook":    hookName,
							"handler": h.Name,
							"panic":   fmt.Sprintf("%v", r),
						})
				}
			}()
			if err := h.Handler(ctx, event); err != nil {
				logger.WarnCF("hooks", "Hook error",
					map[string]any{
						"hook":    hookName,
						"handler": h.Name,
						"error":   err.Error(),
					})
			}
		}()
		if cancelCheck(event) {
			logger.InfoCF("hooks", "Hook canceled operation",
				map[string]any{
					"hook":    hookName,
					"handler": h.Name,
				})
			return
		}
	}
}

// TriggerMessageReceived fires all message_received handlers concurrently.
// Handlers must not mutate the event.
func (r *HookRegistry) TriggerMessageReceived(ctx context.Context, event *MessageReceivedEvent) {
	r.mu.RLock()
	hooks := r.messageReceived
	r.mu.RUnlock()
	triggerVoid(ctx, hooks, event, "message_received")
}

func (r *HookRegistry) TriggerMessageSending(ctx context.Context, event *MessageSendingEvent) {
	r.mu.RLock()
	hooks := r.messageSending
	r.mu.RUnlock()
	triggerModifying(ctx, hooks, event, "message_sending", func(e *MessageSendingEvent) bool {
		return e.Cancel
	})
}

func (r *HookRegistry) TriggerBeforeToolCall(ctx context.Context, event *BeforeToolCallEvent) {
	r.mu.RLock()
	hooks := r.beforeToolCall
	r.mu.RUnlock()
	triggerModifying(ctx, hooks, event, "before_tool_call", func(e *BeforeToolCallEvent) bool {
		return e.Cancel
	})
}

// TriggerAfterToolCall fires all after_tool_call handlers concurrently.
// Handlers must not mutate the event.
func (r *HookRegistry) TriggerAfterToolCall(ctx context.Context, event *AfterToolCallEvent) {
	r.mu.RLock()
	hooks := r.afterToolCall
	r.mu.RUnlock()
	triggerVoid(ctx, hooks, event, "after_tool_call")
}

// TriggerLLMInput fires all llm_input handlers concurrently.
// Handlers must not mutate the event.
func (r *HookRegistry) TriggerLLMInput(ctx context.Context, event *LLMInputEvent) {
	r.mu.RLock()
	hooks := r.llmInput
	r.mu.RUnlock()
	triggerVoid(ctx, hooks, event, "llm_input")
}

// TriggerLLMOutput fires all llm_output handlers concurrently.
// Handlers must not mutate the event.
func (r *HookRegistry) TriggerLLMOutput(ctx context.Context, event *LLMOutputEvent) {
	r.mu.RLock()
	hooks := r.llmOutput
	r.mu.RUnlock()
	triggerVoid(ctx, hooks, event, "llm_output")
}

// TriggerSessionStart fires all session_start handlers concurrently.
// Handlers must not mutate the event.
func (r *HookRegistry) TriggerSessionStart(ctx context.Context, event *SessionEvent) {
	r.mu.RLock()
	hooks := r.sessionStart
	r.mu.RUnlock()
	triggerVoid(ctx, hooks, event, "session_start")
}

// TriggerSessionEnd fires all session_end handlers concurrently.
// Handlers must not mutate the event.
func (r *HookRegistry) TriggerSessionEnd(ctx context.Context, event *SessionEvent) {
	r.mu.RLock()
	hooks := r.sessionEnd
	r.mu.RUnlock()
	triggerVoid(ctx, hooks, event, "session_end")
}
