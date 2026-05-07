package gateway

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/gateway"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/netbind"
	"github.com/sipeed/picoclaw/pkg/utils"
)

func resolveGatewayHostOverride(explicit bool, host string) (string, error) {
	if !explicit {
		return "", nil
	}
	normalized, err := netbind.NormalizeHostInput(host)
	if err != nil {
		return "", fmt.Errorf("invalid --host value: %w", err)
	}
	return normalized, nil
}

func NewGatewayCommand() *cobra.Command {
	var debug bool
	var noTruncate bool
	var allowEmpty bool
	var host string

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
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolvedHost, err := resolveGatewayHostOverride(cmd.Flags().Changed("host"), host)
			if err != nil {
				return err
			}
			if resolvedHost != "" {
				prevHost, hadPrev := os.LookupEnv(config.EnvGatewayHost)
				if err := os.Setenv(config.EnvGatewayHost, resolvedHost); err != nil {
					return fmt.Errorf("failed to set %s: %w", config.EnvGatewayHost, err)
				}
				defer func() {
					if hadPrev {
						_ = os.Setenv(config.EnvGatewayHost, prevHost)
						return
					}
					_ = os.Unsetenv(config.EnvGatewayHost)
				}()
			}

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
	cmd.Flags().StringVar(
		&host,
		"host",
		"",
		"Host address for gateway binding (overrides gateway.host for this run)",
	)

	return cmd
}
