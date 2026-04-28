package evolution

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sipeed/picoclaw/pkg/skills"
)

func NextLifecycleState(profile SkillProfile, now time.Time) SkillStatus {
	if profile.Origin == "manual" || profile.LastUsedAt.IsZero() {
		return profile.Status
	}

	idle := now.Sub(profile.LastUsedAt)
	switch profile.Status {
	case SkillStatusActive:
		if idle > 90*24*time.Hour && profile.RetentionScore < 0.3 {
			return SkillStatusCold
		}
	case SkillStatusCold:
		if idle > 180*24*time.Hour && profile.RetentionScore < 0.2 {
			return SkillStatusArchived
		}
	case SkillStatusArchived:
		if idle > 365*24*time.Hour && profile.RetentionScore < 0.1 {
			return SkillStatusDeleted
		}
	}

	return profile.Status
}

func ApplyLifecycleState(paths Paths, profile SkillProfile, next SkillStatus) error {
	if next != SkillStatusDeleted {
		return nil
	}

	workspace := profile.WorkspaceID
	if workspace == "" {
		workspace = inferWorkspaceFromPaths(paths)
	}
	if workspace == "" {
		return fmt.Errorf("resolve lifecycle delete workspace for skill %q: workspace is required", profile.SkillName)
	}
	if err := skills.ValidateSkillName(profile.SkillName); err != nil {
		return fmt.Errorf("resolve lifecycle delete skill name: %w", err)
	}

	skillPath := filepath.Join(workspace, "skills", profile.SkillName, "SKILL.md")
	err := os.Remove(skillPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func inferWorkspaceFromPaths(paths Paths) string {
	root := filepath.Clean(paths.RootDir)
	if filepath.Base(root) != "evolution" {
		return ""
	}
	stateDir := filepath.Dir(root)
	if filepath.Base(stateDir) != "state" {
		return ""
	}
	return filepath.Dir(stateDir)
}
