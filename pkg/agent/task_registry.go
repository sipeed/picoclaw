package agent

import (
	"strings"

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
		return stored
	}
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
