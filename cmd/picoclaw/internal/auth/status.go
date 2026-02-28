package auth

import "github.com/spf13/cobra"

func newStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current auth status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return authStatusCmd()
		},
	}

	return cmd
}
