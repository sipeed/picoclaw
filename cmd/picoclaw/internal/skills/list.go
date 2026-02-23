package skills

import (
	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/pkg/skills"
)

func newListCommand(skillsLoader *skills.SkillsLoader) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List installed skills",
		Example: `picoclaw skills list`,
		Run: func(_ *cobra.Command, _ []string) {
			skillsListCmd(skillsLoader)
		},
	}

	return cmd
}
