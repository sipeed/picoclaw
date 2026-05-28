package config

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}

	cmd.AddCommand(newResetCommand())
	return cmd
}

func newResetCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset configuration to factory defaults",
		Args:  cobra.NoArgs,
		Example: `  picoclaw config reset
  picoclaw config reset --force`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if !force {
				fmt.Print("Reset config to factory defaults? API keys will be preserved. (y/n): ")
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(strings.TrimSpace(response)) != "y" {
					fmt.Println("Aborted.")
					return nil
				}
			}

			configPath := internal.GetConfigPath()
			if err := config.ResetToDefaults(configPath); err != nil {
				return fmt.Errorf("reset failed: %w", err)
			}
			fmt.Println("Configuration has been reset to factory defaults.")
			fmt.Println("A backup of the previous config was created in the same directory.")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false,
		"Skip confirmation prompt")

	return cmd
}
