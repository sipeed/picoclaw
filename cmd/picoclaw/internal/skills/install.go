package skills

import (
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/spf13/cobra"
)

func newInstallCommand(installer *skills.SkillInstaller) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "install",
		Short:   "Install skill from GitHub",
		Example: `picoclaw skills install sipeed/picoclaw-skills/weather`,
		Args:    cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			skillsInstallCmd(installer, args[0])
		},
	}

	return cmd
}
