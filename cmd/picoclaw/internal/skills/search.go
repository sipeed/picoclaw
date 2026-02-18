package skills

import (
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/spf13/cobra"
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
