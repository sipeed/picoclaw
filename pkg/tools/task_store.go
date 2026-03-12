// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Task represents a task/schedule item
type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	DueDate     string    `json:"due_date,omitempty"`     // YYYY-MM-DD
	DueTime     string    `json:"due_time,omitempty"`     // HH:MM
	DueWeekday  int       `json:"due_weekday,omitempty"`  // 0=Sunday, 1=Monday, ..., 6=Saturday
	Repeat      string    `json:"repeat,omitempty"`        // "none", "daily", "weekly", "monthly", "weekdays"
	Completed   bool      `json:"completed"`
	CompletedAt string   `json:"completed_at,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	RemindBefore int     `json:"remind_before,omitempty"` // minutes before
}

// TaskStore manages tasks
type TaskStore struct {
	workspace string
	tasksFile string
}

// NewTaskStore creates a new TaskStore
func NewTaskStore(workspace string) *TaskStore {
	tasksDir := filepath.Join(workspace, "tasks")
	os.MkdirAll(tasksDir, 0o755)
	
	return &TaskStore{
		workspace: workspace,
		tasksFile: filepath.Join(tasksDir, "tasks.json"),
	}
}

// Load loads tasks from file
func (ts *TaskStore) Load() ([]Task, error) {
	data, err := os.ReadFile(ts.tasksFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Task{}, nil
		}
		return nil, err
	}
	
	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, err
	}
	
	return tasks, nil
}

// Save saves tasks to file
func (ts *TaskStore) Save(tasks []Task) error {
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(ts.tasksFile, data, 0o600)
}

// AddTask adds a new task
func (ts *TaskStore) AddTask(task Task) ([]Task, error) {
	tasks, err := ts.Load()
	if err != nil {
		return nil, err
	}
	
	task.ID = generateTaskID()
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	
	tasks = append(tasks, task)
	
	if err := ts.Save(tasks); err != nil {
		return nil, err
	}
	
	return tasks, nil
}

// UpdateTask updates an existing task
func (ts *TaskStore) UpdateTask(taskID string, updated Task) ([]Task, error) {
	tasks, err := ts.Load()
	if err != nil {
		return nil, err
	}
	
	found := false
	for i, t := range tasks {
		if t.ID == taskID {
			updated.ID = taskID
			updated.CreatedAt = t.CreatedAt // preserve original
			updated.UpdatedAt = time.Now()
			tasks[i] = updated
			found = true
			break
		}
	}
	
	if !found {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	
	if err := ts.Save(tasks); err != nil {
		return nil, err
	}
	
	return tasks, nil
}

// DeleteTask deletes a task
func (ts *TaskStore) DeleteTask(taskID string) ([]Task, error) {
	tasks, err := ts.Load()
	if err != nil {
		return nil, err
	}
	
	originalLen := len(tasks)
	var newTasks []Task
	for _, t := range tasks {
		if t.ID != taskID {
			newTasks = append(newTasks, t)
		}
	}
	
	if len(newTasks) == originalLen {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	
	if err := ts.Save(newTasks); err != nil {
		return nil, err
	}
	
	return newTasks, nil
}

// GetTaskByID gets a task by ID
func (ts *TaskStore) GetTaskByID(taskID string) (*Task, error) {
	tasks, err := ts.Load()
	if err != nil {
		return nil, err
	}
	
	for _, t := range tasks {
		if t.ID == taskID {
			return &t, nil
		}
	}
	
	return nil, nil
}

// GetTasksForDate gets tasks for a specific date
func (ts *TaskStore) GetTasksForDate(date time.Time) ([]Task, error) {
	tasks, err := ts.Load()
	if err != nil {
		return nil, err
	}
	
	var result []Task
	weekday := int(date.Weekday()) // 0=Sunday, 1=Monday, ...
	dateStr := date.Format("2006-01-02")
	
	for _, t := range tasks {
		// Skip completed tasks
		if t.Completed {
			continue
		}
		
		// Check if task matches the date
		match := false
		
		// Exact date match
		if t.DueDate != "" && t.DueDate == dateStr {
			match = true
		}
		
		// Weekday match (for recurring tasks)
		if t.DueWeekday > 0 && t.DueWeekday-1 == weekday {
			match = true
		}
		
		// Weekdays repeat (Monday to Friday)
		if t.Repeat == "weekdays" && weekday >= 1 && weekday <= 5 {
			match = true
		}
		
		// Daily repeat
		if t.Repeat == "daily" {
			match = true
		}
		
		// Weekly repeat (specific weekday)
		if t.Repeat == "weekly" && t.DueWeekday > 0 && t.DueWeekday-1 == weekday {
			match = true
		}
		
		// Monthly repeat (same day of month)
		if t.Repeat == "monthly" && date.Day() == 1 {
			// This is simplified, could be more sophisticated
			match = true
		}
		
		if match {
			result = append(result, t)
		}
	}
	
	return result, nil
}

// GetAllTasks gets all tasks
func (ts *TaskStore) GetAllTasks() ([]Task, error) {
	return ts.Load()
}

// CompleteTask marks a task as completed
func (ts *TaskStore) CompleteTask(taskID string) ([]Task, error) {
	tasks, err := ts.Load()
	if err != nil {
		return nil, err
	}
	
	found := false
	for i, t := range tasks {
		if t.ID == taskID {
			tasks[i].Completed = true
			tasks[i].CompletedAt = time.Now().Format("2006-01-02 15:04")
			tasks[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}
	
	if !found {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	
	if err := ts.Save(tasks); err != nil {
		return nil, err
	}
	
	return tasks, nil
}

// generateTaskID generates a unique task ID
func generateTaskID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}

// weekdayName returns the name of a weekday (1=周一, 2=周二, ..., 7=周日)
func weekdayName(day int) string {
	names := []string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"}
	if day < 1 || day > 7 {
		return ""
	}
	return names[day-1]
}
