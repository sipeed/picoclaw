// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package main

import (
	"fmt"
	"os"

	"github.com/sipeed/picoclaw/pkg/migrate"
	"github.com/spf13/cobra"
)

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate from OpenClaw to PicoClaw",
		Long: `Migrate from OpenClaw to PicoClaw

Examples:
  picoclaw migrate
  picoclaw migrate --dry-run
  picoclaw migrate --refresh
  picoclaw migrate --force`,
		RunE: runMigrate,
	}
	cmd.Flags().Bool("dry-run", false, "Show what would be migrated without making changes")
	cmd.Flags().Bool("config-only", false, "Only migrate config, skip workspace files")
	cmd.Flags().Bool("workspace-only", false, "Only migrate workspace files, skip config")
	cmd.Flags().Bool("force", false, "Skip confirmation prompts")
	cmd.Flags().Bool("refresh", false, "Re-sync workspace files from OpenClaw (repeatable)")
	cmd.Flags().String("openclaw-home", "", "Override OpenClaw home directory (default: ~/.openclaw)")
	cmd.Flags().String("picoclaw-home", "", "Override PicoClaw home directory (default: ~/.picoclaw)")
	return cmd
}

func runMigrate(cmd *cobra.Command, args []string) error {
	opts := migrate.Options{}
	opts.DryRun, _ = cmd.Flags().GetBool("dry-run")
	opts.ConfigOnly, _ = cmd.Flags().GetBool("config-only")
	opts.WorkspaceOnly, _ = cmd.Flags().GetBool("workspace-only")
	opts.Force, _ = cmd.Flags().GetBool("force")
	opts.Refresh, _ = cmd.Flags().GetBool("refresh")
	opts.OpenClawHome, _ = cmd.Flags().GetString("openclaw-home")
	opts.PicoClawHome, _ = cmd.Flags().GetString("picoclaw-home")

	result, err := migrate.Run(opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if !opts.DryRun {
		migrate.PrintSummary(result)
	}
	return nil
}
