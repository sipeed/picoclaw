package research

import "sync"

// FocusTracker tracks which single research task is currently "recalled"
// into the agent's active context. At most one task can be focused at a time
// to avoid context pollution. Thread-safe.
type FocusTracker struct {
	mu     sync.RWMutex
	taskID string // currently focused task ID (empty = none)
	title  string // currently focused task title
}

// NewFocusTracker creates a new FocusTracker.
func NewFocusTracker() *FocusTracker {
	return &FocusTracker{}
}

// Focus sets the single focused task, replacing any previous focus.
func (ft *FocusTracker) Focus(taskID, title string) {
	ft.mu.Lock()
	ft.taskID = taskID
	ft.title = title
	ft.mu.Unlock()
}

// Unfocus removes a specific task from focus.
// Returns true if the task was focused.
func (ft *FocusTracker) Unfocus(taskID string) bool {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	if ft.taskID != taskID {
		return false
	}
	ft.taskID = ""
	ft.title = ""
	return true
}

// UnfocusAll clears focus. Returns 1 if something was focused, 0 otherwise.
func (ft *FocusTracker) UnfocusAll() int {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	if ft.taskID == "" {
		return 0
	}
	ft.taskID = ""
	ft.title = ""
	return 1
}

// Current returns the currently focused task ID and title.
// Returns empty strings if nothing is focused.
func (ft *FocusTracker) Current() (taskID, title string) {
	ft.mu.RLock()
	defer ft.mu.RUnlock()
	return ft.taskID, ft.title
}

// IsFocused returns whether a specific task is currently focused.
func (ft *FocusTracker) IsFocused(taskID string) bool {
	ft.mu.RLock()
	defer ft.mu.RUnlock()
	return ft.taskID == taskID
}
