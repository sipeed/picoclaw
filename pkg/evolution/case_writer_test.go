package evolution_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/evolution"
)

func TestCaseWriter_AppendsOneRecord(t *testing.T) {
	root := t.TempDir()
	paths := evolution.NewPaths(root, "")
	writer := evolution.NewCaseWriter(paths)

	record1 := testRecord("rec-1", "ws-1", true)
	record2 := testRecord("rec-2", "ws-2", false)

	if err := writer.AppendCase(context.Background(), record1); err != nil {
		t.Fatalf("AppendCase: %v", err)
	}
	if err := writer.AppendCase(context.Background(), record2); err != nil {
		t.Fatalf("AppendCase second record: %v", err)
	}

	data, err := os.ReadFile(paths.TaskRecords)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	text := string(data)
	if !strings.HasSuffix(text, "\n") {
		t.Fatalf("record file should end with newline, got %q", text)
	}

	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) != 2 {
		t.Fatalf("record file line count = %d, want 2", len(lines))
	}

	records := []evolution.LearningRecord{record1, record2}
	for i, line := range lines {
		var got evolution.LearningRecord
		if err := json.Unmarshal([]byte(line), &got); err != nil {
			t.Fatalf("Unmarshal line %d: %v", i, err)
		}

		want := records[i]
		if got.ID != want.ID {
			t.Fatalf("record %d ID = %q, want %q", i, got.ID, want.ID)
		}
		if got.Kind != evolution.RecordKindCase {
			t.Fatalf("record %d kind = %q, want %q", i, got.Kind, evolution.RecordKindCase)
		}
		if got.Summary != want.Summary {
			t.Fatalf("record %d summary = %q, want %q", i, got.Summary, want.Summary)
		}
		if got.Success == nil || *got.Success != *want.Success {
			t.Fatalf("record %d success = %v, want %v", i, got.Success, want.Success)
		}
	}
}

func testRecord(id, workspaceID string, success bool) evolution.LearningRecord {
	return evolution.LearningRecord{
		ID:          id,
		Kind:        evolution.RecordKindCase,
		WorkspaceID: workspaceID,
		CreatedAt:   time.Unix(1700000000, 0).UTC(),
		Summary:     "cli turn completed",
		Status:      evolution.RecordStatus("new"),
		Success:     &success,
	}
}
