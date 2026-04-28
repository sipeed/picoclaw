package evolutioncmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/pkg/evolution"
	"github.com/sipeed/picoclaw/pkg/fileutil"
)

func newRollbackCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "rollback <skill-name>",
		Short: "Restore the latest backup for one skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, store, workspace, err := loadEvolutionDeps()
			if err != nil {
				return err
			}

			paths := evolution.NewPaths(workspace, cfg.Evolution.StateDir)
			backupPath, err := latestBackupPath(paths, args[0])
			if err != nil {
				return err
			}

			data, err := os.ReadFile(backupPath)
			if err != nil {
				return err
			}

			skillPath := filepath.Join(workspace, "skills", args[0], "SKILL.md")
			if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
				return err
			}
			if err := fileutil.WriteFileAtomic(skillPath, data, 0o644); err != nil {
				return err
			}

			now := time.Now().UTC()
			rolledBackDraftID, err := saveRollbackProfile(store, workspace, args[0], backupPath, now)
			if err != nil {
				return err
			}
			if err := markDraftRolledBack(store, workspace, cfg.Evolution.StateDir, rolledBackDraftID, backupPath, now); err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "rolled back skill=%s backup=%s\n", args[0], backupPath)
			return err
		},
	}
}

func latestBackupPath(paths evolution.Paths, skillName string) (string, error) {
	backupRoot := filepath.Join(paths.BackupsDir, skillName)
	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		return "", err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	for _, name := range names {
		candidate := filepath.Join(backupRoot, name, "SKILL.md")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no backup found for skill %q", skillName)
}

func saveRollbackProfile(store *evolution.Store, workspace, skillName, backupPath string, now time.Time) (string, error) {
	profile, err := store.LoadProfile(skillName)
	if err != nil {
		return "", err
	}

	rolledBackDraftID := profile.CurrentVersion
	previousVersion := previousStableVersion(profile.VersionHistory, profile.CurrentVersion)
	profile.CurrentVersion = previousVersion
	profile.Status = evolution.SkillStatusActive
	profile.LastUsedAt = now
	profile.VersionHistory = append(profile.VersionHistory, evolution.SkillVersionEntry{
		Version:        previousVersion,
		Action:         "manual_rollback",
		Timestamp:      now,
		DraftID:        rolledBackDraftID,
		Summary:        "manual CLI rollback to latest backup: " + backupPath,
		Rollback:       true,
		RollbackReason: "manual CLI rollback",
	})
	return rolledBackDraftID, store.SaveProfile(profile)
}

func previousStableVersion(history []evolution.SkillVersionEntry, currentVersion string) string {
	if len(history) == 0 {
		return currentVersion
	}

	currentIndex := -1
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Version == currentVersion {
			currentIndex = i
			break
		}
	}
	if currentIndex <= 0 {
		return currentVersion
	}
	for i := currentIndex - 1; i >= 0; i-- {
		if history[i].Rollback {
			continue
		}
		if history[i].Version != "" {
			return history[i].Version
		}
	}
	return currentVersion
}

func markDraftRolledBack(store *evolution.Store, workspace, stateDir, draftID, backupPath string, now time.Time) error {
	if draftID == "" {
		return nil
	}

	drafts, err := store.LoadDrafts()
	if err != nil {
		return err
	}
	paths := evolution.NewPaths(workspace, stateDir)
	for i, draft := range drafts {
		if !draftBelongsToWorkspace(paths, workspace, draft) {
			continue
		}
		if draft.ID != draftID {
			continue
		}
		drafts[i].Status = evolution.DraftStatusQuarantined
		drafts[i].UpdatedAt = timePtr(now)
		drafts[i].ReviewNotes = appendUniqueStrings(
			drafts[i].ReviewNotes,
			"manually rolled back after apply",
			"backup="+backupPath,
		)
		return store.SaveDrafts(drafts)
	}
	return nil
}
