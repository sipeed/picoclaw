// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/utils"
	"github.com/spf13/cobra"
)

type skillsContext struct {
	installer *skills.SkillInstaller
	loader    *skills.SkillsLoader
	workspace string
	cfg       *config.Config
}

func loadSkillsContext() (*skillsContext, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("Error loading config: %w", err)
	}

	workspace := cfg.WorkspacePath()
	installer := skills.NewSkillInstaller(workspace)
	globalDir := filepath.Dir(getConfigPath())
	globalSkillsDir := filepath.Join(globalDir, "skills")
	builtinSkillsDir := filepath.Join(globalDir, "picoclaw", "skills")
	loader := skills.NewSkillsLoader(workspace, globalSkillsDir, builtinSkillsDir)

	return &skillsContext{
		installer: installer,
		loader:    loader,
		workspace: workspace,
		cfg:       cfg,
	}, nil
}

func newSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage skills (install, list, remove)",
		Example: `  picoclaw skills list
  picoclaw skills install sipeed/picoclaw-skills/weather
  picoclaw skills install --registry clawhub github
  picoclaw skills install-builtin
  picoclaw skills list-builtin
  picoclaw skills remove weather`,
	}
	cmd.AddCommand(
		newSkillsListCmd(),
		newSkillsInstallCmd(),
		newSkillsRemoveCmd(),
		newSkillsInstallBuiltinCmd(),
		newSkillsListBuiltinCmd(),
		newSkillsSearchCmd(),
		newSkillsShowCmd(),
	)
	return cmd
}

func newSkillsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed skills",
		RunE:  runSkillsList,
	}
}

func newSkillsInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <repo>",
		Short: "Install skill from GitHub",
		Example: `  picoclaw skills install sipeed/picoclaw-skills/weather
  picoclaw skills install --registry clawhub github`,
		Args: cobra.MinimumNArgs(1),
		RunE: runSkillsInstall,
	}
	// Add --registry flag support
	cmd.Flags().String("registry", "", "Install from registry (e.g., clawhub)")
	return cmd
}

func newSkillsRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"uninstall"},
		Short:   "Remove installed skill",
		Example: `  picoclaw skills remove weather
  picoclaw skills uninstall weather`,
		Args: cobra.ExactArgs(1),
		RunE: runSkillsRemove,
	}
}

func newSkillsInstallBuiltinCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "install-builtin",
		Short:   "Install all builtin skills to workspace",
		Example: `  picoclaw skills install-builtin`,
		RunE:    runSkillsInstallBuiltin,
	}
}

func newSkillsListBuiltinCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list-builtin",
		Short:   "List available builtin skills",
		Example: `  picoclaw skills list-builtin`,
		RunE:    runSkillsListBuiltin,
	}
}

func newSkillsSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search",
		Short: "Search available skills",
		RunE:  runSkillsSearch,
	}
}

func newSkillsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "show <name>",
		Short:   "Show skill details",
		Example: `  picoclaw skills show weather`,
		Args:    cobra.ExactArgs(1),
		RunE:    runSkillsShow,
	}
}

func runSkillsList(cmd *cobra.Command, args []string) error {
	sc, err := loadSkillsContext()
	if err != nil {
		fmt.Println(err)
		return nil
	}

	allSkills := sc.loader.ListSkills()

	if len(allSkills) == 0 {
		fmt.Println("No skills installed.")
		return nil
	}

	fmt.Println("\nInstalled Skills:")
	fmt.Println("------------------")
	for _, skill := range allSkills {
		fmt.Printf("  ✓ %s (%s)\n", skill.Name, skill.Source)
		if skill.Description != "" {
			fmt.Printf("    %s\n", skill.Description)
		}
	}
	return nil
}

func runSkillsInstall(cmd *cobra.Command, args []string) error {
	sc, err := loadSkillsContext()
	if err != nil {
		fmt.Println(err)
		return nil
	}

	// Check for --registry flag
	registry, _ := cmd.Flags().GetString("registry")
	if registry != "" {
		if len(args) < 1 {
			fmt.Println("Usage: picoclaw skills install --registry <name> <slug>")
			fmt.Println("Example: picoclaw skills install --registry clawhub github")
			return nil
		}
		slug := args[0]
		return skillsInstallFromRegistry(sc.cfg, registry, slug)
	}

	// Default: install from GitHub
	if len(args) < 1 {
		fmt.Println("Usage: picoclaw skills install <github-repo>")
		fmt.Println("       picoclaw skills install --registry <name> <slug>")
		return nil
	}
	repo := args[0]
	fmt.Printf("Installing skill from %s...\n", repo)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := sc.installer.InstallFromGitHub(ctx, repo); err != nil {
		fmt.Printf("✗ Failed to install skill: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Skill '%s' installed successfully!\n", filepath.Base(repo))
	return nil
}

// skillsInstallFromRegistry installs a skill from a named registry (e.g. clawhub).
func skillsInstallFromRegistry(cfg *config.Config, registryName, slug string) error {
	err := utils.ValidateSkillIdentifier(registryName)
	if err != nil {
		fmt.Printf("✗ Invalid registry name: %v\n", err)
		os.Exit(1)
	}

	err = utils.ValidateSkillIdentifier(slug)
	if err != nil {
		fmt.Printf("✗ Invalid slug: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Installing skill '%s' from %s registry...\n", slug, registryName)

	registryMgr := skills.NewRegistryManagerFromConfig(skills.RegistryConfig{
		MaxConcurrentSearches: cfg.Tools.Skills.MaxConcurrentSearches,
		ClawHub:               skills.ClawHubConfig(cfg.Tools.Skills.Registries.ClawHub),
	})

	registry := registryMgr.GetRegistry(registryName)
	if registry == nil {
		fmt.Printf("✗ Registry '%s' not found or not enabled. Check your config.json.\n", registryName)
		os.Exit(1)
	}

	workspace := cfg.WorkspacePath()
	targetDir := filepath.Join(workspace, "skills", slug)

	if _, err := os.Stat(targetDir); err == nil {
		fmt.Printf("✗ Skill '%s' already installed at %s\n", slug, targetDir)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := os.MkdirAll(filepath.Join(workspace, "skills"), 0755); err != nil {
		fmt.Printf("✗ Failed to create skills directory: %v\n", err)
		os.Exit(1)
	}

	result, err := registry.DownloadAndInstall(ctx, slug, "", targetDir)
	if err != nil {
		rmErr := os.RemoveAll(targetDir)
		if rmErr != nil {
			fmt.Printf("✗ Failed to remove partial install: %v\n", rmErr)
		}
		fmt.Printf("✗ Failed to install skill: %v\n", err)
		os.Exit(1)
	}

	if result.IsMalwareBlocked {
		rmErr := os.RemoveAll(targetDir)
		if rmErr != nil {
			fmt.Printf("✗ Failed to remove partial install: %v\n", rmErr)
		}
		fmt.Printf("✗ Skill '%s' is flagged as malicious and cannot be installed.\n", slug)
		os.Exit(1)
	}

	if result.IsSuspicious {
		fmt.Printf("⚠️  Warning: skill '%s' is flagged as suspicious.\n", slug)
	}

	fmt.Printf("✓ Skill '%s' v%s installed successfully!\n", slug, result.Version)
	if result.Summary != "" {
		fmt.Printf("  %s\n", result.Summary)
	}
	return nil
}

func runSkillsRemove(cmd *cobra.Command, args []string) error {
	sc, err := loadSkillsContext()
	if err != nil {
		fmt.Println(err)
		return nil
	}

	skillName := args[0]
	fmt.Printf("Removing skill '%s'...\n", skillName)

	if err := sc.installer.Uninstall(skillName); err != nil {
		fmt.Printf("✗ Failed to remove skill: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Skill '%s' removed successfully!\n", skillName)
	return nil
}

func runSkillsInstallBuiltin(cmd *cobra.Command, args []string) error {
	sc, err := loadSkillsContext()
	if err != nil {
		fmt.Println(err)
		return nil
	}

	builtinSkillsDir := "./picoclaw/skills"
	workspaceSkillsDir := filepath.Join(sc.workspace, "skills")

	fmt.Printf("Copying builtin skills to workspace...\n")

	skillsToInstall := []string{
		"weather",
		"news",
		"stock",
		"calculator",
	}

	for _, skillName := range skillsToInstall {
		builtinPath := filepath.Join(builtinSkillsDir, skillName)
		workspacePath := filepath.Join(workspaceSkillsDir, skillName)

		if _, err := os.Stat(builtinPath); err != nil {
			fmt.Printf("⊘ Builtin skill '%s' not found: %v\n", skillName, err)
			continue
		}

		if err := os.MkdirAll(workspacePath, 0755); err != nil {
			fmt.Printf("✗ Failed to create directory for %s: %v\n", skillName, err)
			continue
		}

		if err := copyDirectory(builtinPath, workspacePath); err != nil {
			fmt.Printf("✗ Failed to copy %s: %v\n", skillName, err)
		}
	}

	fmt.Println("\n✓ All builtin skills installed!")
	fmt.Println("Now you can use them in your workspace.")
	return nil
}

func runSkillsListBuiltin(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return nil
	}
	builtinSkillsDir := filepath.Join(filepath.Dir(cfg.WorkspacePath()), "picoclaw", "skills")

	fmt.Println("\nAvailable Builtin Skills:")
	fmt.Println("-----------------------")

	entries, err := os.ReadDir(builtinSkillsDir)
	if err != nil {
		fmt.Printf("Error reading builtin skills: %v\n", err)
		return nil
	}

	if len(entries) == 0 {
		fmt.Println("No builtin skills available.")
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			skillName := entry.Name()
			skillFile := filepath.Join(builtinSkillsDir, skillName, "SKILL.md")

			description := "No description"
			if _, err := os.Stat(skillFile); err == nil {
				data, err := os.ReadFile(skillFile)
				if err == nil {
					content := string(data)
					if idx := strings.Index(content, "\n"); idx > 0 {
						firstLine := content[:idx]
						if strings.Contains(firstLine, "description:") {
							descLine := strings.Index(content[idx:], "\n")
							if descLine > 0 {
								description = strings.TrimSpace(content[idx+descLine : idx+descLine])
							}
						}
					}
				}
			}
			status := "✓"
			fmt.Printf("  %s  %s\n", status, entry.Name())
			if description != "" {
				fmt.Printf("     %s\n", description)
			}
		}
	}
	return nil
}

func runSkillsSearch(cmd *cobra.Command, args []string) error {
	sc, err := loadSkillsContext()
	if err != nil {
		fmt.Println(err)
		return nil
	}

	fmt.Println("Searching for available skills...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	availableSkills, err := sc.installer.ListAvailableSkills(ctx)
	if err != nil {
		fmt.Printf("✗ Failed to fetch skills list: %v\n", err)
		return nil
	}

	if len(availableSkills) == 0 {
		fmt.Println("No skills available.")
		return nil
	}

	fmt.Printf("\nAvailable Skills (%d):\n", len(availableSkills))
	fmt.Println("--------------------")
	for _, skill := range availableSkills {
		fmt.Printf("  📦 %s\n", skill.Name)
		fmt.Printf("     %s\n", skill.Description)
		fmt.Printf("     Repo: %s\n", skill.Repository)
		if skill.Author != "" {
			fmt.Printf("     Author: %s\n", skill.Author)
		}
		if len(skill.Tags) > 0 {
			fmt.Printf("     Tags: %v\n", skill.Tags)
		}
		fmt.Println()
	}
	return nil
}

func runSkillsShow(cmd *cobra.Command, args []string) error {
	sc, err := loadSkillsContext()
	if err != nil {
		fmt.Println(err)
		return nil
	}

	skillName := args[0]
	content, ok := sc.loader.LoadSkill(skillName)
	if !ok {
		fmt.Printf("✗ Skill '%s' not found\n", skillName)
		return nil
	}

	fmt.Printf("\n📦 Skill: %s\n", skillName)
	fmt.Println("----------------------")
	fmt.Println(content)
	return nil
}
