package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultProcessMaxOutputChars  = 30000
	defaultProcessMaxPendingChars = 12000
	defaultProcessLogTailLines    = 200
)

var (
	ErrProcessSessionNotFound = errors.New("process session not found")
	ErrProcessSessionRunning  = errors.New("process session is still running")
)

type processSession struct {
	ID      string
	Command string
	CWD     string

	StartedAt time.Time
	UpdatedAt time.Time
	EndedAt   time.Time

	PID       int
	Status    string
	ExitCode  *int
	ExitError string
	Truncated bool

	output        string
	pending       string
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	cancel        func()
	killRequested bool

	notify chan struct{}
	mu     sync.Mutex
}

type ProcessSessionSnapshot struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
	Command   string `json:"command"`
	CWD       string `json:"cwd,omitempty"`
	PID       int    `json:"pid,omitempty"`

	StartedAt string `json:"started_at"`
	UpdatedAt string `json:"updated_at"`
	EndedAt   string `json:"ended_at,omitempty"`

	ExitCode  *int   `json:"exit_code,omitempty"`
	ExitError string `json:"exit_error,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}

type ProcessPollResult struct {
	Session  ProcessSessionSnapshot `json:"session"`
	Output   string                 `json:"output,omitempty"`
	TimedOut bool                   `json:"timed_out,omitempty"`
}

type ProcessLogResult struct {
	Session    ProcessSessionSnapshot `json:"session"`
	TotalLines int                    `json:"total_lines"`
	Offset     int                    `json:"offset"`
	Limit      int                    `json:"limit,omitempty"`
	Lines      []string               `json:"lines"`
	Output     string                 `json:"output"`
}

type ProcessManager struct {
	mu              sync.RWMutex
	sessions        map[string]*processSession
	nextID          atomic.Uint64
	maxOutputChars  int
	maxPendingChars int
}

func NewProcessManager(maxOutputChars int) *ProcessManager {
	if maxOutputChars <= 0 {
		maxOutputChars = defaultProcessMaxOutputChars
	}

	maxPendingChars := defaultProcessMaxPendingChars
	if maxPendingChars > maxOutputChars {
		maxPendingChars = maxOutputChars
	}

	return &ProcessManager{
		sessions:        make(map[string]*processSession),
		maxOutputChars:  maxOutputChars,
		maxPendingChars: maxPendingChars,
	}
}

func (pm *ProcessManager) StartSession(
	command, cwd string,
	cmd *exec.Cmd,
	stdin io.WriteCloser,
	cancel func(),
) string {
	id := fmt.Sprintf("proc-%d", pm.nextID.Add(1))
	now := time.Now()

	pid := 0
	if cmd != nil && cmd.Process != nil {
		pid = cmd.Process.Pid
	}

	session := &processSession{
		ID:        id,
		Command:   command,
		CWD:       cwd,
		StartedAt: now,
		UpdatedAt: now,
		PID:       pid,
		Status:    "running",
		cmd:       cmd,
		stdin:     stdin,
		cancel:    cancel,
		notify:    make(chan struct{}, 1),
	}

	pm.mu.Lock()
	pm.sessions[id] = session
	pm.mu.Unlock()

	return id
}

func (pm *ProcessManager) AppendOutput(sessionID, chunk string) {
	if chunk == "" {
		return
	}

	session, ok := pm.getSession(sessionID)
	if !ok {
		return
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.output, session.Truncated = appendWithCap(
		session.output,
		chunk,
		pm.maxOutputChars,
		session.Truncated,
	)
	session.pending, session.Truncated = appendWithCap(
		session.pending,
		chunk,
		pm.maxPendingChars,
		session.Truncated,
	)
	session.UpdatedAt = time.Now()
	session.signalNotifyLocked()
}

func (pm *ProcessManager) MarkExited(sessionID string, waitErr error, timedOut bool) {
	session, ok := pm.getSession(sessionID)
	if !ok {
		return
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	now := time.Now()
	session.UpdatedAt = now
	session.EndedAt = now

	switch {
	case timedOut:
		session.Status = "timeout"
	case session.killRequested:
		session.Status = "killed"
	case waitErr != nil:
		session.Status = "failed"
	default:
		session.Status = "completed"
	}

	if waitErr != nil {
		session.ExitError = waitErr.Error()
	}
	if code, ok := extractExitCode(waitErr); ok {
		session.ExitCode = &code
	} else if waitErr == nil {
		code := 0
		session.ExitCode = &code
	}

	if session.stdin != nil {
		_ = session.stdin.Close()
		session.stdin = nil
	}
	if session.cancel != nil {
		session.cancel()
		session.cancel = nil
	}
	session.cmd = nil
	session.signalNotifyLocked()
}

func (pm *ProcessManager) GetSnapshot(sessionID string) (ProcessSessionSnapshot, bool) {
	session, ok := pm.getSession(sessionID)
	if !ok {
		return ProcessSessionSnapshot{}, false
	}

	session.mu.Lock()
	defer session.mu.Unlock()
	return session.snapshotLocked(), true
}

func (pm *ProcessManager) ListSnapshots() []ProcessSessionSnapshot {
	pm.mu.RLock()
	sessions := make([]*processSession, 0, len(pm.sessions))
	for _, session := range pm.sessions {
		sessions = append(sessions, session)
	}
	pm.mu.RUnlock()

	out := make([]ProcessSessionSnapshot, 0, len(sessions))
	for _, session := range sessions {
		session.mu.Lock()
		out = append(out, session.snapshotLocked())
		session.mu.Unlock()
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].StartedAt > out[j].StartedAt
	})
	return out
}

func (pm *ProcessManager) Poll(sessionID string, timeout time.Duration) (ProcessPollResult, error) {
	session, ok := pm.getSession(sessionID)
	if !ok {
		return ProcessPollResult{}, ErrProcessSessionNotFound
	}
	if timeout < 0 {
		timeout = 0
	}

	deadline := time.Now().Add(timeout)
	for {
		output, snapshot := session.drainPending()
		if output != "" || snapshot.Status != "running" || timeout == 0 {
			return ProcessPollResult{
				Session: snapshot,
				Output:  output,
			}, nil
		}

		waitFor := time.Until(deadline)
		if waitFor <= 0 {
			return ProcessPollResult{
				Session:  snapshot,
				TimedOut: true,
			}, nil
		}

		select {
		case <-session.notify:
			// loop and re-check
		case <-time.After(waitFor):
			snapshot := session.snapshot()
			return ProcessPollResult{
				Session:  snapshot,
				TimedOut: true,
			}, nil
		}
	}
}

func (pm *ProcessManager) Log(
	sessionID string,
	offset, limit int,
	useDefaultTail bool,
) (ProcessLogResult, error) {
	session, ok := pm.getSession(sessionID)
	if !ok {
		return ProcessLogResult{}, ErrProcessSessionNotFound
	}

	output, snapshot := session.outputSnapshot()
	lines := normalizeOutputLines(output)
	total := len(lines)

	start := offset
	if start < 0 {
		start = 0
	}
	if start > total {
		start = total
	}

	end := total
	effectiveLimit := limit
	if useDefaultTail {
		if total > defaultProcessLogTailLines {
			start = total - defaultProcessLogTailLines
		} else {
			start = 0
		}
		end = total
		effectiveLimit = defaultProcessLogTailLines
	} else if effectiveLimit > 0 {
		if start+effectiveLimit < end {
			end = start + effectiveLimit
		}
	}

	window := []string{}
	if start < end {
		window = lines[start:end]
	}

	return ProcessLogResult{
		Session:    snapshot,
		TotalLines: total,
		Offset:     start,
		Limit:      effectiveLimit,
		Lines:      window,
		Output:     strings.Join(window, "\n"),
	}, nil
}

func (pm *ProcessManager) Write(sessionID, data string, eof bool) error {
	session, ok := pm.getSession(sessionID)
	if !ok {
		return ErrProcessSessionNotFound
	}

	session.mu.Lock()
	if session.Status != "running" {
		session.mu.Unlock()
		return fmt.Errorf("session %s is not running", sessionID)
	}
	stdin := session.stdin
	session.mu.Unlock()

	if stdin == nil {
		return fmt.Errorf("session %s has no writable stdin", sessionID)
	}

	if data != "" {
		if _, err := io.WriteString(stdin, data); err != nil {
			return err
		}
	}
	if eof {
		return stdin.Close()
	}
	return nil
}

func (pm *ProcessManager) Kill(sessionID string) (bool, error) {
	session, ok := pm.getSession(sessionID)
	if !ok {
		return false, ErrProcessSessionNotFound
	}

	session.mu.Lock()
	if session.Status != "running" {
		session.mu.Unlock()
		return false, nil
	}
	session.killRequested = true
	cmd := session.cmd
	cancel := session.cancel
	session.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if cmd != nil {
		_ = terminateProcessTree(cmd)
	}
	return true, nil
}

func (pm *ProcessManager) Clear(sessionID string) error {
	session, ok := pm.getSession(sessionID)
	if !ok {
		return ErrProcessSessionNotFound
	}

	session.mu.Lock()
	running := session.Status == "running"
	session.mu.Unlock()
	if running {
		return ErrProcessSessionRunning
	}

	pm.mu.Lock()
	delete(pm.sessions, sessionID)
	pm.mu.Unlock()
	return nil
}

func (pm *ProcessManager) Remove(sessionID string) (bool, error) {
	session, ok := pm.getSession(sessionID)
	if !ok {
		return false, ErrProcessSessionNotFound
	}

	session.mu.Lock()
	running := session.Status == "running"
	session.mu.Unlock()
	if running {
		_, err := pm.Kill(sessionID)
		return false, err
	}

	pm.mu.Lock()
	delete(pm.sessions, sessionID)
	pm.mu.Unlock()
	return true, nil
}

func (pm *ProcessManager) getSession(sessionID string) (*processSession, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	session, ok := pm.sessions[sessionID]
	return session, ok
}

func (s *processSession) snapshot() ProcessSessionSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked()
}

func (s *processSession) drainPending() (string, ProcessSessionSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pending := s.pending
	s.pending = ""
	return pending, s.snapshotLocked()
}

func (s *processSession) outputSnapshot() (string, ProcessSessionSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.output, s.snapshotLocked()
}

func (s *processSession) snapshotLocked() ProcessSessionSnapshot {
	snapshot := ProcessSessionSnapshot{
		SessionID: s.ID,
		Status:    s.Status,
		Command:   s.Command,
		CWD:       s.CWD,
		PID:       s.PID,
		StartedAt: s.StartedAt.Format(time.RFC3339),
		UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
		ExitCode:  s.ExitCode,
		ExitError: s.ExitError,
		Truncated: s.Truncated,
	}
	if !s.EndedAt.IsZero() {
		snapshot.EndedAt = s.EndedAt.Format(time.RFC3339)
	}
	return snapshot
}

func (s *processSession) signalNotifyLocked() {
	select {
	case s.notify <- struct{}{}:
	default:
	}
}

func appendWithCap(current, appendChunk string, max int, truncated bool) (string, bool) {
	if max <= 0 {
		return current + appendChunk, truncated
	}

	combined := current + appendChunk
	if len(combined) <= max {
		return combined, truncated
	}

	return combined[len(combined)-max:], true
}

func normalizeOutputLines(output string) []string {
	if output == "" {
		return []string{}
	}
	normalized := strings.ReplaceAll(output, "\r\n", "\n")
	normalized = strings.TrimSuffix(normalized, "\n")
	if normalized == "" {
		return []string{}
	}
	return strings.Split(normalized, "\n")
}

func extractExitCode(err error) (int, bool) {
	if err == nil {
		return 0, false
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), true
	}
	return 0, false
}

type ProcessTool struct {
	processes *ProcessManager
}

func NewProcessTool(processes *ProcessManager) *ProcessTool {
	return &ProcessTool{processes: processes}
}

func (t *ProcessTool) Name() string {
	return "process"
}

func (t *ProcessTool) Description() string {
	return "Manage background exec sessions: list, poll, log, write, kill, clear, remove."
}

func (t *ProcessTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"list", "poll", "log", "write", "kill", "clear", "remove"},
				"description": "Process action",
			},
			"session_id": map[string]any{
				"type":        "string",
				"description": "Session ID returned by exec background mode",
			},
			"data": map[string]any{
				"type":        "string",
				"description": "Input data for write action",
			},
			"eof": map[string]any{
				"type":        "boolean",
				"description": "Close stdin after write",
			},
			"offset": map[string]any{
				"type":        "integer",
				"description": "Log line offset",
				"minimum":     0.0,
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Max log lines to return",
				"minimum":     0.0,
			},
			"timeout_ms": map[string]any{
				"type":        "integer",
				"description": "Poll wait timeout in milliseconds",
				"minimum":     0.0,
			},
		},
		"required": []string{"action"},
	}
}

func (t *ProcessTool) Execute(_ context.Context, args map[string]any) *ToolResult {
	if t.processes == nil {
		return ErrorResult("process manager not configured")
	}

	action, ok := getStringArg(args, "action")
	if !ok || strings.TrimSpace(action) == "" {
		return ErrorResult("action is required")
	}
	action = strings.ToLower(strings.TrimSpace(action))

	switch action {
	case "list":
		sessions := t.processes.ListSnapshots()
		return marshalSilentJSON(map[string]any{
			"count":    len(sessions),
			"sessions": sessions,
		})
	case "poll":
		sessionID, ok := getStringArg(args, "session_id")
		if !ok || strings.TrimSpace(sessionID) == "" {
			return ErrorResult("session_id is required for poll")
		}
		timeoutMS, err := parseOptionalIntArg(args, "timeout_ms", 0, 0, 5*60*1000)
		if err != nil {
			return ErrorResult(err.Error())
		}
		result, err := t.processes.Poll(strings.TrimSpace(sessionID), time.Duration(timeoutMS)*time.Millisecond)
		if err != nil {
			return ErrorResult(err.Error())
		}
		return marshalSilentJSON(result)
	case "log":
		sessionID, ok := getStringArg(args, "session_id")
		if !ok || strings.TrimSpace(sessionID) == "" {
			return ErrorResult("session_id is required for log")
		}
		offset, err := parseOptionalIntArg(args, "offset", 0, 0, 1_000_000)
		if err != nil {
			return ErrorResult(err.Error())
		}
		limit, err := parseOptionalIntArg(args, "limit", 0, 0, 1_000_000)
		if err != nil {
			return ErrorResult(err.Error())
		}
		_, hasOffset := args["offset"]
		_, hasLimit := args["limit"]
		useDefaultTail := !hasOffset && !hasLimit
		result, err := t.processes.Log(strings.TrimSpace(sessionID), offset, limit, useDefaultTail)
		if err != nil {
			return ErrorResult(err.Error())
		}
		return marshalSilentJSON(result)
	case "write":
		sessionID, ok := getStringArg(args, "session_id")
		if !ok || strings.TrimSpace(sessionID) == "" {
			return ErrorResult("session_id is required for write")
		}
		data, _ := getStringArg(args, "data")
		eof, err := parseBoolArg(args, "eof", false)
		if err != nil {
			return ErrorResult(err.Error())
		}
		if err := t.processes.Write(strings.TrimSpace(sessionID), data, eof); err != nil {
			return ErrorResult(err.Error())
		}
		return marshalSilentJSON(map[string]any{
			"status":     "ok",
			"action":     "write",
			"session_id": strings.TrimSpace(sessionID),
		})
	case "kill":
		sessionID, ok := getStringArg(args, "session_id")
		if !ok || strings.TrimSpace(sessionID) == "" {
			return ErrorResult("session_id is required for kill")
		}
		signaled, err := t.processes.Kill(strings.TrimSpace(sessionID))
		if err != nil {
			return ErrorResult(err.Error())
		}
		return marshalSilentJSON(map[string]any{
			"status":      "ok",
			"action":      "kill",
			"session_id":  strings.TrimSpace(sessionID),
			"kill_signal": signaled,
		})
	case "clear":
		sessionID, ok := getStringArg(args, "session_id")
		if !ok || strings.TrimSpace(sessionID) == "" {
			return ErrorResult("session_id is required for clear")
		}
		if err := t.processes.Clear(strings.TrimSpace(sessionID)); err != nil {
			return ErrorResult(err.Error())
		}
		return marshalSilentJSON(map[string]any{
			"status":     "ok",
			"action":     "clear",
			"session_id": strings.TrimSpace(sessionID),
		})
	case "remove":
		sessionID, ok := getStringArg(args, "session_id")
		if !ok || strings.TrimSpace(sessionID) == "" {
			return ErrorResult("session_id is required for remove")
		}
		removed, err := t.processes.Remove(strings.TrimSpace(sessionID))
		if err != nil {
			return ErrorResult(err.Error())
		}
		return marshalSilentJSON(map[string]any{
			"status":     "ok",
			"action":     "remove",
			"session_id": strings.TrimSpace(sessionID),
			"removed":    removed,
		})
	default:
		return ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}
}

func marshalSilentJSON(payload any) *ToolResult {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to encode process payload: %v", err))
	}
	return SilentResult(string(data))
}
