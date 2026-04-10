package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/fileutil"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const skillsSearchMaxResults = 20

type installedSkillOriginMeta struct {
	Version          int    `json:"version"`
	OriginKind       string `json:"origin_kind,omitempty"`
	Registry         string `json:"registry,omitempty"`
	Slug             string `json:"slug,omitempty"`
	RegistryURL      string `json:"registry_url,omitempty"`
	InstalledVersion string `json:"installed_version,omitempty"`
	InstalledAt      int64  `json:"installed_at"`
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

// skillsInstallFromRegistry installs a skill from a named registry (e.g. clawhub).
func skillsInstallFromRegistry(cfg *config.Config, registryName, target string) error {
	err := utils.ValidateSkillIdentifier(registryName)
	if err != nil {
		return fmt.Errorf("✗  invalid registry name: %w", err)
	}

	registryMgr := skills.NewRegistryManagerFromToolsConfig(cfg.Tools.Skills)

	registry := registryMgr.GetRegistry(registryName)
	if registry == nil {
		return fmt.Errorf("✗  registry '%s' not found or not enabled. check your config.json.", registryName)
	}

	dirName, err := registry.ResolveInstallDirName(target)
	if err != nil {
		return fmt.Errorf("✗  invalid install target %q: %w", target, err)
	}

	fmt.Printf("Installing skill '%s' from %s registry...\n", target, registryName)

	workspace := cfg.WorkspacePath()
	targetDir := filepath.Join(workspace, "skills", dirName)

	if _, err = os.Stat(targetDir); err == nil {
		return fmt.Errorf("\u2717 skill '%s' already installed at %s", dirName, targetDir)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err = os.MkdirAll(filepath.Join(workspace, "skills"), 0o755); err != nil {
		return fmt.Errorf("\u2717 failed to create skills directory: %v", err)
	}

	result, err := registry.DownloadAndInstall(ctx, target, "", targetDir)
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

		return fmt.Errorf("\u2717 Skill '%s' is flagged as malicious and cannot be installed.\n", target)
	}

	if result.IsSuspicious {
		fmt.Printf("\u26a0\ufe0f  Warning: skill '%s' is flagged as suspicious.\n", target)
	}

	normalizedSlug := skills.NormalizeInstallTargetForRegistry(cfg.Tools.Skills, registry.Name(), target)
	installedAt := time.Now().UnixMilli()
	if err := writeInstalledSkillOriginMeta(targetDir, installedSkillOriginMeta{
		Version:          1,
		OriginKind:       "third_party",
		Registry:         registry.Name(),
		Slug:             normalizedSlug,
		RegistryURL:      registry.SkillURL(normalizedSlug, result.Version),
		InstalledVersion: result.Version,
		InstalledAt:      installedAt,
	}); err != nil {
		_ = os.RemoveAll(targetDir)
		return fmt.Errorf("✗ failed to persist skill metadata: %w", err)
	}

	fmt.Printf("\u2713 Skill '%s' v%s installed successfully!\n", dirName, result.Version)
	if result.Summary != "" {
		fmt.Printf("  %s\n", result.Summary)
	}

	return nil
}

func writeInstalledSkillOriginMeta(targetDir string, meta installedSkillOriginMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(filepath.Join(targetDir, ".skill-origin.json"), data, 0o600)
}

func skillsRemoveFromWorkspace(workspace, skillName string) error {
	name := strings.TrimSpace(skillName)
	name = strings.Trim(name, "/")
	if name == "" {
		return fmt.Errorf("skill name is required")
	}
	if strings.Contains(name, "/") {
		parts := strings.Split(name, "/")
		name = parts[len(parts)-1]
	}
	skillDir := filepath.Join(workspace, "skills", name)
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return fmt.Errorf("skill '%s' not found", name)
	}
	if err := os.RemoveAll(skillDir); err != nil {
		return fmt.Errorf("failed to remove skill '%s': %w", name, err)
	}
	return nil
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

	registryMgr := skills.NewRegistryManagerFromToolsConfig(cfg.Tools.Skills)

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
