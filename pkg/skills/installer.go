package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/security"
)

// repoNamePattern validates GitHub repository format: "owner/repo"
var repoNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`)

type SkillInstaller struct {
	workspace    string
	policyEngine *security.PolicyEngine
	skillMode    security.PolicyMode
}

type AvailableSkill struct {
	Name        string   `json:"name"`
	Repository  string   `json:"repository"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
}

type BuiltinSkill struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Enabled bool   `json:"enabled"`
}

func NewSkillInstaller(workspace string) *SkillInstaller {
	return &SkillInstaller{
		workspace: workspace,
	}
}

// NewSkillInstallerWithPolicy creates a SkillInstaller with security policy support.
func NewSkillInstallerWithPolicy(workspace string, pe *security.PolicyEngine, mode security.PolicyMode) *SkillInstaller {
	return &SkillInstaller{
		workspace:    workspace,
		policyEngine: pe,
		skillMode:    mode,
	}
}

func (si *SkillInstaller) InstallFromGitHub(ctx context.Context, repo string) error {
	// Validate repo format to prevent URL injection (mode-aware)
	if !si.skillMode.IsOff() {
		if !repoNamePattern.MatchString(repo) {
			reason := fmt.Sprintf("invalid repository format: must be 'owner/repo' (got %q)", repo)
			if si.policyEngine != nil {
				err := si.policyEngine.Evaluate(ctx, si.skillMode, security.Violation{
					Category: "skill_validation",
					Tool:     "skill_install",
					Action:   repo,
					Reason:   reason,
				}, "", "")
				if err != nil {
					return err
				}
				// approved by user, continue
			} else {
				return fmt.Errorf("%s", reason)
			}
		}
	}

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

	resp, err := client.Do(req)
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

	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, body, 0644); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	return nil
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

func (si *SkillInstaller) ListAvailableSkills(ctx context.Context) ([]AvailableSkill, error) {
	url := "https://raw.githubusercontent.com/sipeed/picoclaw-skills/main/skills.json"

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch skills list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch skills list: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var skills []AvailableSkill
	if err := json.Unmarshal(body, &skills); err != nil {
		return nil, fmt.Errorf("failed to parse skills list: %w", err)
	}

	return skills, nil
}

func (si *SkillInstaller) ListBuiltinSkills() []BuiltinSkill {
	builtinSkillsDir := filepath.Join(filepath.Dir(si.workspace), "picoclaw", "skills")

	entries, err := os.ReadDir(builtinSkillsDir)
	if err != nil {
		return nil
	}

	var skills []BuiltinSkill
	for _, entry := range entries {
		if entry.IsDir() {
			_ = entry
			skillName := entry.Name()
			skillFile := filepath.Join(builtinSkillsDir, skillName, "SKILL.md")

			data, err := os.ReadFile(skillFile)
			description := ""
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

			// skill := BuiltinSkill{
			// 	Name:    skillName,
			// 	Path:    description,
			// 	Enabled: true,
			// }

			status := "✓"
			fmt.Printf("  %s  %s\n", status, entry.Name())
			if description != "" {
				fmt.Printf("    %s\n", description)
			}
		}
	}
	return skills
}
