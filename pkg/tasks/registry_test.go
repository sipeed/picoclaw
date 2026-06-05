package tasks

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRegistryPersistsAndReloadsRecords(t *testing.T) {
	store := filepath.Join(t.TempDir(), "state", "task_registry.json")

	registry := NewRegistry(store)
	if err := registry.Upsert(Record{
		TaskID:         "subagent-7",
		Runtime:        RuntimeSubagent,
		Task:           "download media",
		Status:         StatusRunning,
		DeliveryStatus: DeliveryPending,
	}); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if err := registry.Update("subagent-7", func(rec *Record) {
		rec.Status = StatusSucceeded
		rec.DeliveryStatus = DeliveryDelivered
		rec.LastCompletionID = "completion-7"
		rec.DeliveredAt = 123
		rec.TerminalSummary = "done"
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	reloaded := NewRegistry(store)
	rec, ok := reloaded.Get("subagent-7")
	if !ok {
		t.Fatal("expected persisted task after reload")
	}
	if rec.Status != StatusSucceeded {
		t.Fatalf("Status = %q, want %q", rec.Status, StatusSucceeded)
	}
	if rec.DeliveryStatus != DeliveryDelivered {
		t.Fatalf("DeliveryStatus = %q, want %q", rec.DeliveryStatus, DeliveryDelivered)
	}
	if rec.TerminalSummary != "done" {
		t.Fatalf("TerminalSummary = %q, want done", rec.TerminalSummary)
	}
	if rec.LastCompletionID != "completion-7" {
		t.Fatalf("LastCompletionID = %q, want completion-7", rec.LastCompletionID)
	}
	if rec.DeliveredAt != 123 {
		t.Fatalf("DeliveredAt = %d, want 123", rec.DeliveredAt)
	}
}

func TestRegistryPersistsAndReloadsTaskEvents(t *testing.T) {
	store := filepath.Join(t.TempDir(), "state", "task_registry.json")
	registry := NewRegistry(store)

	if err := registry.Upsert(Record{
		TaskID:         "subagent-7",
		Runtime:        RuntimeSubagent,
		Task:           "download media",
		Status:         StatusRunning,
		DeliveryStatus: DeliveryPending,
	}); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if err := registry.Update("subagent-7", func(rec *Record) {
		rec.Status = StatusSucceeded
		rec.DeliveryStatus = DeliveryDelivered
		rec.ProgressSummary = "done"
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	reloaded := NewRegistry(store)
	events := reloaded.ListEvents("subagent-7")
	if len(events) != 4 {
		t.Fatalf("event count = %d, want 4: %+v", len(events), events)
	}
	wantTypes := []EventType{
		EventTaskUpserted,
		EventTaskStatusChanged,
		EventTaskDeliveryChanged,
		EventTaskProgress,
	}
	for i, want := range wantTypes {
		if events[i].Type != want {
			t.Fatalf("events[%d].Type = %q, want %q; events=%+v", i, events[i].Type, want, events)
		}
		if events[i].SchemaVersion != TaskEventSchemaVersion {
			t.Fatalf("events[%d].SchemaVersion = %q, want %q", i, events[i].SchemaVersion, TaskEventSchemaVersion)
		}
		if events[i].Seq != int64(i+1) {
			t.Fatalf("events[%d].Seq = %d, want %d", i, events[i].Seq, i+1)
		}
		if events[i].Fingerprint == "" {
			t.Fatalf("events[%d].Fingerprint is empty", i)
		}
	}
	if events[1].Payload["from"] != string(StatusRunning) || events[1].Payload["to"] != string(StatusSucceeded) {
		t.Fatalf("status event payload = %+v", events[1].Payload)
	}
	if events[2].Payload["from"] != string(DeliveryPending) ||
		events[2].Payload["to"] != string(DeliveryDelivered) {
		t.Fatalf("delivery event payload = %+v", events[2].Payload)
	}
	if events[3].Payload["summary"] != "done" {
		t.Fatalf("progress event payload = %+v", events[3].Payload)
	}
}

func TestRegistryListEventsCanReturnAllTasks(t *testing.T) {
	registry := NewRegistry("")
	for _, id := range []string{"task-a", "task-b"} {
		if err := registry.Upsert(Record{TaskID: id, Runtime: RuntimeTool, Task: id}); err != nil {
			t.Fatalf("Upsert(%s) error = %v", id, err)
		}
	}

	events := registry.ListEvents("")
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2: %+v", len(events), events)
	}
	if events[0].TaskID != "task-a" || events[1].TaskID != "task-b" {
		t.Fatalf("events = %+v, want task-a then task-b", events)
	}
}

func TestRegistryAppendEventPersistsSemanticPayload(t *testing.T) {
	store := filepath.Join(t.TempDir(), "state", "task_registry.json")
	registry := NewRegistry(store)
	if err := registry.Upsert(Record{
		TaskID:         "task-1",
		Runtime:        RuntimeTool,
		Task:           "deliver result",
		Status:         StatusRunning,
		DeliveryStatus: DeliveryPending,
	}); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if err := registry.AppendEvent("task-1", EventTaskDeliveryDecision, map[string]string{
		"mode":          "user_only",
		"completion_id": "completion-1",
	}); err != nil {
		t.Fatalf("AppendEvent() error = %v", err)
	}

	reloaded := NewRegistry(store)
	events := reloaded.ListEvents("task-1")
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2: %+v", len(events), events)
	}
	if events[1].Type != EventTaskDeliveryDecision {
		t.Fatalf("event type = %q, want %q", events[1].Type, EventTaskDeliveryDecision)
	}
	if events[1].Payload["mode"] != "user_only" ||
		events[1].Payload["completion_id"] != "completion-1" {
		t.Fatalf("event payload = %+v", events[1].Payload)
	}
}

func TestRegistryPersistsTaskBoardAndDeliverableFields(t *testing.T) {
	store := filepath.Join(t.TempDir(), "state", "task_registry.json")
	registry := NewRegistry(store)

	if err := registry.Upsert(Record{
		TaskID:       "delegate-1",
		Runtime:      RuntimeDelegate,
		TaskKind:     "delegate",
		BoardID:      "board-1",
		ParentTaskID: "root-1",
		StepID:       "download",
		StepTitle:    "Download media",
		Owner:        "media",
		DependsOn:    []string{"root-1"},
		BlockedBy:    []string{"caption"},
		Task:         "download the reel",
		Status:       StatusSucceeded,
		Deliverable: &DeliverablePayload{
			Text: "video downloaded",
			Artifacts: []DeliverableItem{
				{
					Ref:         "media://video",
					Kind:        "video",
					Filename:    "source.mp4",
					ContentType: "video/mp4",
					Delivered:   true,
				},
			},
			Metadata: map[string]string{"source": "instagram"},
		},
	}); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	reloaded := NewRegistry(store)
	rec, ok := reloaded.Get("delegate-1")
	if !ok {
		t.Fatal("expected persisted task after reload")
	}
	if rec.BoardID != "board-1" || rec.ParentTaskID != "root-1" || rec.Owner != "media" {
		t.Fatalf("unexpected board fields: %+v", rec)
	}
	if len(rec.DependsOn) != 1 || rec.DependsOn[0] != "root-1" {
		t.Fatalf("DependsOn = %#v, want root-1", rec.DependsOn)
	}
	if rec.Deliverable == nil || len(rec.Deliverable.Artifacts) != 1 {
		t.Fatalf("unexpected deliverable: %+v", rec.Deliverable)
	}
	if rec.Deliverable.Artifacts[0].Ref != "media://video" {
		t.Fatalf("artifact ref = %q, want media://video", rec.Deliverable.Artifacts[0].Ref)
	}
	if rec.Deliverable.Metadata["source"] != "instagram" {
		t.Fatalf("metadata source = %q, want instagram", rec.Deliverable.Metadata["source"])
	}
	if rec.Deliverable.Report == nil {
		t.Fatal("expected deliverable report projection")
	}
	if rec.Deliverable.Report.SchemaVersion != DeliverableReportV1 {
		t.Fatalf(
			"report schema = %q, want %q",
			rec.Deliverable.Report.SchemaVersion,
			DeliverableReportV1,
		)
	}
	if rec.Deliverable.Report.Summary != "video downloaded" {
		t.Fatalf("report summary = %q, want video downloaded", rec.Deliverable.Report.Summary)
	}
	if rec.Deliverable.Report.ContentHash == "" || rec.Deliverable.Report.ReportID == "" {
		t.Fatalf("report identity not populated: %+v", rec.Deliverable.Report)
	}
	if len(rec.Deliverable.Report.Claims) != 1 || rec.Deliverable.Report.Claims[0].Kind != "fact" {
		t.Fatalf("unexpected report claims: %+v", rec.Deliverable.Report.Claims)
	}
}

func TestRegistryPreservesExplicitDeliverableReport(t *testing.T) {
	registry := NewRegistry("")
	if err := registry.Upsert(Record{
		TaskID: "delegate-1",
		Task:   "review code",
		Deliverable: &DeliverablePayload{
			Text: "reviewed",
			Report: &DeliverableReport{
				SchemaVersion: DeliverableReportV1,
				ReportID:      "review-1",
				ContentHash:   "abc123",
				Summary:       "No findings",
				Claims: []ReportClaim{{
					Kind:       "negative_evidence",
					Text:       "No high-confidence issues found",
					Confidence: "high",
				}},
				Provenance: map[string]string{"producer": "reviewer"},
			},
		},
	}); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	rec, ok := registry.Get("delegate-1")
	if !ok {
		t.Fatal("expected task")
	}
	report := rec.Deliverable.Report
	if report.ReportID != "review-1" || report.ContentHash != "abc123" {
		t.Fatalf("explicit report identity changed: %+v", report)
	}
	if report.GeneratedAt == 0 {
		t.Fatalf("expected GeneratedAt to be filled: %+v", report)
	}
	if report.Provenance["producer"] != "reviewer" {
		t.Fatalf("explicit provenance lost: %+v", report.Provenance)
	}
}

func TestRegistryDefaultsBoardFields(t *testing.T) {
	registry := NewRegistry("")
	if err := registry.Upsert(Record{
		TaskID:  "task-1",
		AgentID: "media",
		Task:    "download",
	}); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	rec, ok := registry.Get("task-1")
	if !ok {
		t.Fatal("expected task")
	}
	if rec.BoardID != "task-1" {
		t.Fatalf("BoardID = %q, want task-1", rec.BoardID)
	}
	if rec.StepID != "task-1" {
		t.Fatalf("StepID = %q, want task-1", rec.StepID)
	}
	if rec.Owner != "media" {
		t.Fatalf("Owner = %q, want media", rec.Owner)
	}
}

func TestRegistryListBoard(t *testing.T) {
	registry := NewRegistry("")
	for _, rec := range []Record{
		{TaskID: "a-1", BoardID: "board-a", Task: "a1"},
		{TaskID: "a-2", BoardID: "board-a", Task: "a2"},
		{TaskID: "b-1", BoardID: "board-b", Task: "b1"},
	} {
		if err := registry.Upsert(rec); err != nil {
			t.Fatalf("Upsert(%s) error = %v", rec.TaskID, err)
		}
	}

	got := registry.ListBoard("board-a")
	if len(got) != 2 {
		t.Fatalf("ListBoard count = %d, want 2: %+v", len(got), got)
	}
	if got[0].TaskID != "a-1" || got[1].TaskID != "a-2" {
		t.Fatalf("ListBoard tasks = %+v, want a-1,a-2", got)
	}
}

func TestRegistryMaxNumericSuffix(t *testing.T) {
	registry := NewRegistry("")
	for _, id := range []string{"subagent-2", "subagent-10", "other-99"} {
		if err := registry.Upsert(Record{TaskID: id, Runtime: RuntimeSubagent, Task: "t"}); err != nil {
			t.Fatalf("Upsert(%s) error = %v", id, err)
		}
	}

	if got := registry.MaxNumericSuffix("subagent-"); got != 10 {
		t.Fatalf("MaxNumericSuffix() = %d, want 10", got)
	}
}

func TestRegistryStampsCleanupAfterForTerminalTasks(t *testing.T) {
	registry := NewRegistryWithOptions("", Options{TerminalRetention: time.Hour})
	endedAt := time.Now().UnixMilli()
	if err := registry.Upsert(Record{
		TaskID:         "task-1",
		Runtime:        RuntimeSubagent,
		Task:           "done",
		Status:         StatusSucceeded,
		DeliveryStatus: DeliveryDelivered,
		EndedAt:        endedAt,
	}); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	rec, ok := registry.Get("task-1")
	if !ok {
		t.Fatal("expected task")
	}
	if rec.CleanupAfter != endedAt+int64(time.Hour/time.Millisecond) {
		t.Fatalf("CleanupAfter = %d, want %d", rec.CleanupAfter, endedAt+int64(time.Hour/time.Millisecond))
	}
}

func TestRegistryPrunesExpiredTerminalTasks(t *testing.T) {
	store := filepath.Join(t.TempDir(), "state", "task_registry.json")
	registry := NewRegistryWithOptions(store, Options{TerminalRetention: time.Millisecond})

	if err := registry.Upsert(Record{
		TaskID:         "old-done",
		Runtime:        RuntimeSubagent,
		Task:           "old",
		Status:         StatusSucceeded,
		DeliveryStatus: DeliveryDelivered,
		EndedAt:        time.Now().Add(-time.Hour).UnixMilli(),
	}); err != nil {
		t.Fatalf("Upsert(old) error = %v", err)
	}
	if err := registry.Upsert(Record{
		TaskID:         "active",
		Runtime:        RuntimeSubagent,
		Task:           "active",
		Status:         StatusRunning,
		DeliveryStatus: DeliveryPending,
		CreatedAt:      time.Now().Add(-time.Hour).UnixMilli(),
	}); err != nil {
		t.Fatalf("Upsert(active) error = %v", err)
	}

	if _, ok := registry.Get("old-done"); ok {
		t.Fatal("expected expired terminal task to be pruned")
	}
	if _, ok := registry.Get("active"); !ok {
		t.Fatal("expected active task to be preserved")
	}
}

func TestRegistryPrunesOldestTerminalTasksAboveMaxRecords(t *testing.T) {
	registry := NewRegistryWithOptions("", Options{
		TerminalRetention: 24 * time.Hour,
		MaxRecords:        3,
	})

	records := []Record{
		{TaskID: "active-1", Status: StatusRunning, CreatedAt: time.Now().Add(-4 * time.Minute).UnixMilli()},
		{TaskID: "active-2", Status: StatusRunning, CreatedAt: time.Now().Add(-3 * time.Minute).UnixMilli()},
		{
			TaskID:    "old-terminal",
			Status:    StatusSucceeded,
			CreatedAt: time.Now().Add(-2 * time.Minute).UnixMilli(),
			EndedAt:   time.Now().Add(-2 * time.Minute).UnixMilli(),
		},
		{
			TaskID:    "new-terminal",
			Status:    StatusSucceeded,
			CreatedAt: time.Now().Add(-time.Minute).UnixMilli(),
			EndedAt:   time.Now().Add(-time.Minute).UnixMilli(),
		},
	}
	for _, rec := range records {
		rec.Runtime = RuntimeSubagent
		rec.Task = rec.TaskID
		rec.DeliveryStatus = DeliveryPending
		if err := registry.Upsert(rec); err != nil {
			t.Fatalf("Upsert(%s) error = %v", rec.TaskID, err)
		}
	}

	if _, ok := registry.Get("old-terminal"); ok {
		t.Fatal("expected oldest terminal task to be pruned")
	}
	for _, id := range []string{"active-1", "active-2", "new-terminal"} {
		if _, ok := registry.Get(id); !ok {
			t.Fatalf("expected %s to be preserved", id)
		}
	}
}

func TestRegistryListPendingTerminalDelivery(t *testing.T) {
	registry := NewRegistry("")
	records := []Record{
		{TaskID: "pending-done", Status: StatusSucceeded, DeliveryStatus: DeliveryPending},
		{TaskID: "pending-failed", Status: StatusFailed, DeliveryStatus: DeliveryPending},
		{TaskID: "pending-running", Status: StatusRunning, DeliveryStatus: DeliveryPending},
		{TaskID: "delivered-done", Status: StatusSucceeded, DeliveryStatus: DeliveryDelivered},
	}
	for _, rec := range records {
		rec.Runtime = RuntimeSubagent
		rec.Task = rec.TaskID
		if err := registry.Upsert(rec); err != nil {
			t.Fatalf("Upsert(%s) error = %v", rec.TaskID, err)
		}
	}

	got := registry.ListPendingTerminalDelivery()
	if len(got) != 2 {
		t.Fatalf("pending terminal count = %d, want 2: %+v", len(got), got)
	}
	if got[0].TaskID != "pending-done" || got[1].TaskID != "pending-failed" {
		t.Fatalf(
			"pending terminal tasks = %v, want pending-done,pending-failed",
			[]string{got[0].TaskID, got[1].TaskID},
		)
	}
}

func TestRegistryListActive(t *testing.T) {
	registry := NewRegistry("")
	for _, rec := range []Record{
		{TaskID: "queued", Status: StatusQueued},
		{TaskID: "running", Status: StatusRunning},
		{TaskID: "done", Status: StatusSucceeded},
		{TaskID: "lost", Status: StatusLost},
	} {
		rec.Runtime = RuntimeTool
		rec.Task = rec.TaskID
		if err := registry.Upsert(rec); err != nil {
			t.Fatalf("Upsert(%s) error = %v", rec.TaskID, err)
		}
	}

	got := registry.ListActive()
	if len(got) != 2 {
		t.Fatalf("active count = %d, want 2: %+v", len(got), got)
	}
	if got[0].TaskID != "queued" || got[1].TaskID != "running" {
		t.Fatalf("active tasks = %+v, want queued,running", got)
	}
}

func TestRegistryMarkStaleActiveLost(t *testing.T) {
	registry := NewRegistry("")
	old := time.Now().Add(-time.Hour).UnixMilli()
	recent := time.Now().UnixMilli()
	for _, rec := range []Record{
		{TaskID: "old-running", Status: StatusRunning, CreatedAt: old, LastEventAt: old},
		{TaskID: "recent-running", Status: StatusRunning, CreatedAt: recent, LastEventAt: recent},
		{TaskID: "done", Status: StatusSucceeded, CreatedAt: old, LastEventAt: old},
	} {
		rec.Runtime = RuntimeTool
		rec.Task = rec.TaskID
		rec.DeliveryStatus = DeliveryPending
		if err := registry.Upsert(rec); err != nil {
			t.Fatalf("Upsert(%s) error = %v", rec.TaskID, err)
		}
	}

	count, err := registry.MarkStaleActiveLost(30*time.Minute, "stale owner")
	if err != nil {
		t.Fatalf("MarkStaleActiveLost() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("changed count = %d, want 1", count)
	}
	oldRec, _ := registry.Get("old-running")
	if oldRec.Status != StatusLost {
		t.Fatalf("old-running status = %q, want lost", oldRec.Status)
	}
	if oldRec.DeliveryStatus != DeliveryNotApplicable {
		t.Fatalf("old-running delivery = %q, want not_applicable", oldRec.DeliveryStatus)
	}
	if oldRec.Error != "stale owner" {
		t.Fatalf("old-running error = %q, want stale owner", oldRec.Error)
	}
	recentRec, _ := registry.Get("recent-running")
	if recentRec.Status != StatusRunning {
		t.Fatalf("recent-running status = %q, want running", recentRec.Status)
	}
	doneRec, _ := registry.Get("done")
	if doneRec.Status != StatusSucceeded {
		t.Fatalf("done status = %q, want succeeded", doneRec.Status)
	}
}

func TestRegistryHeartbeatUpdatesOnlyActiveTasks(t *testing.T) {
	registry := NewRegistry("")
	old := time.Now().Add(-time.Hour).UnixMilli()
	for _, rec := range []Record{
		{
			TaskID:         "running",
			Runtime:        RuntimeDelegate,
			Task:           "running task",
			Status:         StatusRunning,
			DeliveryStatus: DeliveryPending,
			CreatedAt:      old,
			LastEventAt:    old,
		},
		{
			TaskID:         "done",
			Runtime:        RuntimeDelegate,
			Task:           "done task",
			Status:         StatusSucceeded,
			DeliveryStatus: DeliveryDelivered,
			CreatedAt:      old,
			LastEventAt:    old,
			EndedAt:        old,
		},
	} {
		if err := registry.Upsert(rec); err != nil {
			t.Fatalf("Upsert(%s) error = %v", rec.TaskID, err)
		}
	}

	if err := registry.Heartbeat("running", "still working"); err != nil {
		t.Fatalf("Heartbeat(running) error = %v", err)
	}
	if err := registry.Heartbeat("done", "should not change"); err != nil {
		t.Fatalf("Heartbeat(done) error = %v", err)
	}

	running, _ := registry.Get("running")
	if running.LastEventAt <= old {
		t.Fatalf("running LastEventAt = %d, want > %d", running.LastEventAt, old)
	}
	if running.ProgressSummary != "still working" {
		t.Fatalf("running ProgressSummary = %q, want still working", running.ProgressSummary)
	}
	done, _ := registry.Get("done")
	if done.LastEventAt != old {
		t.Fatalf("done LastEventAt = %d, want unchanged %d", done.LastEventAt, old)
	}
	if done.ProgressSummary != "" {
		t.Fatalf("done ProgressSummary = %q, want empty", done.ProgressSummary)
	}
}

func TestRegistryMarkActiveLost(t *testing.T) {
	registry := NewRegistry("")
	now := time.Now().UnixMilli()
	for _, rec := range []Record{
		{TaskID: "queued", Status: StatusQueued, CreatedAt: now, LastEventAt: now},
		{TaskID: "running", Status: StatusRunning, CreatedAt: now, LastEventAt: now},
		{TaskID: "done", Status: StatusSucceeded, CreatedAt: now, LastEventAt: now},
	} {
		rec.Runtime = RuntimeTool
		rec.Task = rec.TaskID
		rec.DeliveryStatus = DeliveryPending
		if err := registry.Upsert(rec); err != nil {
			t.Fatalf("Upsert(%s) error = %v", rec.TaskID, err)
		}
	}

	count, err := registry.MarkActiveLost("runtime restarted")
	if err != nil {
		t.Fatalf("MarkActiveLost() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("changed count = %d, want 2", count)
	}
	for _, id := range []string{"queued", "running"} {
		rec, _ := registry.Get(id)
		if rec.Status != StatusLost {
			t.Fatalf("%s status = %q, want lost", id, rec.Status)
		}
		if rec.DeliveryStatus != DeliveryNotApplicable {
			t.Fatalf("%s delivery = %q, want not_applicable", id, rec.DeliveryStatus)
		}
		if rec.Error != "runtime restarted" {
			t.Fatalf("%s error = %q, want runtime restarted", id, rec.Error)
		}
	}
	done, _ := registry.Get("done")
	if done.Status != StatusSucceeded {
		t.Fatalf("done status = %q, want succeeded", done.Status)
	}
}

func TestRegistryPlannedRecordsAreNotActiveOrLost(t *testing.T) {
	registry := NewRegistry(WorkspaceStorePath(t.TempDir()))
	if err := registry.Upsert(Record{
		TaskID:         "board:demo:step:one",
		Runtime:        RuntimeTool,
		TaskKind:       "task_board_step",
		Task:           "planned work",
		Status:         StatusPlanned,
		DeliveryStatus: DeliveryNotApplicable,
		CreatedAt:      time.Now().Add(-2 * time.Hour).UnixMilli(),
		LastEventAt:    time.Now().Add(-2 * time.Hour).UnixMilli(),
	}); err != nil {
		t.Fatalf("Upsert(planned) error = %v", err)
	}

	if active := registry.ListActive(); len(active) != 0 {
		t.Fatalf("ListActive returned planned records: %+v", active)
	}
	changed, err := registry.MarkStaleActiveLost(time.Minute, "stale")
	if err != nil {
		t.Fatalf("MarkStaleActiveLost error = %v", err)
	}
	if changed != 0 {
		t.Fatalf("MarkStaleActiveLost changed %d records, want 0", changed)
	}
	rec, ok := registry.Get("board:demo:step:one")
	if !ok {
		t.Fatal("planned record missing")
	}
	if rec.Status != StatusPlanned {
		t.Fatalf("planned status = %q, want %q", rec.Status, StatusPlanned)
	}
}

func TestRegistryMarkActiveLostEmitsTransitionEvents(t *testing.T) {
	registry := NewRegistry("")
	if err := registry.Upsert(Record{
		TaskID:         "running",
		Runtime:        RuntimeSubagent,
		Task:           "running task",
		Status:         StatusRunning,
		DeliveryStatus: DeliveryPending,
	}); err != nil {
		t.Fatalf("Upsert(running) error = %v", err)
	}

	changed, err := registry.MarkActiveLost("runtime restarted")
	if err != nil {
		t.Fatalf("MarkActiveLost error = %v", err)
	}
	if changed != 1 {
		t.Fatalf("changed = %d, want 1", changed)
	}
	events := registry.ListEvents("running")
	wantTypes := []EventType{
		EventTaskUpserted,
		EventTaskStatusChanged,
		EventTaskDeliveryChanged,
		EventTaskReconciled,
	}
	if len(events) != len(wantTypes) {
		t.Fatalf("event count = %d, want %d: %+v", len(events), len(wantTypes), events)
	}
	for i, want := range wantTypes {
		if events[i].Type != want {
			t.Fatalf("events[%d].Type = %q, want %q: %+v", i, events[i].Type, want, events)
		}
	}
	if events[3].Payload["reason"] != "runtime restarted" {
		t.Fatalf("reconciled payload = %+v", events[3].Payload)
	}
}
