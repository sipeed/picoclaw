package gateway

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/daemon"
	"github.com/spf13/cobra"
)

func NewGatewayCommand() *cobra.Command {
	var debug bool
	var runDaemon bool

	cmd := &cobra.Command{
		Use:     "gateway",
		Aliases: []string{"g"},
		Short:   "Manage picoclaw gateway daemon",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			// Check if running as daemon child process
			if runDaemon {
				return runDaemonMode(debug)
			}
			// Default to running in foreground
			return gatewayCmd(debug)
		},
	}

	cmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")
	cmd.Flags().BoolVar(&runDaemon, "run-daemon", false, "Run in daemon mode (internal use)")

	// Add daemon subcommands
	cmd.AddCommand(
		newStartCommand(),
		newStopCommand(),
		newRestartCommand(),
		newStatusCommand(),
	)

	return cmd
}

func newStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the gateway as a daemon",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return startDaemon()
		},
	}
	return cmd
}

func newStopCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the gateway daemon",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return stopDaemon()
		},
	}
	return cmd
}

func newRestartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart the gateway daemon",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return restartDaemon()
		},
	}
	return cmd
}

func newStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the gateway daemon status",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return showStatus()
		},
	}
	return cmd
}

func startDaemon() error {
	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get the current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Create daemon instance
	d := daemon.New(daemon.Config{
		WorkspaceDir:  cfg.WorkspacePath(),
		Executable:    execPath,
		Args:          []string{"gateway", "--run-daemon"},
		MaxRestarts:   3,
		RestartWindow: 5 * time.Minute,
	})

	if err := d.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	fmt.Println("Gateway daemon started successfully")
	fmt.Printf("Logs: %s\n", fmt.Sprintf("%s/%s", cfg.WorkspacePath(), daemon.LogFileName))
	return nil
}

func stopDaemon() error {
	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	d := daemon.New(daemon.Config{
		WorkspaceDir: cfg.WorkspacePath(),
	})

	if err := d.Stop(); err != nil {
		return fmt.Errorf("failed to stop daemon: %w", err)
	}

	fmt.Println("Gateway daemon stopped successfully")
	return nil
}

func restartDaemon() error {
	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	d := daemon.New(daemon.Config{
		WorkspaceDir:  cfg.WorkspacePath(),
		Executable:    execPath,
		Args:          []string{"gateway", "--run-daemon"},
		MaxRestarts:   3,
		RestartWindow: 5 * time.Minute,
	})

	if err := d.Restart(); err != nil {
		return fmt.Errorf("failed to restart daemon: %w", err)
	}

	fmt.Println("Gateway daemon restarted successfully")
	return nil
}

func showStatus() error {
	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	d := daemon.New(daemon.Config{
		WorkspaceDir: cfg.WorkspacePath(),
	})

	state, err := d.Status()
	if err != nil {
		return fmt.Errorf("daemon status: %w", err)
	}

	uptime := time.Since(state.StartTime)
	fmt.Printf("Gateway Daemon Status:\n")
	fmt.Printf("  Status:      Running\n")
	fmt.Printf("  PID:         %d\n", state.PID)
	fmt.Printf("  Uptime:      %s\n", formatDuration(uptime))
	fmt.Printf("  Restarts:    %d\n", state.RestartCount)
	if !state.LastRestart.IsZero() {
		fmt.Printf("  Last Restart: %s ago\n", formatDuration(time.Since(state.LastRestart)))
	}

	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		mins := int(d.Minutes()) % 60
		return fmt.Sprintf("%d hours %d minutes", hours, mins)
	} else {
		days := int(d.Hours() / 24)
		hours := int(d.Hours()) % 24
		return fmt.Sprintf("%d days %d hours", days, hours)
	}
}

// runDaemonMode is the entry point when running as a daemon
func runDaemonMode(debug bool) error {
	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	d := daemon.New(daemon.Config{
		WorkspaceDir:  cfg.WorkspacePath(),
		MaxRestarts:   3,
		RestartWindow: 5 * time.Minute,
	})

	// Run the gateway with auto-restart
	return d.RunWithAutoRestart(func() error {
		return gatewayCmd(debug)
	})
}

// isProcessRunning checks if a process with the given PID is running
func isProcessRunning(pid int) bool {
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
