// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogConfig defines the configuration for daemon logging.
type LogConfig struct {
	// Path is the log file path
	Path string

	// MaxSize is the maximum size in bytes before rotation
	MaxSize int64

	// MaxBackups is the maximum number of backup files to keep
	MaxBackups int

	// MaxAge is the maximum age to keep a log file before deletion
	MaxAge time.Duration
}

// DefaultLogConfig returns a log configuration with sensible defaults.
func DefaultLogConfig(path string) *LogConfig {
	return &LogConfig{
		Path:       path,
		MaxSize:    100 * 1024 * 1024, // 100 MB
		MaxBackups: 3,                  // Keep 3 backups
		MaxAge:     30 * 24 * time.Hour, // 30 days
	}
}

// Logger manages daemon logging with automatic file rotation.
//
// Design rationale:
// - Long-running daemons need log rotation to prevent disk space exhaustion
// - Atomic rotation prevents log loss during rotation
// - Size-based rotation is more predictable than time-based rotation
// - Automatic cleanup of old logs prevents accumulation
type Logger struct {
	config *LogConfig
	file   *os.File
	size   int64
	mu     sync.Mutex
}

// NewLogger creates a new daemon logger with the given configuration.
func NewLogger(config *LogConfig) (*Logger, error) {
	if config == nil {
		return nil, fmt.Errorf("log config cannot be nil")
	}

	l := &Logger{
		config: config,
	}

	// Ensure log directory exists
	dir := filepath.Dir(config.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	if err := l.openLogFile(); err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Get current file size
	info, err := l.file.Stat()
	if err != nil {
		l.file.Close()
		return nil, fmt.Errorf("failed to stat log file: %w", err)
	}
	l.size = info.Size()

	return l, nil
}

// openLogFile opens the log file for appending.
// Must be called with the lock held.
func (l *Logger) openLogFile() error {
	var err error
	l.file, err = os.OpenFile(l.config.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	return err
}

// Write writes a message to the log file.
// If the file size exceeds MaxSize, it rotates the log file first.
func (l *Logger) Write(message string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if rotation is needed
	if l.size+l.fileSizeFor(message) > l.config.MaxSize {
		if err := l.rotate(); err != nil {
			// Log rotation failed, try writing to current file anyway
			// but include error message
			message = fmt.Sprintf("[ERROR] Failed to rotate log: %v\n%s", err, message)
		}
	}

	// Write message
	n, err := l.file.WriteString(message)
	if err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	l.size += int64(n)

	// Sync to ensure data is written to disk
	if err := l.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync log file: %w", err)
	}

	return nil
}

// fileSizeFor estimates the size of the message in bytes.
func (l *Logger) fileSizeFor(message string) int64 {
	return int64(len(message))
}

// rotate performs log file rotation.
// Must be called with the lock held.
func (l *Logger) rotate() error {
	// Close current log file
	if l.file != nil {
		if err := l.file.Close(); err != nil {
			return fmt.Errorf("failed to close current log file: %w", err)
		}
	}

	// Find the next available backup number
	backupNum := 1
	for {
		backupPath := l.backupPath(backupNum)
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			break
		}
		backupNum++
	}

	// Rotate existing backups
	for i := backupNum - 1; i >= 1; i-- {
		oldPath := l.backupPath(i)
		newPath := l.backupPath(i + 1)

		if i >= l.config.MaxBackups {
			// Delete old backup if we have too many
			os.Remove(oldPath)
		} else {
			// Rename backup
			os.Rename(oldPath, newPath)
		}
	}

	// Move current log to backup
	if err := os.Rename(l.config.Path, l.backupPath(1)); err != nil {
		// Reopen current file if rename failed
		l.openLogFile()
		return fmt.Errorf("failed to rotate log file: %w", err)
	}

	// Clean up old log files based on age
	l.cleanupOldLogs()

	// Open new log file
	if err := l.openLogFile(); err != nil {
		return fmt.Errorf("failed to open new log file after rotation: %w", err)
	}

	// Reset size
	l.size = 0

	return nil
}

// backupPath returns the path for a backup file with the given number.
func (l *Logger) backupPath(num int) string {
	return fmt.Sprintf("%s.%d", l.config.Path, num)
}

// cleanupOldLogs removes log files older than MaxAge.
// Must be called with the lock held.
func (l *Logger) cleanupOldLogs() {
	if l.config.MaxAge <= 0 {
		return
	}

	cutoff := time.Now().Add(-l.config.MaxAge)

	// Check all backup files
	for i := 1; i <= l.config.MaxBackups; i++ {
		backupPath := l.backupPath(i)

		info, err := os.Stat(backupPath)
		if err != nil {
			continue
		}

		// Delete if older than cutoff
		if info.ModTime().Before(cutoff) {
			os.Remove(backupPath)
		}
	}
}

// Close closes the log file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}

	return nil
}

// Rotate forces an immediate log rotation.
// Useful for testing or manual log management.
func (l *Logger) Rotate() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.rotate()
}

// GetSize returns the current size of the log file in bytes.
func (l *Logger) GetSize() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.size
}
