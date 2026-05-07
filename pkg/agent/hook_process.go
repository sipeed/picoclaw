package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
	"github.com/sipeed/picoclaw/pkg/isolation"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/tools"
)

const (
	processHookJSONRPCVersion = "2.0"
	processHookReadBufferSize = 1024 * 1024
	processHookCloseTimeout   = 2 * time.Second
)

type ProcessHookOptions struct {
	Command       []string
	Dir           string
	Env           []string
	Observe       bool
	ObserveKinds  []string
	InterceptLLM  bool
	InterceptTool bool
	ApproveTool   bool
}

type ProcessHook struct {
	name string
	opts ProcessHookOptions

	cmd          *exec.Cmd
	stdin        io.WriteCloser
	observeKinds map[string]struct{}

	writeMu sync.Mutex

	pendingMu sync.Mutex
	pending   map[uint64]chan processHookRPCMessage
	nextID    atomic.Uint64

	closed    atomic.Bool
	done      chan struct{}
	closeErr  error
	closeMu   sync.Mutex
	closeOnce sync.Once
}

type processHookRPCMessage struct {
	JSONRPC string               `json:"jsonrpc,omitempty"`
	ID      uint64               `json:"id,omitempty"`
	Method  string               `json:"method,omitempty"`
	Params  json.RawMessage      `json:"params,omitempty"`
	Result  json.RawMessage      `json:"result,omitempty"`
	Error   *processHookRPCError `json:"error,omitempty"`
}

type processHookRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type processHookHelloParams struct {
	Name    string   `json:"name"`
	Version int      `json:"version"`
	Modes   []string `json:"modes,omitempty"`
}

type processHookDecisionResponse struct {
	Action HookAction `json:"action"`
	Reason string     `json:"reason,omitempty"`
}

type processHookBeforeLLMResponse struct {
	processHookDecisionResponse
	Request *LLMHookRequest `json:"request,omitempty"`
}

type processHookAfterLLMResponse struct {
	processHookDecisionResponse
	Response *LLMHookResponse `json:"response,omitempty"`
}

type processHookBeforeToolResponse struct {
	processHookDecisionResponse
	Call   *ToolCallHookRequest `json:"call,omitempty"`
	Result *tools.ToolResult    `json:"result,omitempty"` // Result returned directly by hook (for respond action)
}

type processHookAfterToolResponse struct {
	processHookDecisionResponse
	Result *ToolResultHookResponse `json:"result,omitempty"`
}

func NewProcessHook(ctx context.Context, name string, opts ProcessHookOptions) (*ProcessHook, error) {
	if len(opts.Command) == 0 {
		return nil, fmt.Errorf("process hook command is required")
	}

	cmd := exec.Command(opts.Command[0], opts.Command[1:]...)
	cmd.Dir = opts.Dir
	if len(opts.Env) > 0 {
		cmd.Env = append(os.Environ(), opts.Env...)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("create process hook stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create process hook stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("create process hook stderr: %w", err)
	}
	// Route hook subprocess startup through the shared isolation entry point so
	// process hooks inherit the same isolation behavior as other child processes.
	if err := isolation.Start(cmd); err != nil {
		return nil, fmt.Errorf("start process hook: %w", err)
	}

	ph := &ProcessHook{
		name:         name,
		opts:         opts,
		cmd:          cmd,
		stdin:        stdin,
		observeKinds: newProcessHookObserveKinds(opts.ObserveKinds),
		pending:      make(map[uint64]chan processHookRPCMessage),
		done:         make(chan struct{}),
	}

	go ph.readLoop(stdout)
	go ph.readStderr(stderr)
	go ph.waitLoop()

	helloCtx := ctx
	if helloCtx == nil {
		var cancel context.CancelFunc
		helloCtx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}
	if err := ph.hello(helloCtx); err != nil {
		_ = ph.Close()
		return nil, err
	}

	return ph, nil
}

func (ph *ProcessHook) Close() error {
	if ph == nil {
		return nil
	}

	ph.closeOnce.Do(func() {
		ph.closed.Store(true)
		if ph.stdin != nil {
			_ = ph.stdin.Close()
		}

		select {
		case <-ph.done:
		case <-time.After(processHookCloseTimeout):
			if ph.cmd != nil && ph.cmd.Process != nil {
				_ = ph.cmd.Process.Kill()
			}
			<-ph.done
		}
	})

	ph.closeMu.Lock()
	defer ph.closeMu.Unlock()
	return ph.closeErr
}

func (ph *ProcessHook) OnRuntimeEvent(ctx context.Context, evt runtimeevents.Event) error {
	if ph == nil || !ph.opts.Observe {
		return nil
	}
	if len(ph.observeKinds) > 0 {
		if _, ok := ph.observeKinds[evt.Kind.String()]; !ok {
			return nil
		}
	}
	return ph.notify(ctx, "hook.runtime_event", evt)
}

func (ph *ProcessHook) BeforeLLM(
	ctx context.Context,
	req *LLMHookRequest,
) (*LLMHookRequest, HookDecision, error) {
	if ph == nil || !ph.opts.InterceptLLM {
		return req, HookDecision{Action: HookActionContinue}, nil
	}

	var resp processHookBeforeLLMResponse
	if err := ph.call(ctx, "hook.before_llm", req, &resp); err != nil {
		return nil, HookDecision{}, err
	}
	if resp.Request == nil {
		resp.Request = req
	}
	return resp.Request, HookDecision{Action: resp.Action, Reason: resp.Reason}, nil
}

func (ph *ProcessHook) AfterLLM(
	ctx context.Context,
	resp *LLMHookResponse,
) (*LLMHookResponse, HookDecision, error) {
	if ph == nil || !ph.opts.InterceptLLM {
		return resp, HookDecision{Action: HookActionContinue}, nil
	}

	var result processHookAfterLLMResponse
	if err := ph.call(ctx, "hook.after_llm", resp, &result); err != nil {
		return nil, HookDecision{}, err
	}
	if result.Response == nil {
		result.Response = resp
	}
	return result.Response, HookDecision{Action: result.Action, Reason: result.Reason}, nil
}

func (ph *ProcessHook) BeforeTool(
	ctx context.Context,
	call *ToolCallHookRequest,
) (*ToolCallHookRequest, HookDecision, error) {
	if ph == nil || !ph.opts.InterceptTool {
		return call, HookDecision{Action: HookActionContinue}, nil
	}

	var resp processHookBeforeToolResponse
	if err := ph.call(ctx, "hook.before_tool", call, &resp); err != nil {
		return nil, HookDecision{}, err
	}
	if resp.Call == nil {
		resp.Call = call
	}
	// If hook returned a Result, carry it in ToolCallHookRequest
	if resp.Result != nil {
		resp.Call.HookResult = resp.Result
	}
	return resp.Call, HookDecision{Action: resp.Action, Reason: resp.Reason}, nil
}

func (ph *ProcessHook) AfterTool(
	ctx context.Context,
	result *ToolResultHookResponse,
) (*ToolResultHookResponse, HookDecision, error) {
	if ph == nil || !ph.opts.InterceptTool {
		return result, HookDecision{Action: HookActionContinue}, nil
	}

	var resp processHookAfterToolResponse
	if err := ph.call(ctx, "hook.after_tool", result, &resp); err != nil {
		return nil, HookDecision{}, err
	}
	if resp.Result == nil {
		resp.Result = result
	}
	return resp.Result, HookDecision{Action: resp.Action, Reason: resp.Reason}, nil
}

func (ph *ProcessHook) ApproveTool(ctx context.Context, req *ToolApprovalRequest) (ApprovalDecision, error) {
	if ph == nil || !ph.opts.ApproveTool {
		return ApprovalDecision{Approved: true}, nil
	}

	var resp ApprovalDecision
	if err := ph.call(ctx, "hook.approve_tool", req, &resp); err != nil {
		return ApprovalDecision{}, err
	}
	return resp, nil
}

func (ph *ProcessHook) hello(ctx context.Context) error {
	modes := make([]string, 0, 4)
	if ph.opts.Observe {
		modes = append(modes, "observe")
	}
	if ph.opts.InterceptLLM {
		modes = append(modes, "llm")
	}
	if ph.opts.InterceptTool {
		modes = append(modes, "tool")
	}
	if ph.opts.ApproveTool {
		modes = append(modes, "approve")
	}

	var result map[string]any
	return ph.call(ctx, "hook.hello", processHookHelloParams{
		Name:    ph.name,
		Version: 1,
		Modes:   modes,
	}, &result)
}

func (ph *ProcessHook) notify(ctx context.Context, method string, params any) error {
	msg := processHookRPCMessage{
		JSONRPC: processHookJSONRPCVersion,
		Method:  method,
	}
	if params != nil {
		body, err := json.Marshal(params)
		if err != nil {
			return err
		}
		msg.Params = body
	}
	return ph.send(ctx, msg)
}

func (ph *ProcessHook) call(ctx context.Context, method string, params any, out any) error {
	if ph.closed.Load() {
		return fmt.Errorf("process hook %q is closed", ph.name)
	}

	id := ph.nextID.Add(1)
	respCh := make(chan processHookRPCMessage, 1)
	ph.pendingMu.Lock()
	ph.pending[id] = respCh
	ph.pendingMu.Unlock()

	msg := processHookRPCMessage{
		JSONRPC: processHookJSONRPCVersion,
		ID:      id,
		Method:  method,
	}
	if params != nil {
		body, err := json.Marshal(params)
		if err != nil {
			ph.removePending(id)
			return err
		}
		msg.Params = body
	}

	if err := ph.send(ctx, msg); err != nil {
		ph.removePending(id)
		return err
	}

	select {
	case resp, ok := <-respCh:
		if !ok {
			return fmt.Errorf("process hook %q closed while waiting for %s", ph.name, method)
		}
		if resp.Error != nil {
			return fmt.Errorf("process hook %q %s failed: %s", ph.name, method, resp.Error.Message)
		}
		if out != nil && len(resp.Result) > 0 {
			if err := json.Unmarshal(resp.Result, out); err != nil {
				return fmt.Errorf("decode process hook %q %s result: %w", ph.name, method, err)
			}
		}
		return nil
	case <-ctx.Done():
		ph.removePending(id)
		return ctx.Err()
	}
}

func (ph *ProcessHook) send(ctx context.Context, msg processHookRPCMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	body = append(body, '\n')

	ph.writeMu.Lock()
	defer ph.writeMu.Unlock()

	if ph.closed.Load() {
		return fmt.Errorf("process hook %q is closed", ph.name)
	}

	done := make(chan error, 1)
	go func() {
		_, writeErr := ph.stdin.Write(body)
		done <- writeErr
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("write process hook %q message: %w", ph.name, err)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (ph *ProcessHook) readLoop(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), processHookReadBufferSize)

	for scanner.Scan() {
		var msg processHookRPCMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			logger.WarnCF("hooks", "Failed to decode process hook message", map[string]any{
				"hook":  ph.name,
				"error": err.Error(),
			})
			continue
		}
		if msg.ID == 0 {
			continue
		}
		ph.pendingMu.Lock()
		respCh, ok := ph.pending[msg.ID]
		if ok {
			delete(ph.pending, msg.ID)
		}
		ph.pendingMu.Unlock()
		if ok {
			respCh <- msg
			close(respCh)
		}
	}
}

func (ph *ProcessHook) readStderr(stderr io.Reader) {
	scanner := bufio.NewScanner(stderr)
	scanner.Buffer(make([]byte, 0, 16*1024), processHookReadBufferSize)
	for scanner.Scan() {
		logger.WarnCF("hooks", "Process hook stderr", map[string]any{
			"hook":   ph.name,
			"stderr": scanner.Text(),
		})
	}
}

func (ph *ProcessHook) waitLoop() {
	err := ph.cmd.Wait()
	ph.closeMu.Lock()
	ph.closeErr = err
	ph.closeMu.Unlock()
	ph.failPending(err)
	close(ph.done)
}

func (ph *ProcessHook) failPending(err error) {
	ph.pendingMu.Lock()
	defer ph.pendingMu.Unlock()

	msg := processHookRPCMessage{
		Error: &processHookRPCError{
			Code:    -32000,
			Message: "process exited",
		},
	}
	if err != nil {
		msg.Error.Message = err.Error()
	}

	for id, ch := range ph.pending {
		delete(ph.pending, id)
		ch <- msg
		close(ch)
	}
}

func (ph *ProcessHook) removePending(id uint64) {
	ph.pendingMu.Lock()
	defer ph.pendingMu.Unlock()

	if ch, ok := ph.pending[id]; ok {
		delete(ph.pending, id)
		close(ch)
	}
}

func (al *AgentLoop) MountProcessHook(ctx context.Context, name string, opts ProcessHookOptions) error {
	if al == nil {
		return fmt.Errorf("agent loop is nil")
	}
	processHook, err := NewProcessHook(ctx, name, opts)
	if err != nil {
		return err
	}
	if err := al.MountHook(HookRegistration{
		Name:   name,
		Source: HookSourceProcess,
		Hook:   processHook,
	}); err != nil {
		_ = processHook.Close()
		return err
	}
	return nil
}

func newProcessHookObserveKinds(kinds []string) map[string]struct{} {
	if len(kinds) == 0 {
		return nil
	}

	normalized := make(map[string]struct{}, len(kinds))
	for _, kind := range kinds {
		if kind == "" {
			continue
		}
		normalized[kind] = struct{}{}
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}
