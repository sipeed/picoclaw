package skills

import (
	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/pkg/skills"
)

func newShowCommand(loaderFn func() (*skills.SkillsLoader, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show",
		Short:   "Show skill details",
		Args:    cobra.ExactArgs(1),
		Example: `picoclaw skills show weather`,
		RunE: func(_ *cobra.Command, args []string) error {
			loader, err := loaderFn()
			if err != nil {
				return err
			}
			skillsShowCmd(loader, args[0])
			return nil
		},
	}

	return cmd
}
