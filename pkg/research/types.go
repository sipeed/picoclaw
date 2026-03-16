package research

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TaskStatus represents the lifecycle state of a research task.
type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusActive    TaskStatus = "active"
	StatusCompleted TaskStatus = "completed"
	StatusFailed    TaskStatus = "failed"
	StatusCanceled  TaskStatus = "canceled"
)

// MinFindingsForCompletion is the minimum number of findings required
// before a heartbeat execution can mark a research task as completed.
const MinFindingsForCompletion = 5

// DefaultResearchInterval is the default research interval for new tasks.
const DefaultResearchInterval = "1d"

// DefaultHeartbeatSearchQuota is the max web searches per heartbeat execution.
const DefaultHeartbeatSearchQuota = 3

// Task represents a research task tracked in the database.
type Task struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	Slug             string     `json:"slug"`
	Description      string     `json:"description"`
	Status           TaskStatus `json:"status"`
	OutputDir        string     `json:"output_dir"`
	Interval         string     `json:"interval"`
	LastResearchedAt time.Time  `json:"last_researched_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	CompletedAt      time.Time  `json:"completed_at,omitempty"`
}

// IsDue returns true if enough time has elapsed since the last research.
func (t *Task) IsDue() bool {
	if t.LastResearchedAt.IsZero() {
		return true
	}
	interval, err := ParseInterval(t.Interval)
	if err != nil || interval <= 0 {
		interval = 24 * time.Hour
	}
	return time.Since(t.LastResearchedAt) >= interval
}

// ParseInterval parses a duration string with support for "d" (days) suffix.
// Examples: "30m", "6h", "1d", "7d", "24h".
func ParseInterval(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty interval")
	}
	// Handle "d" suffix: e.g. "1d" -> "24h", "7d" -> "168h"
	if strings.HasSuffix(s, "d") {
		numStr := strings.TrimSuffix(s, "d")
		days, err := strconv.Atoi(numStr)
		if err != nil {
			return 0, fmt.Errorf("invalid day interval %q: %w", s, err)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
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
