// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// PIDFile manages process ID file with atomic operations.
// It provides thread-safe operations for creating, reading, and removing PID files.
//
// Design rationale:
// - PID files are the canonical Unix way to track daemon processes
// - Atomic operations prevent race conditions during concurrent access
// - Proper cleanup prevents stale PID files from accumulating
type PIDFile struct {
	path string
	mu   sync.Mutex
}

// NewPIDFile creates a new PID file manager for the given path.
func NewPIDFile(path string) *PIDFile {
	// Ensure directory exists
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0o755)

	return &PIDFile{
		path: path,
	}
}

// Write atomically writes the current process ID to the PID file.
// Returns an error if a PID file already exists with a running process.
//
// This prevents multiple instances of the gateway from running simultaneously,
// which could cause resource conflicts and undefined behavior.
func (p *PIDFile) Write() error {
	return p.WritePID(os.Getpid())
}

// WritePID atomically writes the specified process ID to the PID file.
// Returns an error if a PID file already exists with a running process.
//
// Use this method when you need to write a PID other than the current process,
// such as when spawning a child process.
func (p *PIDFile) WritePID(pid int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check for existing PID file
	if _, err := os.Stat(p.path); err == nil {
		// PID file exists, check if process is running
		existingPid, err := p.read()
		if err == nil && p.isProcessRunning(existingPid) {
			return &ProcessRunningError{
				pid:  existingPid,
				Path: p.path,
			}
		}
		// Process is not running, stale PID file, continue
	}
	pidStr := strconv.Itoa(pid)

	tempFile := p.path + ".tmp"
	if err := os.WriteFile(tempFile, []byte(pidStr), 0o644); err != nil {
		return fmt.Errorf("failed to write temp PID file: %w", err)
	}

	// Atomic rename from temp to target
	if err := os.Rename(tempFile, p.path); err != nil {
		os.Remove(tempFile) // Cleanup temp file
		return fmt.Errorf("failed to atomically create PID file: %w", err)
	}

	return nil
}

// Remove deletes the PID file.
// Safe to call multiple times (idempotent operation).
func (p *PIDFile) Remove() {
	p.mu.Lock()
	defer p.mu.Unlock()

	_ = os.Remove(p.path)
}

// Read returns the process ID stored in the PID file.
// Returns 0 if the file doesn't exist or cannot be read.
func (p *PIDFile) Read() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	pid, err := p.read()
	if err != nil {
		return 0
	}
	return pid
}

// read must be called with the lock held.
func (p *PIDFile) read() (int, error) {
	data, err := os.ReadFile(p.path)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}

	return pid, nil
}

// IsProcessRunning checks if the process with the given PID is still running.
func (p *PIDFile) IsProcessRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	pid, err := p.read()
	if err != nil {
		return false
	}

	return p.isProcessRunning(pid)
}

// isProcessRunning must be called with the lock held.
func (p *PIDFile) isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	// Send signal 0 to check if process exists
	// On Unix, this doesn't actually send a signal but checks if the process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Try to send signal 0 (no signal, just check existence)
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// Process doesn't exist or we don't have permission to signal it
		return false
	}

	return true
}

// GetUptime returns the uptime of the process if it's running.
// Returns zero duration if the process is not running or uptime cannot be determined.
func (p *PIDFile) GetUptime() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()

	pid, err := p.read()
	if err != nil {
		return 0
	}

	return p.getProcessUptime(pid)
}

// getProcessUptime must be called with the lock held.
func (p *PIDFile) getProcessUptime(pid int) time.Duration {
	if pid <= 0 {
		return 0
	}

	// Read /proc/<pid>/stat to get process start time
	// Format: pid (comm) state ppid pgrp session tty_nr tpgid flags minflt cminflt ...
	// The 22nd field (index 21) is the start time in jiffies
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(statPath)
	if err != nil {
		// Process doesn't exist or /proc not available
		return 0
	}

	// Parse the stat file
	// Find the last ')' to handle command names with spaces
	fields := string(data)
	lastParen := -1
	for i, c := range fields {
		if c == ')' {
			lastParen = i
		}
	}

	if lastParen == -1 {
		return 0
	}

	// Extract fields after the command name
	rest := fields[lastParen+2:] // Skip ") "
	var startTimeJiffies uint64
	_, err = fmt.Sscanf(rest, "%*c %*d %*d %*d %*d %*d %*d %*d %*d %*d %*d %*d %*d %*d %*d %*d %*d %*d %*d %d",
		&startTimeJiffies)
	if err != nil {
		return 0
	}

	// Convert jiffies to nanoseconds
	// Assumes USER_HZ=100 (common on Linux)
	// This is a simplification; for production code, use sysconf(_SC_CLK_TCK)
	clockTick := int64(100)
	startTime := time.Unix(int64(startTimeJiffies)/clockTick, 0)

	return time.Since(startTime)
}

// ProcessRunningError is returned when attempting to write a PID file
// but a process is already running.
type ProcessRunningError struct {
	pid  int
	Path string
}

func (e *ProcessRunningError) Error() string {
	return fmt.Sprintf("process already running with PID %d (PID file: %s)", e.pid, e.Path)
}

// GetPID returns the PID of the running process.
func (e *ProcessRunningError) GetPID() int {
	return e.pid
}
