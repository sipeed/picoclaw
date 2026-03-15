package research

import (
	"os"
	"path/filepath"
	"testing"
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

	task, err := store.CreateTask("Test Research", "A test description")
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

	store.CreateTask("Task A", "")
	store.CreateTask("Task B", "")

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

	task, _ := store.CreateTask("Status Test", "")

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

	task, _ := store.CreateTask("Doc Test", "")

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

	task, _ := store.CreateTask("Count Test", "")

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

	task, _ := store.CreateTask("Delete Test", "")
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

	task, _ := store.CreateTask("Original", "desc")
	if err := store.UpdateTask(task.ID, "Updated", "new desc"); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, _ := store.GetTask(task.ID)
	if got.Title != "Updated" || got.Description != "new desc" {
		t.Errorf("after update: title=%q desc=%q", got.Title, got.Description)
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
