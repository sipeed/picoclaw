package research

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestStore(t *testing.T) (*ResearchStore, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "research.db")
	store, err := OpenResearchStore(dbPath, dir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store, dir
}

func TestCreateAndGetTask(t *testing.T) {
	store, dir := setupTestStore(t)

	task, err := store.CreateTask("Test Research", "A test description", "")
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if task.Title != "Test Research" {
		t.Errorf("title = %q, want %q", task.Title, "Test Research")
	}
	if task.Status != StatusPending {
		t.Errorf("status = %q, want %q", task.Status, StatusPending)
	}

	// Verify output directory was created
	absDir := filepath.Join(dir, task.OutputDir)
	if _, statErr := os.Stat(absDir); os.IsNotExist(statErr) {
		t.Errorf("output dir %q not created", absDir)
	}

	got, err := store.GetTask(task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.Title != task.Title || got.Slug != task.Slug {
		t.Errorf("get mismatch: got %+v", got)
	}
}

func TestListTasks(t *testing.T) {
	store, _ := setupTestStore(t)

	store.CreateTask("Task A", "", "")
	store.CreateTask("Task B", "", "")

	all, err := store.ListTasks("")
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("list all = %d, want 2", len(all))
	}

	pending, err := store.ListTasks(StatusPending)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 2 {
		t.Errorf("list pending = %d, want 2", len(pending))
	}

	active, err := store.ListTasks(StatusActive)
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(active) != 0 {
		t.Errorf("list active = %d, want 0", len(active))
	}
}

func TestSetTaskStatus(t *testing.T) {
	store, _ := setupTestStore(t)

	task, _ := store.CreateTask("Status Test", "", "")

	// Valid: pending → active
	if err := store.SetTaskStatus(task.ID, StatusActive); err != nil {
		t.Fatalf("pending→active: %v", err)
	}

	// Valid: active → completed
	if err := store.SetTaskStatus(task.ID, StatusCompleted); err != nil {
		t.Fatalf("active→completed: %v", err)
	}

	// Verify completed_at is set
	got, _ := store.GetTask(task.ID)
	if got.CompletedAt.IsZero() {
		t.Error("completed_at should be set")
	}

	// Valid: completed → pending (reopen)
	if err := store.SetTaskStatus(task.ID, StatusPending); err != nil {
		t.Fatalf("completed→pending: %v", err)
	}

	// Invalid: pending → completed
	if err := store.SetTaskStatus(task.ID, StatusCompleted); err == nil {
		t.Error("pending→completed should fail")
	}
}

func TestAddAndListDocuments(t *testing.T) {
	store, dir := setupTestStore(t)

	task, _ := store.CreateTask("Doc Test", "", "")

	// Create a test file
	filePath := filepath.Join(dir, task.OutputDir, "001-finding.md")
	os.WriteFile(filePath, []byte("# Finding\nSome content"), 0o644)

	doc, err := store.AddDocument(task.ID, "Finding 1", filePath, "finding", "A brief summary")
	if err != nil {
		t.Fatalf("add document: %v", err)
	}
	if doc.Seq != 1 {
		t.Errorf("seq = %d, want 1", doc.Seq)
	}

	doc2, _ := store.AddDocument(task.ID, "Finding 2", "path2.md", "finding", "")
	if doc2.Seq != 2 {
		t.Errorf("seq = %d, want 2", doc2.Seq)
	}

	docs, err := store.ListDocuments(task.ID)
	if err != nil {
		t.Fatalf("list documents: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("list docs = %d, want 2", len(docs))
	}
}

func TestDocumentCount(t *testing.T) {
	store, _ := setupTestStore(t)

	task, _ := store.CreateTask("Count Test", "", "")

	count, _ := store.DocumentCount(task.ID)
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}

	store.AddDocument(task.ID, "D1", "p1", "finding", "")
	store.AddDocument(task.ID, "D2", "p2", "note", "")

	count, _ = store.DocumentCount(task.ID)
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestDeleteTask(t *testing.T) {
	store, _ := setupTestStore(t)

	task, _ := store.CreateTask("Delete Test", "", "")
	store.AddDocument(task.ID, "D1", "p1", "finding", "")

	if err := store.DeleteTask(task.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Documents should be cascade deleted
	docs, _ := store.ListDocuments(task.ID)
	if len(docs) != 0 {
		t.Errorf("docs after delete = %d, want 0", len(docs))
	}
}

func TestUpdateTask(t *testing.T) {
	store, _ := setupTestStore(t)

	task, _ := store.CreateTask("Original", "desc", "")
	if err := store.UpdateTask(task.ID, "Updated", "new desc"); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, _ := store.GetTask(task.ID)
	if got.Title != "Updated" || got.Description != "new desc" {
		t.Errorf("after update: title=%q desc=%q", got.Title, got.Description)
	}
}

func TestParseInterval(t *testing.T) {
	cases := []struct {
		input string
		want  string // duration string
		err   bool
	}{
		{"1d", "24h0m0s", false},
		{"7d", "168h0m0s", false},
		{"6h", "6h0m0s", false},
		{"30m", "30m0s", false},
		{"", "", true},
		{"abc", "", true},
	}
	for _, tc := range cases {
		got, err := ParseInterval(tc.input)
		if tc.err {
			if err == nil {
				t.Errorf("ParseInterval(%q) expected error", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseInterval(%q) error: %v", tc.input, err)
			continue
		}
		if got.String() != tc.want {
			t.Errorf("ParseInterval(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestTaskIsDue(t *testing.T) {
	// Zero LastResearchedAt → always due
	task := &Task{Interval: "1d"}
	if !task.IsDue() {
		t.Error("zero LastResearchedAt should be due")
	}

	// Recently researched → not due
	task.LastResearchedAt = time.Now().Add(-1 * time.Hour)
	task.Interval = "1d"
	if task.IsDue() {
		t.Error("researched 1h ago with 1d interval should not be due")
	}

	// Long ago → due
	task.LastResearchedAt = time.Now().Add(-25 * time.Hour)
	if !task.IsDue() {
		t.Error("researched 25h ago with 1d interval should be due")
	}
}

func TestListDueTasks(t *testing.T) {
	store, _ := setupTestStore(t)

	// Create tasks with different intervals
	t1, _ := store.CreateTask("Fast", "", "1h")
	t2, _ := store.CreateTask("Slow", "", "7d")

	// Both pending with no LastResearchedAt → both due
	due, err := store.ListDueTasks(10)
	if err != nil {
		t.Fatalf("list due: %v", err)
	}
	if len(due) != 2 {
		t.Errorf("expected 2 due tasks, got %d", len(due))
	}

	// Touch t1 → it should no longer be due (1h not elapsed)
	store.TouchLastResearched(t1.ID)
	due, _ = store.ListDueTasks(10)
	if len(due) != 1 {
		t.Errorf("expected 1 due task after touch, got %d", len(due))
	}
	if len(due) > 0 && due[0].ID != t2.ID {
		t.Errorf("expected Slow task to be due, got %s", due[0].Title)
	}
}

func TestSetInterval(t *testing.T) {
	store, _ := setupTestStore(t)
	task, _ := store.CreateTask("Interval Test", "", "")
	if task.Interval != DefaultResearchInterval {
		t.Errorf("default interval = %q, want %q", task.Interval, DefaultResearchInterval)
	}

	if err := store.SetInterval(task.ID, "6h"); err != nil {
		t.Fatalf("set interval: %v", err)
	}
	got, _ := store.GetTask(task.ID)
	if got.Interval != "6h" {
		t.Errorf("interval = %q, want 6h", got.Interval)
	}

	// Invalid interval
	if err := store.SetInterval(task.ID, "xyz"); err == nil {
		t.Error("expected error for invalid interval")
	}
}

func TestCanTransition(t *testing.T) {
	cases := []struct {
		from, to TaskStatus
		ok       bool
	}{
		{StatusPending, StatusActive, true},
		{StatusPending, StatusCanceled, true},
		{StatusPending, StatusCompleted, false},
		{StatusActive, StatusCompleted, true},
		{StatusActive, StatusFailed, true},
		{StatusActive, StatusCanceled, true},
		{StatusActive, StatusPending, false},
		{StatusCompleted, StatusPending, true},
		{StatusFailed, StatusPending, true},
		{StatusCanceled, StatusPending, false},
	}
	for _, tc := range cases {
		if got := CanTransition(tc.from, tc.to); got != tc.ok {
			t.Errorf("CanTransition(%s, %s) = %v, want %v", tc.from, tc.to, got, tc.ok)
		}
	}
}
