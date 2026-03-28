package onboard

import (
	"embed"

	"github.com/spf13/cobra"
)

//go:generate cp -r ../../../../workspace .
//go:embed workspace
var embeddedFiles embed.FS

func NewOnboardCommand() *cobra.Command {
	var encrypt bool
	var yes bool

	cmd := &cobra.Command{
		Use:     "onboard",
		Aliases: []string{"o"},
		Short:   "Initialize picoclaw configuration and workspace",
		// Run without subcommands → original onboard flow
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				onboard(encrypt, yes)
			} else {
				_ = cmd.Help()
			}
		},
	}

	cmd.AddCommand(NewPurgeCommand())

	cmd.Flags().BoolVar(&encrypt, "enc", false,
		"Enable credential encryption (generates SSH key and prompts for passphrase)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false,
		"Assume 'yes' for all prompts (useful for scripts/Docker non-TTY builds)")

	return cmd
}
