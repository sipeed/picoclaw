package heartbeat

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/tools"
)

// TestExecuteHeartbeat_NoSendResponse verifies that heartbeat results
// do not trigger sendResponse (dedup: response is included in task status instead).
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

	hs.SetHandler(func(prompt, channel, chatID string) *tools.ToolResult {
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

	hs.mu.RLock()
	notified := !hs.lastNotifiedAt.IsZero()
	hs.mu.RUnlock()
	if !notified {
		t.Error("Expected lastNotifiedAt to be set after heartbeat completion")
	}
}

func TestExecuteHeartbeat_TargetPriority_ExplicitTarget(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "heartbeat-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hs := NewHeartbeatService(tmpDir, 30, true)
	hs.stopChan = make(chan struct{})
	hs.SetHeartbeatThreadID(77)
	if err := hs.state.SetHeartbeatTarget("slack:C12345/999"); err != nil {
		t.Fatalf("SetHeartbeatTarget failed: %v", err)
	}
	if err := hs.state.SetLastHeartbeatTarget("telegram:-100500"); err != nil {
		t.Fatalf("SetLastHeartbeatTarget failed: %v", err)
	}

	var gotChannel, gotChatID string
	hs.SetHandler(func(prompt, channel, chatID string) *tools.ToolResult {
		gotChannel, gotChatID = channel, chatID
		return tools.SilentResult("ok")
	})
	os.WriteFile(filepath.Join(tmpDir, "HEARTBEAT.md"), []byte("Test task"), 0o644)

	hs.executeHeartbeat()

	if gotChannel != "slack" || gotChatID != "C12345/999" {
		t.Fatalf("handler target = %s:%s, want slack:C12345/999", gotChannel, gotChatID)
	}
}

func TestExecuteHeartbeat_TargetPriority_TelegramThread(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "heartbeat-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hs := NewHeartbeatService(tmpDir, 30, true)
	hs.stopChan = make(chan struct{})
	hs.SetHeartbeatThreadID(77)
	if err := hs.state.SetLastHeartbeatTarget("telegram:-100500"); err != nil {
		t.Fatalf("SetLastHeartbeatTarget failed: %v", err)
	}

	var gotChannel, gotChatID string
	hs.SetHandler(func(prompt, channel, chatID string) *tools.ToolResult {
		gotChannel, gotChatID = channel, chatID
		return tools.SilentResult("ok")
	})
	os.WriteFile(filepath.Join(tmpDir, "HEARTBEAT.md"), []byte("Test task"), 0o644)

	hs.executeHeartbeat()

	if gotChannel != "telegram" || gotChatID != "-100500/77" {
		t.Fatalf("handler target = %s:%s, want telegram:-100500/77", gotChannel, gotChatID)
	}
}
