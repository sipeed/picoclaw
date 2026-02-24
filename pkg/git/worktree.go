package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// WorktreeInfo describes an active git worktree.
type WorktreeInfo struct {
	Path       string // absolute worktree dir
	Branch     string // e.g. "plan/setup-monitoring"
	BaseBranch string // branch forked from
	RepoRoot   string // main repo root
}

// DisposeResult describes what happened when a worktree was disposed.
type DisposeResult struct {
	Branch        string
	AutoCommitted bool // true if uncommitted changes were saved
	BranchDeleted bool // true if branch had no unique commits
	CommitsAhead  int  // unique commits on branch (0 = safe to delete)
}

// FindRepoRoot returns the git repository root for dir, or "" if not a git repo.
func FindRepoRoot(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// CurrentBranch returns the current branch name, or "" on error.
func CurrentBranch(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

var unsafeBranchRe = regexp.MustCompile(`[^a-z0-9-]`)

// SanitizeBranchName creates a safe branch name from a task description.
// Returns "plan/<safe-40-chars>".
func SanitizeBranchName(task string) string {
	s := strings.ToLower(strings.TrimSpace(task))
	s = unsafeBranchRe.ReplaceAllString(s, "-")

	// Collapse consecutive hyphens
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")

	if s == "" {
		s = "worktree"
	}

	// Truncate to 40 chars
	runes := []rune(s)
	if len(runes) > 40 {
		runes = runes[:40]
	}
	s = strings.TrimRight(string(runes), "-")

	return "plan/" + s
}

// CreateWorktree creates a new git worktree at worktreePath with branchName.
// If the branch already exists, it reuses it.
func CreateWorktree(repoDir, worktreePath, branchName string) (*WorktreeInfo, error) {
	baseBranch := CurrentBranch(repoDir)
	if baseBranch == "" {
		baseBranch = "HEAD"
	}

	if err := os.MkdirAll(filepath.Dir(worktreePath), 0o755); err != nil {
		return nil, fmt.Errorf("create worktree parent: %w", err)
	}

	// Check if branch already exists
	checkCmd := exec.Command("git", "rev-parse", "--verify", branchName)
	checkCmd.Dir = repoDir
	branchExists := checkCmd.Run() == nil

	var cmd *exec.Cmd
	if branchExists {
		// Reuse existing branch
		cmd = exec.Command("git", "worktree", "add", worktreePath, branchName)
	} else {
		// Create new branch
		cmd = exec.Command("git", "worktree", "add", "-b", branchName, worktreePath)
	}
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git worktree add: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return &WorktreeInfo{
		Path:       worktreePath,
		Branch:     branchName,
		BaseBranch: baseBranch,
		RepoRoot:   repoDir,
	}, nil
}

// HasUncommittedChanges returns true if the working tree has staged or unstaged changes.
func HasUncommittedChanges(dir string) bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

// AutoCommit stages all changes and commits with the given message.
func AutoCommit(worktreePath, message string) error {
	addCmd := exec.Command("git", "add", "-A")
	addCmd.Dir = worktreePath
	if out, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %s: %w", strings.TrimSpace(string(out)), err)
	}

	commitCmd := exec.Command("git", "commit", "-m", message, "--allow-empty-message")
	commitCmd.Dir = worktreePath
	if out, err := commitCmd.CombinedOutput(); err != nil {
		// "nothing to commit" is not a real error
		if strings.Contains(string(out), "nothing to commit") {
			return nil
		}
		return fmt.Errorf("git commit: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// CommitsAhead returns the number of commits on branch that are not on base.
func CommitsAhead(repoDir, base, branch string) int {
	cmd := exec.Command("git", "rev-list", "--count", base+".."+branch)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return n
}

// SafeDispose auto-commits uncommitted changes, removes the worktree directory,
// and deletes the branch ONLY if it has no unique commits.
func SafeDispose(repoDir string, wt *WorktreeInfo) DisposeResult {
	result := DisposeResult{Branch: wt.Branch}

	// 1. Auto-commit if there are uncommitted changes
	if HasUncommittedChanges(wt.Path) {
		msg := fmt.Sprintf("auto: save from %s", wt.Branch)
		if err := AutoCommit(wt.Path, msg); err == nil {
			result.AutoCommitted = true
		}
	}

	// 2. Count unique commits
	result.CommitsAhead = CommitsAhead(repoDir, wt.BaseBranch, wt.Branch)

	// 3. Remove worktree
	removeCmd := exec.Command("git", "worktree", "remove", "--force", wt.Path)
	removeCmd.Dir = repoDir
	removeCmd.Run() // best-effort

	// 4. Delete branch if no unique commits
	if result.CommitsAhead == 0 {
		delCmd := exec.Command("git", "branch", "-D", wt.Branch)
		delCmd.Dir = repoDir
		if delCmd.Run() == nil {
			result.BranchDeleted = true
		}
	}

	// 5. Fallback cleanup
	os.RemoveAll(wt.Path)

	return result
}

// PruneOrphaned runs git worktree prune and removes dirs in worktreesDir
// that aren't valid git worktrees.
func PruneOrphaned(repoDir, worktreesDir string) {
	pruneCmd := exec.Command("git", "worktree", "prune")
	pruneCmd.Dir = repoDir
	pruneCmd.Run() // best-effort

	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		wtPath := filepath.Join(worktreesDir, entry.Name())
		// Check if it's still a valid git worktree
		checkCmd := exec.Command("git", "rev-parse", "--git-dir")
		checkCmd.Dir = wtPath
		if err := checkCmd.Run(); err != nil {
			// Not a valid git worktree — remove
			os.RemoveAll(wtPath)
		}
	}
}

// BranchBaseName extracts the last segment of a branch name.
// "plan/add-auth" → "plan-add-auth"
func BranchBaseName(branch string) string {
	s := strings.ReplaceAll(branch, "/", "-")
	// Remove leading/trailing hyphens
	s = strings.Trim(s, "-")
	// Remove non-printable chars
	var b strings.Builder
	for _, r := range s {
		if unicode.IsPrint(r) {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "worktree"
	}
	return b.String()
}
