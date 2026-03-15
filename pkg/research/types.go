package research

import "time"

// TaskStatus represents the lifecycle state of a research task.
type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusActive    TaskStatus = "active"
	StatusCompleted TaskStatus = "completed"
	StatusFailed    TaskStatus = "failed"
	StatusCanceled  TaskStatus = "canceled"
)

// Task represents a research task tracked in the database.
type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	OutputDir   string     `json:"output_dir"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt time.Time  `json:"completed_at,omitempty"`
}

// Document represents a research output document linked to a task.
type Document struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"task_id"`
	Title     string    `json:"title"`
	FilePath  string    `json:"file_path"`
	DocType   string    `json:"doc_type"` // "finding" | "summary" | "note"
	Seq       int       `json:"seq"`
	Summary   string    `json:"summary"`
	CreatedAt time.Time `json:"created_at"`
}

// validTransitions defines which status transitions are allowed.
var validTransitions = map[TaskStatus]map[TaskStatus]bool{
	StatusPending:   {StatusActive: true, StatusCanceled: true},
	StatusActive:    {StatusCompleted: true, StatusFailed: true, StatusCanceled: true},
	StatusCompleted: {StatusPending: true},
	StatusFailed:    {StatusPending: true},
}

// CanTransition checks whether transitioning from one status to another is allowed.
func CanTransition(from, to TaskStatus) bool {
	if m, ok := validTransitions[from]; ok {
		return m[to]
	}
	return false
}
