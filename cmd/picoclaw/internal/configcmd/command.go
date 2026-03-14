package configcmd

import (
	"github.com/spf13/cobra"
)

func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration (model_list, agents)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newModelListCommand())
	cmd.AddCommand(newAgentCommand())
	return cmd
}

func newModelListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model_list",
		Short: "Manage model_list (list, get, set, add, remove, update)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(
		newModelListListCommand(),
		newModelListGetCommand(),
		newModelListSetCommand(),
		newModelListAddCommand(),
		newModelListRemoveCommand(),
		newModelListUpdateCommand(),
	)
	return cmd
}

func newAgentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage agents (defaults, list, add, remove, update)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newAgentDefaultsCommand())
	cmd.AddCommand(newAgentListCommand())
	cmd.AddCommand(newAgentAddCommand())
	cmd.AddCommand(newAgentRemoveCommand())
	cmd.AddCommand(newAgentUpdateCommand())
	return cmd
}
