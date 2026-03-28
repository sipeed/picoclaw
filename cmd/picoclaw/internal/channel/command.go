package channel

import "github.com/spf13/cobra"

func NewChannelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "channel",
		Aliases: []string{"ch"},
		Short:   "Manage channel runtimes",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newStartCommand())

	return cmd
}
