package skills

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/skills"
)

func newInstallCommand(installerFn func() (*skills.SkillInstaller, error), proxy string) *cobra.Command {
	var registry string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install skill from GitHub, URL, or domain with .well-known support",
		Example: `
picoclaw skills install sipeed/picoclaw-skills/weather
picoclaw skills install --registry clawhub github
picoclaw skills install https://example.com
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if registry != "" {
				if len(args) != 1 {
					return fmt.Errorf("when --registry is set, exactly 1 argument is required: <slug>")
				}
				return nil
			}

			if len(args) != 1 {
				return fmt.Errorf("exactly 1 argument is required: <github>, <url>, or <domain>")
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

			return skillsInstallCmdWithProxy(installer, args[0], proxy)
		},
	}

	cmd.Flags().StringVar(&registry, "registry", "", "Install from registry: --registry <name> <slug>")

	return cmd
}

func skillsInstallCmdWithProxy(installer *skills.SkillInstaller, input string, proxy string) error {
	client, err := skills.CreateHTTPClient(proxy, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resolved, err := skills.ResolveSkillSource(ctx, client, input)
	if err != nil {
		return fmt.Errorf("failed to resolve skill source: %w", err)
	}

	switch resolved.Type {
	case "well_known":
		return installFromWellKnown(ctx, installer, client, resolved)
	case "github":
		return skillsInstallCmd(installer, input)
	case "json_url":
		return installFromJSONURL(ctx, installer, client, resolved.URL)
	default:
		return skillsInstallCmd(installer, input)
	}
}

func installFromWellKnown(ctx context.Context, installer *skills.SkillInstaller, client *http.Client, resolved *skills.ResolvedSource) error {
	index := resolved.SkillIndex
	fmt.Printf("Resolving skills from %s...\n", resolved.URL)
	fmt.Printf("Found %d skills:\n", len(index.Skills))

	for _, skill := range index.Skills {
		fmt.Printf("  - %s\n", skill.Name)
	}

	fmt.Println("\nInstalling...")

	var installed []string
	var failed []string

	for _, skill := range index.Skills {
		fmt.Printf("Installing skill '%s' from %s...\n", skill.Name, skill.URL)

		if strings.HasPrefix(skill.URL, "http://") || strings.HasPrefix(skill.URL, "https://") {
			if strings.Contains(skill.URL, "github.com") {
				if err := installer.InstallFromGitHub(ctx, skill.URL); err != nil {
					fmt.Printf("  ✗ Failed to install '%s': %v\n", skill.Name, err)
					failed = append(failed, skill.Name)
					continue
				}
			} else {
				if err := installFromJSONURL(ctx, installer, client, skill.URL); err != nil {
					fmt.Printf("  ✗ Failed to install '%s': %v\n", skill.Name, err)
					failed = append(failed, skill.Name)
					continue
				}
			}
		} else {
			fmt.Printf("  ✗ Invalid URL for skill '%s': %s\n", skill.Name, skill.URL)
			failed = append(failed, skill.Name)
			continue
		}

		fmt.Printf("  ✓ Skill '%s' installed successfully!\n", skill.Name)
		installed = append(installed, skill.Name)
	}

	fmt.Printf("\nInstallation complete: %d succeeded, %d failed\n", len(installed), len(failed))
	if len(failed) > 0 {
		fmt.Printf("Failed skills: %s\n", strings.Join(failed, ", "))
	}

	return nil
}

func installFromJSONURL(ctx context.Context, installer *skills.SkillInstaller, client *http.Client, url string) error {
	fmt.Printf("Installing skill from %s...\n", url)

	if strings.Contains(url, "github.com") {
		return installer.InstallFromGitHub(ctx, url)
	}

	return fmt.Errorf("direct JSON URL installation not yet supported for non-GitHub URLs")
}
