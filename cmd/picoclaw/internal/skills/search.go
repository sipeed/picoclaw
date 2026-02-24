package skills

import (
	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/pkg/skills"
)

func newSearchCommand(installerFn func() (*skills.SkillInstaller, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search available skills",
		RunE: func(_ *cobra.Command, _ []string) error {
			installer, err := installerFn()
			if err != nil {
				return err
			}
			skillsSearchCmd(installer)
			return nil
		},
	}

	return cmd
}
