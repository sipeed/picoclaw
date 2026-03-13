package onboard

import (
	"embed"

	"github.com/spf13/cobra"
)

//go:generate cp -r ../../../../workspace .
//go:embed workspace
var embeddedFiles embed.FS

func NewOnboardCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "onboard",
		Aliases: []string{"o"},
		Short:   "Initialize picoclaw configuration and workspace",
		Run: func(cmd *cobra.Command, args []string) {
			onboard()
		},
	}

	return cmd
}
