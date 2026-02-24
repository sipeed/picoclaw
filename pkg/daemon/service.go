// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package daemon

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	// DefaultPIDFileName is the default PID file name
	DefaultPIDFileName = "gateway.pid"
	// DefaultStateFileName is the default state file name
	DefaultStateFileName = "gateway-state.json"
	// DefaultLogFileName is the default log file name
	DefaultLogFileName = "gateway.log"
)

// Service manages a daemon process with lifecycle control.
//
// Design rationale:
// - Provides a high-level API for daemon operations (start, stop, restart, status)
// - Encapsulates PID file, state management, and logging concerns
// - Follows the same Start/Stop/IsRunning pattern as other services (heartbeat, cron)
// - Thread-safe operations with proper mutex protection
type Service struct {
	// configDir is the directory containing configuration and runtime files
	configDir string

	// binaryPath is the path to the picoclaw binary
	binaryPath string

	// args are the arguments to pass to the daemon process
	args []string

	// version is the picoclaw version
	version string

	pidFile     *PIDFile
	state       *StateManager
	logConfig   *LogConfig
	restartPolicy *RestartPolicy

	mu       sync.Mutex
	quitChan chan struct{}
}

// ServiceConfig is the configuration for creating a new Service.
type ServiceConfig struct {
	// ConfigDir is the directory containing config and runtime files (~/.picoclaw)
	ConfigDir string

	// BinaryPath is the path to the picoclaw binary
	BinaryPath string

	// Args are additional arguments to pass to the gateway process
	Args []string

	// Version is the picoclaw version
	Version string

	// RestartPolicy is the restart policy (nil for default)
	RestartPolicy *RestartPolicy

	// LogConfig is the logging configuration (nil for default)
	LogConfig *LogConfig
}

// NewService creates a new daemon service with the given configuration.
func NewService(config *ServiceConfig) (*Service, error) {
	if config == nil {
		return nil, fmt.Errorf("service config cannot be nil")
	}

	pidFilePath := filepath.Join(config.ConfigDir, DefaultPIDFileName)
	stateFilePath := filepath.Join(config.ConfigDir, DefaultStateFileName)
	logFilePath := filepath.Join(config.ConfigDir, DefaultLogFileName)

	logConfig := config.LogConfig
	if logConfig == nil {
		logConfig = DefaultLogConfig(logFilePath)
	}

	return &Service{
		configDir:     config.ConfigDir,
		binaryPath:    config.BinaryPath,
		args:          config.Args,
		version:       config.Version,
		pidFile:       NewPIDFile(pidFilePath),
		state:         NewStateManager(stateFilePath),
		logConfig:     logConfig,
		restartPolicy: config.RestartPolicy,
		quitChan:      make(chan struct{}),
	}, nil
}

// Start starts the gateway daemon.
//
// Returns an error if:
// - The daemon is already running
// - The binary cannot be executed
// - PID file creation fails
func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	logger.InfoCF("daemon", "Starting gateway daemon", map[string]any{
		"version": s.version,
		"binary":  s.binaryPath,
	})

	// Check if already running
	if s.pidFile.IsProcessRunning() {
		pid := s.pidFile.Read()
		return &AlreadyRunningError{pid: pid}
	}

	// Prepare command
	cmd := exec.Command(s.binaryPath, s.buildArgs()...)

	// Set up environment for daemon mode
	// This tells the gateway process it's running as a daemon
	cmd.Env = append(os.Environ(), "PICOCLAW_DAEMON=1")

	// Redirect stdout and stderr to log file
	logFile, err := os.OpenFile(s.logConfig.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start the process
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("failed to start gateway process: %w", err)
	}

	pid := cmd.Process.Pid

	// Write PID file atomically with the child's PID
	if err := s.pidFile.WritePID(pid); err != nil {
		// Failed to write PID file, kill the process
		cmd.Process.Kill()
		logFile.Close()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Verify the child process is actually running
	// This prevents race conditions where status is checked immediately
	time.Sleep(100 * time.Millisecond)
	if !s.pidFile.IsProcessRunning() {
		// Child process died immediately, likely a startup error
		s.pidFile.Remove()
		logFile.Close()
		return fmt.Errorf("gateway process failed to start (check log file: %s)", s.logConfig.Path)
	}

	// Update state
	s.state.SetPID(pid)
	s.state.SetStartTime(time.Now())
	s.state.SetVersion(s.version)

	logger.InfoCF("daemon", "Gateway daemon started", map[string]any{
		"pid":     pid,
		"version": s.version,
	})

	return nil
}

// Stop stops the running gateway daemon gracefully.
//
// Returns an error if:
// - The daemon is not running
// - The PID file cannot be read
// - Signal delivery fails
func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopLocked()
}

// stopLocked stops the running gateway daemon without acquiring the lock.
// Must be called with the lock already held.
func (s *Service) stopLocked() error {
	logger.InfoC("daemon", "Stopping gateway daemon")

	pid := s.pidFile.Read()
	if pid == 0 {
		return &NotRunningError{}
	}

	// Send SIGTERM for graceful shutdown
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case <-done:
		// Process exited
		logger.InfoCF("daemon", "Gateway daemon stopped", map[string]any{
			"pid": pid,
		})
	case <-time.After(30 * time.Second):
		// Timeout, force kill
		logger.WarnCF("daemon", "Gateway did not stop gracefully, forcing", map[string]any{
			"pid": pid,
		})
		process.Kill()
	}

	// Clean up
	s.pidFile.Remove()
	s.state.Clear()

	return nil
}

// Restart restarts the gateway daemon (stop if running, then start).
// This is a simple one-time restart. For auto-restart with crash recovery,
// use the Run method with proper supervision.
//
// Returns an error if:
// - Stop fails
// - Start fails
func (s *Service) Restart() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	logger.InfoC("daemon", "Restarting gateway daemon")

	// Stop if running (call stopLocked to avoid deadlock)
	if s.pidFile.IsProcessRunning() {
		if err := s.stopLocked(); err != nil {
			return fmt.Errorf("failed to stop daemon: %w", err)
		}
	}

	// Start the gateway (non-blocking, returns immediately)
	if err := s.startLocked(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	logger.InfoC("daemon", "Gateway daemon restarted successfully")
	return nil
}

// RunWithAutoRestart starts the gateway and monitors it for crashes.
// If the gateway crashes, it will automatically restart with exponential backoff.
// This method blocks until the gateway crashes 3 times within 5 minutes.
//
// Use this for long-running daemon supervision.
func (s *Service) RunWithAutoRestart() error {
	// Create restart tracker
	tracker := NewRestartTracker(s.restartPolicy)

	// Restart loop with crash recovery
	for tracker.ShouldRestart() {
		// Start the gateway and wait for it to exit
		err := s.startInternal()
		if err != nil {
			// Gateway exited with an error

			// Check if this is an "already running" error
			if _, ok := err.(*AlreadyRunningError); ok {
				return err
			}

			// Record the failed attempt and get backoff duration
			backoff, backoffErr := tracker.RecordAttempt()
			if backoffErr != nil {
				// Max attempts exceeded
				logger.ErrorCF("daemon", "Maximum restart attempts exceeded", map[string]any{
					"attempts": tracker.GetAttemptCount(),
					"max":      s.restartPolicy.MaxAttempts,
				})
				return backoffErr
			}

			// Update state with restart count
			s.state.IncrementRestartCount()

			logger.WarnCF("daemon", "Gateway daemon crashed, will restart", map[string]any{
				"attempt":       tracker.GetAttemptCount(),
				"backoff":       backoff.String(),
				"error":         err.Error(),
			})

			// Wait before next restart attempt
			select {
			case <-time.After(backoff):
				// Continue to next attempt
				continue
			case <-s.quitChan:
				// Abort restart loop
				return fmt.Errorf("restart aborted")
			}
		}
		// If startInternal returned nil, the gateway was stopped manually
		// Exit the loop and return
		break
	}

	return nil
}

// startInternal starts the daemon without locking and waits for it to exit.
// Must be called with the lock held.
// This is used for the auto-restart loop where we monitor for crashes.
func (s *Service) startInternal() error {
	// Start the gateway (non-blocking)
	if err := s.startLocked(); err != nil {
		return err
	}

	// Wait for the process to exit
	pid := s.pidFile.Read()
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Block until process exits
	_, err = process.Wait()
	return err
}

// startLocked starts the gateway daemon without locking and returns immediately.
// Must be called with the lock held.
func (s *Service) startLocked() error {
	// Prepare command
	cmd := exec.Command(s.binaryPath, s.buildArgs()...)
	cmd.Env = append(os.Environ(), "PICOCLAW_DAEMON=1")

	// Redirect output to log file
	logFile, err := os.OpenFile(s.logConfig.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start the process
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("failed to start gateway process: %w", err)
	}

	pid := cmd.Process.Pid

	// Write PID file with the child's PID
	if err := s.pidFile.WritePID(pid); err != nil {
		cmd.Process.Kill()
		logFile.Close()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Verify the child process is actually running
	time.Sleep(100 * time.Millisecond)
	if !s.pidFile.IsProcessRunning() {
		s.pidFile.Remove()
		logFile.Close()
		return fmt.Errorf("gateway process failed to start (check log file: %s)", s.logConfig.Path)
	}

	// Update state
	s.state.SetPID(pid)
	s.state.SetStartTime(time.Now())
	s.state.SetVersion(s.version)

	logger.InfoCF("daemon", "Gateway daemon started", map[string]any{
		"pid":     pid,
		"version": s.version,
	})

	return nil
}

// Status returns the current status of the gateway daemon.
func (s *Service) Status() *StatusInfo {
	s.mu.Lock()
	defer s.mu.Unlock()

	pid := s.pidFile.Read()
	isRunning := s.pidFile.IsProcessRunning()

	state := s.state.GetState()

	return &StatusInfo{
		IsRunning:    isRunning,
		PID:          pid,
		StartTime:    state.StartTime,
		Uptime:       s.state.GetUptime(),
		RestartCount: state.RestartCount,
		Version:      state.Version,
	}
}

// IsRunning returns true if the daemon is currently running.
func (s *Service) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.pidFile.IsProcessRunning()
}

// buildArgs constructs the command line arguments for the gateway process.
func (s *Service) buildArgs() []string {
	args := append([]string{"gateway"}, s.args...)
	return args
}

// Abort aborts any ongoing restart operation.
func (s *Service) Abort() {
	close(s.quitChan)
}

// StatusInfo contains status information about the daemon.
type StatusInfo struct {
	IsRunning    bool
	PID          int
	StartTime    time.Time
	Uptime       time.Duration
	RestartCount int
	Version      string
}

// AlreadyRunningError is returned when attempting to start a daemon
// that is already running.
type AlreadyRunningError struct {
	pid int
}

func (e *AlreadyRunningError) Error() string {
	return fmt.Sprintf("gateway already running with PID %d", e.pid)
}

// GetPID returns the PID of the running process.
func (e *AlreadyRunningError) GetPID() int {
	return e.pid
}

// NotRunningError is returned when attempting to stop a daemon
// that is not running.
type NotRunningError struct{}

func (e *NotRunningError) Error() string {
	return "gateway is not running"
}

// Run runs the gateway in the foreground (not as a daemon).
// This is the existing behavior when running `picoclaw gateway` without subcommands.
//
// This method blocks until the gateway is stopped via Ctrl+C.
// It sets up signal handlers for graceful shutdown and initializes
// file logging for daemon mode.
func (s *Service) Run(gatewayFunc func(context.Context) error) error {
	// Check if running as daemon
	isDaemon := os.Getenv("PICOCLAW_DAEMON") == "1"

	if isDaemon {
		// Set up file logging
		daemonLogger, err := NewLogger(s.logConfig)
		if err != nil {
			return fmt.Errorf("failed to initialize daemon logging: %w", err)
		}
		defer daemonLogger.Close()

		// Redirect logger output to file
		if err := logger.EnableFileLogging(s.logConfig.Path); err != nil {
			return fmt.Errorf("failed to enable file logging: %w", err)
		}
		defer logger.DisableFileLogging()

		logger.InfoCF("daemon", "Gateway started in daemon mode", map[string]any{
			"pid":     os.Getpid(),
			"version": s.version,
		})
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run the gateway function in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- gatewayFunc(ctx)
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		return nil
	case err := <-errChan:
		return err
	}
}
