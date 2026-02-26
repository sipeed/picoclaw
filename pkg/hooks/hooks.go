// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package hooks

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
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

func cloneMapStringString(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneMapStringAny(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = cloneAny(v)
	}
	return dst
}

func cloneAny(v any) any {
	if v == nil {
		return nil
	}
	cloned := cloneReflectValue(reflect.ValueOf(v))
	if !cloned.IsValid() {
		return nil
	}
	return cloned.Interface()
}

func cloneReflectValue(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}

	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		out := reflect.New(v.Type().Elem())
		out.Elem().Set(cloneReflectValue(v.Elem()))
		return out
	case reflect.Interface:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		out := reflect.New(v.Type()).Elem()
		out.Set(cloneReflectValue(v.Elem()))
		return out
	case reflect.Map:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		out := reflect.MakeMapWithSize(v.Type(), v.Len())
		iter := v.MapRange()
		for iter.Next() {
			out.SetMapIndex(iter.Key(), cloneReflectValue(iter.Value()))
		}
		return out
	case reflect.Slice:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := range v.Len() {
			out.Index(i).Set(cloneReflectValue(v.Index(i)))
		}
		return out
	case reflect.Array:
		out := reflect.New(v.Type()).Elem()
		for i := range v.Len() {
			out.Index(i).Set(cloneReflectValue(v.Index(i)))
		}
		return out
	default:
		return v
	}
}

func cloneToolCall(tc providers.ToolCall) providers.ToolCall {
	out := tc
	out.Arguments = cloneMapStringAny(tc.Arguments)
	if tc.Function != nil {
		f := *tc.Function
		out.Function = &f
	}
	if tc.ExtraContent != nil {
		ec := *tc.ExtraContent
		if tc.ExtraContent.Google != nil {
			g := *tc.ExtraContent.Google
			ec.Google = &g
		}
		out.ExtraContent = &ec
	}
	return out
}

func cloneMessage(msg providers.Message) providers.Message {
	out := msg
	if msg.ToolCalls != nil {
		out.ToolCalls = make([]providers.ToolCall, len(msg.ToolCalls))
		for i := range msg.ToolCalls {
			out.ToolCalls[i] = cloneToolCall(msg.ToolCalls[i])
		}
	}
	if msg.SystemParts != nil {
		out.SystemParts = make([]providers.ContentBlock, len(msg.SystemParts))
		for i := range msg.SystemParts {
			part := msg.SystemParts[i]
			if part.CacheControl != nil {
				cc := *part.CacheControl
				part.CacheControl = &cc
			}
			out.SystemParts[i] = part
		}
	}
	return out
}

func cloneToolDefinition(td providers.ToolDefinition) providers.ToolDefinition {
	out := td
	out.Function = td.Function
	out.Function.Parameters = cloneMapStringAny(td.Function.Parameters)
	return out
}

func cloneVoidEvent[T any](event *T) *T {
	if event == nil {
		return nil
	}

	switch e := any(event).(type) {
	case *MessageReceivedEvent:
		c := *e
		if e.Media != nil {
			c.Media = append([]string(nil), e.Media...)
		}
		c.Metadata = cloneMapStringString(e.Metadata)
		return any(&c).(*T)
	case *AfterToolCallEvent:
		c := *e
		c.Args = cloneMapStringAny(e.Args)
		if e.Result != nil {
			r := *e.Result
			c.Result = &r
		}
		return any(&c).(*T)
	case *LLMInputEvent:
		c := *e
		if e.Messages != nil {
			c.Messages = make([]providers.Message, len(e.Messages))
			for i := range e.Messages {
				c.Messages[i] = cloneMessage(e.Messages[i])
			}
		}
		if e.Tools != nil {
			c.Tools = make([]providers.ToolDefinition, len(e.Tools))
			for i := range e.Tools {
				c.Tools[i] = cloneToolDefinition(e.Tools[i])
			}
		}
		return any(&c).(*T)
	case *LLMOutputEvent:
		c := *e
		if e.ToolCalls != nil {
			c.ToolCalls = make([]providers.ToolCall, len(e.ToolCalls))
			for i := range e.ToolCalls {
				c.ToolCalls[i] = cloneToolCall(e.ToolCalls[i])
			}
		}
		return any(&c).(*T)
	case *SessionEvent:
		c := *e
		return any(&c).(*T)
	default:
		c := *event
		return &c
	}
}

// triggerVoid runs all handlers concurrently and waits for completion.
// Each handler receives a cloned event to avoid shared-state mutation races.
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
			eventCopy := cloneVoidEvent(event)
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
			if err := reg.Handler(ctx, eventCopy); err != nil {
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
func triggerModifying[T any](
	ctx context.Context,
	hooks []HookRegistration[T],
	event *T,
	hookName string,
	cancelCheck func(*T) bool,
) {
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
// Handler mutations are isolated per hook invocation and are not propagated.
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
// Handler mutations are isolated per hook invocation and are not propagated.
func (r *HookRegistry) TriggerAfterToolCall(ctx context.Context, event *AfterToolCallEvent) {
	r.mu.RLock()
	hooks := r.afterToolCall
	r.mu.RUnlock()
	triggerVoid(ctx, hooks, event, "after_tool_call")
}

// TriggerLLMInput fires all llm_input handlers concurrently.
// Handler mutations are isolated per hook invocation and are not propagated.
func (r *HookRegistry) TriggerLLMInput(ctx context.Context, event *LLMInputEvent) {
	r.mu.RLock()
	hooks := r.llmInput
	r.mu.RUnlock()
	triggerVoid(ctx, hooks, event, "llm_input")
}

// TriggerLLMOutput fires all llm_output handlers concurrently.
// Handler mutations are isolated per hook invocation and are not propagated.
func (r *HookRegistry) TriggerLLMOutput(ctx context.Context, event *LLMOutputEvent) {
	r.mu.RLock()
	hooks := r.llmOutput
	r.mu.RUnlock()
	triggerVoid(ctx, hooks, event, "llm_output")
}

// TriggerSessionStart fires all session_start handlers concurrently.
// Handler mutations are isolated per hook invocation and are not propagated.
func (r *HookRegistry) TriggerSessionStart(ctx context.Context, event *SessionEvent) {
	r.mu.RLock()
	hooks := r.sessionStart
	r.mu.RUnlock()
	triggerVoid(ctx, hooks, event, "session_start")
}

// TriggerSessionEnd fires all session_end handlers concurrently.
// Handler mutations are isolated per hook invocation and are not propagated.
func (r *HookRegistry) TriggerSessionEnd(ctx context.Context, event *SessionEvent) {
	r.mu.RLock()
	hooks := r.sessionEnd
	r.mu.RUnlock()
	triggerVoid(ctx, hooks, event, "session_end")
}
