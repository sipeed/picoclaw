package onboard

import "github.com/spf13/cobra"

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
