package evolutioncmd

import (
	"path/filepath"

	"github.com/sipeed/picoclaw/pkg/evolution"
)

func draftBelongsToWorkspace(paths evolution.Paths, workspace string, draft evolution.SkillDraft) bool {
	if draft.WorkspaceID == workspace {
		return true
	}
	return draft.WorkspaceID == "" && usesDefaultWorkspaceState(paths, workspace)
}

func profileBelongsToWorkspace(paths evolution.Paths, workspace string, profile evolution.SkillProfile) bool {
	if profile.WorkspaceID == workspace {
		return true
	}
	return profile.WorkspaceID == "" && usesDefaultWorkspaceState(paths, workspace)
}

func usesDefaultWorkspaceState(paths evolution.Paths, workspace string) bool {
	return paths.RootDir == evolution.NewPaths(workspace, "").RootDir
}

func inferWorkspaceFromPaths(paths evolution.Paths) string {
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
