package integrationtools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/fileutil"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const defaultSkillRegistryName = "github"

var persistInstalledSkillOriginMeta = writeOriginMeta

// InstallSkillTool allows the LLM agent to install skills from registries.
// It shares the same RegistryManager that FindSkillsTool uses,
// so all registries configured in config are available for installation.
type InstallSkillTool struct {
	registryMgr *skills.RegistryManager
	workspace   string
	mu          sync.Mutex
}

// NewInstallSkillTool creates a new InstallSkillTool.
// registryMgr is the shared registry manager (same instance as FindSkillsTool).
// workspace is the root workspace directory; skills install to {workspace}/skills/{slug}/.
func NewInstallSkillTool(registryMgr *skills.RegistryManager, workspace string) *InstallSkillTool {
	return &InstallSkillTool{
		registryMgr: registryMgr,
		workspace:   workspace,
		mu:          sync.Mutex{},
	}
}

func (t *InstallSkillTool) Name() string {
	return "install_skill"
}

func (t *InstallSkillTool) Description() string {
	return "Install a skill from a registry by slug. Defaults to GitHub when registry is omitted. Downloads and extracts the skill into the workspace. Use find_skills first to discover available skills."
}

func (t *InstallSkillTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"slug": map[string]any{
				"type":        "string",
				"description": "The unique slug of the skill to install (e.g., 'github', 'docker-compose')",
			},
			"version": map[string]any{
				"type":        "string",
				"description": "Specific version to install (optional, defaults to latest)",
			},
			"registry": map[string]any{
				"type":        "string",
				"description": "Registry to install from (optional, defaults to 'github')",
			},
			"force": map[string]any{
				"type":        "boolean",
				"description": "Force reinstall if skill already exists (default false)",
			},
		},
		"required": []string{"slug"},
	}
}

func (t *InstallSkillTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	// Install lock to prevent concurrent directory operations.
	// Ideally this should be done at a `slug` level, currently, its at a `workspace` level.
	t.mu.Lock()
	defer t.mu.Unlock()

	slug, _ := args["slug"].(string)
	if strings.TrimSpace(slug) == "" {
		return ErrorResult("identifier is required and must be a non-empty string")
	}

	// Validate registry
	registryName, _ := args["registry"].(string)
	if registryName == "" {
		registryName = defaultSkillRegistryName
	}
	if err := utils.ValidateSkillIdentifier(registryName); err != nil {
		return ErrorResult(fmt.Sprintf("invalid registry %q: error: %s", registryName, err.Error()))
	}

	// Resolve which registry to use.
	registry := t.registryMgr.GetRegistry(registryName)
	if registry == nil {
		return ErrorResult(fmt.Sprintf("registry %q not found", registryName))
	}

	// Validate target and resolve install directory.
	dirName, err := registry.ResolveInstallDirName(slug)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid slug %q: error: %s", slug, err.Error()))
	}

	version, _ := args["version"].(string)
	force, _ := args["force"].(bool)

	// Check if already installed.
	skillsDir := filepath.Join(t.workspace, "skills")
	targetDir := filepath.Join(skillsDir, dirName)
	backupDir := ""
	restorePreviousInstall := func() {
		if backupDir == "" {
			return
		}
		if rmErr := os.RemoveAll(targetDir); rmErr != nil {
			logger.ErrorCF("tool", "Failed to remove failed install before restore",
				map[string]any{
					"tool":       "install_skill",
					"target_dir": targetDir,
					"error":      rmErr.Error(),
				})
			return
		}
		if restoreErr := os.Rename(backupDir, targetDir); restoreErr != nil {
			logger.ErrorCF("tool", "Failed to restore previous install after failed reinstall",
				map[string]any{
					"tool":       "install_skill",
					"backup_dir": backupDir,
					"target_dir": targetDir,
					"error":      restoreErr.Error(),
				})
			return
		}
		backupDir = ""
	}

	if !force {
		if _, statErr := os.Stat(targetDir); statErr == nil {
			return ErrorResult(
				fmt.Sprintf("skill %q already installed at %s. Use force=true to reinstall.", slug, targetDir),
			)
		}
	} else {
		if _, statErr := os.Stat(targetDir); statErr == nil {
			backupDir = filepath.Join(skillsDir, fmt.Sprintf(".%s.picoclaw-backup-%d", dirName, time.Now().UnixNano()))
			if renameErr := os.Rename(targetDir, backupDir); renameErr != nil {
				return ErrorResult(fmt.Sprintf("failed to prepare reinstall for %q: %v", slug, renameErr))
			}
		} else if !os.IsNotExist(statErr) {
			return ErrorResult(fmt.Sprintf("failed to inspect existing install for %q: %v", slug, statErr))
		}
	}

	// Ensure skills directory exists.
	if mkdirErr := os.MkdirAll(skillsDir, 0o755); mkdirErr != nil {
		restorePreviousInstall()
		return ErrorResult(fmt.Sprintf("failed to create skills directory: %v", mkdirErr))
	}

	// Download and install (handles metadata, version resolution, extraction).
	result, err := registry.DownloadAndInstall(ctx, slug, version, targetDir)
	if err != nil {
		// Clean up partial install.
		rmErr := os.RemoveAll(targetDir)
		if rmErr != nil {
			logger.ErrorCF("tool", "Failed to remove partial install",
				map[string]any{
					"tool":       "install_skill",
					"target_dir": targetDir,
					"error":      rmErr.Error(),
				})
		}
		restorePreviousInstall()
		return ErrorResult(fmt.Sprintf("failed to install %q: %v", slug, err))
	}

	// Moderation: block malware.
	if result.IsMalwareBlocked {
		rmErr := os.RemoveAll(targetDir)
		if rmErr != nil {
			logger.ErrorCF("tool", "Failed to remove partial install",
				map[string]any{
					"tool":       "install_skill",
					"target_dir": targetDir,
					"error":      rmErr.Error(),
				})
		}
		restorePreviousInstall()
		return ErrorResult(fmt.Sprintf("skill %q is flagged as malicious and cannot be installed", slug))
	}

	if !workspaceHasValidInstalledSkill(t.workspace, dirName) {
		rmErr := os.RemoveAll(targetDir)
		if rmErr != nil {
			logger.ErrorCF("tool", "Failed to remove invalid installed skill",
				map[string]any{
					"tool":       "install_skill",
					"target_dir": targetDir,
					"error":      rmErr.Error(),
				})
		}
		restorePreviousInstall()
		return ErrorResult(fmt.Sprintf("failed to install %q: registry archive is not a valid skill", slug))
	}

	// Write origin metadata.
	if err := persistInstalledSkillOriginMeta(targetDir, registry, slug, result.Version); err != nil {
		logger.ErrorCF("tool", "Failed to write origin metadata",
			map[string]any{
				"tool":     "install_skill",
				"error":    err.Error(),
				"target":   targetDir,
				"registry": registry.Name(),
				"slug":     slug,
				"version":  result.Version,
			})
		rmErr := os.RemoveAll(targetDir)
		if rmErr != nil {
			logger.ErrorCF("tool", "Failed to roll back install after metadata write failure",
				map[string]any{
					"tool":       "install_skill",
					"target_dir": targetDir,
					"error":      rmErr.Error(),
				})
		}
		restorePreviousInstall()
		return ErrorResult(fmt.Sprintf("failed to persist skill metadata for %q: %v", slug, err))
	}
	if backupDir != "" {
		if rmErr := os.RemoveAll(backupDir); rmErr != nil {
			logger.ErrorCF("tool", "Failed to remove previous install backup after successful reinstall",
				map[string]any{
					"tool":       "install_skill",
					"backup_dir": backupDir,
					"error":      rmErr.Error(),
				})
		}
	}

	// Build result with moderation warning if suspicious.
	var output string
	if result.IsSuspicious {
		output = fmt.Sprintf("⚠️ Warning: skill %q is flagged as suspicious (may contain risky patterns).\n\n", slug)
	}
	output += fmt.Sprintf("Successfully installed skill %q v%s from %s registry.\nLocation: %s\n",
		slug, result.Version, registry.Name(), targetDir)

	if result.Summary != "" {
		output += fmt.Sprintf("Description: %s\n", result.Summary)
	}
	output += "\nThe skill is now available and can be loaded in the current session."

	return SilentResult(output)
}

// originMeta tracks which registry a skill was installed from.
type originMeta struct {
	Version          int    `json:"version"`
	OriginKind       string `json:"origin_kind,omitempty"`
	Registry         string `json:"registry"`
	Slug             string `json:"slug"`
	RegistryURL      string `json:"registry_url,omitempty"`
	InstalledVersion string `json:"installed_version"`
	InstalledAt      int64  `json:"installed_at"`
}

func writeOriginMeta(targetDir string, registry skills.SkillRegistry, slug, version string) error {
	normalizedSlug, registryURL := skills.BuildInstallMetadataForRegistryInstance(registry, slug, version)
	registryName := ""
	if registry != nil {
		registryName = registry.Name()
	}

	meta := originMeta{
		Version:          1,
		OriginKind:       "third_party",
		Registry:         registryName,
		Slug:             normalizedSlug,
		RegistryURL:      registryURL,
		InstalledVersion: version,
		InstalledAt:      time.Now().UnixMilli(),
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	// Use unified atomic write utility with explicit sync for flash storage reliability.
	return fileutil.WriteFileAtomic(filepath.Join(targetDir, ".skill-origin.json"), data, 0o600)
}

func workspaceHasValidInstalledSkill(workspace, directory string) bool {
	loader := skills.NewSkillsLoader(workspace, "", "")
	for _, skill := range loader.ListSkills() {
		if skill.Source != "workspace" {
			continue
		}
		if filepath.Base(filepath.Dir(skill.Path)) == directory {
			return true
		}
	}
	return false
}
