package tools

import (
	"os"
	"os/exec"
	"testing"
)

func TestProcessScope_RegisterAndOwns(t *testing.T) {
	ps := NewProcessScope()

	ps.Register("session-1", 12345)
	ps.Register("session-1", 12346)
	ps.Register("session-2", 99999)

	if !ps.Owns("session-1", 12345) {
		t.Error("session-1 should own PID 12345")
	}
	if !ps.Owns("session-1", 12346) {
		t.Error("session-1 should own PID 12346")
	}
	if ps.Owns("session-1", 99999) {
		t.Error("session-1 should NOT own PID 99999")
	}
	if !ps.Owns("session-2", 99999) {
		t.Error("session-2 should own PID 99999")
	}
}

func TestProcessScope_Deregister(t *testing.T) {
	ps := NewProcessScope()

	ps.Register("session-1", 12345)
	ps.Deregister("session-1", 12345)

	if ps.Owns("session-1", 12345) {
		t.Error("should not own after deregister")
	}
}

func TestProcessScope_CrossSessionIsolation(t *testing.T) {
	ps := NewProcessScope()

	ps.Register("session-a", 100)
	ps.Register("session-b", 200)

	if ps.Owns("session-a", 200) {
		t.Error("session-a should not see session-b's processes")
	}
	if ps.Owns("session-b", 100) {
		t.Error("session-b should not see session-a's processes")
	}
}

func TestProcessScope_ListPIDs_FiltersDeadProcesses(t *testing.T) {
	ps := NewProcessScope()

	// Register current PID (alive) and a fake PID (dead)
	ps.Register("session-1", os.Getpid())
	ps.Register("session-1", 999999999) // almost certainly not a real PID

	live := ps.ListPIDs("session-1")

	// Current process should be in the list
	found := false
	for _, pid := range live {
		if pid == os.Getpid() {
			found = true
		}
		if pid == 999999999 {
			t.Error("dead PID should have been filtered out")
		}
	}
	if !found {
		t.Error("current process PID should be in live list")
	}
}

func TestProcessScope_KillAll(t *testing.T) {
	ps := NewProcessScope()

	// Start a real process we can kill
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start test process: %v", err)
	}
	pid := cmd.Process.Pid

	ps.Register("session-1", pid)

	killed := ps.KillAll("session-1")
	if killed != 1 {
		t.Errorf("expected 1 killed, got %d", killed)
	}

	// Reap the child process to prevent zombie (zombie still responds to signal 0).
	// cmd.Wait() blocks until the process exits and is reaped by the OS.
	err := cmd.Wait()
	if err == nil {
		t.Error("expected wait to return non-nil error after SIGTERM")
	}

	// After reaping, process should no longer be in the process table
	if isProcessAlive(pid) {
		t.Error("process should have been killed")
	}
}

func TestProcessScope_Cleanup(t *testing.T) {
	ps := NewProcessScope()

	ps.Register("session-1", os.Getpid())
	ps.Cleanup("session-1")

	if ps.Owns("session-1", os.Getpid()) {
		t.Error("should not own after cleanup")
	}
}

func TestProcessScope_EmptySession(t *testing.T) {
	ps := NewProcessScope()

	if ps.Owns("nonexistent", 12345) {
		t.Error("nonexistent session should not own anything")
	}

	pids := ps.ListPIDs("nonexistent")
	if len(pids) != 0 {
		t.Error("nonexistent session should have no PIDs")
	}

	killed := ps.KillAll("nonexistent")
	if killed != 0 {
		t.Error("killing nonexistent session should kill 0")
	}
}
