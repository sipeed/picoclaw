// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package main

import (
	"fmt"
	"os"

	"github.com/sipeed/picoclaw/pkg/migrate"
)

func migrateCmd() {
	if len(os.Args) > 2 && (os.Args[2] == "--help" || os.Args[2] == "-h") {
		migrateHelp()
		return
	}

	opts := migrate.Options{}

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dry-run":
			opts.DryRun = true
		case "--config-only":
			opts.ConfigOnly = true
		case "--workspace-only":
			opts.WorkspaceOnly = true
		case "--force":
			opts.Force = true
		case "--refresh":
			opts.Refresh = true
		case "--openclaw-home":
			if i+1 < len(args) {
				opts.OpenClawHome = args[i+1]
				i++
			}
		case "--picoclaw-home":
			if i+1 < len(args) {
				opts.PicoClawHome = args[i+1]
				i++
			}
		default:
			fmt.Printf("Unknown flag: %s\n", args[i])
			migrateHelp()
			os.Exit(1)
		}
	}

	result, err := migrate.Run(opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if !opts.DryRun {
		migrate.PrintSummary(result)
	}
}

func migrateHelp() {
	fmt.Println("\nMigrate from OpenClaw to PicoClaw")
	fmt.Println()
	fmt.Println("Usage: picoclaw migrate [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --dry-run          Show what would be migrated without making changes")
	fmt.Println("  --refresh          Re-sync workspace files from OpenClaw (repeatable)")
	fmt.Println("  --config-only      Only migrate config, skip workspace files")
	fmt.Println("  --workspace-only   Only migrate workspace files, skip config")
	fmt.Println("  --force            Skip confirmation prompts")
	fmt.Println("  --openclaw-home    Override OpenClaw home directory (default: ~/.openclaw)")
	fmt.Println("  --picoclaw-home    Override PicoClaw home directory (default: ~/.picoclaw)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  picoclaw migrate              Detect and migrate from OpenClaw")
	fmt.Println("  picoclaw migrate --dry-run    Show what would be migrated")
	fmt.Println("  picoclaw migrate --refresh    Re-sync workspace files")
	fmt.Println("  picoclaw migrate --force      Migrate without confirmation")
}
