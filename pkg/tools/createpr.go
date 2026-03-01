package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CreatePRTool creates a GitHub pull request from the current worktree branch.
//
// Safety invariants:
//   - Only works inside a worktree (WorktreeInfo must be in context)
//   - Base branch is auto-detected from WorktreeInfo.BaseBranch
//   - Requires the branch to be already pushed (use git_push first)
//   - Checks for merge conflicts with base before creating
//   - Uses `gh pr create` under the hood
type CreatePRTool struct {
	workspace string
}

// NewCreatePRTool creates a CreatePRTool.
func NewCreatePRTool(workspace string) *CreatePRTool {
	return &CreatePRTool{workspace: workspace}
}

func (t *CreatePRTool) Name() string { return "create_pr" }

func (t *CreatePRTool) Description() string {
	return "Create a GitHub pull request from the current worktree branch. " +
		"The base branch is auto-detected from the worktree's parent branch. " +
		"The branch must be pushed to origin first (use git_push). " +
		"Checks for merge conflicts with the base branch before creating. " +
		"Requires the `gh` CLI to be installed and authenticated."
}

func (t *CreatePRTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{
				"type":        "string",
				"description": "Pull request title",
			},
			"body": map[string]any{
				"type":        "string",
				"description": "Pull request body/description (supports markdown)",
			},
			"draft": map[string]any{
				"type":        "boolean",
				"description": "Create as draft PR (default: false)",
			},
		},
		"required": []string{"title"},
	}
}

func (t *CreatePRTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	wt := WorktreeInfoFromCtx(ctx)
	if wt == nil {
		return ErrorResult(
			"create_pr requires an active worktree.\n" +
				"This tool can only be used during worktree-based sessions " +
				"(e.g., heartbeat tasks or plan executing phase).\n" +
				"The worktree provides the branch name and base branch for the PR.")
	}

	branch := wt.Branch
	if branch == "" {
		return ErrorResult(
			"worktree has no branch name.\n" +
				"The WorktreeInfo was set but Branch is empty. " +
				"This is an internal error — the worktree may not have been created correctly.")
	}

	baseBranch := wt.BaseBranch
	if baseBranch == "" {
		baseBranch = "main"
	}

	title, ok := args["title"].(string)
	if !ok || strings.TrimSpace(title) == "" {
		return ErrorResult(
			"title is required.\n" +
				"Provide a concise PR title describing the change (e.g., \"Add rate limiter to API endpoints\").")
	}

	// Verify the branch has been pushed by checking if the remote ref exists
	checkCtx, checkCancel := context.WithTimeout(ctx, 15*time.Second)
	defer checkCancel()
	checkCmd := exec.CommandContext(checkCtx, "git", "ls-remote", "--exit-code", "origin", branch)
	checkCmd.Dir = wt.Path
	if err := checkCmd.Run(); err != nil {
		return ErrorResult(fmt.Sprintf(
			"branch %q not found on origin.\n"+
				"The branch must be pushed before creating a PR. Use the git_push tool first.\n"+
				"git_push will auto-commit uncommitted changes and push the worktree branch to origin.",
			branch))
	}

	// Fetch latest base branch and check for merge conflicts
	fetchCtx, fetchCancel := context.WithTimeout(ctx, 30*time.Second)
	defer fetchCancel()
	fetchCmd := exec.CommandContext(fetchCtx, "git", "fetch", "origin", baseBranch)
	fetchCmd.Dir = wt.Path
	if out, err := fetchCmd.CombinedOutput(); err != nil {
		return ErrorResult(fmt.Sprintf(
			"failed to fetch origin/%s: %s\n%s\n"+
				"Cannot verify merge compatibility without the latest base branch. "+
				"Check network connectivity and that the base branch %q exists on origin.",
			baseBranch, err, strings.TrimSpace(string(out)), baseBranch))
	}

	// Try a merge dry-run to detect conflicts.
	// merge-tree --write-tree is a plumbing command (Git 2.38+) that performs a
	// three-way merge entirely in-memory without touching the working tree.
	// Exit code 0 = clean merge, non-zero = conflicts detected.
	mergeCtx, mergeCancel := context.WithTimeout(ctx, 30*time.Second)
	defer mergeCancel()
	mergeCmd := exec.CommandContext(mergeCtx, "git", "merge-tree",
		"--write-tree", "--no-messages",
		branch, "origin/"+baseBranch)
	mergeCmd.Dir = wt.RepoRoot
	mergeOut, mergeErr := mergeCmd.CombinedOutput()
	if mergeErr != nil {
		conflictInfo := strings.TrimSpace(string(mergeOut))
		return ErrorResult(fmt.Sprintf(
			"merge conflict detected between %q and %s.\n"+
				"The PR cannot be created cleanly. Resolve the conflicts in the worktree first, "+
				"then use git_push to push the resolution before retrying create_pr.\n"+
				"Conflict details:\n%s",
			branch, baseBranch, conflictInfo))
	}

	// Build gh pr create command
	ghArgs := []string{"pr", "create",
		"--base", baseBranch,
		"--head", branch,
		"--title", title,
	}

	if body, ok := args["body"].(string); ok && body != "" {
		ghArgs = append(ghArgs, "--body", body)
	} else {
		ghArgs = append(ghArgs, "--body", "")
	}

	if draft, ok := args["draft"].(bool); ok && draft {
		ghArgs = append(ghArgs, "--draft")
	}

	prCtx, prCancel := context.WithTimeout(ctx, 30*time.Second)
	defer prCancel()

	cmd := exec.CommandContext(prCtx, "gh", ghArgs...)
	cmd.Dir = wt.RepoRoot
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))

	if err != nil {
		return ErrorResult(fmt.Sprintf(
			"gh pr create failed: %s\n%s\n"+
				"Possible causes:\n"+
				"- gh CLI not installed or not authenticated (run `gh auth login`)\n"+
				"- A PR already exists for branch %q (check with `gh pr list`)\n"+
				"- Repository not configured as a GitHub remote",
			err, output, branch))
	}

	return NewToolResult(fmt.Sprintf(
		"Pull request created: %s\n"+
			"Branch: %s -> %s",
		output, branch, baseBranch))
}
