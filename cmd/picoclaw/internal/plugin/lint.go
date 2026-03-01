package plugin

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/cmd/picoclaw/internal/pluginruntime"
	"github.com/sipeed/picoclaw/pkg/config"
)

func newLintSubcommand() *cobra.Command {
	configPath := internal.GetConfigPath()

	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Lint plugin configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("error loading config: %w", err)
			}

			if _, _, err := pluginruntime.ResolveConfiguredPlugins(cfg); err != nil {
				return fmt.Errorf("invalid plugin config: %w", err)
			}

			if _, err := fmt.Fprintln(cmd.OutOrStdout(), "plugin config lint: ok"); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", internal.GetConfigPath(), "Path to config file")

	return cmd
}
