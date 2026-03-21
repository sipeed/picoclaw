package skills

import "github.com/spf13/cobra"

func newListBuiltinCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list-builtin",
		Short:   "List available builtin skills",
		Example: `piconomous skills list-builtin`,
		Run: func(_ *cobra.Command, _ []string) {
			skillsListBuiltinCmd()
		},
	}

	return cmd
}
