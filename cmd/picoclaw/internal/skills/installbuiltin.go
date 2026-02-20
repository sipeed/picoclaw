package skills

import "github.com/spf13/cobra"

func newInstallBuiltinCommand(workspace string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "install-builtin",
		Short:   "Install all builtin skills to workspace",
		Example: `picoclaw skills install-builtin`,
		Run: func(_ *cobra.Command, _ []string) {
			skillsInstallBuiltinCmd(workspace)
		},
	}

	return cmd
}
