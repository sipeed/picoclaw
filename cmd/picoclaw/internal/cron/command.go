package cron

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
)

func NewCronCommand() *cobra.Command {
	var storePath string

	cmd := &cobra.Command{
		Use:     "cron",
		Aliases: []string{"c"},
		Short:   "Manage scheduled tasks",
		Long: `Manage scheduled tasks.

Use 'picoclaw cron list' to view all jobs currently scheduled in your workspace.`,
		Example: `picoclaw cron list
picoclaw cron add "Daily standup" "0 9 * * 1-5"
picoclaw cron disable 1
picoclaw cron remove 1`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		// Resolve storePath at execution time so it reflects the current config
		// and is shared across all subcommands.
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				return fmt.Errorf("error loading config: %w", err)
			}
			storePath = filepath.Join(cfg.WorkspacePath(), "cron", "jobs.json")
			return nil
		},
	}

	cmd.AddCommand(
		newListCommand(func() string { return storePath }),
		newAddCommand(func() string { return storePath }),
		newRemoveCommand(func() string { return storePath }),
		newEnableCommand(func() string { return storePath }),
		newDisableCommand(func() string { return storePath }),
	)

	return cmd
}
