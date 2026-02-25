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
