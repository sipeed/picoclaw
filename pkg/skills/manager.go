package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// SkillManager handles dynamic creation, patching, and discovery of skills.
// It writes SKILL.md files to the workspace skills directory using atomic
// writes (temp file + rename) to prevent partial files.
//
// Inspired by Hermes Agent's tools/skill_manager_tool.py — ported to Go
// for PicoClaw's self-improvement system.
type SkillManager struct {
	mu        sync.Mutex
	skillsDir string // e.g. ~/.picoclaw/workspace/skills/
}

// NewSkillManager creates a manager that writes skills to the given directory.
func NewSkillManager(skillsDir string) *SkillManager {
	return &SkillManager{skillsDir: skillsDir}
}

// CreateSkill validates and atomically writes a new skill.
// category is optional — if provided, creates a subdirectory.
func (sm *SkillManager) CreateSkill(name, content, category string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Validate inputs.
	if err := ValidateName(name); err != nil {
		return fmt.Errorf("validate name: %w", err)
	}
	if err := ValidateFrontmatter(content); err != nil {
		return fmt.Errorf("validate frontmatter: %w", err)
	}
	if err := ValidateSize(content); err != nil {
		return fmt.Errorf("validate size: %w", err)
	}

	// Check for duplicates.
	if _, exists := sm.findSkillLocked(name); exists {
		return fmt.Errorf("skill %q already exists", name)
	}

	// Build target path.
	dir := sm.skillsDir
	if category != "" {
		if err := validateCategory(category); err != nil {
			return fmt.Errorf("validate category: %w", err)
		}
		dir = filepath.Join(dir, category)
	}
	skillDir := filepath.Join(dir, name)
	skillFile := filepath.Join(skillDir, "SKILL.md")

	// Create directory.
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return fmt.Errorf("create skill directory: %w", err)
	}

	// Atomic write: temp file + rename.
	if err := atomicWrite(skillFile, content); err != nil {
		// Cleanup empty directory on failure.
		os.Remove(skillDir)
		return fmt.Errorf("write skill: %w", err)
	}

	logger.DebugCF("skills", "skill created", map[string]any{
		"name":     name,
		"category": category,
		"path":     skillFile,
		"size":     len(content),
	})

	return nil
}

// PatchSkill applies a find-and-replace to an existing skill's SKILL.md.
func (sm *SkillManager) PatchSkill(name, oldStr, newStr string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	info, exists := sm.findSkillLocked(name)
	if !exists {
		return fmt.Errorf("skill %q not found", name)
	}

	data, err := os.ReadFile(info.Path)
	if err != nil {
		return fmt.Errorf("read skill: %w", err)
	}

	content := string(data)
	if !strings.Contains(content, oldStr) {
		return fmt.Errorf("old_string not found in %s", info.Path)
	}

	updated := strings.Replace(content, oldStr, newStr, 1)

	// Validate updated content.
	if err := ValidateFrontmatter(updated); err != nil {
		return fmt.Errorf("patch breaks frontmatter: %w", err)
	}

	if err := atomicWrite(info.Path, updated); err != nil {
		return fmt.Errorf("write patched skill: %w", err)
	}

	logger.DebugCF("skills", "skill patched", map[string]any{
		"name": name,
		"path": info.Path,
	})

	return nil
}

// EditSkill replaces the entire content of a skill's SKILL.md.
func (sm *SkillManager) EditSkill(name, content string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	info, exists := sm.findSkillLocked(name)
	if !exists {
		return fmt.Errorf("skill %q not found", name)
	}

	if err := ValidateFrontmatter(content); err != nil {
		return fmt.Errorf("validate frontmatter: %w", err)
	}
	if err := ValidateSize(content); err != nil {
		return fmt.Errorf("validate size: %w", err)
	}

	if err := atomicWrite(info.Path, content); err != nil {
		return fmt.Errorf("write skill: %w", err)
	}

	return nil
}

// DeleteSkill removes a skill directory and all its contents.
func (sm *SkillManager) DeleteSkill(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	info, exists := sm.findSkillLocked(name)
	if !exists {
		return fmt.Errorf("skill %q not found", name)
	}

	skillDir := filepath.Dir(info.Path)
	if err := os.RemoveAll(skillDir); err != nil {
		return fmt.Errorf("remove skill directory: %w", err)
	}

	logger.DebugCF("skills", "skill deleted", map[string]any{
		"name": name,
		"path": skillDir,
	})

	return nil
}

// FindSkill looks up a skill by name in the skills directory.
func (sm *SkillManager) FindSkill(name string) (*SkillInfo, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.findSkillLocked(name)
}

// ListSkills returns all skills in the managed directory.
func (sm *SkillManager) ListSkills() []SkillInfo {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var skills []SkillInfo
	filepath.WalkDir(sm.skillsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "SKILL.md" {
			return nil
		}
		info, err := loadSkillInfo(path)
		if err != nil {
			return nil
		}
		skills = append(skills, *info)
		return nil
	})
	return skills
}

// findSkillLocked searches for a skill by name. Caller must hold sm.mu.
func (sm *SkillManager) findSkillLocked(name string) (*SkillInfo, bool) {
	var found *SkillInfo
	filepath.WalkDir(sm.skillsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "SKILL.md" {
			return nil
		}
		if filepath.Base(filepath.Dir(path)) == name {
			info, err := loadSkillInfo(path)
			if err == nil {
				found = info
				return filepath.SkipAll
			}
		}
		return nil
	})
	if found != nil {
		return found, true
	}
	return nil, false
}

// loadSkillInfo reads a SKILL.md and extracts its metadata.
func loadSkillInfo(path string) (*SkillInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)

	// Extract name from directory.
	name := filepath.Base(filepath.Dir(path))

	// Extract description from frontmatter.
	desc := ""
	if strings.HasPrefix(content, "---\n") {
		end := strings.Index(content[4:], "\n---")
		if end > 0 {
			// Simple extraction — look for description field.
			for _, line := range strings.Split(content[4:4+end], "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "description:") {
					desc = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "description:"))
					desc = strings.Trim(desc, "\"'>")
					break
				}
			}
		}
	}

	return &SkillInfo{
		Name:        name,
		Path:        path,
		Source:      "workspace",
		Description: desc,
	}, nil
}

// atomicWrite writes content to path via temp file + rename.
func atomicWrite(path, content string) error {
	tmp := path + ".tmp." + fmt.Sprint(os.Getpid())
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// validateCategory checks that a category name is safe.
func validateCategory(cat string) error {
	if strings.Contains(cat, "..") || strings.Contains(cat, "/") || strings.Contains(cat, "\\") {
		return fmt.Errorf("invalid category %q: must not contain path separators or ..", cat)
	}
	return nil
}
