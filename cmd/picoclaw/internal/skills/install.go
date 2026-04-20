package skills

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
)

func newInstallCommand() *cobra.Command {
	var registry string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install skill from GitHub or a registry",
		Example: `
picoclaw skills install sipeed/picoclaw-skills/weather
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
				return fmt.Errorf("exactly 1 argument is required: <github>")
			}

			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				return err
			}
			if registry != "" {
				return skillsInstallFromRegistry(cfg, registry, args[0])
			}

			return skillsInstallFromRegistry(cfg, "github", args[0])
		},
	}

	cmd.Flags().StringVar(&registry, "registry", "", "Install from registry: --registry <name> <slug>")

	return cmd
}
