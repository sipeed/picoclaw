package agent

import (
	"fmt"
	"sync"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/git"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// instanceExt holds fork-specific fields for AgentInstance.
// Embedded in AgentInstance so existing field access continues to work.
type instanceExt struct {
	// SubagentMgr is set during registerSharedTools when orchestration is enabled.
	// Used by runAgentLoop to wait for spawned subagents before worktree cleanup.
	SubagentMgr *tools.SubagentManager

	Subagents    *config.SubagentsConfig
	SkillsFilter []string

	// Interview staleness tracking: consecutive turns where MEMORY.md was not updated.
	interviewStaleCount int
	interviewMemoryLen  int

	// Per-session worktree isolation
	worktrees  map[string]*git.WorktreeInfo // sessionKey → worktree
	worktreeMu sync.RWMutex
}

// ActivateWorktree creates a worktree for a session.
// projectDir is the git repository to create the worktree in.
// If empty, falls back to ai.Workspace.
// Worktree path: <workspace>/.worktrees/<branch-basename>/
func (ai *AgentInstance) ActivateWorktree(sessionKey, taskName, projectDir string) (*git.WorktreeInfo, error) {
	if projectDir == "" {
		projectDir = ai.Workspace
	}

	repoRoot := git.FindRepoRoot(projectDir)
	if repoRoot == "" {
		return nil, fmt.Errorf("directory is not a git repository: %s", projectDir)
	}

	branchName := git.SanitizeBranchName(taskName)
	baseName := git.BranchBaseName(branchName)
	wtPath := ai.worktreePath(baseName)

	wt, err := git.CreateWorktree(repoRoot, wtPath, branchName)
	if err != nil {
		return nil, err
	}

	ai.worktreeMu.Lock()
	if ai.worktrees == nil {
		ai.worktrees = make(map[string]*git.WorktreeInfo)
	}
	ai.worktrees[sessionKey] = wt
	ai.worktreeMu.Unlock()

	return wt, nil
}

// worktreePath returns the standard path for a worktree under the workspace.
func (ai *AgentInstance) worktreePath(baseName string) string {
	return ai.Workspace + "/.worktrees/" + baseName
}

// DeactivateWorktree safe-disposes the session's worktree.
func (ai *AgentInstance) DeactivateWorktree(sessionKey, commitMsg string, discard bool) (*git.DisposeResult, error) {
	ai.worktreeMu.Lock()
	wt, ok := ai.worktrees[sessionKey]
	if ok {
		delete(ai.worktrees, sessionKey)
	}
	ai.worktreeMu.Unlock()

	if !ok || wt == nil {
		return nil, nil
	}

	repoRoot := git.FindRepoRoot(ai.Workspace)
	if repoRoot == "" {
		return nil, fmt.Errorf("workspace is not a git repository")
	}

	// Even on discard, SafeDispose auto-commits first for safety
	if commitMsg != "" && git.HasUncommittedChanges(wt.Path) {
		_ = git.AutoCommit(wt.Path, commitMsg)
	}

	result := git.SafeDispose(repoRoot, wt)
	return &result, nil
}

// GetWorktree returns the session's active worktree, or nil.
func (ai *AgentInstance) GetWorktree(sessionKey string) *git.WorktreeInfo {
	ai.worktreeMu.RLock()
	defer ai.worktreeMu.RUnlock()
	return ai.worktrees[sessionKey]
}

// IsInWorktree returns true if the session has an active worktree.
func (ai *AgentInstance) IsInWorktree(sessionKey string) bool {
	return ai.GetWorktree(sessionKey) != nil
}

// EffectiveWorkspace returns worktree path for session, or original Workspace.
func (ai *AgentInstance) EffectiveWorkspace(sessionKey string) string {
	if wt := ai.GetWorktree(sessionKey); wt != nil {
		return wt.Path
	}
	return ai.Workspace
}

// GetWorktreeBranch returns the branch name for the session's worktree, or "".
func (ai *AgentInstance) GetWorktreeBranch(sessionKey string) string {
	if wt := ai.GetWorktree(sessionKey); wt != nil {
		return wt.Branch
	}
	return ""
}
