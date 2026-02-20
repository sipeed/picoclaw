package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/cron"
)

type mockJobExecutor struct{}

func (m mockJobExecutor) ProcessDirectWithChannel(ctx context.Context, content, sessionKey, channel, chatID string) (string, error) {
	return "ok", nil
}

func TestCronToolListFiltersByChat(t *testing.T) {
	storePath := t.TempDir() + "/jobs.json"
	cs := cron.NewCronService(storePath, nil)

	schedule := cron.CronSchedule{Kind: "every", EveryMS: int64Ptr(60000)}
	if _, err := cs.AddJob("mine", schedule, "hello", true, "telegram", "chat-a"); err != nil {
		t.Fatalf("failed to add job: %v", err)
	}
	if _, err := cs.AddJob("other", schedule, "hi", true, "telegram", "chat-b"); err != nil {
		t.Fatalf("failed to add other job: %v", err)
	}

	cfg := &config.Config{}
	msgBus := bus.NewMessageBus()
	tool := NewCronTool(cs, mockJobExecutor{}, msgBus, t.TempDir(), false, 0, cfg)
	tool.SetContext("telegram", "chat-a")

	res := tool.Execute(context.Background(), map[string]interface{}{
		"action": "list",
	})
	if res.IsError {
		t.Fatalf("unexpected error: %s", res.ForLLM)
	}

	if !strings.Contains(res.ForLLM, "mine") {
		t.Fatalf("filtered list missing own job: %s", res.ForLLM)
	}
	if strings.Contains(res.ForLLM, "other") {
		t.Fatalf("filtered list leaked other chat jobs: %s", res.ForLLM)
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}
