package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/cron"
)

// readOutbound reads the next outbound message with a timeout.
func readOutbound(t *testing.T, msgBus *bus.MessageBus, timeout time.Duration) (bus.OutboundMessage, bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	msg, ok := msgBus.SubscribeOutbound(ctx)
	if !ok {
		t.Fatal("timed out or bus closed waiting for outbound message")
	}
	return msg, ok
}

// stubJobExecutor satisfies JobExecutor for testing without a real agent.
type stubJobExecutor struct{}

func (s *stubJobExecutor) ProcessDirectWithChannel(_ context.Context, _, _, _, _ string) (string, error) {
	return "ok", nil
}

// TestCronTool_ExecuteJob_BlocksDangerousCommands verifies that commands
// executed via the cron scheduler go through the same risk classifier
// as agent-originated ExecTool calls (acceptance criterion AC-8).
func TestCronTool_ExecuteJob_BlocksDangerousCommands(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron.json")

	cronSvc := cron.NewCronService(storePath, func(_ *cron.CronJob) (string, error) {
		return "ok", nil
	})

	msgBus := bus.NewMessageBus()
	defer msgBus.Close()

	workspace := t.TempDir()

	ct, err := NewCronTool(
		cronSvc,
		&stubJobExecutor{},
		msgBus,
		workspace,
		true,
		5*time.Second,
		&config.Config{}, // default config → risk threshold = medium
	)
	if err != nil {
		t.Fatalf("NewCronTool: %v", err)
	}

	tests := []struct {
		name    string
		command string
	}{
		{"sudo", "sudo ls"},
		{"rm -rf", "rm -rf /"},
		{"shutdown", "shutdown -h now"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &cron.CronJob{
				ID:   "test-" + tt.name,
				Name: tt.name,
				Payload: cron.CronPayload{
					Command: tt.command,
					Channel: "test",
					To:      "test-chat",
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			ct.ExecuteJob(ctx, job)

			msg, _ := readOutbound(t, msgBus, 3*time.Second)
			if !strings.Contains(msg.Content, "Error executing scheduled command") {
				t.Errorf("expected blocked error, got: %s", msg.Content)
			}
			if !strings.Contains(msg.Content, "Command blocked by risk classifier") {
				t.Errorf("expected risk classifier message, got: %s", msg.Content)
			}
		})
	}
}

// TestCronTool_ExecuteJob_AllowsSafeCommands verifies harmless commands
// still execute normally through the cron path.
func TestCronTool_ExecuteJob_AllowsSafeCommands(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron.json")

	cronSvc := cron.NewCronService(storePath, func(_ *cron.CronJob) (string, error) {
		return "ok", nil
	})

	msgBus := bus.NewMessageBus()
	defer msgBus.Close()

	workspace := t.TempDir()
	// Create a file to read so the command has visible output.
	os.WriteFile(filepath.Join(workspace, "hello.txt"), []byte("cron-test"), 0o644)

	ct, err := NewCronTool(
		cronSvc,
		&stubJobExecutor{},
		msgBus,
		workspace,
		true,
		5*time.Second,
		&config.Config{},
	)
	if err != nil {
		t.Fatalf("NewCronTool: %v", err)
	}

	job := &cron.CronJob{
		ID:   "safe-echo",
		Name: "safe echo",
		Payload: cron.CronPayload{
			Command: "echo hello",
			Channel: "test",
			To:      "test-chat",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ct.ExecuteJob(ctx, job)

	msg, _ := readOutbound(t, msgBus, 3*time.Second)
	if strings.Contains(msg.Content, "Error executing scheduled command") {
		t.Errorf("safe command should not be blocked: %s", msg.Content)
	}
	if !strings.Contains(msg.Content, "hello") {
		t.Errorf("expected 'hello' in output, got: %s", msg.Content)
	}
}
