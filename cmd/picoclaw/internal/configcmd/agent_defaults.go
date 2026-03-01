package configcmd

import (
	"github.com/spf13/cobra"
)

func newAgentDefaultsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "defaults",
		Short: "Get or set agents.defaults",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newAgentDefaultsGetCommand())
	cmd.AddCommand(newAgentDefaultsSetCommand())
	return cmd
}
