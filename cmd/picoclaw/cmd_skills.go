// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/utils"
)

func skillsCmd() {
	if len(os.Args) < 3 {
		skillsHelp()
		return
	}

	subcommand := os.Args[2]

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	workspace := cfg.WorkspacePath()
	installer := skills.NewSkillInstaller(workspace)
	globalDir := filepath.Dir(getConfigPath())
	globalSkillsDir := filepath.Join(globalDir, "skills")
	builtinSkillsDir := filepath.Join(globalDir, "picoclaw", "skills")
	skillsLoader := skills.NewSkillsLoader(workspace, globalSkillsDir, builtinSkillsDir)

	switch subcommand {
	case "list":
		skillsListCmd(skillsLoader)
	case "install":
		skillsInstallCmd(installer, cfg, false)
	case "reinstall":
		skillsInstallCmd(installer, cfg, true)
	case "remove", "uninstall":
		if len(os.Args) < 4 {
			fmt.Println("Usage: picoclaw skills remove <skill-name>")
			return
		}
		skillsRemoveCmd(installer, os.Args[3])
	case "install-builtin":
		skillsInstallBuiltinCmd(workspace)
	case "list-builtin":
		skillsListBuiltinCmd()
	case "search":
		skillsSearchCmd(installer)
	case "show":
		if len(os.Args) < 4 {
			fmt.Println("Usage: picoclaw skills show <skill-name>")
			return
		}
		skillsShowCmd(skillsLoader, os.Args[3])
	default:
		fmt.Printf("Unknown skills command: %s\n", subcommand)
		skillsHelp()
	}
}

func skillsHelp() {
	fmt.Println("\nSkills commands:")
	fmt.Println("  list                    List installed skills")
	fmt.Println("  install <repo> [subpath]  Install skill from GitHub (repo: owner/repo or owner/repo@branch; subpath e.g. skills/kanban-ai)")
	fmt.Println("  install <path-or-url> [name]  Install from .zip / .tar.gz / .tgz (file path or URL)")
	fmt.Println("  reinstall <repo> [subpath]  Overwrite existing skill (same args as install)")
	fmt.Println("  install-builtin         Install all builtin skills to workspace")
	fmt.Println("  list-builtin            List available builtin skills")
	fmt.Println("  remove <name>           Remove installed skill")
	fmt.Println("  search                  Search available skills")
	fmt.Println("  show <name>             Show skill details")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  picoclaw skills list")
	fmt.Println("  picoclaw skills install sipeed/picoclaw-skills")
	fmt.Println("  picoclaw skills install sipeed/picoclaw-skills weather")
	fmt.Println("  picoclaw skills install ./skill.zip")
	fmt.Println("  picoclaw skills install https://example.com/skill.tar.gz my-skill")
	fmt.Println("  picoclaw skills install --registry clawhub github")
	fmt.Println("  picoclaw skills remove weather")
}

func skillsListCmd(loader *skills.SkillsLoader) {
	allSkills := loader.ListSkills()

	if len(allSkills) == 0 {
		fmt.Println("No skills installed.")
		return
	}

	fmt.Println("\nInstalled Skills:")
	fmt.Println("------------------")
	for _, skill := range allSkills {
		fmt.Printf("  ✓ %s (%s)\n", skill.Name, skill.Source)
		if skill.Description != "" {
			fmt.Printf("    %s\n", skill.Description)
		}
	}
}

// isArchiveInstallArg returns true when arg looks like a path or URL to an archive (.zip, .tar.gz, .tgz),
// so we treat it as archive install and avoid conflicting with GitHub repo names like owner/repo.zip.
func isArchiveInstallArg(arg string) bool {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return false
	}
	// Extension: path or URL path must end with .zip, .tar.gz, or .tgz.
	base := arg
	if u, err := url.Parse(arg); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		base = filepath.Base(u.Path)
	} else {
		base = filepath.Base(arg)
	}
	lower := strings.ToLower(base)
	hasExt := strings.HasSuffix(lower, ".zip") || strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz")
	if !hasExt {
		return false
	}
	// Path or protocol: starts with ./, /, \, or http(s)://, or is an existing file.
	if strings.HasPrefix(arg, "./") || strings.HasPrefix(arg, "/") || strings.HasPrefix(arg, "\\") {
		return true
	}
	if strings.HasPrefix(strings.ToLower(arg), "http://") || strings.HasPrefix(strings.ToLower(arg), "https://") {
		return true
	}
	info, err := os.Stat(arg)
	if err == nil && info.Mode().IsRegular() {
		return true
	}
	return false
}

func skillsInstallCmd(installer *skills.SkillInstaller, cfg *config.Config, force bool) {
	if len(os.Args) < 4 {
		verb := "install"
		if force {
			verb = "reinstall"
		}
		fmt.Printf("Usage: picoclaw skills %s <repo> [subpath]\n", verb)
		fmt.Printf("       picoclaw skills %s <path-or-url> [name]   (for .zip / .tar.gz / .tgz)\n", verb)
		fmt.Println("       picoclaw skills install --registry <name> <slug>")
		fmt.Println("  repo: owner/repo or owner/repo@branch (branch defaults to main)")
		fmt.Println("  subpath: optional path in repo (e.g. weather or skills/kanban-ai)")
		return
	}

	// Check for --registry flag.
	if os.Args[3] == "--registry" {
		if len(os.Args) < 6 {
			fmt.Println("Usage: picoclaw skills install --registry <name> <slug>")
			fmt.Println("Example: picoclaw skills install --registry clawhub github")
			return
		}
		registryName := os.Args[4]
		slug := os.Args[5]
		skillsInstallFromRegistry(cfg, registryName, slug)
		return
	}

	// Archive path or URL: .zip / .tar.gz / .tgz (path or URL only, to avoid conflict with GitHub repo names).
	arg := os.Args[3]
	if isArchiveInstallArg(arg) {
		var skillName string
		if len(os.Args) >= 5 {
			skillName = strings.TrimSpace(os.Args[4])
		}
		if force {
			fmt.Printf("Reinstalling skill from %s...\n", arg)
		} else {
			fmt.Printf("Installing skill from %s...\n", arg)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		installedName, err := installer.InstallFromArchive(ctx, arg, skillName, force)
		if err != nil {
			fmt.Printf("\u2717 Failed to install skill: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\u2713 Skill '%s' installed successfully!\n", installedName)
		return
	}

	// GitHub path: <repo> [subpath]
	spec := os.Args[3]
	repo, branch, err := skills.ParseInstallSpec(spec)
	if err != nil {
		fmt.Printf("\u2717 Invalid install spec: %v\n", err)
		os.Exit(1)
	}

	var subpath string
	if len(os.Args) >= 5 {
		subpath = strings.TrimSpace(os.Args[4])
	}

	display := spec
	if subpath != "" {
		display = spec + " " + subpath
	}
	if force {
		fmt.Printf("Reinstalling skill from %s...\n", display)
	} else {
		fmt.Printf("Installing skill from %s...\n", display)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	skillName, err := installer.InstallFromGitHubEx(ctx, repo, branch, subpath, force)
	if err != nil {
		fmt.Printf("\u2717 Failed to install skill: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\u2713 Skill '%s' installed successfully!\n", skillName)
}

// skillsInstallFromRegistry installs a skill from a named registry (e.g. clawhub).
func skillsInstallFromRegistry(cfg *config.Config, registryName, slug string) {
	err := utils.ValidateSkillIdentifier(registryName)
	if err != nil {
		fmt.Printf("\u2717 Invalid registry name: %v\n", err)
		os.Exit(1)
	}

	err = utils.ValidateSkillIdentifier(slug)
	if err != nil {
		fmt.Printf("\u2717 Invalid slug: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Installing skill '%s' from %s registry...\n", slug, registryName)

	registryMgr := skills.NewRegistryManagerFromConfig(skills.RegistryConfig{
		MaxConcurrentSearches: cfg.Tools.Skills.MaxConcurrentSearches,
		ClawHub:               skills.ClawHubConfig(cfg.Tools.Skills.Registries.ClawHub),
	})

	registry := registryMgr.GetRegistry(registryName)
	if registry == nil {
		fmt.Printf("\u2717 Registry '%s' not found or not enabled. Check your config.json.\n", registryName)
		os.Exit(1)
	}

	workspace := cfg.WorkspacePath()
	targetDir := filepath.Join(workspace, "skills", slug)

	if _, err = os.Stat(targetDir); err == nil {
		fmt.Printf("\u2717 Skill '%s' already installed at %s\n", slug, targetDir)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err = os.MkdirAll(filepath.Join(workspace, "skills"), 0o755); err != nil {
		fmt.Printf("\u2717 Failed to create skills directory: %v\n", err)
		os.Exit(1)
	}

	result, err := registry.DownloadAndInstall(ctx, slug, "", targetDir)
	if err != nil {
		rmErr := os.RemoveAll(targetDir)
		if rmErr != nil {
			fmt.Printf("\u2717 Failed to remove partial install: %v\n", rmErr)
		}
		fmt.Printf("\u2717 Failed to install skill: %v\n", err)
		os.Exit(1)
	}

	if result.IsMalwareBlocked {
		rmErr := os.RemoveAll(targetDir)
		if rmErr != nil {
			fmt.Printf("\u2717 Failed to remove partial install: %v\n", rmErr)
		}
		fmt.Printf("\u2717 Skill '%s' is flagged as malicious and cannot be installed.\n", slug)
		os.Exit(1)
	}

	if result.IsSuspicious {
		fmt.Printf("\u26a0\ufe0f  Warning: skill '%s' is flagged as suspicious.\n", slug)
	}

	fmt.Printf("\u2713 Skill '%s' v%s installed successfully!\n", slug, result.Version)
	if result.Summary != "" {
		fmt.Printf("  %s\n", result.Summary)
	}
}

func skillsRemoveCmd(installer *skills.SkillInstaller, skillName string) {
	fmt.Printf("Removing skill '%s'...\n", skillName)

	if err := installer.Uninstall(skillName); err != nil {
		fmt.Printf("✗ Failed to remove skill: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Skill '%s' removed successfully!\n", skillName)
}

func skillsInstallBuiltinCmd(workspace string) {
	builtinSkillsDir := "./picoclaw/skills"
	workspaceSkillsDir := filepath.Join(workspace, "skills")

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

		if err := os.MkdirAll(workspacePath, 0o755); err != nil {
			fmt.Printf("✗ Failed to create directory for %s: %v\n", skillName, err)
			continue
		}

		if err := copyDirectory(builtinPath, workspacePath); err != nil {
			fmt.Printf("✗ Failed to copy %s: %v\n", skillName, err)
		}
	}

	fmt.Println("\n✓ All builtin skills installed!")
	fmt.Println("Now you can use them in your workspace.")
}

func skillsListBuiltinCmd() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}
	builtinSkillsDir := filepath.Join(filepath.Dir(cfg.WorkspacePath()), "picoclaw", "skills")

	fmt.Println("\nAvailable Builtin Skills:")
	fmt.Println("-----------------------")

	entries, err := os.ReadDir(builtinSkillsDir)
	if err != nil {
		fmt.Printf("Error reading builtin skills: %v\n", err)
		return
	}

	if len(entries) == 0 {
		fmt.Println("No builtin skills available.")
		return
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
}

func skillsSearchCmd(installer *skills.SkillInstaller) {
	fmt.Println("Searching for available skills...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	availableSkills, err := installer.ListAvailableSkills(ctx)
	if err != nil {
		fmt.Printf("✗ Failed to fetch skills list: %v\n", err)
		return
	}

	if len(availableSkills) == 0 {
		fmt.Println("No skills available.")
		return
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
}

func skillsShowCmd(loader *skills.SkillsLoader, skillName string) {
	content, ok := loader.LoadSkill(skillName)
	if !ok {
		fmt.Printf("✗ Skill '%s' not found\n", skillName)
		return
	}

	fmt.Printf("\n📦 Skill: %s\n", skillName)
	fmt.Println("----------------------")
	fmt.Println(content)
}
