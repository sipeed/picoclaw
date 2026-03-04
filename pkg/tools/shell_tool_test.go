package tools

import (
	"context"
	"sync"
	"testing"
	"time"
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
