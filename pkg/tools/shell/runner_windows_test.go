package shell

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRun_Timeout_Windows(t *testing.T) {
	// ping with a high count effectively blocks like sleep on Unix.
	result := Run(context.Background(), RunConfig{
		Command:       "ping -n 60 127.0.0.1",
		Dir:           t.TempDir(),
		Timeout:       500 * time.Millisecond,
		RiskThreshold: RiskMedium,
		RiskOverrides: map[string]string{"ping": "low"},
	})

	if !result.IsError {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(result.Output, "timed out") {
		t.Errorf("expected 'timed out' in output: %s", result.Output)
	}
}

func TestRun_WorkingDir_Windows(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0o644)

	result := Run(context.Background(), RunConfig{
		Command:       "cmd.exe /c type test.txt",
		Dir:           tmpDir,
		Timeout:       5 * time.Second,
		RiskThreshold: RiskHigh, // cmd.exe is risk=critical
		RiskOverrides: map[string]string{"cmd": "low"},
	})

	if result.IsError {
		t.Fatalf("expected success: %s", result.Output)
	}
	if !strings.Contains(result.Output, "test content") {
		t.Errorf("expected 'test content' in output: %s", result.Output)
	}
}

func TestRun_HighThresholdAllowsDel_Windows(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "delete_me.txt")
	os.WriteFile(testFile, []byte("bye"), 0o644)

	result := Run(context.Background(), RunConfig{
		Command:       "cmd.exe /c del delete_me.txt",
		Dir:           tmpDir,
		Timeout:       5 * time.Second,
		RiskThreshold: RiskHigh,
		RiskOverrides: map[string]string{"cmd": "low"},
	})

	if result.IsError {
		t.Fatalf("with threshold=high, del should be allowed: %s", result.Output)
	}
	if _, err := os.Stat(testFile); err == nil {
		t.Error("file should have been deleted")
	}
}

func TestRun_EnvSanitization_Windows(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-secret-test")
	t.Setenv("PATH", os.Getenv("PATH"))

	result := Run(context.Background(), RunConfig{
		Command:       "cmd.exe /c set",
		Dir:           t.TempDir(),
		Timeout:       5 * time.Second,
		RiskThreshold: RiskHigh,
		RiskOverrides: map[string]string{"cmd": "low"},
	})

	if result.IsError {
		t.Fatalf("expected set command to succeed: %s", result.Output)
	}
	if strings.Contains(result.Output, "OPENAI_API_KEY") {
		t.Error("OPENAI_API_KEY should not be in child environment")
	}
	if !strings.Contains(result.Output, "PATH=") {
		t.Error("PATH should be in child environment")
	}
}

func TestRun_NulRedirection_Windows(t *testing.T) {
	result := Run(context.Background(), RunConfig{
		Command:       "echo hello 2>NUL",
		Dir:           t.TempDir(),
		Timeout:       5 * time.Second,
		Restrict:      true,
		WorkspaceDir:  t.TempDir(),
		RiskThreshold: RiskMedium,
	})

	if result.IsError && strings.Contains(result.Output, "sandbox") {
		t.Errorf("NUL should not be blocked: %s", result.Output)
	}
}

func TestRun_RiskOverrides_Windows(t *testing.T) {
	result := Run(context.Background(), RunConfig{
		Command:       "cmd.exe /c del nonexistent_file_xyz 2>NUL & echo done",
		Dir:           t.TempDir(),
		Timeout:       5 * time.Second,
		RiskThreshold: RiskMedium,
		RiskOverrides: map[string]string{"cmd": "low"},
	})

	if result.IsError && strings.Contains(result.Output, "blocked") {
		t.Errorf("cmd.exe should be allowed with override: %s", result.Output)
	}
}
