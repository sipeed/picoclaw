package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestExecTool_SyncExecution(t *testing.T) {
	tool, err := NewExecTool(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}

	result := tool.Execute(context.Background(), map[string]any{
		"command": "echo sync_output",
	})

	if result.IsError {
		t.Fatalf("expected success: %s", result.ForLLM)
	}
	if result.Async {
		t.Error("sync execution should not be async")
	}
}

func TestExecTool_BackgroundWithoutCallback(t *testing.T) {
	tool, err := NewExecTool(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}

	// background=true but no callback set → falls through to sync
	result := tool.Execute(context.Background(), map[string]any{
		"command":    "echo fallback",
		"background": true,
	})

	if result.Async {
		t.Error("should fall back to sync when no callback is set")
	}
	if result.IsError {
		t.Fatalf("expected success: %s", result.ForLLM)
	}
}

func TestExecTool_BackgroundWithCallback(t *testing.T) {
	tool, err := NewExecTool(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}

	var (
		mu       sync.Mutex
		received *ToolResult
	)
	done := make(chan struct{})

	tool.SetCallback(func(_ context.Context, r *ToolResult) {
		mu.Lock()
		received = r
		mu.Unlock()
		close(done)
	})

	result := tool.Execute(context.Background(), map[string]any{
		"command":    "echo async_output",
		"background": true,
	})

	if !result.Async {
		t.Fatal("expected async result")
	}

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for async callback")
	}

	mu.Lock()
	defer mu.Unlock()

	if received == nil {
		t.Fatal("callback was never invoked")
	}
	if received.IsError {
		t.Fatalf("async command failed: %s", received.ForLLM)
	}
}

func TestExecTool_BackgroundBlockedCommand(t *testing.T) {
	tool, err := NewExecTool(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}

	var (
		mu       sync.Mutex
		received *ToolResult
	)
	done := make(chan struct{})

	tool.SetCallback(func(_ context.Context, r *ToolResult) {
		mu.Lock()
		received = r
		mu.Unlock()
		close(done)
	})

	result := tool.Execute(context.Background(), map[string]any{
		"command":    "sudo rm -rf /",
		"background": true,
	})

	if !result.Async {
		t.Fatal("expected async result even for blocked commands")
	}

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for async callback")
	}

	mu.Lock()
	defer mu.Unlock()

	if received == nil {
		t.Fatal("callback was never invoked")
	}
	if !received.IsError {
		t.Error("blocked command should report error via callback")
	}
}

func TestExecTool_ImplementsAsyncTool(t *testing.T) {
	tool, err := NewExecTool(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}

	var _ AsyncTool = tool // compile-time check
}

// captureStdout runs fn and returns whatever it wrote to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	old := os.Stdout
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func boolPtr(b bool) *bool { return &b }

func TestWarnDeprecatedExecConfig_EnableDenyPatternsFalse(t *testing.T) {
	out := captureStdout(t, func() {
		warnDeprecatedExecConfig(config.ExecConfig{
			EnableDenyPatterns: boolPtr(false),
		})
	})

	if !strings.Contains(out, "enable_deny_patterns: false") {
		t.Errorf("expected warning about 'enable_deny_patterns: false', got: %s", out)
	}
	if !strings.Contains(out, "risk_threshold: critical") {
		t.Errorf("expected migration hint to 'risk_threshold: critical', got: %s", out)
	}
}

func TestWarnDeprecatedExecConfig_EnableDenyPatternsTrue(t *testing.T) {
	out := captureStdout(t, func() {
		warnDeprecatedExecConfig(config.ExecConfig{
			EnableDenyPatterns: boolPtr(true),
		})
	})

	if !strings.Contains(out, "enable_deny_patterns") {
		t.Errorf("expected deprecation warning, got: %s", out)
	}
	if !strings.Contains(out, "Remove this field") {
		t.Errorf("expected removal hint, got: %s", out)
	}
}

func TestWarnDeprecatedExecConfig_NilNoWarning(t *testing.T) {
	out := captureStdout(t, func() {
		warnDeprecatedExecConfig(config.ExecConfig{})
	})

	if strings.Contains(out, "enable_deny_patterns") {
		t.Errorf("expected no warning when field is absent, got: %s", out)
	}
}

func TestWarnDeprecatedExecConfig_CustomPatterns(t *testing.T) {
	out := captureStdout(t, func() {
		warnDeprecatedExecConfig(config.ExecConfig{
			CustomDenyPatterns:  []string{"rm"},
			CustomAllowPatterns: []string{"ls"},
		})
	})

	if !strings.Contains(out, "custom_deny_patterns") {
		t.Errorf("expected custom_deny_patterns warning, got: %s", out)
	}
	if !strings.Contains(out, "custom_allow_patterns") {
		t.Errorf("expected custom_allow_patterns warning, got: %s", out)
	}
}

func TestWarnDeprecatedExecConfig_AllDeprecatedFields(t *testing.T) {
	out := captureStdout(t, func() {
		warnDeprecatedExecConfig(config.ExecConfig{
			EnableDenyPatterns:  boolPtr(false),
			CustomDenyPatterns:  []string{"rm"},
			CustomAllowPatterns: []string{"ls"},
		})
	})

	// All three warnings should fire.
	for _, want := range []string{
		"enable_deny_patterns: false",
		"custom_deny_patterns",
		"custom_allow_patterns",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected warning containing %q, got: %s", want, out)
		}
	}
}

func TestNewExecToolWithConfig_EnableDenyPatternsFalseWarning(t *testing.T) {
	out := captureStdout(t, func() {
		cfg := &config.Config{}
		cfg.Tools.Exec.EnableDenyPatterns = boolPtr(false)
		_, err := NewExecToolWithConfig(t.TempDir(), false, cfg)
		if err != nil {
			t.Fatal(err)
		}
	})

	if !strings.Contains(out, "enable_deny_patterns: false") {
		t.Errorf("expected warning in NewExecToolWithConfig output: %s", out)
	}
}

// Suppress unused import lint for fmt (used by captureStdout indirectly).
var _ = fmt.Sprintf
