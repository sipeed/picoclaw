package onboard

import (
	"embed"

	"github.com/spf13/cobra"
)

// Prepare an embedded file system
// 1. Remove any vestigial `workspace` folder that might be present
// 2. Duplicate `workspace` folder into the module
// 3. Embed the module's `workspace` folder into the executable
//
//go:generate rm -rf ./workspace
//go:generate cp -r ../../../../workspace .
//go:embed workspace
var embeddedFiles embed.FS

func NewOnboardCommand() *cobra.Command {
	var encrypt bool

	cmd := &cobra.Command{
		Use:     "onboard",
		Aliases: []string{"o"},
		Short:   "Initialize picoclaw configuration and workspace",
		// Run without subcommands → original onboard flow
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				onboard(encrypt)
			} else {
				_ = cmd.Help()
			}
		},
	}

	cmd.Flags().BoolVar(&encrypt, "enc", false,
		"Enable credential encryption (generates SSH key and prompts for passphrase)")

	return cmd
}
