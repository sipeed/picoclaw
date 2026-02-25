package agent

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/tools"
)

func TestHandleSubagentTaskEvent_PersistsParentTaskID(t *testing.T) {
	ledger := tools.NewTaskLedger(filepath.Join(t.TempDir(), "tasks", "ledger.json"))
	cfg := config.DefaultConfig()
	cfg.Orchestration.DefaultTaskTimeoutSeconds = 60

	now := time.Now().UnixMilli()
	handleSubagentTaskEvent(ledger, cfg, tools.SubagentTaskEvent{
		Type: tools.SubagentTaskRunning,
		Task: tools.SubagentTask{
			ID:            "subagent-2",
			ParentTaskID:  "subagent-1",
			Task:          "child task",
			AgentID:       "worker",
			OriginChannel: "telegram",
			OriginChatID:  "chat-1",
			Status:        "running",
			Created:       now,
		},
		Timestamp: now,
	})

	entry, ok := ledger.Get("subagent-2")
	if !ok {
		t.Fatal("expected task in ledger")
	}
	if entry.ParentTaskID != "subagent-1" {
		t.Fatalf("ParentTaskID = %q, want %q", entry.ParentTaskID, "subagent-1")
	}
	if entry.Status != tools.TaskStatusRunning {
		t.Fatalf("Status = %q, want %q", entry.Status, tools.TaskStatusRunning)
	}
}
