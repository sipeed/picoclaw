package skills

import (
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/spf13/cobra"
)

func newShowCommand(skillsLoader *skills.SkillsLoader) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show",
		Short:   "Show skill details",
		Args:    cobra.ExactArgs(1),
		Example: `picoclaw skills show weather`,
		Run: func(_ *cobra.Command, args []string) {
			skillsShowCmd(skillsLoader, args[0])
		},
	}

	return cmd
}
