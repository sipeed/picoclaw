package tools

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	cronpkg "github.com/sipeed/picoclaw/pkg/cron"
)

func newCronToolForTest(t *testing.T) *CronTool {
	t.Helper()

	workspace := t.TempDir()
	storePath := filepath.Join(workspace, "cron", "jobs.json")
	cronService := cronpkg.NewCronService(storePath, nil)

	tool := NewCronTool(
		cronService,
		nil,
		bus.NewMessageBus(),
		workspace,
		true,
		5*time.Second,
		config.DefaultConfig(),
	)
	tool.SetContext("cli", "direct")
	return tool
}

func TestCronToolAddJob_UsesCronExprWhenZeroNumericFieldsPresent(t *testing.T) {
	tool := newCronToolForTest(t)

	result := tool.Execute(context.Background(), map[string]any{
		"action":        "add",
		"message":       "daily check",
		"at_seconds":    0,
		"every_seconds": 0,
		"cron_expr":     "*/5 * * * *",
		"timezone":      "Asia/Shanghai",
	})
	if result.IsError {
		t.Fatalf("expected add success, got error: %s", result.ForLLM)
	}

	jobs := tool.cronService.ListJobs(false)
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Schedule.Kind != "cron" {
		t.Fatalf("expected cron schedule kind, got %q", jobs[0].Schedule.Kind)
	}
	if jobs[0].Schedule.Expr != "*/5 * * * *" {
		t.Fatalf("expected cron expr to be preserved, got %q", jobs[0].Schedule.Expr)
	}
}

func TestCronToolAddJob_UsesEveryWhenAtIsZero(t *testing.T) {
	tool := newCronToolForTest(t)

	result := tool.Execute(context.Background(), map[string]any{
		"action":        "add",
		"message":       "hourly check",
		"at_seconds":    0,
		"every_seconds": 3600,
	})
	if result.IsError {
		t.Fatalf("expected add success, got error: %s", result.ForLLM)
	}

	jobs := tool.cronService.ListJobs(false)
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Schedule.Kind != "every" {
		t.Fatalf("expected every schedule kind, got %q", jobs[0].Schedule.Kind)
	}
	if jobs[0].Schedule.EveryMS == nil || *jobs[0].Schedule.EveryMS != 3600*1000 {
		t.Fatalf("expected everyMs=3600000, got %+v", jobs[0].Schedule.EveryMS)
	}
}

func TestCronToolAddJob_NegativeSecondsRejected(t *testing.T) {
	tool := newCronToolForTest(t)

	result := tool.Execute(context.Background(), map[string]any{
		"action":     "add",
		"message":    "invalid",
		"at_seconds": -1,
	})
	if !result.IsError {
		t.Fatalf("expected error for negative at_seconds, got success: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "at_seconds must be >= 0") {
		t.Fatalf("unexpected error text: %s", result.ForLLM)
	}
}
