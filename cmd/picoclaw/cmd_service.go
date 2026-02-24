// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sipeed/picoclaw/pkg/daemon"
)

// gatewayServiceCmd handles gateway service management subcommands.
// Usage: picoclaw gateway <start|stop|restart|status>
func gatewayServiceCmd(subcommand string) {
	// Get the picoclaw config directory
	configDir := getConfigDir()

	// Get the path to the current binary
	binaryPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error: Unable to determine binary path: %v\n", err)
		os.Exit(1)
	}

	// Create daemon service configuration
	serviceConfig := &daemon.ServiceConfig{
		ConfigDir:  configDir,
		BinaryPath: binaryPath,
		Version:    formatVersion(),
		// Use default restart policy: 3 attempts, exponential backoff
		RestartPolicy: daemon.DefaultRestartPolicy(),
		// Log files will be stored in config directory
	}

	// Create the daemon service
	service, err := daemon.NewService(serviceConfig)
	if err != nil {
		fmt.Printf("Error: Failed to create daemon service: %v\n", err)
		os.Exit(1)
	}

	// Execute the requested subcommand
	switch subcommand {
	case "start":
		handleStart(service)

	case "stop":
		handleStop(service)

	case "restart":
		handleRestart(service)

	case "status":
		handleStatus(service)

	default:
		fmt.Printf("Error: Unknown gateway subcommand: %s\n", subcommand)
		fmt.Println("Valid subcommands are: start, stop, restart, status")
		os.Exit(1)
	}
}

// handleStart starts the gateway daemon.
func handleStart(service *daemon.Service) {
	fmt.Printf("%s Starting PicoClaw Gateway...\n", logo)

	if err := service.Start(); err != nil {
		switch e := err.(type) {
		case *daemon.AlreadyRunningError:
			fmt.Printf("✗ Gateway is already running with PID %d\n", e.GetPID())
			fmt.Println("\nUse 'picoclaw gateway status' for more information")
		default:
			fmt.Printf("✗ Failed to start gateway: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Println("✓ Gateway started successfully")
	fmt.Printf("  PID: %d\n", service.Status().PID)
	fmt.Printf("  Log: %s\n", filepath.Join(getConfigDir(), daemon.DefaultLogFileName))
	fmt.Println("\nUse 'picoclaw gateway status' to check the gateway status")
	fmt.Println("Use 'picoclaw gateway stop' to stop the gateway")
}

// handleStop stops the gateway daemon.
func handleStop(service *daemon.Service) {
	fmt.Printf("%s Stopping PicoClaw Gateway...\n", logo)

	if err := service.Stop(); err != nil {
		switch err.(type) {
		case *daemon.NotRunningError:
			fmt.Println("✗ Gateway is not running")
			fmt.Println("\nUse 'picoclaw gateway start' to start the gateway")
		default:
			fmt.Printf("✗ Failed to stop gateway: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Println("✓ Gateway stopped successfully")
}

// handleRestart restarts the gateway daemon with automatic crash recovery.
func handleRestart(service *daemon.Service) {
	fmt.Printf("%s Restarting PicoClaw Gateway...\n", logo)

	if err := service.Restart(); err != nil {
		switch e := err.(type) {
		case *daemon.AlreadyRunningError:
			fmt.Printf("✗ Gateway is already running with PID %d\n", e.GetPID())
			fmt.Println("\nUse 'picoclaw gateway stop' first, then 'picoclaw gateway start'")
		case *daemon.MaxRestartsExceededError:
			fmt.Printf("✗ Maximum restart attempts exceeded (%d attempts in %v)\n",
				e.GetAttempts(), e.Window)
			fmt.Println("\nThe gateway is crashing repeatedly. Check the log file for errors:")
			fmt.Printf("  Log: %s\n", filepath.Join(getConfigDir(), daemon.DefaultLogFileName))
		default:
			fmt.Printf("✗ Failed to restart gateway: %v\n", err)
		}
		os.Exit(1)
	}

	status := service.Status()
	fmt.Println("✓ Gateway restarted successfully")
	fmt.Printf("  PID: %d\n", status.PID)
	fmt.Printf("  Restart count: %d\n", status.RestartCount)
	fmt.Printf("  Log: %s\n", filepath.Join(getConfigDir(), daemon.DefaultLogFileName))
}

// handleStatus displays the current status of the gateway daemon.
func handleStatus(service *daemon.Service) {
	status := service.Status()

	fmt.Printf("%s PicoClaw Gateway Status\n", logo)
	fmt.Printf("Version: %s\n\n", formatVersion())

	if !status.IsRunning {
		fmt.Println("Status: Stopped")
		fmt.Println("\nUse 'picoclaw gateway start' to start the gateway")
		return
	}

	fmt.Println("Status: Running")
	fmt.Printf("  PID: %d\n", status.PID)

	if !status.StartTime.IsZero() {
		fmt.Printf("  Started: %s\n", status.StartTime.Format("2006-01-02 15:04:05"))

		if status.Uptime > 0 {
			fmt.Printf("  Uptime: %s\n", formatUptime(status.Uptime))
		}
	}

	if status.RestartCount > 0 {
		fmt.Printf("  Restarts: %d\n", status.RestartCount)
	}

	if status.Version != "" {
		fmt.Printf("  Version: %s\n", status.Version)
	}

	// Log file location
	logPath := filepath.Join(getConfigDir(), daemon.DefaultLogFileName)
	fmt.Printf("\nLog file: %s\n", logPath)
}

// getConfigDir returns the picoclaw configuration directory.
func getConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".picoclaw"
	}
	return filepath.Join(home, ".picoclaw")
}

// formatUptime formats a duration in a human-readable way.
func formatUptime(d time.Duration) string {
	if d < time.Minute {
		return d.String()
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}
