package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/agent/sandbox"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/cron"
)

type cronStubSandbox struct {
	calls int
	last  sandbox.ExecRequest
	res   *sandbox.ExecResult
	err   error
}

func (s *cronStubSandbox) Start(ctx context.Context) error { return nil }
func (s *cronStubSandbox) Prune(ctx context.Context) error { return nil }
func (s *cronStubSandbox) Fs() sandbox.FsBridge            { return nil }
func (s *cronStubSandbox) Exec(ctx context.Context, req sandbox.ExecRequest) (*sandbox.ExecResult, error) {
	return s.ExecStream(ctx, req, nil)
}

func (s *cronStubSandbox) ExecStream(ctx context.Context, req sandbox.ExecRequest, onEvent func(sandbox.ExecEvent) error) (*sandbox.ExecResult, error) {
	s.calls++
	s.last = req
	if s.err != nil {
		return nil, s.err
	}
	if s.res != nil {
		if onEvent != nil {
			if s.res.Stdout != "" {
				if err := onEvent(sandbox.ExecEvent{Type: sandbox.ExecEventStdout, Chunk: []byte(s.res.Stdout)}); err != nil {
					return nil, err
				}
			}
			if s.res.Stderr != "" {
				if err := onEvent(sandbox.ExecEvent{Type: sandbox.ExecEventStderr, Chunk: []byte(s.res.Stderr)}); err != nil {
					return nil, err
				}
			}
			if err := onEvent(sandbox.ExecEvent{Type: sandbox.ExecEventExit, ExitCode: s.res.ExitCode}); err != nil {
				return nil, err
			}
		}
		return s.res, nil
	}
	if onEvent != nil {
		if err := onEvent(sandbox.ExecEvent{Type: sandbox.ExecEventStdout, Chunk: []byte("ok")}); err != nil {
			return nil, err
		}
		if err := onEvent(sandbox.ExecEvent{Type: sandbox.ExecEventExit, ExitCode: 0}); err != nil {
			return nil, err
		}
	}
	return &sandbox.ExecResult{Stdout: "ok", ExitCode: 0}, nil
}

type noopExecutor struct{}

func (n *noopExecutor) ProcessDirectWithChannel(ctx context.Context, content, sessionKey, channel, chatID string) (string, error) {
	return "ok", nil
}

func TestCronTool_ExecuteJob_BlocksDangerousCommandViaGuard(t *testing.T) {
	msgBus := bus.NewMessageBus()
	sb := &cronStubSandbox{}
	tool := &CronTool{
		msgBus:    msgBus,
		sandbox:   sb,
		execGuard: NewExecTool("", true),
	}

	job := &cron.CronJob{
		ID: "j1",
		Payload: cron.CronPayload{
			Command: "rm -rf /",
			Channel: "cli",
			To:      "direct",
		},
	}

	tool.ExecuteJob(context.Background(), job)
	if sb.calls != 0 {
		t.Fatalf("sandbox should not be called for blocked command, got %d calls", sb.calls)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	out, ok := msgBus.SubscribeOutbound(ctx)
	if !ok {
		t.Fatal("expected outbound message")
	}
	if !strings.Contains(out.Content, "blocked") {
		t.Fatalf("expected blocked message, got: %s", out.Content)
	}
}

func TestCronTool_ExecuteJob_AllowsSafeCommand(t *testing.T) {
	msgBus := bus.NewMessageBus()
	sb := &cronStubSandbox{res: &sandbox.ExecResult{Stdout: "safe", ExitCode: 0}}
	tool := &CronTool{
		msgBus:    msgBus,
		sandbox:   sb,
		execGuard: NewExecTool("/tmp/ws", true),
	}

	job := &cron.CronJob{
		ID: "j2",
		Payload: cron.CronPayload{
			Command: "echo safe",
			Channel: "cli",
			To:      "direct",
		},
	}
	tool.ExecuteJob(context.Background(), job)
	if sb.calls != 1 {
		t.Fatalf("expected sandbox call, got %d", sb.calls)
	}
	if sb.last.WorkingDir != "." {
		t.Fatalf("expected sandbox working dir '.', got %q", sb.last.WorkingDir)
	}
}
