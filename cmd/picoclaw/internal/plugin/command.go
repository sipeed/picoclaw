package plugin

import "github.com/spf13/cobra"

func NewPluginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Inspect and validate plugins",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newLintSubcommand())

	return cmd
}
