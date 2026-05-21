package agent

import (
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	taskregistry "github.com/sipeed/picoclaw/pkg/tasks"
)

func (al *AgentLoop) taskRegistryForWorkspace(workspace string) *taskregistry.Registry {
	if al == nil {
		return nil
	}
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return nil
	}
	if existing, ok := al.taskRegistries.Load(workspace); ok {
		if registry, ok := existing.(*taskregistry.Registry); ok {
			return registry
		}
	}
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(workspace))
	actual, _ := al.taskRegistries.LoadOrStore(workspace, registry)
	if stored, ok := actual.(*taskregistry.Registry); ok {
		if stored == registry {
			al.reconcilePendingTerminalTaskDelivery(workspace, stored)
		}
		return stored
	}
	al.reconcilePendingTerminalTaskDelivery(workspace, registry)
	return registry
}

func (al *AgentLoop) updateAsyncTaskDeliveryStatus(
	workspace string,
	taskID string,
	status taskregistry.DeliveryStatus,
	errorSummary string,
) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" || status == "" {
		return
	}
	registry := al.taskRegistryForWorkspace(workspace)
	if registry == nil {
		return
	}
	_ = registry.Update(taskID, func(rec *taskregistry.Record) {
		rec.DeliveryStatus = status
		if strings.TrimSpace(errorSummary) != "" {
			rec.Error = strings.TrimSpace(errorSummary)
		}
	})
}

func (al *AgentLoop) reconcilePendingTerminalTaskDelivery(workspace string, registry *taskregistry.Registry) {
	if registry == nil {
		return
	}
	pending := registry.ListPendingTerminalDelivery()
	if len(pending) == 0 {
		return
	}
	now := time.Now().UnixMilli()
	for _, rec := range pending {
		taskID := rec.TaskID
		_ = registry.Update(taskID, func(rec *taskregistry.Record) {
			rec.DeliveryStatus = taskregistry.DeliveryParentMissing
			rec.LastEventAt = now
			if strings.TrimSpace(rec.Error) == "" {
				rec.Error = "pending delivery was not completed before runtime restart/reload"
			}
		})
	}
	logger.WarnCF("agent", "Reconciled stale pending task deliveries",
		map[string]any{
			"workspace": workspace,
			"count":     len(pending),
		})
}
