package tools

import (
	"context"
	"time"

	taskregistry "github.com/sipeed/picoclaw/pkg/tasks"
)

const taskRegistryHeartbeatInterval = 30 * time.Second

func startTaskRegistryHeartbeat(
	ctx context.Context,
	registry *taskregistry.Registry,
	taskID string,
	progress string,
) func() {
	return startTaskRegistryHeartbeatEvery(ctx, registry, taskID, progress, taskRegistryHeartbeatInterval)
}

func startTaskRegistryHeartbeatEvery(
	ctx context.Context,
	registry *taskregistry.Registry,
	taskID string,
	progress string,
	interval time.Duration,
) func() {
	if registry == nil || taskID == "" || interval <= 0 {
		return func() {}
	}
	heartbeatCtx, cancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				_ = registry.Heartbeat(taskID, progress)
			}
		}
	}()
	return cancel
}
