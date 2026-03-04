package tools

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestBgMonitor_List(t *testing.T) {
	tool, _ := NewExecTool("", false)

	monitor := NewBgMonitorTool(tool)

	// List with no processes

	result := monitor.Execute(context.Background(), map[string]any{"action": "list"})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForLLM, "No background") {
		t.Errorf("expected 'No background' message, got: %s", result.ForLLM)
	}

	// Start two bg processes

	var cmd1, cmd2 string

	if runtime.GOOS == "windows" {
		cmd1 = "Start-Sleep -Seconds 30"

		cmd2 = "Start-Sleep -Seconds 30"
	} else {
		cmd1 = "sleep 30"

		cmd2 = "sleep 30"
	}

	r1 := tool.Execute(context.Background(), map[string]any{
		"command": cmd1,

		"background": true,
	})

	if r1.IsError {
		t.Fatalf("failed to start bg-1: %s", r1.ForLLM)
	}

	r2 := tool.Execute(context.Background(), map[string]any{
		"command": cmd2,

		"background": true,
	})

	if r2.IsError {
		t.Fatalf("failed to start bg-2: %s", r2.ForLLM)
	}

	// List should show both

	result = monitor.Execute(context.Background(), map[string]any{"action": "list"})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForLLM, "bg-1") {
		t.Errorf("expected bg-1 in list, got: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForLLM, "bg-2") {
		t.Errorf("expected bg-2 in list, got: %s", result.ForLLM)
	}

	// Cleanup

	tool.Shutdown()
}

func TestBgMonitor_Watch_Match(t *testing.T) {
	tool, _ := NewExecTool("", false)

	monitor := NewBgMonitorTool(tool)

	var cmd string

	if runtime.GOOS == "windows" {
		cmd = "Write-Output 'Server ready on port 3000'; Start-Sleep -Seconds 30"
	} else {
		cmd = "echo 'Server ready on port 3000'; sleep 30"
	}

	r := tool.Execute(context.Background(), map[string]any{
		"command": cmd,

		"background": true,
	})

	if r.IsError {
		t.Fatalf("failed to start bg: %s", r.ForLLM)
	}

	// Watch for "ready" pattern — should match quickly

	result := monitor.Execute(context.Background(), map[string]any{
		"action": "watch",

		"bg_id": "bg-1",

		"pattern": "ready",

		"watch_timeout": float64(10),
	})

	if result.IsError {
		t.Fatalf("expected watch to match, got error: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForLLM, "Match found") {
		t.Errorf("expected 'Match found' message, got: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForLLM, "ready") {
		t.Errorf("expected match to contain 'ready', got: %s", result.ForLLM)
	}

	tool.Shutdown()
}

func TestBgMonitor_Watch_Timeout(t *testing.T) {
	tool, _ := NewExecTool("", false)

	monitor := NewBgMonitorTool(tool)

	var cmd string

	if runtime.GOOS == "windows" {
		cmd = "Start-Sleep -Seconds 30"
	} else {
		cmd = "sleep 30"
	}

	r := tool.Execute(context.Background(), map[string]any{
		"command": cmd,

		"background": true,
	})

	if r.IsError {
		t.Fatalf("failed to start bg: %s", r.ForLLM)
	}

	// Watch for a pattern that won't appear, with short timeout

	result := monitor.Execute(context.Background(), map[string]any{
		"action": "watch",

		"bg_id": "bg-1",

		"pattern": "never_going_to_match",

		"watch_timeout": float64(1),
	})

	if !result.IsError {
		t.Fatalf("expected watch to timeout with error, got success: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForLLM, "timed out") {
		t.Errorf("expected 'timed out' message, got: %s", result.ForLLM)
	}

	tool.Shutdown()
}

func TestBgMonitor_Watch_ProcessExit(t *testing.T) {
	tool, _ := NewExecTool("", false)

	monitor := NewBgMonitorTool(tool)

	var cmd string

	if runtime.GOOS == "windows" {
		cmd = "Write-Output 'done quickly'"
	} else {
		cmd = "echo 'done quickly'"
	}

	r := tool.Execute(context.Background(), map[string]any{
		"command": cmd,

		"background": true,
	})

	if r.IsError {
		t.Fatalf("failed to start bg: %s", r.ForLLM)
	}

	// Wait a bit for the process to exit

	time.Sleep(4 * time.Second)

	// Watch for a pattern that doesn't match — process should have exited

	result := monitor.Execute(context.Background(), map[string]any{
		"action": "watch",

		"bg_id": "bg-1",

		"pattern": "never_match",

		"watch_timeout": float64(5),
	})

	if !result.IsError {
		t.Fatalf("expected error when process exits, got: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForLLM, "exited") {
		t.Errorf("expected 'exited' message, got: %s", result.ForLLM)
	}

	tool.Shutdown()
}

func TestBgMonitor_Tail(t *testing.T) {
	tool, _ := NewExecTool("", false)

	monitor := NewBgMonitorTool(tool)

	var cmd string

	if runtime.GOOS == "windows" {
		cmd = "1..5 | ForEach-Object { Write-Output \"line $_\" }; Start-Sleep -Seconds 30"
	} else {
		cmd = "for i in 1 2 3 4 5; do echo \"line $i\"; done; sleep 30"
	}

	r := tool.Execute(context.Background(), map[string]any{
		"command": cmd,

		"background": true,
	})

	if r.IsError {
		t.Fatalf("failed to start bg: %s", r.ForLLM)
	}

	// Wait for initial output to be captured

	time.Sleep(4 * time.Second)

	// Tail last 3 lines

	result := monitor.Execute(context.Background(), map[string]any{
		"action": "tail",

		"bg_id": "bg-1",

		"lines": float64(3),
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForLLM, "line 5") {
		t.Errorf("expected tail to contain 'line 5', got: %s", result.ForLLM)
	}

	tool.Shutdown()
}

func TestBgMonitor_InvalidAction(t *testing.T) {
	tool, _ := NewExecTool("", false)

	monitor := NewBgMonitorTool(tool)

	result := monitor.Execute(context.Background(), map[string]any{"action": "invalid"})

	if !result.IsError {
		t.Fatalf("expected error for invalid action")
	}

	if !strings.Contains(result.ForLLM, "unknown action") {
		t.Errorf("expected 'unknown action' message, got: %s", result.ForLLM)
	}
}
