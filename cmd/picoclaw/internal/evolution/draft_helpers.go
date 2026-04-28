package evolutioncmd

import (
	"fmt"
	"time"

	"github.com/sipeed/picoclaw/pkg/evolution"
)

func loadWorkspaceDraft(store *evolution.Store, workspace, stateDir, id string) (evolution.SkillDraft, error) {
	drafts, err := store.LoadDrafts()
	if err != nil {
		return evolution.SkillDraft{}, err
	}
	paths := evolution.NewPaths(workspace, stateDir)
	_, draft, err := findWorkspaceDraft(drafts, paths, workspace, id)
	return draft, err
}

func findWorkspaceDraft(
	drafts []evolution.SkillDraft,
	paths evolution.Paths,
	workspace, id string,
) (int, evolution.SkillDraft, error) {
	for i, draft := range drafts {
		if !draftBelongsToWorkspace(paths, workspace, draft) {
			continue
		}
		if draft.ID == id {
			return i, draft, nil
		}
	}
	return -1, evolution.SkillDraft{}, fmt.Errorf("draft %q not found for workspace", id)
}

func appendUniqueStrings(existing []string, values ...string) []string {
	seen := make(map[string]struct{}, len(existing))
	for _, value := range existing {
		seen[value] = struct{}{}
	}
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		existing = append(existing, value)
		seen[value] = struct{}{}
	}
	return existing
}

func timePtr(v time.Time) *time.Time {
	return &v
}
