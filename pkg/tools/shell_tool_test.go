package tools

import (
	"bytes"
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/tools/shell"
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
	if cbResult.Async {
		t.Error("callback result should not be async (it is a completion)")
	}
	if cbResult.IsError {
		t.Errorf("callback result should not be an error: %s", cbResult.ForLLM)
	}
	if !strings.Contains(cbResult.ForLLM, "bg_output") {
		t.Errorf("expected output in callback ForLLM: %s", cbResult.ForLLM)
	}
	if !strings.Contains(cbResult.ForLLM, "completed") {
		t.Errorf("expected 'completed' in callback ForLLM: %s", cbResult.ForLLM)
	}
	if cbResult.ForUser == "" {
		t.Error("callback ForUser should be populated for user notification")
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
	if cbResult.Async {
		t.Error("callback result should not be async (it is a completion)")
	}
	if !cbResult.IsError {
		t.Error("callback result should be an error for blocked commands")
	}
	if !strings.Contains(cbResult.ForLLM, "failed") {
		t.Errorf("expected 'failed' in callback ForLLM: %s", cbResult.ForLLM)
	}
	if cbResult.ForUser == "" {
		t.Error("callback ForUser should be populated for user notification")
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

func ptr[T any](v T) *T { return &v }

func TestWarnDeprecatedExecConfig_EnableDenyPatternsFalse(t *testing.T) {
	out := captureStdout(t, func() {
		warnDeprecatedExecConfig(config.ExecConfig{
			EnableDenyPatterns: ptr(false),
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
			EnableDenyPatterns: ptr(true),
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
			EnableDenyPatterns:  ptr(false),
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
		cfg.Tools.Exec.EnableDenyPatterns = ptr(false)
		_, err := NewExecToolWithConfig(t.TempDir(), false, cfg)
		if err != nil {
			t.Fatal(err)
		}
	})

	if !strings.Contains(out, "enable_deny_patterns: false") {
		t.Errorf("expected warning in NewExecToolWithConfig output: %s", out)
	}
}

func TestParseArgProfiles(t *testing.T) {
	profiles := parseArgProfiles(map[string]config.ArgProfileConfig{
		"curl": {
			SplitCombinedShort: true,
			SplitLongEquals:    true,
			ShortAttachedValue: map[string]string{"-X": "upper"},
			SeparateValueFlags: map[string]string{"--request": "upper"},
		},
	})

	profile, ok := profiles["curl"]
	if !ok {
		t.Fatal("expected curl profile")
	}
	if !profile.SplitCombinedShort || !profile.SplitLongEquals {
		t.Fatal("expected split flags to be enabled")
	}
	if got := profile.ShortAttachedValue["-X"]; got != shell.FlagValueUpper {
		t.Fatalf("ShortAttachedValue[-X] = %q, want %q", got, shell.FlagValueUpper)
	}
	if got := profile.SeparateValueFlags["--request"]; got != shell.FlagValueUpper {
		t.Fatalf("SeparateValueFlags[--request] = %q, want %q", got, shell.FlagValueUpper)
	}
}

func TestParseArgProfiles_InvalidTransformWarning(t *testing.T) {
	out := captureStdout(t, func() {
		profiles := parseArgProfiles(map[string]config.ArgProfileConfig{
			"curl": {
				ShortAttachedValue: map[string]string{"-X": "bogus"},
			},
		})
		if got := len(profiles["curl"].ShortAttachedValue); got != 0 {
			t.Fatalf("expected invalid transform to be skipped, got %d entries", got)
		}
	})

	if !strings.Contains(out, "invalid short_attached_value_flags transform") {
		t.Fatalf("expected invalid transform warning, got: %s", out)
	}
}
