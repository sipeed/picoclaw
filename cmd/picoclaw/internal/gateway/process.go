package gateway

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
)

// getPidFilePath returns the path to the gateway PID file
func getPidFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".picoclaw", "gateway.pid")
}

// savePID saves the process ID to the PID file
func savePID(pid int) error {
	pidFile := getPidFilePath()
	if err := os.MkdirAll(filepath.Dir(pidFile), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0o644)
}

// loadPID loads the process ID from the PID file
func loadPID() (int, error) {
	data, err := os.ReadFile(getPidFilePath())
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(data))
}

// removePIDFile removes the PID file
func removePIDFile() {
	_ = os.Remove(getPidFilePath())
}

// isProcessRunning checks if a process with the given PID is running
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix systems, FindProcess always succeeds, so we need to send signal 0 to check
	if runtime.GOOS != "windows" {
		err = process.Signal(syscall.Signal(0))
		return err == nil
	}

	// On Windows, we rely on FindProcess result
	return true
}

// isGatewayRunning checks if the gateway is currently running
func isGatewayRunning() (bool, int) {
	pid, err := loadPID()
	if err != nil {
		return false, 0
	}

	if isProcessRunning(pid) {
		return true, pid
	}

	// PID file exists but process is not running, clean up
	removePIDFile()
	return false, 0
}

// startGatewayBackground starts the gateway in the background
func startGatewayBackground(debug bool) error {
	// Check if already running
	if running, pid := isGatewayRunning(); running {
		return fmt.Errorf("gateway is already running (PID: %d)", pid)
	}

	// Get the current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Build command arguments
	args := []string{"gateway"}
	if debug {
		args = append(args, "--debug")
	}

	cmd := exec.Command(execPath, args...)

	// Redirect output to log file
	logPath := getLogFilePath()
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Set process group for proper cleanup on Unix systems
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return fmt.Errorf("failed to start gateway: %w", err)
	}

	// Close log file handle (process inherits it)
	_ = logFile.Close()

	// Save PID
	if err := savePID(cmd.Process.Pid); err != nil {
		// Try to kill the process if we can't save the PID
		_ = cmd.Process.Kill()
		return fmt.Errorf("failed to save PID: %w", err)
	}

	return nil
}

// stopGatewayProcess stops the running gateway process
func stopGatewayProcess() error {
	running, pid := isGatewayRunning()
	if !running {
		return fmt.Errorf("gateway is not running")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		removePIDFile()
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Send interrupt signal first for graceful shutdown
	if runtime.GOOS != "windows" {
		if err := process.Signal(os.Interrupt); err != nil {
			// If interrupt fails, try kill
			if killErr := process.Kill(); killErr != nil {
				removePIDFile()
				return fmt.Errorf("failed to stop gateway: %w", killErr)
			}
		}
	} else {
		// Windows doesn't support Interrupt, use Kill directly
		if err := process.Kill(); err != nil {
			removePIDFile()
			return fmt.Errorf("failed to stop gateway: %w", err)
		}
	}

	removePIDFile()
	return nil
}

// getLogFilePath returns the path to the gateway log file
func getLogFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".picoclaw", "logs", "gateway.log")
}

// getGatewayStatus returns the status information of the gateway
func getGatewayStatus() (bool, int, string) {
	running, pid := isGatewayRunning()
	logPath := getLogFilePath()
	return running, pid, logPath
}
