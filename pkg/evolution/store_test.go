package evolution_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/evolution"
)

func TestStore_AppendLearningRecordsPersistsCaseAndRule(t *testing.T) {
	root := t.TempDir()
	paths := evolution.NewPaths(root, "")
	store := evolution.NewStore(paths)

	records := []evolution.LearningRecord{
		{
			ID:          "case-1",
			Kind:        evolution.RecordKindCase,
			WorkspaceID: "ws-1",
			CreatedAt:   time.Unix(1700000000, 0).UTC(),
			Summary:     "weather task completed",
			Status:      evolution.RecordStatus("new"),
		},
		{
			ID:          "rule-1",
			Kind:        evolution.RecordKindRule,
			WorkspaceID: "ws-1",
			CreatedAt:   time.Unix(1700000100, 0).UTC(),
			Summary:     "prefer native-name weather path",
			Status:      evolution.RecordStatus("ready"),
		},
	}

	if err := store.AppendLearningRecords(records); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	loaded, err := store.LoadLearningRecords()
	if err != nil {
		t.Fatalf("LoadLearningRecords: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("len(loaded) = %d, want 2", len(loaded))
	}
	if loaded[1].Kind != evolution.RecordKindRule {
		t.Fatalf("loaded[1].Kind = %q, want %q", loaded[1].Kind, evolution.RecordKindRule)
	}
}

func TestStore_SaveDraftsOverwritesByID(t *testing.T) {
	root := t.TempDir()
	paths := evolution.NewPaths(root, "")
	store := evolution.NewStore(paths)

	first := evolution.SkillDraft{
		ID:              "draft-1",
		WorkspaceID:     "ws-1",
		CreatedAt:       time.Unix(1700000000, 0).UTC(),
		SourceRecordID:  "rule-1",
		TargetSkillName: "weather",
		DraftType:       evolution.DraftTypeShortcut,
		ChangeKind:      evolution.ChangeKindAppend,
		HumanSummary:    "prefer native-name path first",
		BodyOrPatch:     "## Start Here",
		Status:          evolution.DraftStatusCandidate,
	}
	second := first
	second.HumanSummary = "updated summary"

	if err := store.SaveDrafts([]evolution.SkillDraft{first}); err != nil {
		t.Fatalf("SaveDrafts(first): %v", err)
	}
	if err := store.SaveDrafts([]evolution.SkillDraft{second}); err != nil {
		t.Fatalf("SaveDrafts(second): %v", err)
	}

	loaded, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("len(loaded) = %d, want 1", len(loaded))
	}
	if loaded[0].HumanSummary != "updated summary" {
		t.Fatalf("HumanSummary = %q, want %q", loaded[0].HumanSummary, "updated summary")
	}
}

func TestStore_SaveDraftsKeepsSameIDDifferentWorkspace(t *testing.T) {
	root := t.TempDir()
	paths := evolution.NewPaths(root, "")
	store := evolution.NewStore(paths)

	first := evolution.SkillDraft{
		ID:              "draft-1",
		WorkspaceID:     "ws-1",
		CreatedAt:       time.Unix(1700000000, 0).UTC(),
		SourceRecordID:  "rule-1",
		TargetSkillName: "weather",
		DraftType:       evolution.DraftTypeShortcut,
		ChangeKind:      evolution.ChangeKindAppend,
		HumanSummary:    "workspace one",
		BodyOrPatch:     "## Start Here",
		Status:          evolution.DraftStatusCandidate,
	}
	second := first
	second.WorkspaceID = "ws-2"
	second.HumanSummary = "workspace two"

	if err := store.SaveDrafts([]evolution.SkillDraft{first}); err != nil {
		t.Fatalf("SaveDrafts(first): %v", err)
	}
	if err := store.SaveDrafts([]evolution.SkillDraft{second}); err != nil {
		t.Fatalf("SaveDrafts(second): %v", err)
	}

	loaded, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("len(loaded) = %d, want 2", len(loaded))
	}
	if loaded[0].WorkspaceID == loaded[1].WorkspaceID {
		t.Fatalf("loaded drafts should keep distinct workspace IDs: %+v", loaded)
	}
}

func TestStore_LoadLearningRecordsIgnoresTruncatedTrailingLine(t *testing.T) {
	root := t.TempDir()
	paths := evolution.NewPaths(root, "")
	store := evolution.NewStore(paths)

	record := evolution.LearningRecord{
		ID:          "case-1",
		Kind:        evolution.RecordKindCase,
		WorkspaceID: "ws-1",
		CreatedAt:   time.Unix(1700000000, 0).UTC(),
		Summary:     "weather task completed",
		Status:      evolution.RecordStatus("new"),
	}
	if err := store.AppendLearningRecords([]evolution.LearningRecord{record}); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	f, err := os.OpenFile(paths.LearningRecords, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	if _, err := f.WriteString("{\"id\":\"broken\""); err != nil {
		f.Close()
		t.Fatalf("WriteString: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	loaded, err := store.LoadLearningRecords()
	if err != nil {
		t.Fatalf("LoadLearningRecords: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("len(loaded) = %d, want 1", len(loaded))
	}
	if loaded[0].ID != "case-1" {
		t.Fatalf("loaded[0].ID = %q, want %q", loaded[0].ID, "case-1")
	}

	data, err := os.ReadFile(paths.LearningRecords)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "\"broken\"") {
		t.Fatalf("expected test fixture to include broken trailing line")
	}
}
