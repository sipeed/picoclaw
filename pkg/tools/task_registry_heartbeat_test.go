package tools

import (
	"context"
	"testing"
	"time"

	taskregistry "github.com/sipeed/picoclaw/pkg/tasks"
)

func TestStartTaskRegistryHeartbeatEveryUpdatesRunningTask(t *testing.T) {
	registry := taskregistry.NewRegistry("")
	old := time.Now().Add(-time.Hour).UnixMilli()
	if err := registry.Upsert(taskregistry.Record{
		TaskID:         "delegate-1",
		Runtime:        taskregistry.RuntimeDelegate,
		Task:           "long task",
		Status:         taskregistry.StatusRunning,
		DeliveryStatus: taskregistry.DeliveryPending,
		CreatedAt:      old,
		LastEventAt:    old,
	}); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	stop := startTaskRegistryHeartbeatEvery(
		context.Background(),
		registry,
		"delegate-1",
		"delegate child turn is still running",
		time.Millisecond,
	)
	defer stop()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		rec, _ := registry.Get("delegate-1")
		if rec.LastEventAt > old {
			if rec.ProgressSummary != "delegate child turn is still running" {
				t.Fatalf("ProgressSummary = %q", rec.ProgressSummary)
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	rec, _ := registry.Get("delegate-1")
	t.Fatalf("LastEventAt = %d, want > %d", rec.LastEventAt, old)
}
