package agent

import (
	"context"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
)

const (
	defaultHookObserverTimeout    = 500 * time.Millisecond
	defaultHookInterceptorTimeout = 5 * time.Second
	defaultHookApprovalTimeout    = 60 * time.Second
	hookObserverBufferSize        = 64
)

type HookAction string

const (
	HookActionContinue  HookAction = "continue"
	HookActionModify    HookAction = "modify"
	HookActionRespond   HookAction = "respond" // Return result directly, skip tool execution. SECURITY: This bypasses ApproveTool checks, allowing hooks to return results for any tool (including sensitive ones like bash) without approval. Use with caution.
	HookActionDenyTool  HookAction = "deny_tool"
	HookActionAbortTurn HookAction = "abort_turn"
	HookActionHardAbort HookAction = "hard_abort"
)

type HookDecision struct {
	Action HookAction `json:"action"`
	Reason string     `json:"reason,omitempty"`
}

func (d HookDecision) normalizedAction() HookAction {
	if d.Action == "" {
		return HookActionContinue
	}
	return d.Action
}

type ApprovalDecision struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason,omitempty"`
}

type HookSource uint8

const (
	HookSourceInProcess HookSource = iota
	HookSourceProcess
)

type HookRegistration struct {
	Name     string
	Priority int
	Source   HookSource
	Hook     any
}

func NamedHook(name string, hook any) HookRegistration {
	return HookRegistration{
		Name:   name,
		Source: HookSourceInProcess,
		Hook:   hook,
	}
}

type EventObserver interface {
	OnEvent(ctx context.Context, evt Event) error
}

type LLMInterceptor interface {
	BeforeLLM(ctx context.Context, req *LLMHookRequest) (*LLMHookRequest, HookDecision, error)
	AfterLLM(ctx context.Context, resp *LLMHookResponse) (*LLMHookResponse, HookDecision, error)
}

type ToolInterceptor interface {
	BeforeTool(ctx context.Context, call *ToolCallHookRequest) (*ToolCallHookRequest, HookDecision, error)
	AfterTool(ctx context.Context, result *ToolResultHookResponse) (*ToolResultHookResponse, HookDecision, error)
}

type ToolApprover interface {
	ApproveTool(ctx context.Context, req *ToolApprovalRequest) (ApprovalDecision, error)
}

type LLMHookRequest struct {
	Meta             EventMeta                  `json:"meta"`
	Model            string                     `json:"model"`
	Messages         []providers.Message        `json:"messages,omitempty"`
	Tools            []providers.ToolDefinition `json:"tools,omitempty"`
	Options          map[string]any             `json:"options,omitempty"`
	Channel          string                     `json:"channel,omitempty"`
	ChatID           string                     `json:"chat_id,omitempty"`
	GracefulTerminal bool                       `json:"graceful_terminal,omitempty"`
}

func (r *LLMHookRequest) Clone() *LLMHookRequest {
	if r == nil {
		return nil
	}
	cloned := *r
	cloned.Messages = cloneProviderMessages(r.Messages)
	cloned.Tools = cloneToolDefinitions(r.Tools)
	cloned.Options = cloneStringAnyMap(r.Options)
	return &cloned
}

type LLMHookResponse struct {
	Meta     EventMeta              `json:"meta"`
	Model    string                 `json:"model"`
	Response *providers.LLMResponse `json:"response,omitempty"`
	Channel  string                 `json:"channel,omitempty"`
	ChatID   string                 `json:"chat_id,omitempty"`
}

func (r *LLMHookResponse) Clone() *LLMHookResponse {
	if r == nil {
		return nil
	}
	cloned := *r
	cloned.Response = cloneLLMResponse(r.Response)
	return &cloned
}

type ToolCallHookRequest struct {
	Meta       EventMeta         `json:"meta"`
	Tool       string            `json:"tool"`
	Arguments  map[string]any    `json:"arguments,omitempty"`
	Channel    string            `json:"channel,omitempty"`
	ChatID     string            `json:"chat_id,omitempty"`
	HookResult *tools.ToolResult `json:"hook_result,omitempty"` // Result returned directly by hook (for respond action). Media is supported - see Media handling section in docs.
}

func (r *ToolCallHookRequest) Clone() *ToolCallHookRequest {
	if r == nil {
		return nil
	}
	cloned := *r
	cloned.Arguments = cloneStringAnyMap(r.Arguments)
	cloned.HookResult = cloneToolResult(r.HookResult)
	return &cloned
}

type ToolApprovalRequest struct {
	Meta      EventMeta      `json:"meta"`
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments,omitempty"`
	Channel   string         `json:"channel,omitempty"`
	ChatID    string         `json:"chat_id,omitempty"`
}

func (r *ToolApprovalRequest) Clone() *ToolApprovalRequest {
	if r == nil {
		return nil
	}
	cloned := *r
	cloned.Arguments = cloneStringAnyMap(r.Arguments)
	return &cloned
}

type ToolResultHookResponse struct {
	Meta      EventMeta         `json:"meta"`
	Tool      string            `json:"tool"`
	Arguments map[string]any    `json:"arguments,omitempty"`
	Result    *tools.ToolResult `json:"result,omitempty"`
	Duration  time.Duration     `json:"duration"`
	Channel   string            `json:"channel,omitempty"`
	ChatID    string            `json:"chat_id,omitempty"`
}

func (r *ToolResultHookResponse) Clone() *ToolResultHookResponse {
	if r == nil {
		return nil
	}
	cloned := *r
	cloned.Arguments = cloneStringAnyMap(r.Arguments)
	cloned.Result = cloneToolResult(r.Result)
	return &cloned
}

type HookManager struct {
	eventBus           *EventBus
	observerTimeout    time.Duration
	interceptorTimeout time.Duration
	approvalTimeout    time.Duration

	mu      sync.RWMutex
	hooks   map[string]HookRegistration
	ordered []HookRegistration

	sub       EventSubscription
	done      chan struct{}
	closeOnce sync.Once
}

func NewHookManager(eventBus *EventBus) *HookManager {
	hm := &HookManager{
		eventBus:           eventBus,
		observerTimeout:    defaultHookObserverTimeout,
		interceptorTimeout: defaultHookInterceptorTimeout,
		approvalTimeout:    defaultHookApprovalTimeout,
		hooks:              make(map[string]HookRegistration),
		done:               make(chan struct{}),
	}

	if eventBus == nil {
		close(hm.done)
		return hm
	}

	hm.sub = eventBus.Subscribe(hookObserverBufferSize)
	go hm.dispatchEvents()
	return hm
}

func (hm *HookManager) Close() {
	if hm == nil {
		return
	}

	hm.closeOnce.Do(func() {
		if hm.eventBus != nil {
			hm.eventBus.Unsubscribe(hm.sub.ID)
		}
		<-hm.done
		hm.closeAllHooks()
	})
}

func (hm *HookManager) ConfigureTimeouts(observer, interceptor, approval time.Duration) {
	if hm == nil {
		return
	}
	if observer > 0 {
		hm.observerTimeout = observer
	}
	if interceptor > 0 {
		hm.interceptorTimeout = interceptor
	}
	if approval > 0 {
		hm.approvalTimeout = approval
	}
}

func (hm *HookManager) Mount(reg HookRegistration) error {
	if hm == nil {
		return fmt.Errorf("hook manager is nil")
	}
	if reg.Name == "" {
		return fmt.Errorf("hook name is required")
	}
	if reg.Hook == nil {
		return fmt.Errorf("hook %q is nil", reg.Name)
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	if existing, ok := hm.hooks[reg.Name]; ok {
		closeHookIfPossible(existing.Hook)
	}
	hm.hooks[reg.Name] = reg
	hm.rebuildOrdered()
	return nil
}

func (hm *HookManager) Unmount(name string) {
	if hm == nil || name == "" {
		return
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	if existing, ok := hm.hooks[name]; ok {
		closeHookIfPossible(existing.Hook)
	}
	delete(hm.hooks, name)
	hm.rebuildOrdered()
}

func (hm *HookManager) dispatchEvents() {
	defer close(hm.done)

	for evt := range hm.sub.C {
		for _, reg := range hm.snapshotHooks() {
			observer, ok := reg.Hook.(EventObserver)
			if !ok {
				continue
			}
			hm.runObserver(reg.Name, observer, evt)
		}
	}
}

func (hm *HookManager) BeforeLLM(ctx context.Context, req *LLMHookRequest) (*LLMHookRequest, HookDecision) {
	if hm == nil || req == nil {
		return req, HookDecision{Action: HookActionContinue}
	}

	current := req.Clone()
	for _, reg := range hm.snapshotHooks() {
		interceptor, ok := reg.Hook.(LLMInterceptor)
		if !ok {
			continue
		}

		next, decision, ok := hm.callBeforeLLM(ctx, reg.Name, interceptor, current.Clone())
		if !ok {
			continue
		}

		switch decision.normalizedAction() {
		case HookActionContinue, HookActionModify:
			if next != nil {
				current = next
			}
		case HookActionAbortTurn, HookActionHardAbort:
			return current, decision
		default:
			hm.logUnsupportedAction(reg.Name, "before_llm", decision.Action)
		}
	}
	return current, HookDecision{Action: HookActionContinue}
}

func (hm *HookManager) AfterLLM(ctx context.Context, resp *LLMHookResponse) (*LLMHookResponse, HookDecision) {
	if hm == nil || resp == nil {
		return resp, HookDecision{Action: HookActionContinue}
	}

	current := resp.Clone()
	for _, reg := range hm.snapshotHooks() {
		interceptor, ok := reg.Hook.(LLMInterceptor)
		if !ok {
			continue
		}

		next, decision, ok := hm.callAfterLLM(ctx, reg.Name, interceptor, current.Clone())
		if !ok {
			continue
		}

		switch decision.normalizedAction() {
		case HookActionContinue, HookActionModify:
			if next != nil {
				current = next
			}
		case HookActionAbortTurn, HookActionHardAbort:
			return current, decision
		default:
			hm.logUnsupportedAction(reg.Name, "after_llm", decision.Action)
		}
	}
	return current, HookDecision{Action: HookActionContinue}
}

func (hm *HookManager) BeforeTool(
	ctx context.Context,
	call *ToolCallHookRequest,
) (*ToolCallHookRequest, HookDecision) {
	if hm == nil || call == nil {
		return call, HookDecision{Action: HookActionContinue}
	}

	current := call.Clone()
	for _, reg := range hm.snapshotHooks() {
		interceptor, ok := reg.Hook.(ToolInterceptor)
		if !ok {
			continue
		}

		next, decision, ok := hm.callBeforeTool(ctx, reg.Name, interceptor, current.Clone())
		if !ok {
			continue
		}

		switch decision.normalizedAction() {
		case HookActionContinue, HookActionModify:
			if next != nil {
				current = next
			}
		case HookActionRespond:
			// Hook returns result directly, skip tool execution
			// Carry HookResult in ToolCallHookRequest and return
			return next, decision
		case HookActionDenyTool, HookActionAbortTurn, HookActionHardAbort:
			return current, decision
		default:
			hm.logUnsupportedAction(reg.Name, "before_tool", decision.Action)
		}
	}
	return current, HookDecision{Action: HookActionContinue}
}

func (hm *HookManager) AfterTool(
	ctx context.Context,
	result *ToolResultHookResponse,
) (*ToolResultHookResponse, HookDecision) {
	if hm == nil || result == nil {
		return result, HookDecision{Action: HookActionContinue}
	}

	current := result.Clone()
	for _, reg := range hm.snapshotHooks() {
		interceptor, ok := reg.Hook.(ToolInterceptor)
		if !ok {
			continue
		}

		next, decision, ok := hm.callAfterTool(ctx, reg.Name, interceptor, current.Clone())
		if !ok {
			continue
		}

		switch decision.normalizedAction() {
		case HookActionContinue, HookActionModify:
			if next != nil {
				current = next
			}
		case HookActionAbortTurn, HookActionHardAbort:
			return current, decision
		default:
			hm.logUnsupportedAction(reg.Name, "after_tool", decision.Action)
		}
	}
	return current, HookDecision{Action: HookActionContinue}
}

func (hm *HookManager) ApproveTool(ctx context.Context, req *ToolApprovalRequest) ApprovalDecision {
	if hm == nil || req == nil {
		return ApprovalDecision{Approved: true}
	}

	for _, reg := range hm.snapshotHooks() {
		approver, ok := reg.Hook.(ToolApprover)
		if !ok {
			continue
		}

		decision, ok := hm.callApproveTool(ctx, reg.Name, approver, req.Clone())
		if !ok {
			return ApprovalDecision{
				Approved: false,
				Reason:   fmt.Sprintf("tool approval hook %q failed", reg.Name),
			}
		}
		if !decision.Approved {
			return decision
		}
	}

	return ApprovalDecision{Approved: true}
}

func (hm *HookManager) rebuildOrdered() {
	hm.ordered = hm.ordered[:0]
	for _, reg := range hm.hooks {
		hm.ordered = append(hm.ordered, reg)
	}
	sort.SliceStable(hm.ordered, func(i, j int) bool {
		if hm.ordered[i].Source != hm.ordered[j].Source {
			return hm.ordered[i].Source < hm.ordered[j].Source
		}
		if hm.ordered[i].Priority == hm.ordered[j].Priority {
			return hm.ordered[i].Name < hm.ordered[j].Name
		}
		return hm.ordered[i].Priority < hm.ordered[j].Priority
	})
}

func (hm *HookManager) snapshotHooks() []HookRegistration {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	snapshot := make([]HookRegistration, len(hm.ordered))
	copy(snapshot, hm.ordered)
	return snapshot
}

func (hm *HookManager) closeAllHooks() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	for name, reg := range hm.hooks {
		closeHookIfPossible(reg.Hook)
		delete(hm.hooks, name)
	}
	hm.ordered = nil
}

func (hm *HookManager) runObserver(name string, observer EventObserver, evt Event) {
	ctx, cancel := context.WithTimeout(context.Background(), hm.observerTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- observer.OnEvent(ctx, evt)
	}()

	select {
	case err := <-done:
		if err != nil {
			logger.WarnCF("hooks", "Event observer failed", map[string]any{
				"hook":  name,
				"event": evt.Kind.String(),
				"error": err.Error(),
			})
		}
	case <-ctx.Done():
		logger.WarnCF("hooks", "Event observer timed out", map[string]any{
			"hook":       name,
			"event":      evt.Kind.String(),
			"timeout_ms": hm.observerTimeout.Milliseconds(),
		})
	}
}

func (hm *HookManager) callBeforeLLM(
	parent context.Context,
	name string,
	interceptor LLMInterceptor,
	req *LLMHookRequest,
) (*LLMHookRequest, HookDecision, bool) {
	return runInterceptorHook(
		parent,
		hm.interceptorTimeout,
		name,
		"before_llm",
		func(ctx context.Context) (*LLMHookRequest, HookDecision, error) {
			return interceptor.BeforeLLM(ctx, req)
		},
	)
}

func (hm *HookManager) callAfterLLM(
	parent context.Context,
	name string,
	interceptor LLMInterceptor,
	resp *LLMHookResponse,
) (*LLMHookResponse, HookDecision, bool) {
	return runInterceptorHook(
		parent,
		hm.interceptorTimeout,
		name,
		"after_llm",
		func(ctx context.Context) (*LLMHookResponse, HookDecision, error) {
			return interceptor.AfterLLM(ctx, resp)
		},
	)
}

func (hm *HookManager) callBeforeTool(
	parent context.Context,
	name string,
	interceptor ToolInterceptor,
	call *ToolCallHookRequest,
) (*ToolCallHookRequest, HookDecision, bool) {
	return runInterceptorHook(
		parent,
		hm.interceptorTimeout,
		name,
		"before_tool",
		func(ctx context.Context) (*ToolCallHookRequest, HookDecision, error) {
			return interceptor.BeforeTool(ctx, call)
		},
	)
}

func (hm *HookManager) callAfterTool(
	parent context.Context,
	name string,
	interceptor ToolInterceptor,
	resultView *ToolResultHookResponse,
) (*ToolResultHookResponse, HookDecision, bool) {
	return runInterceptorHook(
		parent,
		hm.interceptorTimeout,
		name,
		"after_tool",
		func(ctx context.Context) (*ToolResultHookResponse, HookDecision, error) {
			return interceptor.AfterTool(ctx, resultView)
		},
	)
}

func (hm *HookManager) callApproveTool(
	parent context.Context,
	name string,
	approver ToolApprover,
	req *ToolApprovalRequest,
) (ApprovalDecision, bool) {
	return runApprovalHook(
		parent,
		hm.approvalTimeout,
		name,
		"approve_tool",
		func(ctx context.Context) (ApprovalDecision, error) {
			return approver.ApproveTool(ctx, req)
		},
	)
}

func runInterceptorHook[T any](
	parent context.Context,
	timeout time.Duration,
	name string,
	stage string,
	fn func(ctx context.Context) (T, HookDecision, error),
) (T, HookDecision, bool) {
	var zero T

	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	type result struct {
		value    T
		decision HookDecision
		err      error
	}
	done := make(chan result, 1)
	go func() {
		value, decision, err := fn(ctx)
		done <- result{value: value, decision: decision, err: err}
	}()

	select {
	case res := <-done:
		if res.err != nil {
			logger.WarnCF("hooks", "Interceptor hook failed", map[string]any{
				"hook":  name,
				"stage": stage,
				"error": res.err.Error(),
			})
			return zero, HookDecision{}, false
		}
		return res.value, res.decision, true
	case <-ctx.Done():
		logger.WarnCF("hooks", "Interceptor hook timed out", map[string]any{
			"hook":       name,
			"stage":      stage,
			"timeout_ms": timeout.Milliseconds(),
		})
		return zero, HookDecision{}, false
	}
}

func runApprovalHook(
	parent context.Context,
	timeout time.Duration,
	name string,
	stage string,
	fn func(ctx context.Context) (ApprovalDecision, error),
) (ApprovalDecision, bool) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	type result struct {
		decision ApprovalDecision
		err      error
	}
	done := make(chan result, 1)
	go func() {
		decision, err := fn(ctx)
		done <- result{decision: decision, err: err}
	}()

	select {
	case res := <-done:
		if res.err != nil {
			logger.WarnCF("hooks", "Approval hook failed", map[string]any{
				"hook":  name,
				"stage": stage,
				"error": res.err.Error(),
			})
			return ApprovalDecision{}, false
		}
		return res.decision, true
	case <-ctx.Done():
		logger.WarnCF("hooks", "Approval hook timed out", map[string]any{
			"hook":       name,
			"stage":      stage,
			"timeout_ms": timeout.Milliseconds(),
		})
		return ApprovalDecision{
			Approved: false,
			Reason:   fmt.Sprintf("tool approval hook %q timed out", name),
		}, true
	}
}

func (hm *HookManager) logUnsupportedAction(name, stage string, action HookAction) {
	logger.WarnCF("hooks", "Hook returned unsupported action for stage", map[string]any{
		"hook":   name,
		"stage":  stage,
		"action": action,
	})
}

func cloneProviderMessages(messages []providers.Message) []providers.Message {
	if len(messages) == 0 {
		return nil
	}

	cloned := make([]providers.Message, len(messages))
	for i, msg := range messages {
		cloned[i] = msg
		if len(msg.Media) > 0 {
			cloned[i].Media = append([]string(nil), msg.Media...)
		}
		if len(msg.SystemParts) > 0 {
			cloned[i].SystemParts = append([]providers.ContentBlock(nil), msg.SystemParts...)
		}
		if len(msg.ToolCalls) > 0 {
			cloned[i].ToolCalls = cloneProviderToolCalls(msg.ToolCalls)
		}
	}
	return cloned
}

func cloneProviderToolCalls(calls []providers.ToolCall) []providers.ToolCall {
	if len(calls) == 0 {
		return nil
	}

	cloned := make([]providers.ToolCall, len(calls))
	for i, call := range calls {
		cloned[i] = call
		if call.Function != nil {
			fn := *call.Function
			cloned[i].Function = &fn
		}
		if call.Arguments != nil {
			cloned[i].Arguments = cloneStringAnyMap(call.Arguments)
		}
		if call.ExtraContent != nil {
			extra := *call.ExtraContent
			if call.ExtraContent.Google != nil {
				google := *call.ExtraContent.Google
				extra.Google = &google
			}
			cloned[i].ExtraContent = &extra
		}
	}
	return cloned
}

func cloneToolDefinitions(defs []providers.ToolDefinition) []providers.ToolDefinition {
	if len(defs) == 0 {
		return nil
	}

	cloned := make([]providers.ToolDefinition, len(defs))
	for i, def := range defs {
		cloned[i] = def
		cloned[i].Function.Parameters = cloneStringAnyMap(def.Function.Parameters)
	}
	return cloned
}

func cloneLLMResponse(resp *providers.LLMResponse) *providers.LLMResponse {
	if resp == nil {
		return nil
	}
	cloned := *resp
	cloned.ToolCalls = cloneProviderToolCalls(resp.ToolCalls)
	if len(resp.ReasoningDetails) > 0 {
		cloned.ReasoningDetails = append(cloned.ReasoningDetails[:0:0], resp.ReasoningDetails...)
	}
	if resp.Usage != nil {
		usage := *resp.Usage
		cloned.Usage = &usage
	}
	return &cloned
}

func cloneStringAnyMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}

	cloned := make(map[string]any, len(src))
	for k, v := range src {
		cloned[k] = v
	}
	return cloned
}

func cloneToolResult(result *tools.ToolResult) *tools.ToolResult {
	if result == nil {
		return nil
	}

	cloned := *result
	if len(result.Media) > 0 {
		cloned.Media = append([]string(nil), result.Media...)
	}
	if len(result.ArtifactTags) > 0 {
		cloned.ArtifactTags = append([]string(nil), result.ArtifactTags...)
	}
	if len(result.Messages) > 0 {
		cloned.Messages = make([]providers.Message, len(result.Messages))
		copy(cloned.Messages, result.Messages)
	}
	return &cloned
}

func closeHookIfPossible(hook any) {
	closer, ok := hook.(io.Closer)
	if !ok {
		return
	}
	if err := closer.Close(); err != nil {
		logger.WarnCF("hooks", "Failed to close hook", map[string]any{
			"error": err.Error(),
		})
	}
}
