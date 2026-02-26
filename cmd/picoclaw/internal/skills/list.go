package skills

import (
	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/pkg/skills"
)

func newListCommand(loaderFn func() (*skills.SkillsLoader, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List installed skills",
		Example: `picoclaw skills list`,
		RunE: func(_ *cobra.Command, _ []string) error {
			loader, err := loaderFn()
			if err != nil {
				return err
			}
			skillsListCmd(loader)
			return nil
		},
	}

	return cmd
}
