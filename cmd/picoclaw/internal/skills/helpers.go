package skills

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const skillsSearchMaxResults = 20

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

func skillsInstallCmd(installer *skills.SkillInstaller, repo string) error {
	fmt.Printf("Installing skill from %s...\n", repo)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := installer.InstallFromGitHub(ctx, repo); err != nil {
		return fmt.Errorf("failed to install skill: %w", err)
	}

	fmt.Printf("✓ Skill '%s' installed successfully!\n", filepath.Base(repo))

	return nil
}

// skillsInstallFromGitCmd installs a skill from a Git repository URL.
func skillsInstallFromGitCmd(installer *skills.SkillInstaller, gitURL string, force bool) error {
	fmt.Printf("Cloning repository: %s...\n", gitURL)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	tempDir, discoveredSkills, err := installer.CloneAndDiscoverSkills(ctx, gitURL)
	if err != nil {
		return fmt.Errorf("failed to clone and discover skills: %w", err)
	}

	if len(discoveredSkills) == 0 {
		_ = os.RemoveAll(tempDir)
		return fmt.Errorf("no skills found in repository")
	}

	fmt.Printf("Found %d skill(s)\n\n", len(discoveredSkills))

	// Interactive selection.
	selectedSkills, err := interactiveSkillSelect(discoveredSkills)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return err
	}

	if len(selectedSkills) == 0 {
		_ = os.RemoveAll(tempDir)
		fmt.Println("No skills selected. Installation cancelled.")
		return nil
	}

	// Install selected skills.
	fmt.Printf("\nInstalling %d skill(s)...\n", len(selectedSkills))
	installed, err := installer.InstallSelectedSkills(tempDir, selectedSkills, force)
	if err != nil {
		return err
	}

	fmt.Println()
	for _, name := range installed {
		fmt.Printf("✓ Skill '%s' installed successfully!\n", name)
	}

	return nil
}

// interactiveSkillSelect provides an interactive terminal UI for selecting skills.
// Press Space to toggle selection, 'a' to toggle all, Enter to confirm.
func interactiveSkillSelect(discovered []skills.DiscoveredSkill) ([]skills.DiscoveredSkill, error) {
	if len(discovered) == 0 {
		return nil, nil
	}

	// If only one skill, ask for simple confirmation.
	if len(discovered) == 1 {
		fmt.Print("Install this skill? [Y/n]: ")
		var input string
		fmt.Scanln(&input)
		input = strings.TrimSpace(strings.ToLower(input))
		if input == "" || input == "y" || input == "yes" {
			return discovered, nil
		}
		return nil, nil
	}

	selected := make([]bool, len(discovered))
	cursor := 0

	// Save terminal state and enable raw mode.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		// Fallback to simple input mode.
		return fallbackSkillSelect(discovered)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Hide cursor.
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h")

	// ANSI color codes.
	const (
		colorReset  = "\033[0m"
		colorOrange = "\033[38;5;208m" // Mantis shrimp orange for selected items.
		colorCyan   = "\033[36m"       // Cyan for cursor highlight.
		colorDim    = "\033[2m"        // Dim for description.
	)

	// Total lines: header(1) + list(len).
	totalLines := len(discovered) + 2

	// Render function.
	render := func(initial bool) {
		if !initial {
			// Move cursor up to beginning.
			fmt.Printf("\033[%dA\r", totalLines)
		}

		// Header with instructions.
		fmt.Print("\033[2K") // Clear line.
		fmt.Print("Select skills to install:\r\n")
		fmt.Print("\033[2K") // Clear line.
		fmt.Printf("%s↑/↓ move  space toggle  a toggle all  enter confirm  q/esc quit%s\r\n", colorDim, colorReset)

		// List items.
		for i, skill := range discovered {
			fmt.Print("\033[2K") // Clear line.

			checkbox := "◻"
			if selected[i] {
				checkbox = "◼"
			}

			// Build the line content.
			var line string
			if i == cursor {
				// Current cursor - show description after name.
				desc := skill.Description
				if desc == "" {
					desc = "no description"
				}
				line = fmt.Sprintf("│  %s%s %s%s %s(%s)%s", colorCyan, checkbox, skill.Name, colorReset, colorDim, desc, colorReset)
			} else if selected[i] {
				// Selected item - orange color.
				line = fmt.Sprintf("│  %s%s %s%s", colorOrange, checkbox, skill.Name, colorReset)
			} else {
				// Normal item.
				line = fmt.Sprintf("│  %s %s", checkbox, skill.Name)
			}
			fmt.Print(line + "\r\n")
		}
	}

	// Initial render.
	render(true)

	buf := make([]byte, 3)
	for {
		// Read first byte.
		n, err := os.Stdin.Read(buf[:1])
		if err != nil || n == 0 {
			break
		}

		switch buf[0] {
		case ' ': // Space - toggle current.
			selected[cursor] = !selected[cursor]
			render(false)

		case 'a', 'A': // Toggle all.
			// Check if all are selected.
			allSelected := true
			for _, s := range selected {
				if !s {
					allSelected = false
					break
				}
			}
			// Toggle.
			for i := range selected {
				selected[i] = !allSelected
			}
			render(false)

		case 13, 10: // Enter - confirm.
			fmt.Print("\r\n")
			var result []skills.DiscoveredSkill
			for i, sel := range selected {
				if sel {
					result = append(result, discovered[i])
				}
			}
			return result, nil

		case 'q', 'Q': // q - cancel.
			fmt.Print("\r\n")
			return nil, nil

		case 27: // Escape - could be standalone or start of arrow key sequence.
			// Read remaining bytes of escape sequence.
			os.Stdin.Read(buf[1:3])
			if buf[1] == '[' {
				// Arrow keys.
				switch buf[2] {
				case 'A': // Up.
					if cursor > 0 {
						cursor--
						render(false)
					}
				case 'B': // Down.
					if cursor < len(discovered)-1 {
						cursor++
						render(false)
					}
				}
			} else {
				// Standalone Escape - quit.
				fmt.Print("\r\n")
				return nil, nil
			}

		case 'j': // vim-style down.
			if cursor < len(discovered)-1 {
				cursor++
				render(false)
			}

		case 'k': // vim-style up.
			if cursor > 0 {
				cursor--
				render(false)
			}
		}
	}

	return nil, nil
}

// fallbackSkillSelect provides a simple text-based fallback for skill selection.
func fallbackSkillSelect(discovered []skills.DiscoveredSkill) ([]skills.DiscoveredSkill, error) {
	fmt.Println("\nEnter skill numbers to install (comma-separated), 'a' for all, or 'q' to cancel:")
	fmt.Print("> ")

	var input string
	fmt.Scanln(&input)
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" || input == "q" {
		return nil, nil
	}

	if input == "a" || input == "all" {
		return discovered, nil
	}

	var result []skills.DiscoveredSkill
	parts := strings.Split(input, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		var idx int
		if _, err := fmt.Sscanf(part, "%d", &idx); err == nil {
			if idx >= 1 && idx <= len(discovered) {
				result = append(result, discovered[idx-1])
			}
		}
	}

	return result, nil
}

// skillsInstallFromRegistry installs a skill from a named registry (e.g. clawhub).
func skillsInstallFromRegistry(cfg *config.Config, registryName, slug string) error {
	err := utils.ValidateSkillIdentifier(registryName)
	if err != nil {
		return fmt.Errorf("✗  invalid registry name: %w", err)
	}

	err = utils.ValidateSkillIdentifier(slug)
	if err != nil {
		return fmt.Errorf("✗  invalid slug: %w", err)
	}

	fmt.Printf("Installing skill '%s' from %s registry...\n", slug, registryName)

	registryMgr := skills.NewRegistryManagerFromConfig(skills.RegistryConfig{
		MaxConcurrentSearches: cfg.Tools.Skills.MaxConcurrentSearches,
		ClawHub:               skills.ClawHubConfig(cfg.Tools.Skills.Registries.ClawHub),
	})

	registry := registryMgr.GetRegistry(registryName)
	if registry == nil {
		return fmt.Errorf("✗  registry '%s' not found or not enabled. check your config.json.", registryName)
	}

	workspace := cfg.WorkspacePath()
	targetDir := filepath.Join(workspace, "skills", slug)

	if _, err = os.Stat(targetDir); err == nil {
		return fmt.Errorf("\u2717 skill '%s' already installed at %s", slug, targetDir)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err = os.MkdirAll(filepath.Join(workspace, "skills"), 0o755); err != nil {
		return fmt.Errorf("\u2717 failed to create skills directory: %v", err)
	}

	result, err := registry.DownloadAndInstall(ctx, slug, "", targetDir)
	if err != nil {
		rmErr := os.RemoveAll(targetDir)
		if rmErr != nil {
			fmt.Printf("\u2717 Failed to remove partial install: %v\n", rmErr)
		}
		return fmt.Errorf("✗ failed to install skill: %w", err)
	}

	if result.IsMalwareBlocked {
		rmErr := os.RemoveAll(targetDir)
		if rmErr != nil {
			fmt.Printf("\u2717 Failed to remove partial install: %v\n", rmErr)
		}

		return fmt.Errorf("\u2717 Skill '%s' is flagged as malicious and cannot be installed.\n", slug)
	}

	if result.IsSuspicious {
		fmt.Printf("\u26a0\ufe0f  Warning: skill '%s' is flagged as suspicious.\n", slug)
	}

	fmt.Printf("\u2713 Skill '%s' v%s installed successfully!\n", slug, result.Version)
	if result.Summary != "" {
		fmt.Printf("  %s\n", result.Summary)
	}

	return nil
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
	cfg, err := internal.LoadConfig()
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

func skillsSearchCmd(query string) {
	fmt.Println("Searching for available skills...")

	cfg, err := internal.LoadConfig()
	if err != nil {
		fmt.Printf("✗ Failed to load config: %v\n", err)
		return
	}

	registryMgr := skills.NewRegistryManagerFromConfig(skills.RegistryConfig{
		MaxConcurrentSearches: cfg.Tools.Skills.MaxConcurrentSearches,
		ClawHub:               skills.ClawHubConfig(cfg.Tools.Skills.Registries.ClawHub),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := registryMgr.SearchAll(ctx, query, skillsSearchMaxResults)
	if err != nil {
		fmt.Printf("✗ Failed to fetch skills list: %v\n", err)
		return
	}

	if len(results) == 0 {
		fmt.Println("No skills available.")
		return
	}

	fmt.Printf("\nAvailable Skills (%d):\n", len(results))
	fmt.Println("--------------------")
	for _, result := range results {
		fmt.Printf("  📦 %s\n", result.DisplayName)
		fmt.Printf("     %s\n", result.Summary)
		fmt.Printf("     Slug: %s\n", result.Slug)
		fmt.Printf("     Registry: %s\n", result.RegistryName)
		if result.Version != "" {
			fmt.Printf("     Version: %s\n", result.Version)
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

func copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}
