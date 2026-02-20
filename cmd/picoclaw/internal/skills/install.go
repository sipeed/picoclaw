package skills

import (
	"fmt"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/spf13/cobra"
)

func newInstallCommand(installer *skills.SkillInstaller) *cobra.Command {
	var registry string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install skill from GitHub",
		Example: `
picoclaw skills install sipeed/picoclaw-skills/weather
picoclaw skills install --registry clawhub github
`,
		Args: func(cmd *cobra.Command, args []string) error {
			reg, _ := cmd.Flags().GetString("registry")

			if reg != "" {
				if len(args) != 2 {
					return fmt.Errorf("when --registry is set, exactly 2 arguments are required: <name> <slug>")
				}
				return nil
			}

			if len(args) != 1 {
				return fmt.Errorf("exactly 1 argument is required: <github>")
			}

			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			if registry != "" {
				cfg, err := internal.LoadConfig()
				if err != nil {
					return err
				}

				return skillsInstallFromRegistry(cfg, args[0], args[1])
			}

			return skillsInstallCmd(installer, args[0])
		},
	}

	cmd.Flags().StringVar(&registry, "registry", "", "--registry <name> <slug>")

	return cmd
}
