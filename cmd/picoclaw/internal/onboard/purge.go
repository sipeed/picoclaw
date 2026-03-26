package onboard

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
)

func NewPurgeCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "purge",
		Short: "Delete the picoclaw workspace and logs",
		Long:  "Completely deletes the .picoclaw/workspace and .picoclaw/logs directories. Use with caution.",
		Run: func(cmd *cobra.Command, args []string) {
			home := internal.GetPicoclawHome()
			workspace := filepath.Join(home, "workspace")
			logs := filepath.Join(home, "logs")

			fmt.Printf("This will delete:\n  - %s\n  - %s\n", workspace, logs)

			if !force {
				fmt.Print("Are you sure? (y/n): ")
				var response string
				fmt.Scanln(&response)
				if response != "y" {
					fmt.Println("Aborted.")
					return
				}
			}

			fmt.Println("Purging...")
			
			if err := os.RemoveAll(workspace); err != nil {
				fmt.Printf("Error deleting workspace: %v\n", err)
			} else {
				fmt.Println("✓ Workspace deleted")
			}

			if err := os.RemoveAll(logs); err != nil {
				fmt.Printf("Error deleting logs: %v\n", err)
			} else {
				fmt.Println("✓ Logs deleted")
			}
			
			fmt.Println("Purge complete.")
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}
