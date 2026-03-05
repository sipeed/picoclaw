package tools

import (
	"bytes"
	"context"
	"os"
	"strings"
	"sync"
	"testing"

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

	// background=true but nil callback → falls back to synchronous
	result := tool.ExecuteAsync(context.Background(), map[string]any{
		"command":    "echo fallback",
		"background": true,
	}, nil)

	if result.Async {
		t.Error("should fall back to sync when no callback is provided")
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

	var wg sync.WaitGroup
	wg.Add(1)

	var cbResult *ToolResult
	cb := func(_ context.Context, result *ToolResult) {
		cbResult = result
		wg.Done()
	}

	result := tool.ExecuteAsync(context.Background(), map[string]any{
		"command":    "echo bg_output",
		"background": true,
	}, cb)

	if !result.Async {
		t.Fatal("expected async result")
	}

	wg.Wait()

	if cbResult == nil {
		t.Fatal("callback was not invoked")
	}
	if !strings.Contains(cbResult.ForLLM, "bg_output") {
		t.Errorf("expected output in callback result: %s", cbResult.ForLLM)
	}
	if !strings.Contains(cbResult.ForLLM, "completed") {
		t.Errorf("expected 'completed' in callback result: %s", cbResult.ForLLM)
	}
}

func TestExecTool_BackgroundBlockedCommand(t *testing.T) {
	tool, err := NewExecTool(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	var cbResult *ToolResult
	cb := func(_ context.Context, result *ToolResult) {
		cbResult = result
		wg.Done()
	}

	result := tool.ExecuteAsync(context.Background(), map[string]any{
		"command":    "sudo rm -rf /",
		"background": true,
	}, cb)

	if !result.Async {
		t.Fatal("expected async result even for blocked commands")
	}

	wg.Wait()

	if cbResult == nil {
		t.Fatal("callback was not invoked")
	}
	if !strings.Contains(cbResult.ForLLM, "failed") {
		t.Errorf("expected 'failed' in callback result: %s", cbResult.ForLLM)
	}
}

func TestExecTool_ImplementsAsyncExecutor(t *testing.T) {
	tool, err := NewExecTool(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}

	var _ AsyncExecutor = tool // compile-time check
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
	defer func() {
		os.Stdout = old
		_ = w.Close()
		_ = r.Close()
	}()

	fn()

	_ = w.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
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
