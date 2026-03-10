package skills

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/skills"
)

func newInstallCommand(installerFn func() (*skills.SkillInstaller, error)) *cobra.Command {
	var registry string
	var force bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install skill from GitHub or Git repository",
		Example: `
picoclaw skills install sipeed/picoclaw-skills/weather
picoclaw skills install git@gitlab.com:user/my-skill.git
picoclaw skills install https://gitlab.com/user/my-skill.git
picoclaw skills install --force git@gitlab.com:user/my-skill.git
picoclaw skills install --registry clawhub github
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if registry != "" {
				if len(args) != 1 {
					return fmt.Errorf("when --registry is set, exactly 1 argument is required: <slug>")
				}
				return nil
			}

			if len(args) != 1 {
				return fmt.Errorf("exactly 1 argument is required: <github-repo> or <git-url>")
			}

			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			installer, err := installerFn()
			if err != nil {
				return err
			}

			if registry != "" {
				cfg, err := internal.LoadConfig()
				if err != nil {
					return err
				}

				return skillsInstallFromRegistry(cfg, registry, args[0])
			}

			// Check if input is a Git URL.
			if skills.IsGitURL(args[0]) {
				return skillsInstallFromGitCmd(installer, args[0], force)
			}

			return skillsInstallCmd(installer, args[0])
		},
	}

	cmd.Flags().StringVar(&registry, "registry", "", "Install from registry: --registry <name> <slug>")
	cmd.Flags().BoolVarP(&force, "force", "f", true, "Overwrite existing skills (default: true, use --force=false to skip)")

	return cmd
}
