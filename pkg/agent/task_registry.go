package agent

import (
	"strconv"
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
			al.reconcileActiveTasksAfterRegistryRestore(workspace, stored)
			al.reconcilePendingTerminalTaskDelivery(workspace, stored)
		}
		return stored
	}
	al.reconcileActiveTasksAfterRegistryRestore(workspace, registry)
	al.reconcilePendingTerminalTaskDelivery(workspace, registry)
	return registry
}

// TaskRegistryForWorkspace returns the durable task registry shared by agent
// tools and gateway-managed runtimes for the given workspace.
func (al *AgentLoop) TaskRegistryForWorkspace(workspace string) *taskregistry.Registry {
	return al.taskRegistryForWorkspace(workspace)
}

func (al *AgentLoop) updateAsyncTaskDeliveryStatus(
	workspace string,
	taskID string,
	status taskregistry.DeliveryStatus,
	completionID string,
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
		if strings.TrimSpace(completionID) != "" {
			rec.LastCompletionID = strings.TrimSpace(completionID)
		}
		if status == taskregistry.DeliveryDelivered || status == taskregistry.DeliverySessionQueued ||
			status == taskregistry.DeliveryNotApplicable {
			rec.DeliveredAt = time.Now().UnixMilli()
			rec.DeliveryError = ""
		}
		if strings.TrimSpace(errorSummary) != "" {
			rec.DeliveryError = strings.TrimSpace(errorSummary)
			if strings.TrimSpace(rec.Error) == "" {
				rec.Error = strings.TrimSpace(errorSummary)
			}
		}
	})
}

func (al *AgentLoop) recordAsyncTaskDeliveryDecision(
	workspace string,
	decision AsyncDeliveryDecision,
	completionID string,
	sourceTool string,
) {
	taskID := strings.TrimSpace(decision.TaskID)
	if taskID == "" {
		return
	}
	registry := al.taskRegistryForWorkspace(workspace)
	if registry == nil {
		return
	}
	_ = registry.AppendEvent(taskID, taskregistry.EventTaskDeliveryDecision, map[string]string{
		"completion_id":  completionID,
		"source_tool":    sourceTool,
		"mode":           string(decision.DeliveryMode),
		"will_user":      boolString(decision.PublishToUser),
		"will_parent":    boolString(decision.QueueParent),
		"parent_handled": boolString(decision.ParentHandled),
		"is_error":       boolString(decision.IsError),
		"content_len":    strconv.Itoa(decision.ContentLen),
		"for_user_len":   strconv.Itoa(decision.ForUserLen),
		"media_count":    strconv.Itoa(decision.MediaCount),
	})
}

func (al *AgentLoop) asyncTaskDeliveryAlreadyHandled(
	workspace string,
	taskID string,
	completionID string,
) bool {
	taskID = strings.TrimSpace(taskID)
	completionID = strings.TrimSpace(completionID)
	if taskID == "" || completionID == "" {
		return false
	}
	registry := al.taskRegistryForWorkspace(workspace)
	if registry == nil {
		return false
	}
	rec, ok := registry.Get(taskID)
	if !ok || strings.TrimSpace(rec.LastCompletionID) != completionID {
		return false
	}
	switch rec.DeliveryStatus {
	case taskregistry.DeliveryDelivered, taskregistry.DeliverySessionQueued, taskregistry.DeliveryNotApplicable:
		return true
	default:
		return false
	}
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

func (al *AgentLoop) reconcileActiveTasksAfterRegistryRestore(workspace string, registry *taskregistry.Registry) {
	if registry == nil {
		return
	}
	active := registry.ListActive()
	if len(active) == 0 {
		return
	}
	reason := "task was still active when the runtime registry was restored; previous runtime owner is no longer alive"
	count, err := registry.MarkActiveLost(reason)
	if count == 0 {
		return
	}
	logger.WarnCF("agent", "Reconciled active tasks from previous runtime as lost",
		map[string]any{
			"workspace": workspace,
			"count":     count,
			"error":     errString(err),
		})
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
