package skills

import (
	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/pkg/skills"
)

func newSearchCommand(installer *skills.SkillInstaller) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search available skills",
		Run: func(_ *cobra.Command, _ []string) {
			skillsSearchCmd(installer)
		},
	}

	return cmd
}
