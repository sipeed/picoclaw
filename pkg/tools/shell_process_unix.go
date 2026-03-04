//go:build !windows

package tools

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func prepareCommandForTermination(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func terminateProcessTree(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	pid := cmd.Process.Pid
	if pid <= 0 {
		return nil
	}

	// Kill the entire process group spawned by the shell command.
	_ = syscall.Kill(-pid, syscall.SIGKILL)

	// Some shells/background jobs may still leave descendants around
	// briefly; aggressively walk /proc and kill child processes too.
	killDescendants(pid)

	// Fallback kill on the shell process itself.
	_ = cmd.Process.Kill()

	return nil
}

func killDescendants(ppid int) {
	if ppid <= 0 {
		return
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		childPID, err := strconv.Atoi(e.Name())
		if err != nil || childPID <= 0 || childPID == ppid {
			continue
		}

		statPath := "/proc/" + e.Name() + "/stat"
		data, err := os.ReadFile(statPath)
		if err != nil {
			continue
		}

		// /proc/<pid>/stat: pid (comm) state ppid ...
		raw := string(data)
		end := strings.LastIndex(raw, ")")
		if end == -1 || end+2 >= len(raw) {
			continue
		}
		fields := strings.Fields(raw[end+2:])
		if len(fields) < 2 {
			continue
		}
		parent, err := strconv.Atoi(fields[1])
		if err != nil || parent != ppid {
			continue
		}

		// Recurse first, then kill child process/group.
		killDescendants(childPID)
		_ = syscall.Kill(-childPID, syscall.SIGKILL)
		_ = syscall.Kill(childPID, syscall.SIGKILL)
	}
}
