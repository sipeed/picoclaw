package migrate

import (
	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/pkg/migrate"
)

func NewMigrateCommand() *cobra.Command {
	var opts migrate.Options

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate from OpenClaw to PicoClaw",
		Args:  cobra.NoArgs,
		Example: `  picoclaw migrate
  picoclaw migrate --dry-run
  picoclaw migrate --refresh
  picoclaw migrate --force`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := migrate.Run(opts)
			if err != nil {
				return err
			}
			if !opts.DryRun {
				migrate.PrintSummary(result)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false,
		"Show what would be migrated without making changes")
	cmd.Flags().BoolVar(&opts.Refresh, "refresh", false,
		"Re-sync workspace files from OpenClaw (repeatable)")
	cmd.Flags().BoolVar(&opts.ConfigOnly, "config-only", false,
		"Only migrate config, skip workspace files")
	cmd.Flags().BoolVar(&opts.WorkspaceOnly, "workspace-only", false,
		"Only migrate workspace files, skip config")
	cmd.Flags().BoolVar(&opts.Force, "force", false,
		"Skip confirmation prompts")
	cmd.Flags().StringVar(&opts.OpenClawHome, "openclaw-home", "",
		"Override OpenClaw home directory (default: ~/.openclaw)")
	cmd.Flags().StringVar(&opts.PicoClawHome, "picoclaw-home", "",
		"Override PicoClaw home directory (default: ~/.picoclaw)")

	return cmd
}
