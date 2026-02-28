package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	// Default configuration values
	DefaultMaxRestarts    = 3
	DefaultRestartWindow  = 5 * time.Minute
	DefaultMaxLogSize     = 100 * 1024 * 1024 // 100MB
	DefaultMaxLogBackups  = 3
	DefaultShutdownTimeout = 30 * time.Second

	// File names
	PIDFileName      = "gateway.pid"
	StateFileName    = "gateway_state.json"
	LogFileName      = "gateway.log"
)

// State represents the persistent daemon state
type State struct {
	PID           int       `json:"pid"`
	StartTime     time.Time `json:"start_time"`
	RestartCount  int       `json:"restart_count"`
	LastRestart   time.Time `json:"last_restart,omitempty"`
	CrashCount    int       `json:"crash_count"`
	FirstCrashTime time.Time `json:"first_crash_time,omitempty"`
}

// Daemon manages the gateway daemon lifecycle
type Daemon struct {
	pidFile      string
	stateFile    string
	logFile      string
	executable   string
	args         []string
	maxRestarts  int
	restartWindow time.Duration
	maxLogSize   int64
	maxLogBackups int
	mu           sync.RWMutex
}

// Config holds daemon configuration
type Config struct {
	WorkspaceDir  string
	Executable    string
	Args          []string
	MaxRestarts   int
	RestartWindow time.Duration
	MaxLogSize    int64
	MaxLogBackups int
}

// New creates a new Daemon instance
func New(cfg Config) *Daemon {
	if cfg.MaxRestarts <= 0 {
		cfg.MaxRestarts = DefaultMaxRestarts
	}
	if cfg.RestartWindow <= 0 {
		cfg.RestartWindow = DefaultRestartWindow
	}
	if cfg.MaxLogSize <= 0 {
		cfg.MaxLogSize = DefaultMaxLogSize
	}
	if cfg.MaxLogBackups <= 0 {
		cfg.MaxLogBackups = DefaultMaxLogBackups
	}

	return &Daemon{
		pidFile:      filepath.Join(cfg.WorkspaceDir, PIDFileName),
		stateFile:    filepath.Join(cfg.WorkspaceDir, StateFileName),
		logFile:      filepath.Join(cfg.WorkspaceDir, LogFileName),
		executable:   cfg.Executable,
		args:         cfg.Args,
		maxRestarts:  cfg.MaxRestarts,
		restartWindow: cfg.RestartWindow,
		maxLogSize:   cfg.MaxLogSize,
		maxLogBackups: cfg.MaxLogBackups,
	}
}

// Start starts the gateway as a daemon
func (d *Daemon) Start() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if already running
	if state, err := d.loadState(); err == nil && d.isProcessRunning(state.PID) {
		return fmt.Errorf("gateway is already running (PID: %d)", state.PID)
	}

	// Setup logging
	if err := d.setupLogging(); err != nil {
		return fmt.Errorf("failed to setup logging: %w", err)
	}

	logger.InfoC("daemon", "Starting gateway daemon")

	// Start the process
	cmd := exec.Command(d.executable, d.args...)
	cmd.Dir = filepath.Dir(d.executable)

	// Set up output to log file
	logFH, err := os.OpenFile(d.logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFH.Close()

	cmd.Stdout = logFH
	cmd.Stderr = logFH

	// Set process group for cleanup
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start gateway: %w", err)
	}

	pid := cmd.Process.Pid

	// Save state
	state := &State{
		PID:          pid,
		StartTime:    time.Now(),
		RestartCount: 0,
	}

	if err := d.saveState(state); err != nil {
		// If we can't save state, kill the process
		_ = cmd.Process.Kill()
		return fmt.Errorf("failed to save daemon state: %w", err)
	}

	logger.InfoCF("daemon", "Gateway daemon started", map[string]any{
		"pid": pid,
	})

	return nil
}

// Stop stops the running gateway daemon
func (d *Daemon) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	state, err := d.loadState()
	if err != nil {
		return fmt.Errorf("daemon not running: %w", err)
	}

	if !d.isProcessRunning(state.PID) {
		_ = d.cleanup()
		return fmt.Errorf("daemon not running (stale PID file)")
	}

	logger.InfoCF("daemon", "Stopping gateway daemon", map[string]any{
		"pid": state.PID,
	})

	// Try graceful shutdown first
	process, err := os.FindProcess(state.PID)
	if err != nil {
		_ = d.cleanup()
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Send SIGTERM for graceful shutdown
	if err := process.Signal(syscall.SIGTERM); err != nil {
		logger.WarnCF("daemon", "Failed to send SIGTERM, forcing kill", map[string]any{
			"pid": state.PID,
			"error": err.Error(),
		})
		// Force kill if SIGTERM fails
		if err := process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case <-time.After(DefaultShutdownTimeout):
		logger.WarnCF("daemon", "Shutdown timeout, forcing kill", map[string]any{
			"pid": state.PID,
		})
		_ = process.Kill()
		<-done // Drain the channel
	case err := <-done:
		if err != nil {
			logger.WarnCF("daemon", "Process wait error", map[string]any{
				"error": err.Error(),
			})
		}
	}

	if err := d.cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup daemon files: %w", err)
	}

	logger.InfoC("daemon", "Gateway daemon stopped")
	return nil
}

// Restart restarts the gateway daemon
func (d *Daemon) Restart() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	state, err := d.loadState()
	wasRunning := err == nil && d.isProcessRunning(state.PID)

	if wasRunning {
		logger.InfoC("daemon", "Restarting gateway daemon")
		if err := d.stopLocked(); err != nil {
			return fmt.Errorf("failed to stop daemon: %w", err)
		}
		// Small delay to ensure ports are released
		time.Sleep(1 * time.Second)
	} else {
		logger.InfoC("daemon", "Starting gateway daemon (was not running)")
	}

	if err := d.startLocked(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	return nil
}

// Status returns the current daemon status
func (d *Daemon) Status() (*State, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	state, err := d.loadState()
	if err != nil {
		return nil, fmt.Errorf("daemon not running: %w", err)
	}

	if !d.isProcessRunning(state.PID) {
		return nil, fmt.Errorf("daemon not running (stale PID file)")
	}

	// Update uptime
	state.StartTime = state.StartTime

	return state, nil
}

// RunWithAutoRestart runs the gateway with automatic restart on crash
func (d *Daemon) RunWithAutoRestart(gatewayFunc func() error) error {
	// Save current PID
	state := &State{
		PID:       os.Getpid(),
		StartTime: time.Now(),
	}

	if err := d.saveState(state); err != nil {
		return fmt.Errorf("failed to save initial state: %w", err)
	}

	logger.InfoCF("daemon", "Gateway started with auto-recovery", map[string]any{
		"pid": state.PID,
		"max_restarts": d.maxRestarts,
		"restart_window": d.restartWindow,
	})

	for {
		if err := gatewayFunc(); err != nil {
			logger.ErrorCF("daemon", "Gateway crashed", map[string]any{
				"error": err.Error(),
			})

			// Update crash state
			if err := d.recordCrash(); err != nil {
				logger.ErrorCF("daemon", "Failed to record crash", map[string]any{
					"error": err.Error(),
				})
			}

			// Check if we should restart
			shouldRestart, delay := d.shouldRestart()
			if !shouldRestart {
				logger.ErrorC("daemon", "Too many crashes, giving up")
				return fmt.Errorf("gateway crashed too many times")
			}

			logger.InfoCF("daemon", "Restarting gateway", map[string]any{
				"delay": delay,
			})

			time.Sleep(delay)
		} else {
			// Clean shutdown
			logger.InfoC("daemon", "Gateway shutdown cleanly")
			return d.cleanup()
		}
	}
}

// startLocked starts the daemon (caller must hold lock)
func (d *Daemon) startLocked() error {
	// Check if already running
	if state, err := d.loadState(); err == nil && d.isProcessRunning(state.PID) {
		return fmt.Errorf("gateway is already running (PID: %d)", state.PID)
	}

	// Setup logging
	if err := d.setupLogging(); err != nil {
		return fmt.Errorf("failed to setup logging: %w", err)
	}

	// Start the process
	cmd := exec.Command(d.executable, d.args...)
	cmd.Dir = filepath.Dir(d.executable)

	// Set up output to log file
	logFH, err := os.OpenFile(d.logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFH.Close()

	cmd.Stdout = logFH
	cmd.Stderr = logFH

	// Set process group for cleanup
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start gateway: %w", err)
	}

	pid := cmd.Process.Pid

	// Save state
	state := &State{
		PID:       pid,
		StartTime: time.Now(),
	}

	if err := d.saveState(state); err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("failed to save daemon state: %w", err)
	}

	logger.InfoCF("daemon", "Gateway daemon started", map[string]any{
		"pid": pid,
	})

	return nil
}

// stopLocked stops the daemon (caller must hold lock)
func (d *Daemon) stopLocked() error {
	state, err := d.loadState()
	if err != nil {
		return fmt.Errorf("daemon not running: %w", err)
	}

	if !d.isProcessRunning(state.PID) {
		_ = d.cleanup()
		return fmt.Errorf("daemon not running (stale PID file)")
	}

	process, err := os.FindProcess(state.PID)
	if err != nil {
		_ = d.cleanup()
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Send SIGTERM for graceful shutdown
	if err := process.Signal(syscall.SIGTERM); err != nil {
		logger.WarnCF("daemon", "Failed to send SIGTERM, forcing kill", map[string]any{
			"pid": state.PID,
			"error": err.Error(),
		})
		_ = process.Kill()
	}

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case <-time.After(DefaultShutdownTimeout):
		logger.WarnCF("daemon", "Shutdown timeout, forcing kill", map[string]any{
			"pid": state.PID,
		})
		_ = process.Kill()
		<-done
	case <-done:
	}

	return d.cleanup()
}

// recordCrash records a crash and updates restart state
func (d *Daemon) recordCrash() error {
	state, err := d.loadState()
	if err != nil {
		state = &State{
			PID:        os.Getpid(),
			StartTime:  time.Now(),
			CrashCount: 0,
		}
	}

	state.CrashCount++
	now := time.Now()

	// Reset crash count if outside window
	if !state.FirstCrashTime.IsZero() && now.Sub(state.FirstCrashTime) > d.restartWindow {
		state.CrashCount = 1
		state.FirstCrashTime = now
	} else if state.FirstCrashTime.IsZero() {
		state.FirstCrashTime = now
	}

	state.RestartCount++
	state.LastRestart = now

	return d.saveState(state)
}

// shouldRestart determines if the daemon should restart and the delay
func (d *Daemon) shouldRestart() (bool, time.Duration) {
	state, err := d.loadState()
	if err != nil {
		return true, time.Second
	}

	// Check if we've exceeded max restarts
	if state.CrashCount >= d.maxRestarts {
		// Check if we're outside the restart window
		if !state.FirstCrashTime.IsZero() && time.Since(state.FirstCrashTime) > d.restartWindow {
			// Reset and allow restart
			state.CrashCount = 0
			state.FirstCrashTime = time.Time{}
			_ = d.saveState(state)
			return true, time.Second
		}
		return false, 0
	}

	// Exponential backoff: 2^n seconds, max 30 seconds
	backoff := time.Duration(1<<uint(state.CrashCount)) * time.Second
	if backoff > 30*time.Second {
		backoff = 30 * time.Second
	}

	return true, backoff
}

// loadState loads the daemon state from disk
func (d *Daemon) loadState() (*State, error) {
	data, err := os.ReadFile(d.stateFile)
	if err != nil {
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// saveState saves the daemon state to disk atomically
func (d *Daemon) saveState(state *State) error {
	// Write to temp file first
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	tmpFile := d.stateFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tmpFile, d.stateFile)
}

// cleanup removes daemon state files
func (d *Daemon) cleanup() error {
	if err := os.Remove(d.pidFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(d.stateFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// isProcessRunning checks if a process with the given PID is running
func (d *Daemon) isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return false
	}

	return true
}

// setupLogging sets up file logging with rotation
func (d *Daemon) setupLogging() error {
	// Rotate log if too large
	if info, err := os.Stat(d.logFile); err == nil {
		if info.Size() >= d.maxLogSize {
			d.rotateLog()
		}
	}

	return logger.EnableFileLogging(d.logFile)
}

// rotateLog rotates the log file
func (d *Daemon) rotateLog() {
	// Remove oldest backup
	oldestBackup := d.logFile + "." + strconv.Itoa(d.maxLogBackups)
	os.Remove(oldestBackup)

	// Rotate existing backups
	for i := d.maxLogBackups - 1; i >= 1; i-- {
		oldFile := d.logFile + "." + strconv.Itoa(i)
		newFile := d.logFile + "." + strconv.Itoa(i+1)
		os.Rename(oldFile, newFile)
	}

	// Move current log to .1
	os.Rename(d.logFile, d.logFile+".1")
}

// WaitForShutdown waits for SIGTERM and returns a channel
func (d *Daemon) WaitForShutdown() <-chan struct{} {
	ch := make(chan struct{})

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
		<-sigChan
		close(ch)
	}()

	return ch
}
