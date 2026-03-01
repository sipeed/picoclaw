package tools

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/git"
)

// worktreeInfoKey is the context key for passing WorktreeInfo to tools.
type worktreeInfoKey struct{}

// WithWorktreeInfo returns a context carrying the active WorktreeInfo.
func WithWorktreeInfo(ctx context.Context, wt *git.WorktreeInfo) context.Context {
	return context.WithValue(ctx, worktreeInfoKey{}, wt)
}

// WorktreeInfoFromCtx extracts the WorktreeInfo from context, or nil.
func WorktreeInfoFromCtx(ctx context.Context) *git.WorktreeInfo {
	if v, ok := ctx.Value(worktreeInfoKey{}).(*git.WorktreeInfo); ok {
		return v
	}
	return nil
}

// protectedBranches are branch names that can never be pushed to.
var protectedBranches = regexp.MustCompile(`^(main|master|develop|release/.*)$`)

// GitPushTool implements safe git push restricted to worktree branches.
//
// Safety invariants:
//   - Only works inside a worktree (WorktreeInfo must be in context)
//   - Pushes only the worktree's branch — no arbitrary branch targets
//   - Protected branches (main, master, develop, release/*) are blocked
//   - Force push is never allowed
//   - Auto-commits uncommitted changes before pushing
type GitPushTool struct {
	workspace string
}

// NewGitPushTool creates a GitPushTool.
func NewGitPushTool(workspace string) *GitPushTool {
	return &GitPushTool{workspace: workspace}
}

func (t *GitPushTool) Name() string { return "git_push" }

func (t *GitPushTool) Description() string {
	return "Push the current worktree branch to origin. Only works inside a git worktree. " +
		"Auto-commits uncommitted changes before pushing. " +
		"Protected branches (main, master, develop) cannot be pushed to. Force push is not allowed."
}

func (t *GitPushTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"commit_message": map[string]any{
				"type":        "string",
				"description": "Commit message for uncommitted changes. If omitted, uncommitted changes are auto-committed with a default message.",
			},
		},
		"required": []string{},
	}
}

func (t *GitPushTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	wt := WorktreeInfoFromCtx(ctx)
	if wt == nil {
		return ErrorResult(
			"git_push requires an active worktree.\n" +
				"This tool can only be used during worktree-based sessions " +
				"(e.g., heartbeat tasks or plan executing phase).\n" +
				"The worktree provides the branch name and isolation boundary — " +
				"without it, git_push cannot determine which branch to push.")
	}

	branch := wt.Branch
	if branch == "" {
		return ErrorResult(
			"worktree has no branch name.\n" +
				"The WorktreeInfo was set but Branch is empty. " +
				"This is an internal error — the worktree may not have been created correctly.")
	}

	// Block protected branches
	if protectedBranches.MatchString(branch) {
		return ErrorResult(fmt.Sprintf(
			"cannot push to protected branch %q.\n"+
				"Protected branches (main, master, develop, release/*) are blocked to prevent "+
				"accidental overwrites. Work should be done on feature branches created by worktrees.",
			branch))
	}

	// Auto-commit uncommitted changes
	if git.HasUncommittedChanges(wt.Path) {
		commitMsg := "auto: save before push"
		if msg, ok := args["commit_message"].(string); ok && msg != "" {
			commitMsg = msg
		}
		if err := git.AutoCommit(wt.Path, commitMsg); err != nil {
			return ErrorResult(fmt.Sprintf(
				"auto-commit failed before push: %v\n"+
					"git_push auto-commits uncommitted changes before pushing. "+
					"The commit failed, so no push was attempted. "+
					"Check if the worktree at %q is in a valid state (e.g., no merge conflicts).",
				err, wt.Path))
		}
	}

	// Check there are commits to push
	ahead := git.CommitsAhead(wt.RepoRoot, wt.BaseBranch, branch)
	if ahead == 0 {
		return NewToolResult(fmt.Sprintf(
			"Nothing to push: branch %q has no commits ahead of %s.\n"+
				"The branch is identical to the base. Make changes and commit before pushing.",
			branch, wt.BaseBranch))
	}

	// Push with -u (set upstream tracking)
	pushCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(pushCtx, "git", "push", "-u", "origin", branch)
	cmd.Dir = wt.Path
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))

	if err != nil {
		return ErrorResult(fmt.Sprintf(
			"git push failed for branch %q: %s\n%s\n"+
				"Possible causes: network error, authentication failure, or remote rejected the push. "+
				"If the remote branch has diverged, resolve the divergence in the worktree first — "+
				"force push is not available.",
			branch, err, output))
	}

	return NewToolResult(fmt.Sprintf("Pushed branch %q to origin (%d commit(s) ahead of %s)\n%s",
		branch, ahead, wt.BaseBranch, output))
}
