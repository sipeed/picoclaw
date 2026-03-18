package tools

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/cron"
)

func newTestCronToolWithConfig(t *testing.T, cfg *config.Config) *CronTool {
	t.Helper()
	storePath := filepath.Join(t.TempDir(), "cron.json")
	cronService := cron.NewCronService(storePath, nil)
	msgBus := bus.NewMessageBus()
	tool, err := NewCronTool(cronService, nil, msgBus, t.TempDir(), true, 0, cfg)
	if err != nil {
		t.Fatalf("NewCronTool() error: %v", err)
	}
	return tool
}

func newTestCronTool(t *testing.T) *CronTool {
	t.Helper()
	return newTestCronToolWithConfig(t, config.DefaultConfig())
}

// TestCronTool_MinIntervalSeconds validates that min_interval_seconds config prevents excessive scheduling
func TestCronTool_MinIntervalSeconds(t *testing.T) {
	// Test with default config (min_interval_seconds = 60)
	tool := newTestCronTool(t)
	ctx := WithToolContext(context.Background(), "cli", "direct")

	// Test every_seconds below minimum (should fail)
	result := tool.Execute(ctx, map[string]any{
		"action":        "add",
		"message":       "test reminder",
		"every_seconds": float64(30), // 30 seconds, below default 60
	})

	if !result.IsError {
		t.Fatal("expected error for interval below minimum")
	}
	if !strings.Contains(result.ForLLM, "interval too short") {
		t.Errorf("expected 'interval too short' error, got: %s", result.ForLLM)
	}

	// Test every_seconds at minimum (should succeed)
	result = tool.Execute(ctx, map[string]any{
		"action":        "add",
		"message":       "test reminder",
		"every_seconds": float64(60), // 60 seconds, at default minimum
	})

	if result.IsError {
		t.Fatalf("expected success for interval at minimum, got: %s", result.ForLLM)
	}
}

// TestCronTool_MinIntervalSecondsDisabled validates that min_interval_seconds = 0 disables the check
func TestCronTool_MinIntervalSecondsDisabled(t *testing.T) {
	// Create config with min_interval_seconds = 0 (disabled)
	cfg := config.DefaultConfig()
	cfg.Tools.Cron.MinIntervalSeconds = 0

	tool := newTestCronToolWithConfig(t, cfg)
	ctx := WithToolContext(context.Background(), "cli", "direct")

	// Test every_seconds below default minimum (should succeed when check is disabled)
	result := tool.Execute(ctx, map[string]any{
		"action":        "add",
		"message":       "test reminder",
		"every_seconds": float64(10), // 10 seconds, would fail if check was enabled
	})

	if result.IsError {
		t.Fatalf("expected success when min_interval_seconds is disabled, got: %s", result.ForLLM)
	}
}

// TestCronTool_MinIntervalSecondsCron validates cron expression interval checking
func TestCronTool_MinIntervalSecondsCron(t *testing.T) {
	// Test with default config (min_interval_seconds = 60)
	tool := newTestCronTool(t)
	ctx := WithToolContext(context.Background(), "cli", "direct")

	// Test cron expression with interval below minimum (every 30 seconds)
	result := tool.Execute(ctx, map[string]any{
		"action":    "add",
		"message":   "test reminder",
		"cron_expr": "*/30 * * * * *", // every 30 seconds
	})

	if !result.IsError {
		t.Fatal("expected error for cron interval below minimum")
	}
	if !strings.Contains(result.ForLLM, "cron interval too short") {
		t.Errorf("expected 'cron interval too short' error, got: %s", result.ForLLM)
	}

	// Test cron expression with interval at minimum (every minute)
	result = tool.Execute(ctx, map[string]any{
		"action":    "add",
		"message":   "test reminder",
		"cron_expr": "0 * * * * *", // every minute
	})

	if result.IsError {
		t.Fatalf("expected success for cron interval at minimum, got: %s", result.ForLLM)
	}
}

// TestCronTool_AtSecondsNotAffected validates that at_seconds is not affected by min_interval_seconds
func TestCronTool_AtSecondsNotAffected(t *testing.T) {
	tool := newTestCronTool(t)
	ctx := WithToolContext(context.Background(), "cli", "direct")

	// Test at_seconds (one-time) should not be affected by min_interval_seconds
	result := tool.Execute(ctx, map[string]any{
		"action":     "add",
		"message":    "test reminder",
		"at_seconds": float64(10), // 10 seconds from now
	})

	if result.IsError {
		t.Fatalf("expected success for at_seconds regardless of min_interval_seconds, got: %s", result.ForLLM)
	}
}

// TestCronTool_MinIntervalSecondsCustom validates custom min_interval_seconds values
func TestCronTool_MinIntervalSecondsCustom(t *testing.T) {
	// Create config with custom min_interval_seconds = 120
	cfg := config.DefaultConfig()
	cfg.Tools.Cron.MinIntervalSeconds = 120

	tool := newTestCronToolWithConfig(t, cfg)
	ctx := WithToolContext(context.Background(), "cli", "direct")

	// Test every_seconds below custom minimum (should fail)
	result := tool.Execute(ctx, map[string]any{
		"action":        "add",
		"message":       "test reminder",
		"every_seconds": float64(90), // 90 seconds, below custom minimum 120
	})

	if !result.IsError {
		t.Fatal("expected error for interval below custom minimum")
	}

	// Test every_seconds at custom minimum (should succeed)
	result = tool.Execute(ctx, map[string]any{
		"action":        "add",
		"message":       "test reminder",
		"every_seconds": float64(120), // 120 seconds, at custom minimum
	})

	if result.IsError {
		t.Fatalf("expected success for interval at custom minimum, got: %s", result.ForLLM)
	}
}

// TestCronTool_MinIntervalSecondsNegative validates that negative values are treated as default
func TestCronTool_MinIntervalSecondsNegative(t *testing.T) {
	// Create config with negative min_interval_seconds (should use default 60)
	cfg := config.DefaultConfig()
	cfg.Tools.Cron.MinIntervalSeconds = -1

	tool := newTestCronToolWithConfig(t, cfg)
	ctx := WithToolContext(context.Background(), "cli", "direct")

	// Test every_seconds below default minimum (should fail with default 60)
	result := tool.Execute(ctx, map[string]any{
		"action":        "add",
		"message":       "test reminder",
		"every_seconds": float64(30), // 30 seconds, below default 60
	})

	if !result.IsError {
		t.Fatal("expected error for interval below default minimum when negative value provided")
	}
}
