package sandbox

import (
	"context"
	"os"
	"strings"

	"github.com/sipeed/picoclaw/pkg/logger"
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

// Manager is implemented by the scoped sandbox manager.
// It resolves the specific Sandbox instance to use for the current execution context
// (e.g. based on session key / scope). Only the manager needs this; leaf sandboxes
// (HostSandbox, ContainerSandbox) implement Sandbox directly and are the resolved result.
type Manager interface {
	Sandbox
	// Resolve returns the concrete Sandbox instance to use for the given context.
	Resolve(ctx context.Context) (Sandbox, error)
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

type sessionContextKey struct{}

// WithSessionKey returns a derived context carrying the current routing session key.
func WithSessionKey(ctx context.Context, sessionKey string) context.Context {
	return context.WithValue(ctx, sessionContextKey{}, sessionKey)
}

// SessionKeyFromContext returns the session key attached by WithSessionKey.
func SessionKeyFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(sessionContextKey{}).(string)
	return v
}

type sandboxContextKey struct{}

// WithSandbox returns a derived context carrying the current sandbox instance.
func WithSandbox(ctx context.Context, sb Sandbox) context.Context {
	return context.WithValue(ctx, sandboxContextKey{}, sb)
}

// FromContext returns the sandbox instance attached by WithSandbox.
// If no pre-resolved sandbox exists, it attempts to resolve one via the Manager in context.
func FromContext(ctx context.Context) Sandbox {
	if ctx == nil {
		return nil
	}
	// 1. Try pre-resolved sandbox
	if v, ok := ctx.Value(sandboxContextKey{}).(Sandbox); ok && v != nil {
		return v
	}
	// 2. Try on-demand resolution via Manager
	if m := managerFromContext(ctx); m != nil {
		sb, err := m.Resolve(ctx)
		if err != nil {
			logger.WarnCF("sandbox", "FromContext: manager.Resolve failed", map[string]any{"error": err.Error()})
		} else if sb != nil {
			return sb
		}
	}
	return nil
}

type managerContextKey struct{}

// WithManager returns a derived context carrying the sandbox manager.
func WithManager(ctx context.Context, m Manager) context.Context {
	return context.WithValue(ctx, managerContextKey{}, m)
}

// managerFromContext returns the manager attached by WithManager.
func managerFromContext(ctx context.Context) Manager {
	if ctx == nil {
		return nil
	}
	v, _ := ctx.Value(managerContextKey{}).(Manager)
	return v
}

// FsBridge abstracts sandbox-scoped file I/O.
type FsBridge interface {
	// ReadFile reads a file from sandbox-visible filesystem.
	ReadFile(ctx context.Context, path string) ([]byte, error)
	// WriteFile writes data to a sandbox-visible path.
	// When mkdir is true, missing parent directories should be created.
	WriteFile(ctx context.Context, path string, data []byte, mkdir bool) error
	// ReadDir reads the named directory and returns a list of directory entries.
	ReadDir(ctx context.Context, path string) ([]os.DirEntry, error)
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
