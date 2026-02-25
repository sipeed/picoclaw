package tools

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type TaskStatus string

const (
	TaskStatusPlanned   TaskStatus = "planned"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
	TaskStatusDegraded  TaskStatus = "degraded"
)

type TaskEvidence struct {
	TimestampMS   int64          `json:"timestamp_ms"`
	Iteration     int            `json:"iteration,omitempty"`
	ToolName      string         `json:"tool_name,omitempty"`
	Arguments     map[string]any `json:"arguments,omitempty"`
	ResultPreview string         `json:"result_preview,omitempty"`
	IsError       bool           `json:"is_error,omitempty"`
	DurationMS    int64          `json:"duration_ms,omitempty"`
}

type TaskRemediation struct {
	Action      string `json:"action"`
	Status      string `json:"status"`
	Note        string `json:"note,omitempty"`
	CreatedAtMS int64  `json:"created_at_ms"`
}

type TaskLedgerEntry struct {
	ID            string            `json:"id"`
	ParentTaskID  string            `json:"parent_task_id,omitempty"`
	AgentID       string            `json:"agent_id,omitempty"`
	Source        string            `json:"source,omitempty"`
	Intent        string            `json:"intent,omitempty"`
	OriginChannel string            `json:"origin_channel,omitempty"`
	OriginChatID  string            `json:"origin_chat_id,omitempty"`
	Status        TaskStatus        `json:"status"`
	CreatedAtMS   int64             `json:"created_at_ms"`
	UpdatedAtMS   int64             `json:"updated_at_ms"`
	DeadlineAtMS  *int64            `json:"deadline_at_ms,omitempty"`
	RetryCount    int               `json:"retry_count,omitempty"`
	Result        string            `json:"result,omitempty"`
	Error         string            `json:"error,omitempty"`
	Evidence      []TaskEvidence    `json:"evidence,omitempty"`
	Remediations  []TaskRemediation `json:"remediations,omitempty"`
}

type taskLedgerStore struct {
	Version int               `json:"version"`
	Tasks   []TaskLedgerEntry `json:"tasks"`
}

type TaskLedger struct {
	path  string
	tasks map[string]*TaskLedgerEntry
	mu    sync.RWMutex
}

func NewTaskLedger(path string) *TaskLedger {
	l := &TaskLedger{
		path:  path,
		tasks: make(map[string]*TaskLedgerEntry),
	}
	_ = l.load()
	return l
}

func (l *TaskLedger) CreateTask(entry TaskLedgerEntry) error {
	if strings.TrimSpace(entry.ID) == "" {
		return errors.New("task id is required")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UnixMilli()
	entry.ID = strings.TrimSpace(entry.ID)
	if _, exists := l.tasks[entry.ID]; exists {
		return errors.New("task id already exists")
	}
	if entry.Status == "" {
		entry.Status = TaskStatusPlanned
	}
	if entry.CreatedAtMS == 0 {
		entry.CreatedAtMS = now
	}
	entry.UpdatedAtMS = now

	snapshot := entry
	l.tasks[entry.ID] = &snapshot
	return l.saveLocked()
}

func (l *TaskLedger) UpsertTask(entry TaskLedgerEntry) error {
	if strings.TrimSpace(entry.ID) == "" {
		return errors.New("task id is required")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UnixMilli()
	entry.ID = strings.TrimSpace(entry.ID)
	if existing, ok := l.tasks[entry.ID]; ok {
		if entry.ParentTaskID != "" {
			existing.ParentTaskID = entry.ParentTaskID
		}
		if entry.AgentID != "" {
			existing.AgentID = entry.AgentID
		}
		if entry.Source != "" {
			existing.Source = entry.Source
		}
		if entry.Intent != "" {
			existing.Intent = entry.Intent
		}
		if entry.OriginChannel != "" {
			existing.OriginChannel = entry.OriginChannel
		}
		if entry.OriginChatID != "" {
			existing.OriginChatID = entry.OriginChatID
		}
		if entry.Status != "" {
			existing.Status = entry.Status
		}
		if entry.DeadlineAtMS != nil {
			existing.DeadlineAtMS = entry.DeadlineAtMS
		}
		if entry.Result != "" {
			existing.Result = entry.Result
		}
		if entry.Error != "" {
			existing.Error = entry.Error
		}
		existing.UpdatedAtMS = now
		return l.saveLocked()
	}

	if entry.Status == "" {
		entry.Status = TaskStatusPlanned
	}
	if entry.CreatedAtMS == 0 {
		entry.CreatedAtMS = now
	}
	entry.UpdatedAtMS = now

	snapshot := entry
	l.tasks[entry.ID] = &snapshot
	return l.saveLocked()
}

func (l *TaskLedger) SetStatus(taskID string, status TaskStatus, result, taskErr string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.tasks[strings.TrimSpace(taskID)]
	if !ok {
		return errors.New("task not found")
	}
	entry.Status = status
	entry.UpdatedAtMS = time.Now().UnixMilli()
	entry.Result = result
	entry.Error = taskErr
	return l.saveLocked()
}

func (l *TaskLedger) IncrementRetry(taskID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.tasks[strings.TrimSpace(taskID)]
	if !ok {
		return errors.New("task not found")
	}
	entry.RetryCount++
	entry.UpdatedAtMS = time.Now().UnixMilli()
	return l.saveLocked()
}

func (l *TaskLedger) AddEvidence(taskID string, evidence TaskEvidence) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.tasks[strings.TrimSpace(taskID)]
	if !ok {
		return errors.New("task not found")
	}
	if evidence.TimestampMS == 0 {
		evidence.TimestampMS = time.Now().UnixMilli()
	}
	if evidence.Arguments != nil {
		clonedArgs := make(map[string]any, len(evidence.Arguments))
		for k, v := range evidence.Arguments {
			clonedArgs[k] = v
		}
		evidence.Arguments = clonedArgs
	}
	entry.Evidence = append(entry.Evidence, evidence)
	entry.UpdatedAtMS = evidence.TimestampMS
	return l.saveLocked()
}

func (l *TaskLedger) AddRemediation(taskID string, remediation TaskRemediation) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.tasks[strings.TrimSpace(taskID)]
	if !ok {
		return errors.New("task not found")
	}
	if remediation.CreatedAtMS == 0 {
		remediation.CreatedAtMS = time.Now().UnixMilli()
	}
	entry.Remediations = append(entry.Remediations, remediation)
	entry.UpdatedAtMS = remediation.CreatedAtMS
	return l.saveLocked()
}

func (l *TaskLedger) Get(taskID string) (TaskLedgerEntry, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	entry, ok := l.tasks[strings.TrimSpace(taskID)]
	if !ok {
		return TaskLedgerEntry{}, false
	}
	return cloneTaskEntry(*entry), true
}

func (l *TaskLedger) List() []TaskLedgerEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.listLocked(0)
}

func (l *TaskLedger) ListSince(since time.Time) []TaskLedgerEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.listLocked(since.UnixMilli())
}

func (l *TaskLedger) listLocked(minCreatedAtMS int64) []TaskLedgerEntry {
	entries := make([]TaskLedgerEntry, 0, len(l.tasks))
	for _, entry := range l.tasks {
		if minCreatedAtMS > 0 && entry.CreatedAtMS < minCreatedAtMS {
			continue
		}
		entries = append(entries, cloneTaskEntry(*entry))
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].CreatedAtMS == entries[j].CreatedAtMS {
			return entries[i].ID < entries[j].ID
		}
		return entries[i].CreatedAtMS < entries[j].CreatedAtMS
	})
	return entries
}

func (l *TaskLedger) load() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var store taskLedgerStore
	if err := json.Unmarshal(data, &store); err != nil {
		return err
	}

	l.tasks = make(map[string]*TaskLedgerEntry, len(store.Tasks))
	for i := range store.Tasks {
		entry := store.Tasks[i]
		snapshot := cloneTaskEntry(entry)
		l.tasks[entry.ID] = &snapshot
	}
	return nil
}

func (l *TaskLedger) saveLocked() error {
	store := taskLedgerStore{
		Version: 1,
		Tasks:   l.listLocked(0),
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}

	tmp := l.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, l.path)
}

func cloneTaskEntry(entry TaskLedgerEntry) TaskLedgerEntry {
	snapshot := entry
	if entry.Evidence != nil {
		snapshot.Evidence = make([]TaskEvidence, len(entry.Evidence))
		for i := range entry.Evidence {
			snapshot.Evidence[i] = entry.Evidence[i]
			if entry.Evidence[i].Arguments != nil {
				clonedArgs := make(map[string]any, len(entry.Evidence[i].Arguments))
				for k, v := range entry.Evidence[i].Arguments {
					clonedArgs[k] = v
				}
				snapshot.Evidence[i].Arguments = clonedArgs
			}
		}
	}
	if entry.Remediations != nil {
		snapshot.Remediations = make([]TaskRemediation, len(entry.Remediations))
		copy(snapshot.Remediations, entry.Remediations)
	}
	return snapshot
}
