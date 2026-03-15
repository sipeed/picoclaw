package research

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

const sqliteDriver = "sqlite"

const schema = `
CREATE TABLE IF NOT EXISTS research_tasks (
	id           TEXT PRIMARY KEY,
	title        TEXT NOT NULL,
	slug         TEXT NOT NULL UNIQUE,
	description  TEXT NOT NULL DEFAULT '',
	status       TEXT NOT NULL DEFAULT 'pending',
	output_dir   TEXT NOT NULL DEFAULT '',
	created_at   TEXT NOT NULL,
	updated_at   TEXT NOT NULL,
	completed_at TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS research_documents (
	id         TEXT PRIMARY KEY,
	task_id    TEXT NOT NULL REFERENCES research_tasks(id) ON DELETE CASCADE,
	title      TEXT NOT NULL,
	file_path  TEXT NOT NULL,
	doc_type   TEXT NOT NULL DEFAULT 'finding',
	seq        INTEGER NOT NULL DEFAULT 0,
	summary    TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_research_documents_task ON research_documents(task_id, seq);
`

// ResearchStore manages research tasks and documents in SQLite.
type ResearchStore struct {
	db        *sql.DB
	workspace string
}

// OpenResearchStore opens (or creates) the research SQLite database.
func OpenResearchStore(dbPath, workspace string) (*ResearchStore, error) {
	connStr := "file:" + dbPath + "?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000"
	db, err := sql.Open(sqliteDriver, connStr)
	if err != nil {
		return nil, fmt.Errorf("open research db: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}
	return &ResearchStore{db: db, workspace: workspace}, nil
}

// Close closes the database connection.
func (s *ResearchStore) Close() error {
	return s.db.Close()
}

// CreateTask creates a new research task with auto-generated slug and output directory.
func (s *ResearchStore) CreateTask(title, description string) (*Task, error) {
	id := uuid.New().String()
	slug := slugify(title)
	now := time.Now().UTC()
	outputDir := filepath.Join("research", slug)

	// Ensure output directory exists
	absDir := filepath.Join(s.workspace, outputDir)
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	nowStr := now.Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT INTO research_tasks (id, title, slug, description, status, output_dir, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, title, slug, description, string(StatusPending), outputDir, nowStr, nowStr,
	)
	if err != nil {
		return nil, fmt.Errorf("insert task: %w", err)
	}

	return &Task{
		ID:          id,
		Title:       title,
		Slug:        slug,
		Description: description,
		Status:      StatusPending,
		OutputDir:   outputDir,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// GetTask retrieves a single task by ID.
func (s *ResearchStore) GetTask(id string) (*Task, error) {
	row := s.db.QueryRow(
		`SELECT id, title, slug, description, status, output_dir, created_at, updated_at, completed_at
		 FROM research_tasks WHERE id = ?`, id)
	return scanTask(row)
}

// ListTasks returns tasks filtered by status. Empty status returns all.
func (s *ResearchStore) ListTasks(status TaskStatus) ([]*Task, error) {
	var rows *sql.Rows
	var err error
	if status == "" {
		rows, err = s.db.Query(
			`SELECT id, title, slug, description, status, output_dir, created_at, updated_at, completed_at
			 FROM research_tasks ORDER BY created_at DESC`)
	} else {
		rows, err = s.db.Query(
			`SELECT id, title, slug, description, status, output_dir, created_at, updated_at, completed_at
			 FROM research_tasks WHERE status = ? ORDER BY created_at DESC`, string(status))
	}
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		t, err := scanTaskRow(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// SetTaskStatus updates task status with transition validation.
func (s *ResearchStore) SetTaskStatus(id string, status TaskStatus) error {
	task, err := s.GetTask(id)
	if err != nil {
		return err
	}
	if !CanTransition(task.Status, status) {
		return fmt.Errorf("invalid transition: %s → %s", task.Status, status)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	completedAt := ""
	if status == StatusCompleted || status == StatusFailed {
		completedAt = now
	}
	_, err = s.db.Exec(
		`UPDATE research_tasks SET status = ?, updated_at = ?, completed_at = ? WHERE id = ?`,
		string(status), now, completedAt, id)
	return err
}

// UpdateTask updates a task's title and/or description.
func (s *ResearchStore) UpdateTask(id, title, description string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`UPDATE research_tasks SET title = ?, description = ?, updated_at = ? WHERE id = ?`,
		title, description, now, id)
	return err
}

// DeleteTask deletes a task and its documents (cascade).
func (s *ResearchStore) DeleteTask(id string) error {
	_, err := s.db.Exec(`DELETE FROM research_tasks WHERE id = ?`, id)
	return err
}

// AddDocument adds a document record linked to a task with auto-incrementing seq.
func (s *ResearchStore) AddDocument(taskID, title, filePath, docType, summary string) (*Document, error) {
	id := uuid.New().String()
	now := time.Now().UTC()

	// Auto-increment seq
	var maxSeq int
	err := s.db.QueryRow(
		`SELECT COALESCE(MAX(seq), 0) FROM research_documents WHERE task_id = ?`, taskID).Scan(&maxSeq)
	if err != nil {
		return nil, fmt.Errorf("get max seq: %w", err)
	}
	seq := maxSeq + 1

	nowStr := now.Format(time.RFC3339)
	_, err = s.db.Exec(
		`INSERT INTO research_documents (id, task_id, title, file_path, doc_type, seq, summary, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, taskID, title, filePath, docType, seq, summary, nowStr)
	if err != nil {
		return nil, fmt.Errorf("insert document: %w", err)
	}

	return &Document{
		ID:        id,
		TaskID:    taskID,
		Title:     title,
		FilePath:  filePath,
		DocType:   docType,
		Seq:       seq,
		Summary:   summary,
		CreatedAt: now,
	}, nil
}

// ListDocuments returns all documents for a task, ordered by seq.
func (s *ResearchStore) ListDocuments(taskID string) ([]*Document, error) {
	rows, err := s.db.Query(
		`SELECT id, task_id, title, file_path, doc_type, seq, summary, created_at
		 FROM research_documents WHERE task_id = ? ORDER BY seq`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	var docs []*Document
	for rows.Next() {
		var d Document
		var createdStr string
		if err := rows.Scan(
			&d.ID, &d.TaskID, &d.Title, &d.FilePath,
			&d.DocType, &d.Seq, &d.Summary, &createdStr,
		); err != nil {
			return nil, err
		}
		d.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		docs = append(docs, &d)
	}
	return docs, rows.Err()
}

// DocumentCount returns the number of documents for a task.
func (s *ResearchStore) DocumentCount(taskID string) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM research_documents WHERE task_id = ?`, taskID).Scan(&count)
	return count, err
}

// --- helpers ---

func scanTask(row *sql.Row) (*Task, error) {
	var t Task
	var statusStr, createdStr, updatedStr, completedStr string
	err := row.Scan(&t.ID, &t.Title, &t.Slug, &t.Description, &statusStr,
		&t.OutputDir, &createdStr, &updatedStr, &completedStr)
	if err != nil {
		return nil, err
	}
	t.Status = TaskStatus(statusStr)
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	if completedStr != "" {
		t.CompletedAt, _ = time.Parse(time.RFC3339, completedStr)
	}
	return &t, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTaskRow(row rowScanner) (*Task, error) {
	var t Task
	var statusStr, createdStr, updatedStr, completedStr string
	err := row.Scan(&t.ID, &t.Title, &t.Slug, &t.Description, &statusStr,
		&t.OutputDir, &createdStr, &updatedStr, &completedStr)
	if err != nil {
		return nil, err
	}
	t.Status = TaskStatus(statusStr)
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	if completedStr != "" {
		t.CompletedAt, _ = time.Parse(time.RFC3339, completedStr)
	}
	return &t, nil
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	slug := nonAlphaNum.ReplaceAllString(b.String(), "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 60 {
		slug = slug[:60]
	}
	if slug == "" {
		slug = "task"
	}
	// Append short UUID suffix for uniqueness
	suffix := uuid.New().String()[:8]
	return slug + "-" + suffix
}
