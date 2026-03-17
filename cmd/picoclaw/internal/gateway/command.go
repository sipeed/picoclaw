package gateway

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/gateway"
	"github.com/sipeed/picoclaw/pkg/logger"
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
			return gateway.Run(debug, internal.GetConfigPath(), allowEmpty)
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

	return cmd
}
