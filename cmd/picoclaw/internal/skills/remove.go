package skills

import (
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/spf13/cobra"
)

func newRemoveCommand(installer *skills.SkillInstaller) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm", "uninstall"},
		Short:   "Remove installed skill",
		Args:    cobra.ExactArgs(1),
		Example: `picoclaw skills remove weather`,
		Run: func(_ *cobra.Command, args []string) {
			skillsRemoveCmd(installer, args[0])
		},
	}

	return cmd
}
