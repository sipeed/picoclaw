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
	if _, statErr := os.Stat(paths.LearningRecords); !os.IsNotExist(statErr) {
		t.Fatalf("legacy learning records file should not be written, stat err = %v", statErr)
	}
	if _, statErr := os.Stat(paths.TaskRecords); statErr != nil {
		t.Fatalf("task records file should exist: %v", statErr)
	}
	if _, statErr := os.Stat(paths.PatternRecords); statErr != nil {
		t.Fatalf("pattern records file should exist: %v", statErr)
	}
}

func TestStore_LoadTaskRecordsMergesLegacyWhenSplitFileExists(t *testing.T) {
	root := t.TempDir()
	paths := evolution.NewPaths(root, "")
	store := evolution.NewStore(paths)

	legacy := evolution.LearningRecord{
		ID:          "legacy-task",
		Kind:        evolution.RecordKindTask,
		WorkspaceID: "ws-1",
		CreatedAt:   time.Unix(1700000000, 0).UTC(),
		Summary:     "legacy task",
		Status:      evolution.RecordStatus("new"),
	}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("Marshal legacy: %v", err)
	}
	if mkdirErr := os.MkdirAll(paths.RootDir, 0o755); mkdirErr != nil {
		t.Fatalf("MkdirAll: %v", mkdirErr)
	}
	if writeErr := os.WriteFile(paths.LearningRecords, append(data, '\n'), 0o644); writeErr != nil {
		t.Fatalf("WriteFile legacy: %v", writeErr)
	}

	current := evolution.LearningRecord{
		ID:          "current-task",
		Kind:        evolution.RecordKindTask,
		WorkspaceID: "ws-1",
		CreatedAt:   time.Unix(1700000100, 0).UTC(),
		Summary:     "current task",
		Status:      evolution.RecordStatus("new"),
	}
	if appendErr := store.AppendTaskRecord(context.Background(), current); appendErr != nil {
		t.Fatalf("AppendTaskRecord: %v", appendErr)
	}

	records, err := store.LoadTaskRecords()
	if err != nil {
		t.Fatalf("LoadTaskRecords: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2: %+v", len(records), records)
	}
	ids := records[0].ID + "," + records[1].ID
	if !strings.Contains(ids, "legacy-task") || !strings.Contains(ids, "current-task") {
		t.Fatalf("records should include legacy and current task IDs, got %q", ids)
	}
}

func TestStore_LoadPatternRecordsMergesLegacyWhenSplitFileExists(t *testing.T) {
	root := t.TempDir()
	paths := evolution.NewPaths(root, "")
	store := evolution.NewStore(paths)

	legacy := evolution.LearningRecord{
		ID:          "legacy-pattern",
		Kind:        evolution.RecordKindPattern,
		WorkspaceID: "ws-1",
		CreatedAt:   time.Unix(1700000000, 0).UTC(),
		Summary:     "legacy pattern",
		Status:      evolution.RecordStatus("ready"),
	}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("Marshal legacy: %v", err)
	}
	if mkdirErr := os.MkdirAll(paths.RootDir, 0o755); mkdirErr != nil {
		t.Fatalf("MkdirAll: %v", mkdirErr)
	}
	if writeErr := os.WriteFile(paths.LearningRecords, append(data, '\n'), 0o644); writeErr != nil {
		t.Fatalf("WriteFile legacy: %v", writeErr)
	}

	current := evolution.LearningRecord{
		ID:          "current-pattern",
		Kind:        evolution.RecordKindPattern,
		WorkspaceID: "ws-1",
		CreatedAt:   time.Unix(1700000100, 0).UTC(),
		Summary:     "current pattern",
		Status:      evolution.RecordStatus("ready"),
	}
	if appendErr := store.AppendPatternRecords([]evolution.LearningRecord{current}); appendErr != nil {
		t.Fatalf("AppendPatternRecords: %v", appendErr)
	}

	records, err := store.LoadPatternRecords()
	if err != nil {
		t.Fatalf("LoadPatternRecords: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2: %+v", len(records), records)
	}
	ids := records[0].ID + "," + records[1].ID
	if !strings.Contains(ids, "legacy-pattern") || !strings.Contains(ids, "current-pattern") {
		t.Fatalf("records should include legacy and current pattern IDs, got %q", ids)
	}
}

func TestStore_MarkTaskRecordsClusteredPreservesNewerAppendedRecords(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))

	first := evolution.LearningRecord{
		ID:          "task-1",
		Kind:        evolution.RecordKindTask,
		WorkspaceID: "ws-1",
		CreatedAt:   time.Unix(1700000000, 0).UTC(),
		Summary:     "first task",
		Status:      evolution.RecordStatus("new"),
	}
	if err := store.AppendTaskRecord(context.Background(), first); err != nil {
		t.Fatalf("AppendTaskRecord(first): %v", err)
	}
	if _, err := store.LoadTaskRecords(); err != nil {
		t.Fatalf("LoadTaskRecords snapshot: %v", err)
	}

	second := evolution.LearningRecord{
		ID:          "task-2",
		Kind:        evolution.RecordKindTask,
		WorkspaceID: "ws-1",
		CreatedAt:   time.Unix(1700000100, 0).UTC(),
		Summary:     "second task",
		Status:      evolution.RecordStatus("new"),
	}
	if err := store.AppendTaskRecord(context.Background(), second); err != nil {
		t.Fatalf("AppendTaskRecord(second): %v", err)
	}

	if err := store.MarkTaskRecordsClustered([]string{"task-1"}); err != nil {
		t.Fatalf("MarkTaskRecordsClustered: %v", err)
	}

	records, err := store.LoadTaskRecords()
	if err != nil {
		t.Fatalf("LoadTaskRecords: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2: %+v", len(records), records)
	}
	statusByID := map[string]evolution.RecordStatus{}
	for _, record := range records {
		statusByID[record.ID] = record.Status
	}
	if statusByID["task-1"] != evolution.RecordStatus("clustered") {
		t.Fatalf("task-1 status = %q, want clustered", statusByID["task-1"])
	}
	if statusByID["task-2"] != evolution.RecordStatus("new") {
		t.Fatalf("task-2 status = %q, want new", statusByID["task-2"])
	}
}

func TestStore_MergeKeepsSameRecordIDAcrossWorkspaces(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths("workspace-a", root))

	records := []evolution.LearningRecord{
		{
			ID:          "main-turn-1",
			Kind:        evolution.RecordKindTask,
			WorkspaceID: "workspace-a",
			CreatedAt:   time.Unix(1700000000, 0).UTC(),
			Summary:     "workspace a task",
			Status:      evolution.RecordStatus("new"),
		},
		{
			ID:          "main-turn-1",
			Kind:        evolution.RecordKindTask,
			WorkspaceID: "workspace-b",
			CreatedAt:   time.Unix(1700000100, 0).UTC(),
			Summary:     "workspace b task",
			Status:      evolution.RecordStatus("new"),
		},
	}
	if err := store.AppendTaskRecords(context.Background(), records); err != nil {
		t.Fatalf("AppendTaskRecords: %v", err)
	}

	loaded, err := store.LoadTaskRecords()
	if err != nil {
		t.Fatalf("LoadTaskRecords: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("len(loaded) = %d, want 2: %+v", len(loaded), loaded)
	}

	if markErr := store.MarkTaskRecordsClustered([]string{"main-turn-1"}); markErr != nil {
		t.Fatalf("MarkTaskRecordsClustered: %v", markErr)
	}
	loaded, err = store.LoadTaskRecords()
	if err != nil {
		t.Fatalf("LoadTaskRecords after clustered: %v", err)
	}
	statusByWorkspace := map[string]evolution.RecordStatus{}
	for _, record := range loaded {
		statusByWorkspace[record.WorkspaceID] = record.Status
	}
	if statusByWorkspace["workspace-a"] != evolution.RecordStatus("clustered") {
		t.Fatalf("workspace-a status = %q, want clustered", statusByWorkspace["workspace-a"])
	}
	if statusByWorkspace["workspace-b"] != evolution.RecordStatus("new") {
		t.Fatalf("workspace-b status = %q, want new", statusByWorkspace["workspace-b"])
	}
}

func TestStore_MergePatternRecordsPreservesNewerWorkspaceRecords(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))

	first := evolution.LearningRecord{
		ID:          "pattern-a",
		Kind:        evolution.RecordKindPattern,
		WorkspaceID: "workspace-a",
		CreatedAt:   time.Unix(1700000000, 0).UTC(),
		Summary:     "workspace a pattern",
		Status:      evolution.RecordStatus("ready"),
	}
	if err := store.SavePatternRecords([]evolution.LearningRecord{first}); err != nil {
		t.Fatalf("SavePatternRecords(first): %v", err)
	}
	if _, err := store.LoadPatternRecords(); err != nil {
		t.Fatalf("LoadPatternRecords snapshot: %v", err)
	}

	second := evolution.LearningRecord{
		ID:          "pattern-b",
		Kind:        evolution.RecordKindPattern,
		WorkspaceID: "workspace-b",
		CreatedAt:   time.Unix(1700000100, 0).UTC(),
		Summary:     "workspace b pattern",
		Status:      evolution.RecordStatus("ready"),
	}
	if err := store.MergePatternRecords([]evolution.LearningRecord{second}); err != nil {
		t.Fatalf("MergePatternRecords: %v", err)
	}

	records, err := store.LoadPatternRecords()
	if err != nil {
		t.Fatalf("LoadPatternRecords: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2: %+v", len(records), records)
	}
	ids := records[0].ID + "," + records[1].ID
	if !strings.Contains(ids, "pattern-a") || !strings.Contains(ids, "pattern-b") {
		t.Fatalf("records should include both workspace patterns, got %q", ids)
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

	f, err := os.OpenFile(paths.TaskRecords, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	if _, writeErr := f.WriteString("{\"id\":\"broken\""); writeErr != nil {
		f.Close()
		t.Fatalf("WriteString: %v", writeErr)
	}
	if closeErr := f.Close(); closeErr != nil {
		t.Fatalf("Close: %v", closeErr)
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

	data, err := os.ReadFile(paths.TaskRecords)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "\"broken\"") {
		t.Fatalf("expected test fixture to include broken trailing line")
	}
}
