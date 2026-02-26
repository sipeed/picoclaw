package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/tools"
)

func TestRunTaskAudit_DetectsMissedAndQualityIssues(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.Audit.Enabled = true
	cfg.Audit.LookbackMinutes = 360
	cfg.Audit.Supervisor.Enabled = false
	cfg.Orchestration.DefaultTaskTimeoutSeconds = 1
	cfg.Orchestration.RetryLimitPerTask = 2

	loop := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})
	ledger := loop.GetTaskLedger()
	if ledger == nil {
		t.Fatal("expected task ledger")
	}

	oldTS := time.Now().Add(-10 * time.Minute).UnixMilli()
	_ = ledger.UpsertTask(tools.TaskLedgerEntry{
		ID:          "task-planned-old",
		Status:      tools.TaskStatusPlanned,
		Intent:      "do something",
		CreatedAtMS: oldTS,
		UpdatedAtMS: oldTS,
	})
	_ = ledger.UpsertTask(tools.TaskLedgerEntry{
		ID:          "task-completed-empty",
		Status:      tools.TaskStatusCompleted,
		CreatedAtMS: oldTS,
		UpdatedAtMS: oldTS,
		Result:      "",
	})

	report, err := loop.RunTaskAudit(context.Background())
	if err != nil {
		t.Fatalf("RunTaskAudit error: %v", err)
	}
	if report == nil {
		t.Fatal("expected non-nil report")
	}
	if len(report.Findings) < 2 {
		t.Fatalf("expected at least 2 findings, got %d", len(report.Findings))
	}

	hasMissed := false
	hasQuality := false
	for _, f := range report.Findings {
		if f.TaskID == "task-planned-old" && f.Category == "missed" {
			hasMissed = true
		}
		if f.TaskID == "task-completed-empty" && f.Category == "quality" {
			hasQuality = true
		}
	}
	if !hasMissed {
		t.Fatal("expected missed finding for overdue planned task")
	}
	if !hasQuality {
		t.Fatal("expected quality finding for empty completed task")
	}
}

func TestParseSupervisorReview_EmbeddedJSON(t *testing.T) {
	raw := "review result:\n{\"score\":0.42,\"issues\":[{\"category\":\"quality\",\"severity\":\"high\",\"message\":\"missing evidence\"}]}"
	review, err := parseSupervisorReview(raw)
	if err != nil {
		t.Fatalf("parseSupervisorReview error: %v", err)
	}
	if review.Score != 0.42 {
		t.Fatalf("score = %v, want 0.42", review.Score)
	}
	if len(review.Issues) != 1 {
		t.Fatalf("issues len = %d, want 1", len(review.Issues))
	}
	if !strings.EqualFold(review.Issues[0].Category, "quality") {
		t.Fatalf("issue category = %q", review.Issues[0].Category)
	}
}

func TestApplyAutoRemediation_RetriesMissedTasksWithCooldown(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.Audit.Enabled = true
	cfg.Audit.AutoRemediation = "retry_missed"
	cfg.Audit.MaxAutoRemediationsPerCycle = 5
	cfg.Audit.RemediationCooldownMinutes = 60

	loop := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})
	ledger := loop.GetTaskLedger()
	if ledger == nil {
		t.Fatal("expected task ledger")
	}

	taskID := "task-failed-1"
	_ = ledger.UpsertTask(tools.TaskLedgerEntry{
		ID:            taskID,
		Status:        tools.TaskStatusFailed,
		Intent:        "Write a short status update about the project.",
		OriginChannel: "telegram",
		OriginChatID:  "chat-1",
		CreatedAtMS:   time.Now().Add(-10 * time.Minute).UnixMilli(),
	})

	report := &AuditReport{
		GeneratedAt: time.Now(),
		Lookback:    180 * time.Minute,
		TotalTasks:  1,
		Findings: []AuditFinding{
			{
				TaskID:         taskID,
				Category:       "missed",
				Severity:       "medium",
				Message:        "Task failed and still has retry budget.",
				Recommendation: "Retry this task automatically or manually.",
			},
		},
	}

	loop.applyAutoRemediation(context.Background(), report)

	entry, ok := ledger.Get(taskID)
	if !ok {
		t.Fatal("expected task in ledger")
	}
	if entry.RetryCount != 1 {
		t.Fatalf("RetryCount = %d, want %d", entry.RetryCount, 1)
	}
	hasSpawned := false
	for _, r := range entry.Remediations {
		if strings.EqualFold(r.Action, "retry") && strings.EqualFold(r.Status, "spawned") {
			hasSpawned = true
			break
		}
	}
	if !hasSpawned {
		t.Fatalf("expected spawned retry remediation, got: %+v", entry.Remediations)
	}

	// Second run should respect cooldown and not spawn another retry.
	loop.applyAutoRemediation(context.Background(), report)
	entry2, _ := ledger.Get(taskID)
	if entry2.RetryCount != 1 {
		t.Fatalf("RetryCount after cooldown check = %d, want %d", entry2.RetryCount, 1)
	}
	spawnedCount := 0
	spawnedTaskID := ""
	for _, r := range entry2.Remediations {
		if strings.EqualFold(r.Action, "retry") && strings.EqualFold(r.Status, "spawned") {
			spawnedCount++
			parts := strings.Fields(r.Note)
			if len(parts) >= 2 {
				spawnedTaskID = parts[1]
			}
		}
	}
	if spawnedCount != 1 {
		t.Fatalf("spawned remediation count = %d, want %d", spawnedCount, 1)
	}

	// Ensure the spawned subagent finishes before TempDir cleanup.
	if spawnedTaskID == "" {
		t.Fatal("expected spawned task id in remediation note")
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		subTask, ok := ledger.Get(spawnedTaskID)
		if ok && (subTask.Status == tools.TaskStatusCompleted ||
			subTask.Status == tools.TaskStatusFailed ||
			subTask.Status == tools.TaskStatusCancelled) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for spawned task %s to finish", spawnedTaskID)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
