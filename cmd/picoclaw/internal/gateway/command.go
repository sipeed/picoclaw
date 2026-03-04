package gateway

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewGatewayCommand() *cobra.Command {
	var debug bool

	cmd := &cobra.Command{
		Use:     "gateway",
		Aliases: []string{"g"},
		Short:   "Start picoclaw gateway",
		Long: `Start picoclaw gateway in foreground mode.

Use 'picoclaw gateway start' to run in background mode.
Use 'picoclaw gateway stop' to stop the background service.`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return gatewayCmd(debug)
		},
	}

	cmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")

	// Add subcommands
	cmd.AddCommand(newStartCommand(&debug))
	cmd.AddCommand(newStopCommand())
	cmd.AddCommand(newStatusCommand())

	return cmd
}

func newStartCommand(debug *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start picoclaw gateway in background",
		Long: `Start picoclaw gateway as a background service.

The gateway will run in the background and logs will be written to:
  ~/.picoclaw/logs/gateway.log

Use 'picoclaw gateway stop' to stop the background service.
Use 'picoclaw gateway status' to check if the gateway is running.`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			// Check if already running
			if running, pid := isGatewayRunning(); running {
				fmt.Printf("⚠️  Gateway is already running (PID: %d)\n", pid)
				fmt.Println("Use 'picoclaw gateway stop' to stop it first.")
				return nil
			}

			fmt.Println("🚀 Starting picoclaw gateway in background...")

			if err := startGatewayBackground(*debug); err != nil {
				return fmt.Errorf("failed to start gateway: %w", err)
			}

			running, pid, logPath := getGatewayStatus()
			if running {
				fmt.Printf("✓ Gateway started successfully (PID: %d)\n", pid)
				fmt.Printf("📝 Logs: %s\n", logPath)
				fmt.Println("\nUse 'picoclaw gateway stop' to stop the service.")
				fmt.Println("Use 'picoclaw gateway status' to check the status.")
			} else {
				fmt.Println("⚠️  Gateway may have failed to start. Check the logs:")
				fmt.Printf("   %s\n", logPath)
			}

			return nil
		},
	}

	return cmd
}

func newStopCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop picoclaw gateway background service",
		Long: `Stop the picoclaw gateway running in the background.

This command will gracefully stop the gateway service that was started
with 'picoclaw gateway start'.`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			running, pid := isGatewayRunning()
			if !running {
				fmt.Println("ℹ️  Gateway is not running.")
				return nil
			}

			fmt.Printf("🛑 Stopping picoclaw gateway (PID: %d)...\n", pid)

			if err := stopGatewayProcess(); err != nil {
				return fmt.Errorf("failed to stop gateway: %w", err)
			}

			fmt.Println("✓ Gateway stopped successfully.")
			return nil
		},
	}

	return cmd
}

func newStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show picoclaw gateway status",
		Long:  `Show the current status of the picoclaw gateway background service.`,
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, _ []string) {
			running, pid, logPath := getGatewayStatus()

			if running {
				fmt.Printf("✓ Gateway is running (PID: %d)\n", pid)
				fmt.Printf("📝 Logs: %s\n", logPath)
				fmt.Println("\nUse 'picoclaw gateway stop' to stop the service.")
			} else {
				fmt.Println("○ Gateway is not running.")
				fmt.Println("\nUse 'picoclaw gateway start' to start the service.")
			}
		},
	}

	return cmd
}
