// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package hooks

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	// DefaultTimeout is the maximum time a hook script can run.
	DefaultTimeout = 30 * time.Second

	// MaxOutputBytes is the maximum bytes captured from hook stdout.
	MaxOutputBytes = 64 * 1024 // 64KB
)

// Executor runs hook commands via os/exec.
type Executor struct {
	Timeout        time.Duration
	MaxOutputBytes int
}

// NewExecutor creates an Executor with default settings.
func NewExecutor() *Executor {
	return &Executor{
		Timeout:        DefaultTimeout,
		MaxOutputBytes: MaxOutputBytes,
	}
}

// Run executes a command with the given stdin data and extra environment variables.
// It returns the stdout output (truncated to MaxOutputBytes) and any error.
// The command inherits the current process environment plus the extra env vars.
func (e *Executor) Run(ctx context.Context, command string, stdinData []byte, extraEnv []string) HookResult {
	if strings.TrimSpace(command) == "" {
		return HookResult{Err: fmt.Errorf("empty hook command")}
	}

	// Expand ~ in command path
	command = expandHome(command)

	timeout := e.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(cmdCtx, "powershell", "-NoProfile", "-NonInteractive", "-Command", command)
	} else {
		cmd = exec.CommandContext(cmdCtx, "sh", "-c", command)
	}

	// Inherit current env + hook-specific vars
	cmd.Env = append(os.Environ(), extraEnv...)

	// Pass payload via stdin
	if len(stdinData) > 0 {
		cmd.Stdin = bytes.NewReader(stdinData)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	maxOutput := e.MaxOutputBytes
	if maxOutput <= 0 {
		maxOutput = MaxOutputBytes
	}
	if len(output) > maxOutput {
		output = output[:maxOutput]
	}

	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			return HookResult{
				Output: output,
				Err:    fmt.Errorf("hook timed out after %v: %s", timeout, command),
			}
		}
		stderrStr := stderr.String()
		if len(stderrStr) > 1024 {
			stderrStr = stderrStr[:1024]
		}
		return HookResult{
			Output: output,
			Err:    fmt.Errorf("hook failed: %w (stderr: %s)", err, stderrStr),
		}
	}

	return HookResult{Output: output}
}

// expandHome replaces leading ~ with the user's home directory.
func expandHome(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if len(path) > 1 && path[1] == '/' {
		return home + path[1:]
	}
	return home
}
