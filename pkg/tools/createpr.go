package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	ciPollInterval = 30 * time.Second
	ciPollTimeout  = 15 * time.Minute
)

// CreatePRTool creates a GitHub pull request from the current worktree branch.
//
// Safety invariants:
//   - Only works inside a worktree (WorktreeInfo must be in context)
//   - Base branch is auto-detected from WorktreeInfo.BaseBranch
//   - Requires the branch to be already pushed (use git_push first)
//   - Checks for merge conflicts with base before creating
//   - Uses `gh pr create` under the hood
//
// Async behavior:
//   - PR creation itself is synchronous and returns immediately with the PR URL
//   - If CI runs are triggered, a background goroutine polls `gh pr checks`
//     and calls the AsyncCallback when CI completes (pass or fail)
type CreatePRTool struct {
	workspace string
	callback  AsyncCallback
}

// NewCreatePRTool creates a CreatePRTool.
func NewCreatePRTool(workspace string) *CreatePRTool {
	return &CreatePRTool{workspace: workspace}
}

func (t *CreatePRTool) Name() string { return "create_pr" }

// SetCallback implements AsyncTool for CI completion notification.
func (t *CreatePRTool) SetCallback(cb AsyncCallback) {
	t.callback = cb
}

func (t *CreatePRTool) Description() string {
	return "Create a GitHub pull request from the current worktree branch. " +
		"The base branch is auto-detected from the worktree's parent branch. " +
		"The branch must be pushed to origin first (use git_push). " +
		"Checks for merge conflicts with the base branch before creating. " +
		"After PR creation, polls CI status in the background and notifies when complete. " +
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

	prURL := output // gh pr create outputs the PR URL

	// Start background CI polling if callback is set
	if t.callback != nil && prURL != "" {
		cb := t.callback
		repoRoot := wt.RepoRoot
		go pollCIStatus(repoRoot, prURL, cb)
	}

	return AsyncResult(fmt.Sprintf(
		"Pull request created: %s\n"+
			"Branch: %s -> %s\n"+
			"CI status will be reported asynchronously when checks complete.",
		prURL, branch, baseBranch))
}

// pollCIStatus polls `gh pr checks` in the background until all checks
// pass, fail, or the timeout is reached. Reports back via AsyncCallback.
func pollCIStatus(repoRoot, prURL string, callback AsyncCallback) {
	// Detached context with hard timeout — this goroutine outlives the tool call.
	ctx, cancel := context.WithTimeout(context.Background(), ciPollTimeout)
	defer cancel()

	// Initial wait: CI runs take a few seconds to register after PR creation
	select {
	case <-time.After(10 * time.Second):
	case <-ctx.Done():
		return
	}

	ticker := time.NewTicker(ciPollInterval)
	defer ticker.Stop()

	for {
		status, detail := checkPRChecks(ctx, repoRoot, prURL)
		switch status {
		case ciStatusPass:
			callback(ctx, NewToolResult(fmt.Sprintf(
				"CI passed for %s\n%s",
				prURL, detail)))
			return
		case ciStatusFail:
			callback(ctx, &ToolResult{
				ForLLM: fmt.Sprintf(
					"CI failed for %s\n%s\n"+
						"Run `gh run view` for detailed logs.",
					prURL, detail),
				IsError: true,
			})
			return
		case ciStatusNone:
			callback(ctx, NewToolResult(fmt.Sprintf(
				"No CI checks configured for %s. PR is ready for review.",
				prURL)))
			return
		case ciStatusPending:
			// Still running, continue polling
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			callback(ctx, &ToolResult{
				ForLLM: fmt.Sprintf(
					"CI polling timed out after %s for %s.\n"+
						"Checks may still be running. Run `gh pr checks %s` to check.",
					ciPollTimeout, prURL, prURL),
				IsError: true,
			})
			return
		}
	}
}

type ciStatus int

const (
	ciStatusPending ciStatus = iota
	ciStatusPass
	ciStatusFail
	ciStatusNone
)

// checkPRChecks runs `gh pr checks` and parses the result.
// Returns the aggregate status and raw output for the caller to include.
func checkPRChecks(ctx context.Context, repoRoot, prURL string) (ciStatus, string) {
	checkCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(checkCtx, "gh", "pr", "checks", prURL)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))

	if err != nil {
		// gh pr checks exits 1 when any check has failed
		if strings.Contains(output, "fail") || strings.Contains(output, "X ") {
			return ciStatusFail, output
		}
		// "no checks" case
		if strings.Contains(output, "no checks") || output == "" {
			return ciStatusNone, ""
		}
		// Transient error or still pending — keep polling
		return ciStatusPending, output
	}

	// Exit 0: all checks completed. Check for pending.
	if strings.Contains(output, "pending") || strings.Contains(output, "- ") {
		return ciStatusPending, output
	}

	return ciStatusPass, output
}
