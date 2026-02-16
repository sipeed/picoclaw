package tools

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/cron"
)

func TestCronTool_ExecuteJobCommandRespectsWorkspaceRestriction(t *testing.T) {
	msgBus := bus.NewMessageBus()
	service := cron.NewCronService(filepath.Join(t.TempDir(), "jobs.json"), nil)
	tool := NewCronTool(service, nil, msgBus, t.TempDir())

	job := &cron.CronJob{}
	job.Payload.Command = "cat /etc/passwd"
	job.Payload.Channel = "cli"
	job.Payload.To = "direct"

	tool.ExecuteJob(context.Background(), job)
	outbound, ok := msgBus.SubscribeOutbound(context.Background())
	if !ok {
		t.Fatalf("expected outbound message")
	}
	if outbound.Content == "" || outbound.Content == "ok" {
		t.Fatalf("expected error output from blocked command")
	}
}
