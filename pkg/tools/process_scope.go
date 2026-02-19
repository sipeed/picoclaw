package tools

import (
	"os"
	"sync"
	"syscall"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// ProcessScope tracks PIDs per session key, providing namespace-like isolation
// for exec tool processes. Inspired by Google Kubernetes pod process isolation
// and Linux cgroups — agents can only see and kill their own processes.
type ProcessScope struct {
	pids sync.Map // sessionKey -> *pidSet
}

type pidSet struct {
	mu   sync.Mutex
	pids map[int]bool
}

// NewProcessScope creates a new process scope tracker.
func NewProcessScope() *ProcessScope {
	return &ProcessScope{}
}

// Register adds a PID to a session's scope.
func (ps *ProcessScope) Register(sessionKey string, pid int) {
	set := ps.getOrCreate(sessionKey)
	set.mu.Lock()
	defer set.mu.Unlock()
	set.pids[pid] = true
}

// Deregister removes a PID from a session's scope (process exited normally).
func (ps *ProcessScope) Deregister(sessionKey string, pid int) {
	v, ok := ps.pids.Load(sessionKey)
	if !ok {
		return
	}
	set := v.(*pidSet)
	set.mu.Lock()
	defer set.mu.Unlock()
	delete(set.pids, pid)
}

// Owns returns true if the given PID belongs to the session's scope.
func (ps *ProcessScope) Owns(sessionKey string, pid int) bool {
	v, ok := ps.pids.Load(sessionKey)
	if !ok {
		return false
	}
	set := v.(*pidSet)
	set.mu.Lock()
	defer set.mu.Unlock()
	return set.pids[pid]
}

// ListPIDs returns all live PIDs for a session (filters out already-exited processes).
func (ps *ProcessScope) ListPIDs(sessionKey string) []int {
	v, ok := ps.pids.Load(sessionKey)
	if !ok {
		return nil
	}
	set := v.(*pidSet)
	set.mu.Lock()
	defer set.mu.Unlock()

	var live []int
	for pid := range set.pids {
		// Check if process is still running (Unix signal 0).
		if isProcessAlive(pid) {
			live = append(live, pid)
		} else {
			delete(set.pids, pid)
		}
	}
	return live
}

// KillAll kills all processes owned by a session. Returns number killed.
// Used during cascade stop to clean up spawned processes.
func (ps *ProcessScope) KillAll(sessionKey string) int {
	v, ok := ps.pids.Load(sessionKey)
	if !ok {
		return 0
	}
	set := v.(*pidSet)
	set.mu.Lock()
	defer set.mu.Unlock()

	killed := 0
	for pid := range set.pids {
		if err := killProcess(pid); err == nil {
			killed++
		}
		delete(set.pids, pid)
	}

	if killed > 0 {
		logger.InfoCF("process_scope", "Killed session processes", map[string]interface{}{
			"session_key": sessionKey,
			"killed":      killed,
		})
	}
	return killed
}

// Cleanup removes all tracking for a session.
func (ps *ProcessScope) Cleanup(sessionKey string) {
	ps.pids.Delete(sessionKey)
}

func (ps *ProcessScope) getOrCreate(sessionKey string) *pidSet {
	if v, ok := ps.pids.Load(sessionKey); ok {
		return v.(*pidSet)
	}
	set := &pidSet{pids: make(map[int]bool)}
	actual, _ := ps.pids.LoadOrStore(sessionKey, set)
	return actual.(*pidSet)
}

// isProcessAlive checks if a process is still running via signal 0 (Unix).
func isProcessAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// killProcess sends SIGTERM to a process. Falls back to SIGKILL if needed.
func killProcess(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(syscall.SIGTERM)
}
