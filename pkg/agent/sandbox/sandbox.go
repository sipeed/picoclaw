package sandbox

import (
	"context"
	"strings"
)

// Sandbox abstracts command execution and filesystem access.
type Sandbox interface {
	// Start initializes sandbox runtime dependencies.
	// Implementations should prepare resources that are expensive to set up lazily
	// (for example, container client connectivity checks).
	Start(ctx context.Context) error
	// Prune performs sandbox resource reclamation.
	// Implementations should release reclaimable runtime resources and remove
	// sandbox artifacts (for example containers) according to their policy.
	// It should be safe to call multiple times.
	Prune(ctx context.Context) error
	// Exec runs a command in sandbox context.
	// Command/Args semantics follow ExecRequest; a non-zero exit code should be
	// returned in ExecResult.ExitCode, while transport/runtime failures return error.
	Exec(ctx context.Context, req ExecRequest) (*ExecResult, error)
	// ExecStream runs a command and emits runtime events.
	// Implementations should emit stdout/stderr chunks as they arrive and a final
	// exit event when command execution completes.
	ExecStream(ctx context.Context, req ExecRequest, onEvent func(ExecEvent) error) (*ExecResult, error)
	// Fs returns the sandbox-aware filesystem bridge.
	Fs() FsBridge
}

// ExecRequest describes a command execution request for Sandbox.Exec.
type ExecRequest struct {
	// Command is the program or shell command to execute.
	Command string
	// Args are optional argv items; when empty, implementations may execute
	// Command through a shell.
	Args []string
	// WorkingDir is an optional path scoped to the sandbox workspace.
	WorkingDir string
	// TimeoutMs is an optional timeout in milliseconds; 0 means implementation default.
	TimeoutMs int64
}

// ExecResult is the normalized result returned by Sandbox.Exec.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// ExecEventType identifies the class of event emitted by Sandbox.ExecStream.
type ExecEventType string

const (
	ExecEventStdout ExecEventType = "stdout"
	ExecEventStderr ExecEventType = "stderr"
	ExecEventExit   ExecEventType = "exit"
)

// ExecEvent is the streaming payload emitted during Sandbox.ExecStream.
type ExecEvent struct {
	Type     ExecEventType
	Chunk    []byte
	ExitCode int
}

// FsBridge abstracts sandbox-scoped file I/O.
type FsBridge interface {
	// ReadFile reads a file from sandbox-visible filesystem.
	ReadFile(ctx context.Context, path string) ([]byte, error)
	// WriteFile writes data to a sandbox-visible path.
	// When mkdir is true, missing parent directories should be created.
	WriteFile(ctx context.Context, path string, data []byte, mkdir bool) error
}

func aggregateExecStream(execFn func(onEvent func(ExecEvent) error) (*ExecResult, error)) (*ExecResult, error) {
	var stdoutBuilder strings.Builder
	var stderrBuilder strings.Builder
	exitCode := 0

	res, err := execFn(func(event ExecEvent) error {
		switch event.Type {
		case ExecEventStdout:
			_, _ = stdoutBuilder.Write(event.Chunk)
		case ExecEventStderr:
			_, _ = stderrBuilder.Write(event.Chunk)
		case ExecEventExit:
			exitCode = event.ExitCode
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	out := &ExecResult{
		Stdout:   stdoutBuilder.String(),
		Stderr:   stderrBuilder.String(),
		ExitCode: exitCode,
	}
	if res == nil {
		return out, nil
	}
	if res.Stdout != "" || out.Stdout == "" {
		out.Stdout = res.Stdout
	}
	if res.Stderr != "" || out.Stderr == "" {
		out.Stderr = res.Stderr
	}
	if res.ExitCode != 0 || exitCode == 0 {
		out.ExitCode = res.ExitCode
	}
	return out, nil
}
