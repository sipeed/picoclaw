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

	cmd := &cobra.Command{
		Use:     "onboard",
		Aliases: []string{"o"},
		Short:   "Initialize picoclaw configuration and workspace",
		Run: func(cmd *cobra.Command, args []string) {
			onboard(encrypt)
		},
	}

	cmd.Flags().BoolVar(&encrypt, "enc", false,
		"Enable credential encryption (generates SSH key and prompts for passphrase)")

	return cmd
}
