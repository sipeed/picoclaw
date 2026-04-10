package skills

import (
	"github.com/spf13/cobra"
)

func newRemoveCommand(workspaceFn func() (string, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm", "uninstall"},
		Short:   "Remove installed skill",
		Args:    cobra.ExactArgs(1),
		Example: `picoclaw skills remove weather`,
		RunE: func(_ *cobra.Command, args []string) error {
			workspace, err := workspaceFn()
			if err != nil {
				return err
			}
			return skillsRemoveFromWorkspace(workspace, args[0])
		},
	}

	return cmd
}
