package gateway

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/gateway"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/pid"
	"github.com/sipeed/picoclaw/pkg/utils"
)

func NewGatewayCommand() *cobra.Command {
	var debug bool
	var noTruncate bool
	var allowEmpty bool

	cmd := &cobra.Command{
		Use:     "gateway",
		Aliases: []string{"g"},
		Short:   "Start picoclaw gateway",
		Args:    cobra.NoArgs,
		PreRunE: func(_ *cobra.Command, _ []string) error {
			if noTruncate && !debug {
				return fmt.Errorf("the --no-truncate option can only be used in conjunction with --debug (-d)")
			}

			if noTruncate {
				utils.SetDisableTruncation(true)
				logger.Info("String truncation is globally disabled via 'no-truncate' flag")
			}

			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return gateway.Run(debug, internal.GetPicoclawHome(), internal.GetConfigPath(), allowEmpty)
		},
	}

	cmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")
	cmd.Flags().BoolVarP(&noTruncate, "no-truncate", "T", false, "Disable string truncation in debug logs")
	cmd.Flags().BoolVarP(
		&allowEmpty,
		"allow-empty",
		"E",
		false,
		"Continue starting even when no default model is configured",
	)

	// Add stop subcommand
	cmd.AddCommand(newGatewayStopCommand())

	return cmd
}

func newGatewayStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the running picoclaw gateway",
		Long:  `Stop the running picoclaw gateway by reading its PID file and sending a termination signal.`,
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			homePath := internal.GetPicoclawHome()

			// Read PID file and check if process is running
			data := pid.ReadPidFileWithCheck(homePath)
			if data == nil {
				fmt.Println("gateway is not running")
				return nil
			}

			// Find the process
			process, err := os.FindProcess(data.PID)
			if err != nil {
				return fmt.Errorf("failed to find gateway process (PID: %d): %w", data.PID, err)
			}

			// Send termination signal
			fmt.Printf("stopping gateway (PID: %d)...\n", data.PID)
			if err := stopProcess(process); err != nil {
				return fmt.Errorf("failed to stop gateway (PID: %d): %w", data.PID, err)
			}

			fmt.Println("gateway stopped successfully")
			return nil
		},
	}
}
