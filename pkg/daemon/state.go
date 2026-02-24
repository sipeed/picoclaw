// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// State represents the persistent state of a daemon process.
//
// Design rationale:
// - Persistent state allows status reporting across process restarts
// - Atomic saves prevent corruption from crashes or power loss
// - Tracking restart history helps identify systemic issues
type State struct {
	// PID is the process ID of the running daemon
	PID int `json:"pid"`

	// StartTime is when the daemon was started
	StartTime time.Time `json:"start_time"`

	// RestartCount is the number of times the daemon has been restarted
	// This is cumulative and is reset when the daemon runs successfully
	// for longer than the restart window duration.
	RestartCount int `json:"restart_count"`

	// LastRestartTime is the timestamp of the most recent restart
	LastRestartTime *time.Time `json:"last_restart_time,omitempty"`

	// Version is the picoclaw version that started this daemon
	Version string `json:"version,omitempty"`
}

// StateManager manages daemon state with atomic saves.
// Follows the same atomic save pattern as pkg/state/state.go
type StateManager struct {
	stateFile string
	state     *State
	mu        sync.RWMutex
}

// NewStateManager creates a new state manager for the given state file path.
func NewStateManager(stateFile string) *StateManager {
	// Ensure directory exists
	dir := filepath.Dir(stateFile)
	os.MkdirAll(dir, 0o755)

	sm := &StateManager{
		stateFile: stateFile,
		state:     &State{},
	}

	// Load existing state if present
	sm.load()

	return sm
}

// Save atomically saves the current state to disk.
// Uses the temp file + rename pattern for atomic writes.
func (sm *StateManager) Save() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	return sm.saveAtomic()
}

// saveAtomic performs an atomic save using temp file + rename.
// Must be called with the lock held.
func (sm *StateManager) saveAtomic() error {
	// Create temp file in the same directory as the target
	tempFile := sm.stateFile + ".tmp"

	// Marshal state to JSON
	data, err := json.MarshalIndent(sm.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temp file
	if err := os.WriteFile(tempFile, data, 0o644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename from temp to target
	if err := os.Rename(tempFile, sm.stateFile); err != nil {
		// Cleanup temp file if rename fails
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// load loads the state from disk.
// Must be called with the lock held.
func (sm *StateManager) load() error {
	data, err := os.ReadFile(sm.stateFile)
	if err != nil {
		// File doesn't exist yet, that's OK
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read state file: %w", err)
	}

	if err := json.Unmarshal(data, sm.state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return nil
}

// SetPID updates the PID in the state and saves atomically.
func (sm *StateManager) SetPID(pid int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.state.PID = pid
	return sm.saveAtomic()
}

// GetPID returns the PID from the state.
func (sm *StateManager) GetPID() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.PID
}

// SetStartTime sets the start time in the state and saves atomically.
func (sm *StateManager) SetStartTime(startTime time.Time) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.state.StartTime = startTime
	return sm.saveAtomic()
}

// GetStartTime returns the start time from the state.
func (sm *StateManager) GetStartTime() time.Time {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.StartTime
}

// IncrementRestartCount increments the restart counter and saves atomically.
func (sm *StateManager) IncrementRestartCount() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.state.RestartCount++
	now := time.Now()
	sm.state.LastRestartTime = &now

	return sm.saveAtomic()
}

// GetRestartCount returns the restart count from the state.
func (sm *StateManager) GetRestartCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.RestartCount
}

// ResetRestartCount resets the restart counter to zero and saves atomically.
// Call this when the daemon has been running successfully for a while
// to indicate that crashes are no longer a concern.
func (sm *StateManager) ResetRestartCount() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.state.RestartCount = 0
	sm.state.LastRestartTime = nil

	return sm.saveAtomic()
}

// SetVersion sets the picoclaw version in the state and saves atomically.
func (sm *StateManager) SetVersion(version string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.state.Version = version
	return sm.saveAtomic()
}

// GetVersion returns the version from the state.
func (sm *StateManager) GetVersion() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.Version
}

// GetUptime returns the duration since the daemon started.
// Returns zero if start time is not set.
func (sm *StateManager) GetUptime() time.Duration {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.state.StartTime.IsZero() {
		return 0
	}

	return time.Since(sm.state.StartTime)
}

// GetState returns a copy of the current state.
func (sm *StateManager) GetState() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return *sm.state
}

// Clear removes the state file entirely.
// Use this when stopping the daemon gracefully.
func (sm *StateManager) Clear() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.state = &State{}

	if err := os.Remove(sm.stateFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove state file: %w", err)
	}

	return nil
}
