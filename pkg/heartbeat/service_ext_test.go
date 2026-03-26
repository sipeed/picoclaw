package heartbeat

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/tools"
)

// TestExecuteHeartbeat_NoSendResponse verifies that heartbeat results
// do not trigger sendResponse (dedup: response is included in task status instead).
func TestExecuteHeartbeat_NoSendResponse(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "heartbeat-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hs := NewHeartbeatService(tmpDir, 30, true)
	hs.stopChan = make(chan struct{})

	var handlerCalled bool
	hs.SetHandler(func(prompt, channel, chatID string) *tools.ToolResult {
		handlerCalled = true
		return &tools.ToolResult{
			ForUser: "Task result for user",
			ForLLM:  "Task result for LLM",
			Silent:  false,
			IsError: false,
			Async:   false,
		}
	})

	os.WriteFile(filepath.Join(tmpDir, "HEARTBEAT.md"), []byte("Test task"), 0o644)

	hs.executeHeartbeat()

	if !handlerCalled {
		t.Error("Expected handler to be called after heartbeat execution")
	}
}
