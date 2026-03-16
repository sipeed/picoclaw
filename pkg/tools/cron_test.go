package tools

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/cron"
)

func newTestCronTool(t *testing.T) *CronTool {
	t.Helper()
	return newTestCronToolWithMinInterval(t, 0)
}

func newTestCronToolWithMinInterval(t *testing.T, minInterval int) *CronTool {
	t.Helper()
	storePath := filepath.Join(t.TempDir(), "cron.json")
	cronService := cron.NewCronService(storePath, nil)
	msgBus := bus.NewMessageBus()
	cfg := config.DefaultConfig()
	cfg.Tools.Cron.MinIntervalSeconds = minInterval
	tool, err := NewCronTool(cronService, nil, msgBus, t.TempDir(), true, 0, cfg)
	if err != nil {
		t.Fatalf("NewCronTool() error: %v", err)
	}
	return tool
}

// TestCronTool_CommandBlockedFromRemoteChannel verifies command scheduling is restricted to internal channels
func TestCronTool_CommandBlockedFromRemoteChannel(t *testing.T) {
	tool := newTestCronTool(t)
	ctx := WithToolContext(context.Background(), "telegram", "chat-1")
	result := tool.Execute(ctx, map[string]any{
		"action":          "add",
		"message":         "check disk",
		"command":         "df -h",
		"command_confirm": true,
		"at_seconds":      float64(60),
	})

	if !result.IsError {
		t.Fatal("expected command scheduling to be blocked from remote channel")
	}
	if !strings.Contains(result.ForLLM, "restricted to internal channels") {
		t.Errorf("expected 'restricted to internal channels', got: %s", result.ForLLM)
	}
}

// TestCronTool_CommandRequiresConfirm verifies command_confirm=true is required
func TestCronTool_CommandRequiresConfirm(t *testing.T) {
	tool := newTestCronTool(t)
	ctx := WithToolContext(context.Background(), "cli", "direct")
	result := tool.Execute(ctx, map[string]any{
		"action":     "add",
		"message":    "check disk",
		"command":    "df -h",
		"at_seconds": float64(60),
	})

	if !result.IsError {
		t.Fatal("expected error when command_confirm is missing")
	}
	if !strings.Contains(result.ForLLM, "command_confirm=true") {
		t.Errorf("expected 'command_confirm=true' message, got: %s", result.ForLLM)
	}
}

// TestCronTool_CommandAllowedFromInternalChannel verifies command scheduling works from internal channels
func TestCronTool_CommandAllowedFromInternalChannel(t *testing.T) {
	tool := newTestCronTool(t)
	ctx := WithToolContext(context.Background(), "cli", "direct")
	result := tool.Execute(ctx, map[string]any{
		"action":          "add",
		"message":         "check disk",
		"command":         "df -h",
		"command_confirm": true,
		"at_seconds":      float64(60),
	})

	if result.IsError {
		t.Fatalf("expected command scheduling to succeed from internal channel, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Cron job added") {
		t.Errorf("expected 'Cron job added', got: %s", result.ForLLM)
	}
}

// TestCronTool_AddJobRequiresSessionContext verifies fail-closed when channel/chatID missing
func TestCronTool_AddJobRequiresSessionContext(t *testing.T) {
	tool := newTestCronTool(t)
	result := tool.Execute(context.Background(), map[string]any{
		"action":     "add",
		"message":    "reminder",
		"at_seconds": float64(60),
	})

	if !result.IsError {
		t.Fatal("expected error when session context is missing")
	}
	if !strings.Contains(result.ForLLM, "no session context") {
		t.Errorf("expected 'no session context' message, got: %s", result.ForLLM)
	}
}

// TestCronTool_NonCommandJobAllowedFromRemoteChannel verifies regular reminders work from any channel
func TestCronTool_NonCommandJobAllowedFromRemoteChannel(t *testing.T) {
	tool := newTestCronTool(t)
	ctx := WithToolContext(context.Background(), "telegram", "chat-1")
	result := tool.Execute(ctx, map[string]any{
		"action":     "add",
		"message":    "time to stretch",
		"at_seconds": float64(600),
	})

	if result.IsError {
		t.Fatalf("expected non-command reminder to succeed from remote channel, got: %s", result.ForLLM)
	}
}

func TestCronTool_NonCommandJobDefaultsDeliverToFalse(t *testing.T) {
	tool := newTestCronTool(t)
	ctx := WithToolContext(context.Background(), "telegram", "chat-1")
	result := tool.Execute(ctx, map[string]any{
		"action":     "add",
		"message":    "send me a poem",
		"at_seconds": float64(600),
	})

	if result.IsError {
		t.Fatalf("expected non-command reminder to succeed, got: %s", result.ForLLM)
	}

	jobs := tool.cronService.ListJobs(false)
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Payload.Deliver {
		t.Fatal("expected deliver=false by default for non-command jobs")
	}
}

// TestCronTool_MinInterval_EveryBelowLimit verifies that every_seconds below min_interval is rejected
func TestCronTool_MinInterval_EveryBelowLimit(t *testing.T) {
	tool := newTestCronToolWithMinInterval(t, 60)
	ctx := WithToolContext(context.Background(), "cli", "direct")
	result := tool.Execute(ctx, map[string]any{
		"action":        "add",
		"message":       "too fast",
		"every_seconds": float64(5),
	})

	if !result.IsError {
		t.Fatal("expected error for every_seconds below min_interval")
	}
	if !strings.Contains(result.ForLLM, "below the minimum allowed interval") {
		t.Errorf("expected min interval error, got: %s", result.ForLLM)
	}
}

// TestCronTool_MinInterval_EveryAboveLimit verifies that every_seconds at or above min_interval is accepted
func TestCronTool_MinInterval_EveryAboveLimit(t *testing.T) {
	tool := newTestCronToolWithMinInterval(t, 60)
	ctx := WithToolContext(context.Background(), "cli", "direct")
	result := tool.Execute(ctx, map[string]any{
		"action":        "add",
		"message":       "fast enough",
		"every_seconds": float64(60),
	})

	if result.IsError {
		t.Fatalf("expected every_seconds at min_interval to succeed, got: %s", result.ForLLM)
	}
}

// TestCronTool_MinInterval_CronExprBelowLimit verifies that cron expressions firing too frequently are rejected
func TestCronTool_MinInterval_CronExprBelowLimit(t *testing.T) {
	tool := newTestCronToolWithMinInterval(t, 120)
	ctx := WithToolContext(context.Background(), "cli", "direct")
	// "* * * * *" fires every minute (60s), which is below 120s min
	result := tool.Execute(ctx, map[string]any{
		"action":    "add",
		"message":   "every minute",
		"cron_expr": "* * * * *",
	})

	if !result.IsError {
		t.Fatal("expected error for cron expression below min_interval")
	}
	if !strings.Contains(result.ForLLM, "below the minimum allowed interval") {
		t.Errorf("expected min interval error, got: %s", result.ForLLM)
	}
}

// TestCronTool_MinInterval_CronExprAboveLimit verifies that cron expressions with sufficient intervals are accepted
func TestCronTool_MinInterval_CronExprAboveLimit(t *testing.T) {
	tool := newTestCronToolWithMinInterval(t, 60)
	ctx := WithToolContext(context.Background(), "cli", "direct")
	// "0 * * * *" fires every hour (3600s), well above 60s min
	result := tool.Execute(ctx, map[string]any{
		"action":    "add",
		"message":   "every hour",
		"cron_expr": "0 * * * *",
	})

	if result.IsError {
		t.Fatalf("expected cron expression above min_interval to succeed, got: %s", result.ForLLM)
	}
}

// TestCronTool_MinInterval_AtJobNotAffected verifies that one-time (at) jobs are not affected by min_interval
func TestCronTool_MinInterval_AtJobNotAffected(t *testing.T) {
	tool := newTestCronToolWithMinInterval(t, 3600)
	ctx := WithToolContext(context.Background(), "cli", "direct")
	result := tool.Execute(ctx, map[string]any{
		"action":     "add",
		"message":    "one time only",
		"at_seconds": float64(5),
	})

	if result.IsError {
		t.Fatalf("expected one-time at_seconds job to bypass min_interval, got: %s", result.ForLLM)
	}
}

// TestCronTool_MinInterval_ZeroDisablesCheck verifies that min_interval=0 disables the check
func TestCronTool_MinInterval_ZeroDisablesCheck(t *testing.T) {
	tool := newTestCronToolWithMinInterval(t, 0)
	ctx := WithToolContext(context.Background(), "cli", "direct")
	result := tool.Execute(ctx, map[string]any{
		"action":        "add",
		"message":       "very fast",
		"every_seconds": float64(1),
	})

	if result.IsError {
		t.Fatalf("expected every_seconds=1 to succeed when min_interval=0, got: %s", result.ForLLM)
	}
}
