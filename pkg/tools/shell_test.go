package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

// TestShellTool_Success verifies successful command execution
func TestShellTool_Success(t *testing.T) {
	tool, err := NewExecTool("", false)
	if err != nil {
		t.Errorf("unable to configure exec tool: %s", err)
	}

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
	tool, err := NewExecTool("", false)
	if err != nil {
		t.Errorf("unable to configure exec tool: %s", err)
	}

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
	tool, err := NewExecTool("", false)
	if err != nil {
		t.Errorf("unable to configure exec tool: %s", err)
	}

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

	tool, err := NewExecTool("", false)
	if err != nil {
		t.Errorf("unable to configure exec tool: %s", err)
	}

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
	tool, err := NewExecTool("", false)
	if err != nil {
		t.Errorf("unable to configure exec tool: %s", err)
	}

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

func TestShellTool_DangerousCommand_KillBlocked(t *testing.T) {
	tool, err := NewExecTool("", false)
	if err != nil {
		t.Errorf("unable to configure exec tool: %s", err)
	}

	ctx := context.Background()
	args := map[string]any{
		"command": "kill 12345",
	}

	result := tool.Execute(ctx, args)
	if !result.IsError {
		t.Errorf("Expected kill command to be blocked")
	}
	if !strings.Contains(result.ForLLM, "blocked") && !strings.Contains(result.ForUser, "blocked") {
		t.Errorf("Expected blocked message, got ForLLM: %s, ForUser: %s", result.ForLLM, result.ForUser)
	}
}

// TestShellTool_MissingCommand verifies error handling for missing command
func TestShellTool_MissingCommand(t *testing.T) {
	tool, err := NewExecTool("", false)
	if err != nil {
		t.Errorf("unable to configure exec tool: %s", err)
	}

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
	tool, err := NewExecTool("", false)
	if err != nil {
		t.Errorf("unable to configure exec tool: %s", err)
	}

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
	tool, err := NewExecTool("", false)
	if err != nil {
		t.Errorf("unable to configure exec tool: %s", err)
	}

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

	tool, err := NewExecTool(workspace, true)
	if err != nil {
		t.Errorf("unable to configure exec tool: %s", err)
	}

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

	tool, err := NewExecTool(workspace, true)
	if err != nil {
		t.Errorf("unable to configure exec tool: %s", err)
	}

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

// TestShellTool_RemoteChannelBlockedByDefault verifies exec is blocked for remote channels
func TestShellTool_RemoteChannelBlockedByDefault(t *testing.T) {
	cfg := &config.Config{}
	cfg.Tools.Exec.EnableDenyPatterns = true
	cfg.Tools.Exec.AllowRemote = false

	tool, err := NewExecToolWithConfig("", false, cfg)
	if err != nil {
		t.Fatalf("NewExecToolWithConfig() error: %v", err)
	}
	ctx := WithToolContext(context.Background(), "telegram", "chat-1")
	result := tool.Execute(ctx, map[string]any{"command": "echo hi"})

	if !result.IsError {
		t.Fatal("expected remote-channel exec to be blocked")
	}
	if !strings.Contains(result.ForLLM, "restricted to internal channels") {
		t.Errorf("expected 'restricted to internal channels' message, got: %s", result.ForLLM)
	}
}

// TestShellTool_InternalChannelAllowed verifies exec is allowed for internal channels
func TestShellTool_InternalChannelAllowed(t *testing.T) {
	cfg := &config.Config{}
	cfg.Tools.Exec.EnableDenyPatterns = true
	cfg.Tools.Exec.AllowRemote = false

	tool, err := NewExecToolWithConfig("", false, cfg)
	if err != nil {
		t.Fatalf("NewExecToolWithConfig() error: %v", err)
	}
	ctx := WithToolContext(context.Background(), "cli", "direct")
	result := tool.Execute(ctx, map[string]any{"command": "echo hi"})

	if result.IsError {
		t.Fatalf("expected internal channel exec to succeed, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "hi") {
		t.Errorf("expected output to contain 'hi', got: %s", result.ForLLM)
	}
}

// TestShellTool_EmptyChannelBlockedWhenNotAllowRemote verifies fail-closed when no channel context
func TestShellTool_EmptyChannelBlockedWhenNotAllowRemote(t *testing.T) {
	cfg := &config.Config{}
	cfg.Tools.Exec.EnableDenyPatterns = true
	cfg.Tools.Exec.AllowRemote = false

	tool, err := NewExecToolWithConfig("", false, cfg)
	if err != nil {
		t.Fatalf("NewExecToolWithConfig() error: %v", err)
	}
	result := tool.Execute(context.Background(), map[string]any{
		"command": "echo hi",
	})

	if !result.IsError {
		t.Fatal("expected exec with empty channel to be blocked when allowRemote=false")
	}
}

// TestShellTool_AllowRemoteBypassesChannelCheck verifies allowRemote=true permits any channel
func TestShellTool_AllowRemoteBypassesChannelCheck(t *testing.T) {
	cfg := &config.Config{}
	cfg.Tools.Exec.EnableDenyPatterns = true
	cfg.Tools.Exec.AllowRemote = true

	tool, err := NewExecToolWithConfig("", false, cfg)
	if err != nil {
		t.Fatalf("NewExecToolWithConfig() error: %v", err)
	}
	ctx := WithToolContext(context.Background(), "telegram", "chat-1")
	result := tool.Execute(ctx, map[string]any{"command": "echo hi"})

	if result.IsError {
		t.Fatalf("expected allowRemote=true to permit remote channel, got: %s", result.ForLLM)
	}
}

// TestShellTool_RestrictToWorkspace verifies workspace restriction
func TestShellTool_RestrictToWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	tool, err := NewExecTool(tmpDir, false)
	if err != nil {
		t.Errorf("unable to configure exec tool: %s", err)
	}

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

// TestShellTool_DevNullAllowed verifies that /dev/null redirections are not blocked (issue #964).
func TestShellTool_DevNullAllowed(t *testing.T) {
	tmpDir := t.TempDir()
	tool, err := NewExecTool(tmpDir, true)
	if err != nil {
		t.Fatalf("unable to configure exec tool: %s", err)
	}

	commands := []string{
		"echo hello 2>/dev/null",
		"echo hello >/dev/null",
		"echo hello > /dev/null",
		"echo hello 2> /dev/null",
		"echo hello >/dev/null 2>&1",
		"find " + tmpDir + " -name '*.go' 2>/dev/null",
	}

	for _, cmd := range commands {
		result := tool.Execute(context.Background(), map[string]any{"command": cmd})
		if result.IsError && strings.Contains(result.ForLLM, "blocked") {
			t.Errorf("command should not be blocked: %s\n  error: %s", cmd, result.ForLLM)
		}
	}
}

// TestShellTool_BlockDevices verifies that writes to block devices are blocked (issue #965).
func TestShellTool_BlockDevices(t *testing.T) {
	tool, err := NewExecTool("", false)
	if err != nil {
		t.Fatalf("unable to configure exec tool: %s", err)
	}

	blocked := []string{
		"echo x > /dev/sda",
		"echo x > /dev/hda",
		"echo x > /dev/vda",
		"echo x > /dev/xvda",
		"echo x > /dev/nvme0n1",
		"echo x > /dev/mmcblk0",
		"echo x > /dev/loop0",
		"echo x > /dev/dm-0",
		"echo x > /dev/md0",
		"echo x > /dev/sr0",
		"echo x > /dev/nbd0",
	}

	for _, cmd := range blocked {
		result := tool.Execute(context.Background(), map[string]any{"command": cmd})
		if !result.IsError {
			t.Errorf("expected block device write to be blocked: %s", cmd)
		}
	}
}

// TestShellTool_SafePathsInWorkspaceRestriction verifies that safe kernel pseudo-devices
// are allowed even when workspace restriction is active.
func TestShellTool_SafePathsInWorkspaceRestriction(t *testing.T) {
	tmpDir := t.TempDir()
	tool, err := NewExecTool(tmpDir, true)
	if err != nil {
		t.Fatalf("unable to configure exec tool: %s", err)
	}

	// These reference paths outside workspace but should be allowed via safePaths.
	commands := []string{
		"cat /dev/urandom | head -c 16 | od",
		"echo test > /dev/null",
		"dd if=/dev/zero bs=1 count=1",
	}

	for _, cmd := range commands {
		result := tool.Execute(context.Background(), map[string]any{"command": cmd})
		if result.IsError && strings.Contains(result.ForLLM, "path outside working dir") {
			t.Errorf("safe path should not be blocked by workspace check: %s\n  error: %s", cmd, result.ForLLM)
		}
	}
}

// TestShellTool_CustomAllowPatterns verifies that custom allow patterns exempt
// commands from deny pattern checks.
func TestShellTool_CustomAllowPatterns(t *testing.T) {
	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Exec: config.ExecConfig{
				EnableDenyPatterns:  true,
				CustomAllowPatterns: []string{`\bgit\s+push\s+origin\b`},
			},
		},
	}

	tool, err := NewExecToolWithConfig("", false, cfg)
	if err != nil {
		t.Fatalf("unable to configure exec tool: %s", err)
	}

	// "git push origin main" should be allowed by custom allow pattern.
	result := tool.Execute(context.Background(), map[string]any{
		"command": "git push origin main",
	})
	if result.IsError && strings.Contains(result.ForLLM, "blocked") {
		t.Errorf("custom allow pattern should exempt 'git push origin main', got: %s", result.ForLLM)
	}

	// "git push upstream main" should still be blocked (does not match allow pattern).
	result = tool.Execute(context.Background(), map[string]any{
		"command": "git push upstream main",
	})
	if !result.IsError {
		t.Errorf("'git push upstream main' should still be blocked by deny pattern")
	}
}

// TestShellTool_URLsNotBlocked verifies that commands containing URLs are not
// incorrectly blocked by the workspace restriction safety guard (issue #1203).
func TestShellTool_URLsNotBlocked(t *testing.T) {
	tmpDir := t.TempDir()
	tool, err := NewExecTool(tmpDir, true)
	if err != nil {
		t.Fatalf("unable to configure exec tool: %s", err)
	}

	// These commands contain URLs and should NOT be blocked by workspace restriction.
	// The URL path components (e.g., "//github.com") should be recognized as URLs,
	// not as file system paths.
	commands := []string{
		"agent-browser open https://github.com",
		"curl https://api.example.com/data",
		"wget http://example.com/file",
		"browser open https://github.com/user/repo",
		"fetch ftp://ftp.example.com/file.txt",
		"git clone https://github.com/sipeed/picoclaw.git",
	}

	for _, cmd := range commands {
		result := tool.Execute(context.Background(), map[string]any{"command": cmd})
		if result.IsError && strings.Contains(result.ForLLM, "path outside working dir") {
			t.Errorf("command with URL should not be blocked by workspace check: %s\n  error: %s", cmd, result.ForLLM)
		}
	}
}

// TestShellTool_FileURISandboxing verifies that file:// URIs that escape the
// workspace are still blocked, even though other URLs are allowed (issue #1254).
func TestShellTool_FileURISandboxing(t *testing.T) {
	tmpDir := t.TempDir()
	tool, err := NewExecTool(tmpDir, true)
	if err != nil {
		t.Fatalf("unable to configure exec tool: %s", err)
	}

	// These file:// URIs should be blocked if they reference paths outside the workspace.
	// Unlike web URLs (http://, https://, ftp://), file:// URIs can be used to escape the sandbox.
	blockedCommands := []string{
		"cat file:///etc/passwd",
		"cat file:///etc/hosts",
		"cat file:///root/.ssh/id_rsa",
	}

	for _, cmd := range blockedCommands {
		result := tool.Execute(context.Background(), map[string]any{"command": cmd})
		if !result.IsError || !strings.Contains(result.ForLLM, "path outside working dir") {
			t.Errorf("file:// URI outside workspace should be blocked: %s", cmd)
		}
	}

	// These file:// URIs should be allowed if they reference paths inside the workspace.
	// Create a test file inside the temp directory
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %s", err)
	}

	allowedCommands := []string{
		"cat file://" + testFile,
	}

	for _, cmd := range allowedCommands {
		result := tool.Execute(context.Background(), map[string]any{"command": cmd})
		if result.IsError && strings.Contains(result.ForLLM, "path outside working dir") {
			t.Errorf("file:// URI inside workspace should be allowed: %s\n  error: %s", cmd, result.ForLLM)
		}
	}
}

// TestShellTool_URLBypassPrevented verifies that a command cannot bypass the workspace
// sandbox by smuggling a real path after a URL that contains the same //path substring.
// e.g. "echo https://etc/passwd && cat //etc/passwd" must still be blocked.
func TestShellTool_URLBypassPrevented(t *testing.T) {
	tmpDir := t.TempDir()
	tool, err := NewExecTool(tmpDir, true)
	if err != nil {
		t.Fatalf("unable to configure exec tool: %s", err)
	}

	// The path //etc/passwd appears twice: once as the host part of an https URL
	// and once as a real (escaped) absolute path. The guard must block the command
	// because the second occurrence is a genuine out-of-workspace path.
	blockedCommands := []string{
		"echo https://etc/passwd && cat //etc/passwd",
		"curl https://host/file && ls //etc",
	}

	for _, cmd := range blockedCommands {
		result := tool.Execute(context.Background(), map[string]any{"command": cmd})
		if !result.IsError || !strings.Contains(result.ForLLM, "path outside working dir") {
			t.Errorf("bypass attempt should be blocked: %q\n  got: %s", cmd, result.ForLLM)
		}
	}
}
