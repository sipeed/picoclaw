package tools

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestShellTool_Success verifies successful command execution
func TestShellTool_Success(t *testing.T) {
	tool := NewExecTool("", false)

	ctx := context.Background()
	args := map[string]any{
		"command": "echo 'hello world'",
	}

	result := tool.Execute(ctx, args)

	// Success should not be an error
	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	// ForUser should contain command output
	if !strings.Contains(result.ForUser, "hello world") {
		t.Errorf("Expected ForUser to contain 'hello world', got: %s", result.ForUser)
	}

	// ForLLM should contain full output
	if !strings.Contains(result.ForLLM, "hello world") {
		t.Errorf("Expected ForLLM to contain 'hello world', got: %s", result.ForLLM)
	}
}

// TestShellTool_Failure verifies failed command execution
func TestShellTool_Failure(t *testing.T) {
	tool := NewExecTool("", false)

	ctx := context.Background()
	args := map[string]any{
		"command": "ls /nonexistent_directory_12345",
	}

	result := tool.Execute(ctx, args)

	// Failure should be marked as error
	if !result.IsError {
		t.Errorf("Expected error for failed command, got IsError=false")
	}

	// ForUser should contain error information
	if result.ForUser == "" {
		t.Errorf("Expected ForUser to contain error info, got empty string")
	}

	// ForLLM should contain exit code or error
	if !strings.Contains(result.ForLLM, "Exit code") && result.ForUser == "" {
		t.Errorf("Expected ForLLM to contain exit code or error, got: %s", result.ForLLM)
	}
}

// TestShellTool_Timeout verifies command timeout handling
func TestShellTool_Timeout(t *testing.T) {
	tool := NewExecTool("", false)
	tool.SetTimeout(100 * time.Millisecond)

	ctx := context.Background()
	args := map[string]any{
		"command": "sleep 10",
	}

	result := tool.Execute(ctx, args)

	// Timeout should be marked as error
	if !result.IsError {
		t.Errorf("Expected error for timeout, got IsError=false")
	}

	// Should mention timeout
	if !strings.Contains(result.ForLLM, "timed out") && !strings.Contains(result.ForUser, "timed out") {
		t.Errorf("Expected timeout message, got ForLLM: %s, ForUser: %s", result.ForLLM, result.ForUser)
	}
}

// TestShellTool_WorkingDir verifies custom working directory
func TestShellTool_WorkingDir(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0o644)

	tool := NewExecTool("", false)

	ctx := context.Background()
	args := map[string]any{
		"command":     "cat test.txt",
		"working_dir": tmpDir,
	}

	result := tool.Execute(ctx, args)

	if result.IsError {
		t.Errorf("Expected success in custom working dir, got error: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForUser, "test content") {
		t.Errorf("Expected output from custom dir, got: %s", result.ForUser)
	}
}

// TestShellTool_DangerousCommand verifies safety guard blocks dangerous commands
func TestShellTool_DangerousCommand(t *testing.T) {
	tool := NewExecTool("", false)

	ctx := context.Background()
	args := map[string]any{
		"command": "rm -rf /",
	}

	result := tool.Execute(ctx, args)

	// Dangerous command should be blocked
	if !result.IsError {
		t.Errorf("Expected dangerous command to be blocked (IsError=true)")
	}

	if !strings.Contains(result.ForLLM, "blocked") && !strings.Contains(result.ForUser, "blocked") {
		t.Errorf("Expected 'blocked' message, got ForLLM: %s, ForUser: %s", result.ForLLM, result.ForUser)
	}
}

// TestShellTool_MissingCommand verifies error handling for missing command
func TestShellTool_MissingCommand(t *testing.T) {
	tool := NewExecTool("", false)

	ctx := context.Background()
	args := map[string]any{}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error when command is missing")
	}
}

// TestShellTool_StderrCapture verifies stderr is captured and included
func TestShellTool_StderrCapture(t *testing.T) {
	tool := NewExecTool("", false)

	ctx := context.Background()
	args := map[string]any{
		"command": "sh -c 'echo stdout; echo stderr >&2'",
	}

	result := tool.Execute(ctx, args)

	// Both stdout and stderr should be in output
	if !strings.Contains(result.ForLLM, "stdout") {
		t.Errorf("Expected stdout in output, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "stderr") {
		t.Errorf("Expected stderr in output, got: %s", result.ForLLM)
	}
}

// TestShellTool_OutputTruncation verifies long output is truncated
func TestShellTool_OutputTruncation(t *testing.T) {
	tool := NewExecTool("", false)

	ctx := context.Background()
	// Generate long output (>10000 chars)
	args := map[string]any{
		"command": "python3 -c \"print('x' * 20000)\" || echo " + strings.Repeat("x", 20000),
	}

	result := tool.Execute(ctx, args)

	// Should have truncation message or be truncated
	if len(result.ForLLM) > 15000 {
		t.Errorf("Expected output to be truncated, got length: %d", len(result.ForLLM))
	}
}

// TestShellTool_WorkingDir_OutsideWorkspace verifies that working_dir cannot escape the workspace directly
func TestShellTool_WorkingDir_OutsideWorkspace(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	outsideDir := filepath.Join(root, "outside")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatalf("failed to create outside dir: %v", err)
	}

	tool := NewExecTool(workspace, true)
	result := tool.Execute(context.Background(), map[string]any{
		"command":     "pwd",
		"working_dir": outsideDir,
	})

	if !result.IsError {
		t.Fatalf("expected working_dir outside workspace to be blocked, got output: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "blocked") {
		t.Errorf("expected 'blocked' in error, got: %s", result.ForLLM)
	}
}

// TestShellTool_WorkingDir_SymlinkEscape verifies that a symlink inside the workspace
// pointing outside cannot be used as working_dir to escape the sandbox.
func TestShellTool_WorkingDir_SymlinkEscape(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	secretDir := filepath.Join(root, "secret")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	if err := os.MkdirAll(secretDir, 0o755); err != nil {
		t.Fatalf("failed to create secret dir: %v", err)
	}
	os.WriteFile(filepath.Join(secretDir, "secret.txt"), []byte("top secret"), 0o644)

	// symlink lives inside the workspace but resolves to secretDir outside it
	link := filepath.Join(workspace, "escape")
	if err := os.Symlink(secretDir, link); err != nil {
		t.Skipf("symlinks not supported in this environment: %v", err)
	}

	tool := NewExecTool(workspace, true)
	result := tool.Execute(context.Background(), map[string]any{
		"command":     "cat secret.txt",
		"working_dir": link,
	})

	if !result.IsError {
		t.Fatalf("expected symlink working_dir escape to be blocked, got output: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "blocked") {
		t.Errorf("expected 'blocked' in error, got: %s", result.ForLLM)
	}
}

// TestShellTool_RestrictToWorkspace verifies workspace restriction
func TestShellTool_RestrictToWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewExecTool(tmpDir, false)
	tool.SetRestrictToWorkspace(true)

	ctx := context.Background()
	args := map[string]any{
		"command": "cat ../../etc/passwd",
	}

	result := tool.Execute(ctx, args)

	// Path traversal should be blocked
	if !result.IsError {
		t.Errorf("Expected path traversal to be blocked with restrictToWorkspace=true")
	}

	if !strings.Contains(result.ForLLM, "blocked") && !strings.Contains(result.ForUser, "blocked") {
		t.Errorf(
			"Expected 'blocked' message for path traversal, got ForLLM: %s, ForUser: %s",
			result.ForLLM,
			result.ForUser,
		)
	}
}

// --- guardCommand unit tests ---

// TestGuardCommand_RelativePathWithSlashes verifies that relative paths
// containing slashes (e.g., tests/cold/test.py, projects/terra-py-form)
// are NOT falsely blocked. This was a regression caused by the old regex
// matching "/cold/test.py" from "tests/cold/test.py" as an absolute path.
func TestGuardCommand_RelativePathWithSlashes(t *testing.T) {
	workspace := t.TempDir()
	tool := NewExecTool(workspace, true)

	cmds := []string{
		"pytest tests/cold/test_solver.py -v --tb=short",
		"cd projects/terra-py-form && pytest",
		"uv run pytest tests/cold/test_solver.py -v --tb=short",
		"cat src/terra_py_form/cold/parser.py",
		"python src/main.py --config config/dev.json",
	}

	for _, cmd := range cmds {
		result := tool.guardCommand(cmd, workspace)
		if result != "" {
			t.Errorf("Relative path should not be blocked: %q → %s", cmd, result)
		}
	}
}

// TestGuardCommand_VenvBinary verifies that .venv/bin/... paths are allowed
// (they are relative paths, not absolute).
func TestGuardCommand_VenvBinary(t *testing.T) {
	workspace := t.TempDir()
	tool := NewExecTool(workspace, true)

	cmds := []string{
		".venv/bin/python -m pytest",
		".venv/bin/pytest tests/ -v",
		".venv/bin/pip install -e .",
	}

	for _, cmd := range cmds {
		result := tool.guardCommand(cmd, workspace)
		if result != "" {
			t.Errorf("Venv relative path should not be blocked: %q → %s", cmd, result)
		}
	}
}

// TestGuardCommand_ExecutableBinaryAllowed verifies that absolute paths
// to executable files outside the workspace are allowed (system binaries).
func TestGuardCommand_ExecutableBinaryAllowed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix executable permission test not applicable on Windows")
	}

	workspace := t.TempDir()
	externalDir := t.TempDir()

	// Create a fake executable outside the workspace
	execPath := filepath.Join(externalDir, "mybin")
	os.WriteFile(execPath, []byte("#!/bin/sh\necho ok"), 0755)

	tool := NewExecTool(workspace, true)

	cmd := execPath + " --help"
	result := tool.guardCommand(cmd, workspace)
	if result != "" {
		t.Errorf("Executable binary outside workspace should be allowed: %q → %s", cmd, result)
	}
}

// TestGuardCommand_ExecutableBinaryAllowed_Windows verifies that .exe files
// outside the workspace are allowed on Windows.
func TestGuardCommand_ExecutableBinaryAllowed_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	workspace := t.TempDir()
	externalDir := t.TempDir()

	// Create a fake .exe outside the workspace
	execPath := filepath.Join(externalDir, "tool.exe")
	os.WriteFile(execPath, []byte("MZ"), 0644)

	tool := NewExecTool(workspace, true)

	cmd := execPath + " --version"
	result := tool.guardCommand(cmd, workspace)
	if result != "" {
		t.Errorf("Windows .exe outside workspace should be allowed: %q → %s", cmd, result)
	}
}

// TestGuardCommand_NonExecutableOutsideBlocked verifies that non-executable
// files outside the workspace are blocked (e.g., reading /etc/shadow).
func TestGuardCommand_NonExecutableOutsideBlocked(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission test not applicable on Windows")
	}

	workspace := t.TempDir()
	externalDir := t.TempDir()

	// Create a regular (non-executable) file outside workspace
	dataFile := filepath.Join(externalDir, "secret.txt")
	os.WriteFile(dataFile, []byte("secret data"), 0644)

	tool := NewExecTool(workspace, true)

	cmd := "cat " + dataFile
	result := tool.guardCommand(cmd, workspace)
	if result == "" {
		t.Errorf("Non-executable file outside workspace should be blocked: %q", cmd)
	}
	if !strings.Contains(result, "path outside working dir") {
		t.Errorf("Expected 'path outside working dir' message, got: %s", result)
	}
}

// TestGuardCommand_NonExistentAbsolutePathBlocked verifies that absolute
// paths that don't exist are blocked (could be file creation outside workspace).
func TestGuardCommand_NonExistentAbsolutePathBlocked(t *testing.T) {
	workspace := t.TempDir()
	tool := NewExecTool(workspace, true)

	// Use platform-appropriate absolute path
	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "echo hello > C:\\nonexistent_picoclaw_test_output"
	} else {
		cmd = "echo hello > /tmp/nonexistent_picoclaw_test_output"
	}
	result := tool.guardCommand(cmd, workspace)
	if result == "" {
		t.Errorf("Non-existent absolute path outside workspace should be blocked: %q", cmd)
	}
}

// TestGuardCommand_FlagEmbeddedPathSkipped verifies that paths embedded in
// flags (e.g., -I/usr/local/include) are NOT extracted as absolute paths
// because the token starts with "-", not "/".
func TestGuardCommand_FlagEmbeddedPathSkipped(t *testing.T) {
	workspace := t.TempDir()
	tool := NewExecTool(workspace, true)

	cmds := []string{
		"gcc -I/usr/local/include -L/usr/lib main.c",
		"g++ -std=c++17 -I/opt/include file.cpp",
		"python --prefix=/usr/local script.py",
	}

	for _, cmd := range cmds {
		result := tool.guardCommand(cmd, workspace)
		if result != "" {
			t.Errorf("Flag-embedded path should not be blocked: %q → %s", cmd, result)
		}
	}
}

// TestGuardCommand_AbsolutePathInsideWorkspace verifies that absolute paths
// within the workspace are always allowed.
func TestGuardCommand_AbsolutePathInsideWorkspace(t *testing.T) {
	workspace := t.TempDir()
	tool := NewExecTool(workspace, true)

	innerDir := filepath.Join(workspace, "projects", "myapp")
	os.MkdirAll(innerDir, 0755)

	cmd := "ls " + innerDir
	result := tool.guardCommand(cmd, workspace)
	if result != "" {
		t.Errorf("Absolute path inside workspace should be allowed: %q → %s", cmd, result)
	}
}

// TestGuardCommand_PathTraversal verifies that various path traversal
// patterns are blocked.
func TestGuardCommand_PathTraversal(t *testing.T) {
	workspace := t.TempDir()
	tool := NewExecTool(workspace, true)

	cmds := []string{
		"cat ../../etc/passwd",
		"cat ../../../etc/shadow",
		"ls projects/../../../../etc",
	}

	for _, cmd := range cmds {
		result := tool.guardCommand(cmd, workspace)
		if result == "" {
			t.Errorf("Path traversal should be blocked: %q", cmd)
		}
		if !strings.Contains(result, "path traversal") {
			t.Errorf("Expected 'path traversal' message, got: %s", result)
		}
	}
}

// TestGuardCommand_CdWithAbsoluteWorkspacePath verifies that cd to an
// absolute path within the workspace followed by other commands is allowed.
func TestGuardCommand_CdWithAbsoluteWorkspacePath(t *testing.T) {
	workspace := t.TempDir()
	innerDir := filepath.Join(workspace, "projects", "foo")
	os.MkdirAll(innerDir, 0755)

	tool := NewExecTool(workspace, true)

	cmd := "cd " + innerDir + " && ls -la"
	result := tool.guardCommand(cmd, workspace)
	if result != "" {
		t.Errorf("cd to workspace subdir should be allowed: %q → %s", cmd, result)
	}
}
