package mcp

import "github.com/spf13/cobra"

func NewMCPCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP server configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(
		newAddCommand(),
		newRemoveCommand(),
		newListCommand(),
		newEditCommand(),
		newTestCommand(),
		newShowCommand(),
	)

	return cmd
}
