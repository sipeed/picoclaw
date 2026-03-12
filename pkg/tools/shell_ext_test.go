package tools

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestGuardCommand_RelativePathWithSlashes(t *testing.T) {
	workspace := t.TempDir()

	tool, _ := NewExecTool(workspace, true)

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

func TestGuardCommand_VenvBinary(t *testing.T) {
	workspace := t.TempDir()

	tool, _ := NewExecTool(workspace, true)

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

func TestGuardCommand_ExecutableBinaryAllowed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix executable permission test not applicable on Windows")
	}

	workspace := t.TempDir()

	externalDir := t.TempDir()

	execPath := filepath.Join(externalDir, "mybin")

	os.WriteFile(execPath, []byte("#!/bin/sh\necho ok"), 0o755)

	tool, _ := NewExecTool(workspace, true)

	cmd := execPath + " --help"

	result := tool.guardCommand(cmd, workspace)

	if result != "" {
		t.Errorf("Executable binary outside workspace should be allowed: %q → %s", cmd, result)
	}
}

func TestGuardCommand_ExecutableBinaryAllowed_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	workspace := t.TempDir()

	externalDir := t.TempDir()

	execPath := filepath.Join(externalDir, "tool.exe")

	os.WriteFile(execPath, []byte("MZ"), 0o644)

	tool, _ := NewExecTool(workspace, true)

	cmd := execPath + " --version"

	result := tool.guardCommand(cmd, workspace)

	if result != "" {
		t.Errorf("Windows .exe outside workspace should be allowed: %q → %s", cmd, result)
	}
}

func TestGuardCommand_NonExecutableOutsideBlocked(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission test not applicable on Windows")
	}

	workspace := t.TempDir()

	externalDir := t.TempDir()

	dataFile := filepath.Join(externalDir, "secret.txt")

	os.WriteFile(dataFile, []byte("secret data"), 0o644)

	tool, _ := NewExecTool(workspace, true)

	cmd := "cat " + dataFile

	result := tool.guardCommand(cmd, workspace)

	if result == "" {
		t.Errorf("Non-executable file outside workspace should be blocked: %q", cmd)
	}

	if !strings.Contains(result, "path outside working dir") {
		t.Errorf("Expected 'path outside working dir' message, got: %s", result)
	}
}

func TestGuardCommand_NonExistentAbsolutePathBlocked(t *testing.T) {
	workspace := t.TempDir()

	tool, _ := NewExecTool(workspace, true)

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

func TestGuardCommand_FlagEmbeddedPathSkipped(t *testing.T) {
	workspace := t.TempDir()

	tool, _ := NewExecTool(workspace, true)

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

func TestGuardCommand_AbsolutePathInsideWorkspace(t *testing.T) {
	workspace := t.TempDir()

	tool, _ := NewExecTool(workspace, true)

	innerDir := filepath.Join(workspace, "projects", "myapp")

	os.MkdirAll(innerDir, 0o755)

	cmd := "ls " + innerDir

	result := tool.guardCommand(cmd, workspace)

	if result != "" {
		t.Errorf("Absolute path inside workspace should be allowed: %q → %s", cmd, result)
	}
}

func TestGuardCommand_PathTraversal(t *testing.T) {
	workspace := t.TempDir()

	tool, _ := NewExecTool(workspace, true)

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

func TestGuardCommand_CdWithAbsoluteWorkspacePath(t *testing.T) {
	workspace := t.TempDir()

	innerDir := filepath.Join(workspace, "projects", "foo")

	os.MkdirAll(innerDir, 0o755)

	tool, _ := NewExecTool(workspace, true)

	cmd := "cd " + innerDir + " && ls -la"

	result := tool.guardCommand(cmd, workspace)

	if result != "" {
		t.Errorf("cd to workspace subdir should be allowed: %q → %s", cmd, result)
	}
}

func TestGuardCommand_AgentCLISlashCommand(t *testing.T) {
	workspace := t.TempDir()

	tool, _ := NewExecTool(workspace, true)

	cmds := []string{
		`codex exec --yolo "/review skip-git-repo-check"`,

		`claude "/review"`,

		`gemini "/help"`,
	}

	for _, cmd := range cmds {
		result := tool.guardCommand(cmd, workspace)

		if result != "" {
			t.Errorf("Agent CLI slash command should not be blocked: %q → %s", cmd, result)
		}
	}

	if runtime.GOOS != "windows" {
		blocked := `cat /etc/hosts`

		result := tool.guardCommand(blocked, workspace)

		if result == "" {
			t.Errorf("Non-agent command with absolute path should be blocked: %q", blocked)
		}
	}
}

func TestGuardCommand_DenyPattern_IncludesPattern(t *testing.T) {
	workspace := t.TempDir()

	tool, _ := NewExecTool(workspace, true)

	tool.denyPatterns = append(tool.denyPatterns, regexp.MustCompile(`\bdangerous_cmd\b`))

	result := tool.guardCommand("dangerous_cmd --force", workspace)

	if result == "" {
		t.Fatal("expected deny pattern to block the command")
	}

	if !strings.Contains(result, "deny pattern") {
		t.Errorf("expected 'deny pattern' in message, got: %s", result)
	}

	if !strings.Contains(result, `\bdangerous_cmd\b`) {
		t.Errorf("expected pattern string in message, got: %s", result)
	}
}

func TestGuardCommand_Allowlist_ShowsRules(t *testing.T) {
	workspace := t.TempDir()

	tool, _ := NewExecTool(workspace, true)

	tool.SetAllowRules([]string{"go test", "git"})

	result := tool.guardCommand("curl http://example.com", workspace)

	if result == "" {
		t.Fatal("expected allowlist to block the command")
	}

	if !strings.Contains(result, "not in allowlist") {
		t.Errorf("expected 'not in allowlist' in message, got: %s", result)
	}

	if !strings.Contains(result, "go test") || !strings.Contains(result, "git") {
		t.Errorf("expected allowlist rules in message, got: %s", result)
	}
}

func TestGuardCommand_PathOutside_IncludesPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix absolute path test not applicable on Windows")
	}

	workspace := t.TempDir()

	externalDir := t.TempDir()

	dataFile := filepath.Join(externalDir, "secret.txt")

	os.WriteFile(dataFile, []byte("secret"), 0o644)

	tool, _ := NewExecTool(workspace, true)

	result := tool.guardCommand("cat "+dataFile, workspace)

	if result == "" {
		t.Fatal("expected path outside workspace to be blocked")
	}

	if !strings.Contains(result, "path outside working dir") {
		t.Errorf("expected 'path outside working dir' in message, got: %s", result)
	}

	if !strings.Contains(result, dataFile) {
		t.Errorf("expected offending path %q in message, got: %s", dataFile, result)
	}
}

func TestExecTool_Bg_StartAndOutput(t *testing.T) {
	tool, _ := NewExecTool("", false)

	defer tool.Shutdown()

	var cmd string

	if runtime.GOOS == "windows" {
		cmd = "Write-Output 'hello from bg'; Start-Sleep -Seconds 30"
	} else {
		cmd = "echo 'hello from bg'; sleep 30"
	}

	result := tool.Execute(context.Background(), map[string]any{
		"command": cmd,

		"background": true,
	})

	if result.IsError {
		t.Fatalf("failed to start bg process: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForLLM, "bg-1") {
		t.Errorf("expected bg-1 in result, got: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForLLM, "Background process started") {
		t.Errorf("expected start message, got: %s", result.ForLLM)
	}

	outputResult := tool.Execute(context.Background(), map[string]any{
		"bg_action": "output",

		"bg_id": "bg-1",
	})

	if outputResult.IsError {
		t.Fatalf("failed to get output: %s", outputResult.ForLLM)
	}

	if !strings.Contains(outputResult.ForLLM, "hello from bg") {
		t.Errorf("expected 'hello from bg' in output, got: %s", outputResult.ForLLM)
	}

	if !strings.Contains(outputResult.ForLLM, "running") {
		t.Errorf("expected 'running' status, got: %s", outputResult.ForLLM)
	}
}

func TestExecTool_Bg_Kill(t *testing.T) {
	tool, _ := NewExecTool("", false)

	defer tool.Shutdown()

	var cmd string

	if runtime.GOOS == "windows" {
		cmd = "Start-Sleep -Seconds 60"
	} else {
		cmd = "sleep 60"
	}

	result := tool.Execute(context.Background(), map[string]any{
		"command": cmd,

		"background": true,
	})

	if result.IsError {
		t.Fatalf("failed to start bg process: %s", result.ForLLM)
	}

	killResult := tool.Execute(context.Background(), map[string]any{
		"bg_action": "kill",

		"bg_id": "bg-1",
	})

	if killResult.IsError {
		t.Fatalf("failed to kill: %s", killResult.ForLLM)
	}

	if !strings.Contains(killResult.ForLLM, "terminated") {
		t.Errorf("expected 'terminated' message, got: %s", killResult.ForLLM)
	}

	procs := tool.BgProcesses()

	if _, ok := procs["bg-1"]; ok {
		t.Errorf("expected bg-1 to be removed after kill")
	}
}

func TestExecTool_Bg_ExitedProcess(t *testing.T) {
	tool, _ := NewExecTool("", false)

	defer tool.Shutdown()

	var cmd string

	if runtime.GOOS == "windows" {
		cmd = "Write-Output 'quick exit'"
	} else {
		cmd = "echo 'quick exit'"
	}

	result := tool.Execute(context.Background(), map[string]any{
		"command": cmd,

		"background": true,
	})

	if result.IsError {
		t.Fatalf("failed to start bg process: %s", result.ForLLM)
	}

	time.Sleep(4 * time.Second)

	outputResult := tool.Execute(context.Background(), map[string]any{
		"bg_action": "output",

		"bg_id": "bg-1",
	})

	if outputResult.IsError {
		t.Fatalf("failed to get output: %s", outputResult.ForLLM)
	}

	if !strings.Contains(outputResult.ForLLM, "exited") {
		t.Errorf("expected 'exited' in output, got: %s", outputResult.ForLLM)
	}

	if !strings.Contains(outputResult.ForLLM, "quick exit") {
		t.Errorf("expected 'quick exit' in output, got: %s", outputResult.ForLLM)
	}
}

func TestExecTool_Bg_InvalidID(t *testing.T) {
	tool, _ := NewExecTool("", false)

	defer tool.Shutdown()

	result := tool.Execute(context.Background(), map[string]any{
		"bg_action": "output",

		"bg_id": "bg-999",
	})

	if !result.IsError {
		t.Fatalf("expected error for invalid bg_id")
	}

	if !strings.Contains(result.ForLLM, "not found") {
		t.Errorf("expected 'not found' message, got: %s", result.ForLLM)
	}

	result = tool.Execute(context.Background(), map[string]any{
		"bg_action": "kill",

		"bg_id": "bg-999",
	})

	if !result.IsError {
		t.Fatalf("expected error for invalid bg_id")
	}
}

func TestExecTool_Bg_InitialOutputCapture(t *testing.T) {
	tool, _ := NewExecTool("", false)

	defer tool.Shutdown()

	var cmd string

	if runtime.GOOS == "windows" {
		cmd = "Write-Output 'initial line 1'; Write-Output 'initial line 2'; Start-Sleep -Seconds 30"
	} else {
		cmd = "echo 'initial line 1'; echo 'initial line 2'; sleep 30"
	}

	result := tool.Execute(context.Background(), map[string]any{
		"command": cmd,

		"background": true,
	})

	if result.IsError {
		t.Fatalf("failed to start bg process: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForLLM, "initial line 1") {
		t.Errorf("expected 'initial line 1' in initial output, got: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForLLM, "initial line 2") {
		t.Errorf("expected 'initial line 2' in initial output, got: %s", result.ForLLM)
	}
}

func TestExecTool_Bg_RuntimeStatus(t *testing.T) {
	tool, _ := NewExecTool("", false)

	defer tool.Shutdown()

	if s := tool.RuntimeStatus(); s != "" {
		t.Errorf("expected empty runtime status with no bg processes, got: %s", s)
	}

	var cmd string

	if runtime.GOOS == "windows" {
		cmd = "Start-Sleep -Seconds 30"
	} else {
		cmd = "sleep 30"
	}

	tool.Execute(context.Background(), map[string]any{
		"command": cmd,

		"background": true,
	})

	status := tool.RuntimeStatus()

	if !strings.Contains(status, "Background Processes") {
		t.Errorf("expected 'Background Processes' section, got: %s", status)
	}

	if !strings.Contains(status, "bg-1") {
		t.Errorf("expected 'bg-1' in status, got: %s", status)
	}

	if !strings.Contains(status, "running") {
		t.Errorf("expected 'running' in status, got: %s", status)
	}
}

func TestExecTool_Bg_Shutdown(t *testing.T) {
	tool, _ := NewExecTool("", false)

	var cmd string

	if runtime.GOOS == "windows" {
		cmd = "Start-Sleep -Seconds 60"
	} else {
		cmd = "sleep 60"
	}

	tool.Execute(context.Background(), map[string]any{
		"command": cmd,

		"background": true,
	})

	tool.Execute(context.Background(), map[string]any{
		"command": cmd,

		"background": true,
	})

	procs := tool.BgProcesses()

	for _, bp := range procs {
		if !bp.isRunning() {
			t.Errorf("expected process to be running before shutdown")
		}
	}

	tool.Shutdown()

	procs = tool.BgProcesses()

	for _, bp := range procs {
		if bp.isRunning() {
			t.Errorf("expected process to be stopped after shutdown")
		}
	}
}

func TestRingBuffer(t *testing.T) {
	t.Run("Write and String", func(t *testing.T) {
		rb := newRingBuffer(100)

		rb.Write([]byte("hello "))

		rb.Write([]byte("world"))

		if got := rb.String(); got != "hello world" {
			t.Errorf("expected 'hello world', got %q", got)
		}
	})

	t.Run("Lines", func(t *testing.T) {
		rb := newRingBuffer(100)

		rb.Write([]byte("line1\nline2\nline3\nline4\nline5\n"))

		lines := rb.Lines(3)

		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d", len(lines))
		}

		if lines[0] != "line3" || lines[1] != "line4" || lines[2] != "line5" {
			t.Errorf("unexpected lines: %v", lines)
		}
	})

	t.Run("Match", func(t *testing.T) {
		rb := newRingBuffer(100)

		rb.Write([]byte("starting...\nServer ready on port 3000\nwaiting...\n"))

		re := regexp.MustCompile(`ready.*port`)

		match := rb.Match(re)

		if match == "" {
			t.Fatal("expected match but got empty string")
		}

		if !strings.Contains(match, "ready") {
			t.Errorf("expected match to contain 'ready', got: %s", match)
		}

		re2 := regexp.MustCompile(`never_match`)

		match2 := rb.Match(re2)

		if match2 != "" {
			t.Errorf("expected no match, got: %s", match2)
		}
	})

	t.Run("Overflow", func(t *testing.T) {
		rb := newRingBuffer(10)

		rb.Write([]byte("1234567890ABCDEF"))

		got := rb.String()

		if len(got) != 10 {
			t.Errorf("expected buffer to be 10 bytes, got %d", len(got))
		}

		if got != "7890ABCDEF" {
			t.Errorf("expected '7890ABCDEF', got %q", got)
		}
	})

	t.Run("Len", func(t *testing.T) {
		rb := newRingBuffer(100)

		if rb.Len() != 0 {
			t.Errorf("expected 0 length initially")
		}

		rb.Write([]byte("hello"))

		if rb.Len() != 5 {
			t.Errorf("expected 5, got %d", rb.Len())
		}
	})

	t.Run("Empty Lines", func(t *testing.T) {
		rb := newRingBuffer(100)

		lines := rb.Lines(5)

		if lines != nil {
			t.Errorf("expected nil for empty buffer, got: %v", lines)
		}
	})
}

func TestExecTool_Bg_RingBufferOverflow(t *testing.T) {
	tool, _ := NewExecTool("", false)

	defer tool.Shutdown()

	var cmd string

	if runtime.GOOS == "windows" {
		cmd = "1..2000 | ForEach-Object { Write-Output ('x' * 50) }; Start-Sleep -Seconds 30"
	} else {
		cmd = "yes 'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx' | head -n 2000; sleep 30"
	}

	result := tool.Execute(context.Background(), map[string]any{
		"command": cmd,

		"background": true,
	})

	if result.IsError {
		t.Fatalf("failed to start bg process: %s", result.ForLLM)
	}

	time.Sleep(5 * time.Second)

	outputResult := tool.Execute(context.Background(), map[string]any{
		"bg_action": "output",

		"bg_id": "bg-1",
	})

	if outputResult.IsError {
		t.Fatalf("failed to get output: %s", outputResult.ForLLM)
	}

	procs := tool.BgProcesses()

	bp := procs["bg-1"]

	if bp == nil {
		t.Fatal("bg-1 not found")
	}

	bufLen := bp.output.Len()

	if bufLen > bgRingBufSize {
		t.Errorf("ring buffer exceeded max size: %d > %d", bufLen, bgRingBufSize)
	}
}

func TestIsLocalHost(t *testing.T) {
	tests := []struct {
		host string

		want bool
	}{

		{"localhost", true},

		{"LOCALHOST", true},

		{"127.0.0.1", true},

		{"127.0.0.2", true},

		{"::1", true},

		{"10.0.0.1", true},

		{"10.255.255.255", true},

		{"172.16.0.1", true},

		{"172.31.255.255", true},

		{"192.168.0.1", true},

		{"192.168.1.100", true},

		{"8.8.8.8", false},

		{"1.1.1.1", false},

		{"example.com", false},

		{"api.github.com", false},

		{"172.15.255.255", false},

		{"172.32.0.0", false},
	}

	for _, tt := range tests {
		got := isLocalHost(tt.host)

		if got != tt.want {
			t.Errorf("isLocalHost(%q) = %v, want %v", tt.host, got, tt.want)
		}
	}
}

func TestCheckCurlLocalNet(t *testing.T) {
	tests := []struct {
		cmd string

		wantErr bool
	}{

		{"curl http://localhost:3000/health", false},

		{"curl -v http://127.0.0.1:8080/api/status", false},

		{"wget http://192.168.1.10/file.bin", false},

		{"curl -X POST http://10.0.0.5:9000/webhook", false},

		{"curl http://example.com", true},

		{"wget https://releases.github.com/v1.tar.gz", true},

		{"curl http://8.8.8.8/data", true},

		{"curl --help", false},

		{"curl --version", false},

		{"wget --help", false},
	}

	for _, tt := range tests {
		errMsg := checkCurlLocalNet(tt.cmd)

		gotErr := errMsg != ""

		if gotErr != tt.wantErr {
			t.Errorf("checkCurlLocalNet(%q): gotErr=%v wantErr=%v (msg: %q)",

				tt.cmd, gotErr, tt.wantErr, errMsg)
		}
	}
}

func TestExecTool_LocalNetOnly(t *testing.T) {
	tool, _ := NewExecTool("", false)

	tool.SetLocalNetOnly(true)

	tests := []struct {
		cmd string

		wantErr bool
	}{
		{"curl http://localhost:3000", false},

		{"curl http://example.com", true},

		{"echo hello", false},
	}

	ctx := context.Background()

	for _, tt := range tests {
		result := tool.Execute(ctx, map[string]any{"command": tt.cmd})

		if tt.wantErr && !result.IsError {
			t.Errorf("cmd %q: expected blocked, but succeeded", tt.cmd)
		}

		if !tt.wantErr && result.IsError && strings.Contains(result.ForLLM, "safety guard") {
			t.Errorf("cmd %q: expected allowed, but safety guard blocked: %s", tt.cmd, result.ForLLM)
		}
	}
}
