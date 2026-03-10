package skills

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/fileutil"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type SkillInstaller struct {
	workspace string
}

func NewSkillInstaller(workspace string) *SkillInstaller {
	return &SkillInstaller{
		workspace: workspace,
	}
}

func (si *SkillInstaller) InstallFromGitHub(ctx context.Context, repo string) error {
	skillDir := filepath.Join(si.workspace, "skills", filepath.Base(repo))

	if _, err := os.Stat(skillDir); err == nil {
		return fmt.Errorf("skill '%s' already exists", filepath.Base(repo))
	}

	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/main/SKILL.md", repo)

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := utils.DoRequestWithRetry(client, req)
	if err != nil {
		return fmt.Errorf("failed to fetch skill: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to fetch skill: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")

	// Use unified atomic write utility with explicit sync for flash storage reliability.
	if err := fileutil.WriteFileAtomic(skillPath, body, 0o600); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	return nil
}

// InstallFromGit clones a Git repository to install a skill.
// Supports SSH URLs (git@host:user/repo.git) and HTTPS URLs (https://host/user/repo.git).
func (si *SkillInstaller) InstallFromGit(ctx context.Context, gitURL string) error {
	skillName := extractSkillNameFromGitURL(gitURL)
	if skillName == "" {
		return fmt.Errorf("failed to extract skill name from git URL: %s", gitURL)
	}

	skillDir := filepath.Join(si.workspace, "skills", skillName)

	if _, err := os.Stat(skillDir); err == nil {
		return fmt.Errorf("skill '%s' already exists", skillName)
	}

	// Ensure parent directory exists.
	skillsDir := filepath.Join(si.workspace, "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	// Execute git clone.
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", gitURL, skillDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w\nOutput: %s", err, string(output))
	}

	// Verify SKILL.md exists.
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		// Clean up if SKILL.md doesn't exist.
		_ = os.RemoveAll(skillDir)
		return fmt.Errorf("repository does not contain SKILL.md file")
	}

	// Remove .git directory to save space.
	gitDir := filepath.Join(skillDir, ".git")
	_ = os.RemoveAll(gitDir)

	return nil
}

// DiscoveredSkill represents a skill found in a Git repository.
type DiscoveredSkill struct {
	Name        string // Skill name from SKILL.md (or directory name as fallback)
	DirName     string // Directory name (used for installation path)
	Path        string // Full path in the cloned repository
	Description string // Description from SKILL.md (if available)
}

// CloneAndDiscoverSkills clones a Git repository and discovers all skills in it.
// Skills are expected to be in subdirectories of the repository root, each containing a SKILL.md file.
// Returns the temporary directory path and the list of discovered skills.
func (si *SkillInstaller) CloneAndDiscoverSkills(ctx context.Context, gitURL string) (string, []DiscoveredSkill, error) {
	// Create temporary directory for cloning.
	tempDir, err := os.MkdirTemp("", "picoclaw-skills-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Clone the repository.
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", gitURL, tempDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return "", nil, fmt.Errorf("failed to clone repository: %w\nOutput: %s", err, string(output))
	}

	// Discover skills in the repository.
	skills, err := si.discoverSkillsInDir(tempDir)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return "", nil, err
	}

	if len(skills) == 0 {
		_ = os.RemoveAll(tempDir)
		return "", nil, fmt.Errorf("no skills found in repository (no subdirectories with SKILL.md)")
	}

	return tempDir, skills, nil
}

// discoverSkillsInDir scans a directory for skills (subdirectories containing SKILL.md).
func (si *SkillInstaller) discoverSkillsInDir(dir string) ([]DiscoveredSkill, error) {
	var skills []DiscoveredSkill

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip hidden directories.
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		skillDir := filepath.Join(dir, entry.Name())
		skillFile := filepath.Join(skillDir, "SKILL.md")

		if _, err := os.Stat(skillFile); err == nil {
			skill := DiscoveredSkill{
				Name:    entry.Name(), // Default to directory name.
				DirName: entry.Name(),
				Path:    skillDir,
			}

			// Try to extract metadata from SKILL.md.
			if content, err := os.ReadFile(skillFile); err == nil {
				contentStr := string(content)
				metadata := extractSkillMetadata(contentStr)
				if metadata.Name != "" {
					skill.Name = metadata.Name
				}
				if metadata.Description != "" {
					skill.Description = metadata.Description
				} else {
					skill.Description = extractFirstLine(contentStr)
				}
			}

			skills = append(skills, skill)
		}
	}

	return skills, nil
}

// skillMetadata holds parsed metadata from SKILL.md frontmatter.
type skillMetadata struct {
	Name        string
	Description string
}

// extractSkillMetadata parses the frontmatter of a SKILL.md file.
func extractSkillMetadata(content string) skillMetadata {
	var meta skillMetadata

	// Normalize line endings.
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	// Check for YAML frontmatter (--- delimited).
	if !strings.HasPrefix(normalized, "---") {
		return meta
	}

	// Find end of frontmatter.
	endIdx := strings.Index(normalized[3:], "\n---")
	if endIdx == -1 {
		return meta
	}

	frontmatter := normalized[3 : 3+endIdx]

	// Parse simple YAML key: value format.
	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)

		switch key {
		case "name":
			meta.Name = value
		case "description":
			meta.Description = value
		}
	}

	return meta
}

// extractFirstLine extracts the description from SKILL.md content.
func extractFirstLine(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for "description:" field.
		if strings.HasPrefix(strings.ToLower(line), "description:") {
			desc := strings.TrimSpace(line[len("description:"):])
			// Remove surrounding quotes if present.
			desc = strings.Trim(desc, `"'`)
			if len(desc) > 100 {
				return desc[:100] + "..."
			}
			return desc
		}
	}
	return ""
}

// InstallSelectedSkills installs the selected skills from a cloned repository.
// It copies the skill directories to the workspace and cleans up the temp directory.
// If force is true, existing skills will be overwritten.
func (si *SkillInstaller) InstallSelectedSkills(tempDir string, skills []DiscoveredSkill, force bool) ([]string, error) {
	skillsDir := filepath.Join(si.workspace, "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create skills directory: %w", err)
	}

	var installed []string
	var updated []string
	var errors []string

	for _, skill := range skills {
		// Use DirName for target directory (preserves original folder structure).
		targetDir := filepath.Join(skillsDir, skill.DirName)

		// Check if skill already exists.
		exists := false
		if _, err := os.Stat(targetDir); err == nil {
			if !force {
				errors = append(errors, fmt.Sprintf("'%s' already exists, use --force to overwrite", skill.Name))
				continue
			}
			// Remove existing directory for force update.
			if err := os.RemoveAll(targetDir); err != nil {
				errors = append(errors, fmt.Sprintf("'%s' failed to remove existing: %v", skill.Name, err))
				continue
			}
			exists = true
		}

		// Copy skill directory.
		if err := copyDir(skill.Path, targetDir); err != nil {
			errors = append(errors, fmt.Sprintf("'%s' failed: %v", skill.Name, err))
			continue
		}

		if exists {
			updated = append(updated, skill.Name)
		} else {
			installed = append(installed, skill.Name)
		}
	}

	// Clean up temp directory.
	_ = os.RemoveAll(tempDir)

	if len(errors) > 0 && len(installed) == 0 && len(updated) == 0 {
		return nil, fmt.Errorf("failed to install any skills:\n%s", strings.Join(errors, "\n"))
	}

	// Combine installed and updated for the return value.
	result := make([]string, 0, len(installed)+len(updated))
	for _, name := range installed {
		result = append(result, name+" (new)")
	}
	for _, name := range updated {
		result = append(result, name+" (updated)")
	}

	return result, nil
}

// copyDir recursively copies a directory.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory.
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
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

// IsGitURL checks if the given string is a Git URL.
// Supports SSH format (git@host:user/repo.git) and HTTPS/HTTP format.
func IsGitURL(url string) bool {
	// SSH format: git@host:user/repo.git
	sshPattern := regexp.MustCompile(`^git@[^:]+:.+\.git$`)
	if sshPattern.MatchString(url) {
		return true
	}

	// SSH URL format: ssh://git@host/user/repo.git
	if strings.HasPrefix(url, "ssh://") && strings.HasSuffix(url, ".git") {
		return true
	}

	// HTTPS/HTTP format: https://host/user/repo.git
	if (strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://")) && strings.HasSuffix(url, ".git") {
		return true
	}

	return false
}

// extractSkillNameFromGitURL extracts the repository name from a Git URL.
func extractSkillNameFromGitURL(gitURL string) string {
	// Remove .git suffix.
	url := strings.TrimSuffix(gitURL, ".git")

	// SSH format: git@host:user/repo -> repo
	if strings.HasPrefix(url, "git@") {
		parts := strings.Split(url, ":")
		if len(parts) == 2 {
			pathParts := strings.Split(parts[1], "/")
			return pathParts[len(pathParts)-1]
		}
	}

	// HTTPS/HTTP/SSH URL format: extract last path segment.
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return ""
}

func (si *SkillInstaller) Uninstall(skillName string) error {
	skillDir := filepath.Join(si.workspace, "skills", skillName)

	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return fmt.Errorf("skill '%s' not found", skillName)
	}

	if err := os.RemoveAll(skillDir); err != nil {
		return fmt.Errorf("failed to remove skill: %w", err)
	}

	return nil
}
