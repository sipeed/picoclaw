package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/fileutil"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
)

type Task struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	Result      string     `json:"result,omitempty"`
}

type SessionTasks struct {
	SessionKey string    `json:"session_key"`
	MessageID  string    `json:"message_id,omitempty"` // ID of the message to edit with progress
	Tasks      []Task    `json:"tasks"`
	Updated    time.Time `json:"updated"`
}

type TaskManager struct {
	storage string
	tasks   map[string]*SessionTasks
	mu      sync.RWMutex
}

func NewTaskManager(storage string) *TaskManager {
	tm := &TaskManager{
		storage: storage,
		tasks:   make(map[string]*SessionTasks),
	}
	if storage != "" {
		if err := tm.loadAll(); err != nil {
			logger.ErrorCF("tasks", "Failed to load session tasks on startup", map[string]any{
				"error": err.Error(),
			})
		}
	}
	return tm
}

func (tm *TaskManager) Get(sessionKey string) *SessionTasks {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasks, ok := tm.tasks[sessionKey]
	if ok {
		return tasks
	}
	return nil
}

func (tm *TaskManager) GetOrCreate(sessionKey string) *SessionTasks {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tasks, ok := tm.tasks[sessionKey]
	if ok {
		return tasks
	}

	tasks = &SessionTasks{
		SessionKey: sessionKey,
		Tasks:      []Task{},
		Updated:    time.Now(),
	}
	tm.tasks[sessionKey] = tasks
	return tasks
}

func (tm *TaskManager) CreatePlan(sessionKey string, tasks []Task) *SessionTasks {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	st := &SessionTasks{
		SessionKey: sessionKey,
		Tasks:      make([]Task, len(tasks)),
		Updated:    time.Now(),
	}
	copy(st.Tasks, tasks)
	tm.tasks[sessionKey] = st
	return st
}

func (tm *TaskManager) UpdateTask(sessionKey, taskID string, status TaskStatus, result string) (*SessionTasks, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	st, ok := tm.tasks[sessionKey]
	if !ok || len(st.Tasks) == 0 {
		return nil, fmt.Errorf("no active plan for session %s", sessionKey)
	}

	found := false
	for i := range st.Tasks {
		if st.Tasks[i].ID == taskID {
			st.Tasks[i].Status = status
			if result != "" {
				st.Tasks[i].Result = result
			}
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("task %s not found in plan", taskID)
	}

	st.Updated = time.Now()
	// Attempt to save immediately but do not block return on error.
	go func() {
		// Just saving this session, we need a separate lock and method for fine-grained
		_ = tm.Save(sessionKey)
	}()

	return st, nil
}

func (tm *TaskManager) SetMessageID(sessionKey, messageID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if st, ok := tm.tasks[sessionKey]; ok {
		st.MessageID = messageID
		st.Updated = time.Now()
		go func() { _ = tm.Save(sessionKey) }()
	}
}

func (tm *TaskManager) Save(key string) error {
	if tm.storage == "" {
		return nil
	}

	if err := os.MkdirAll(tm.storage, 0o755); err != nil {
		return err
	}
	filename := fileutil.SanitizeFilename(key) + "_tasks.json"

	if filename == "." || !filepath.IsLocal(filename) || strings.ContainsAny(filename, "/\\") {
		return os.ErrInvalid
	}

	tm.mu.RLock()
	stored, ok := tm.tasks[key]
	if !ok {
		tm.mu.RUnlock()
		return nil
	}

	// Make a safe copy to marshal
	snapshot := SessionTasks{
		SessionKey: stored.SessionKey,
		MessageID:  stored.MessageID,
		Updated:    stored.Updated,
		Tasks:      make([]Task, len(stored.Tasks)),
	}
	copy(snapshot.Tasks, stored.Tasks)
	tm.mu.RUnlock()

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	sessionPath := filepath.Join(tm.storage, filename)
	return fileutil.WriteFileAtomic(sessionPath, data, 0o644)
}

func (tm *TaskManager) loadAll() error {
	files, err := os.ReadDir(tm.storage)
	if err != nil {
		return err
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), "_tasks.json") {
			continue
		}

		path := filepath.Join(tm.storage, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var st SessionTasks
		if err := json.Unmarshal(data, &st); err != nil {
			continue
		}

		tm.tasks[st.SessionKey] = &st
	}
	return nil
}
