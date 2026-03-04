package git

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// WorktreeInfo describes an active git worktree.
type WorktreeInfo struct {
	Path       string // absolute worktree dir
	Branch     string // e.g. "plan/setup-monitoring"
	BaseBranch string // branch forked from
	RepoRoot   string // main repo root
}

// ManagedWorktree is a user-facing summary for worktree management commands/UI.
type ManagedWorktree struct {
	Name              string `json:"name"`
	Path              string `json:"-"`
	Branch            string `json:"branch"`
	LastCommitHash    string `json:"last_commit_hash"`
	LastCommitSubject string `json:"last_commit_subject"`
	LastCommitAge     string `json:"last_commit_age"`
	HasUncommitted    bool   `json:"has_uncommitted"`
}

var (
	// ErrInvalidWorktreeName is returned when the given worktree name is unsafe.
	ErrInvalidWorktreeName = errors.New("invalid worktree name")
	// ErrWorktreeNotFound is returned when the named worktree cannot be found.
	ErrWorktreeNotFound = errors.New("worktree not found")
)

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

	// Truncate to 40 chars (ASCII fast path: branch names are ASCII after sanitization)
	if len(s) > 40 {
		s = s[:40]
	}
	s = strings.TrimRight(s, "-")

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

// MergeResult describes the outcome of a worktree branch merge attempt.
type MergeResult struct {
	Merged   bool   // true if merge succeeded
	Branch   string // branch name that was merged
	Conflict bool   // true if merge failed due to conflict
}

// MergeWorktreeBranch attempts to merge the worktree branch into the base branch.
// On conflict, it aborts the merge and returns Conflict=true.
// Must be called AFTER auto-commit and BEFORE SafeDispose.
func MergeWorktreeBranch(repoDir string, wt *WorktreeInfo) MergeResult {
	result := MergeResult{Branch: wt.Branch}

	mergeCmd := exec.Command("git", "merge", "--no-edit", wt.Branch)
	mergeCmd.Dir = repoDir
	if err := mergeCmd.Run(); err != nil {
		// Merge failed — abort and report conflict
		abortCmd := exec.Command("git", "merge", "--abort")
		abortCmd.Dir = repoDir
		abortCmd.Run() // best-effort
		result.Conflict = true
		return result
	}

	result.Merged = true
	return result
}

// ListManagedWorktrees returns active worktree summaries under worktreesDir.
func ListManagedWorktrees(repoDir, worktreesDir string) ([]ManagedWorktree, error) {
	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ManagedWorktree{}, nil
		}
		return nil, err
	}

	result := make([]ManagedWorktree, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		wtPath := filepath.Join(worktreesDir, name)
		if !isLinkedWorktree(wtPath) {
			continue
		}
		result = append(result, buildManagedWorktree(repoDir, name, wtPath))
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// GetManagedWorktree resolves one managed worktree by name.
func GetManagedWorktree(repoDir, worktreesDir, name string) (*ManagedWorktree, error) {
	if !isSafeWorktreeName(name) {
		return nil, ErrInvalidWorktreeName
	}
	wtPath := filepath.Join(worktreesDir, name)
	if !isLinkedWorktree(wtPath) {
		return nil, ErrWorktreeNotFound
	}
	wt := buildManagedWorktree(repoDir, name, wtPath)
	return &wt, nil
}

// MergeManagedWorktree merges a named worktree branch into baseBranch.
// When baseBranch is empty, DetectDefaultBranch(repoDir) is used.
func MergeManagedWorktree(repoDir, worktreesDir, name, baseBranch string) (MergeResult, string, error) {
	var zero MergeResult

	wt, err := GetManagedWorktree(repoDir, worktreesDir, name)
	if err != nil {
		return zero, "", err
	}
	if wt.Branch == "" || wt.Branch == "HEAD" {
		return zero, "", fmt.Errorf("worktree %q has no mergeable branch", name)
	}

	if baseBranch == "" {
		baseBranch = DetectDefaultBranch(repoDir)
	}
	if baseBranch == "" {
		return zero, "", fmt.Errorf("failed to resolve base branch")
	}

	current := CurrentBranch(repoDir)
	if current == "" {
		return zero, "", fmt.Errorf("failed to detect current branch")
	}

	if current != baseBranch {
		if err := checkoutBranch(repoDir, baseBranch); err != nil {
			return zero, "", err
		}
		defer func() {
			if err := checkoutBranch(repoDir, current); err != nil {
				logger.WarnCF("git", "Failed to restore original branch after merge", map[string]any{
					"branch": current,
					"error":  err.Error(),
				})
			}
		}()
	}

	res := MergeWorktreeBranch(repoDir, &WorktreeInfo{
		Path:       wt.Path,
		Branch:     wt.Branch,
		BaseBranch: baseBranch,
		RepoRoot:   repoDir,
	})
	return res, baseBranch, nil
}

// DisposeManagedWorktree removes a named worktree with SafeDispose.
// When baseBranch is empty, DetectDefaultBranch(repoDir) is used.
func DisposeManagedWorktree(repoDir, worktreesDir, name, baseBranch string) (DisposeResult, error) {
	var zero DisposeResult

	wt, err := GetManagedWorktree(repoDir, worktreesDir, name)
	if err != nil {
		return zero, err
	}
	if wt.Branch == "" {
		return zero, fmt.Errorf("worktree %q has no branch", name)
	}
	if baseBranch == "" {
		baseBranch = DetectDefaultBranch(repoDir)
	}
	if baseBranch == "" {
		baseBranch = "main"
	}
	res := SafeDispose(repoDir, &WorktreeInfo{
		Path:       wt.Path,
		Branch:     wt.Branch,
		BaseBranch: baseBranch,
		RepoRoot:   repoDir,
	})
	return res, nil
}

// WorktreeStatusShort returns "git status --short" output for a worktree.
func WorktreeStatusShort(worktreePath string) (string, error) {
	cmd := exec.Command("git", "status", "--short")
	cmd.Dir = worktreePath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git status --short: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// WorktreeRecentLog returns recent oneline commits for a worktree branch.
func WorktreeRecentLog(worktreePath string, n int) (string, error) {
	if n <= 0 {
		n = 10
	}
	cmd := exec.Command("git", "log", "--oneline", fmt.Sprintf("-%d", n))
	cmd.Dir = worktreePath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git log --oneline: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// WorktreeDiffStat returns a compact diff stat for a worktree.
func WorktreeDiffStat(worktreePath string) (string, error) {
	cmd := exec.Command("git", "diff", "--stat")
	cmd.Dir = worktreePath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git diff --stat: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// DetectDefaultBranch returns the preferred base branch name for merges/dispose.
func DetectDefaultBranch(repoDir string) string {
	if localBranchExists(repoDir, "main") {
		return "main"
	}
	if localBranchExists(repoDir, "master") {
		return "master"
	}

	// Try origin/HEAD -> origin/<branch>
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = repoDir
	if out, err := cmd.Output(); err == nil {
		ref := strings.TrimSpace(string(out))
		if idx := strings.LastIndex(ref, "/"); idx >= 0 && idx < len(ref)-1 {
			return ref[idx+1:]
		}
	}

	current := CurrentBranch(repoDir)
	if current != "" && current != "HEAD" {
		return current
	}
	return "main"
}

// PruneOrphaned removes stale worktree directories under worktreesDir.
// For orphaned linked worktrees with uncommitted changes, it auto-commits before removal.
func PruneOrphaned(repoDir, worktreesDir string) {
	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		// Still attempt git's own metadata prune.
		pruneCmd := exec.Command("git", "worktree", "prune")
		pruneCmd.Dir = repoDir
		pruneCmd.Run() // best-effort
		return
	}

	active := map[string]bool{}
	activeKnown := false
	if activePaths, err := listGitWorktreePaths(repoDir); err == nil {
		activeKnown = true
		for _, p := range activePaths {
			active[filepath.Clean(p)] = true
		}
	} else {
		logger.WarnCF("git", "Skip linked-worktree prune: failed to enumerate active worktrees", map[string]any{
			"error": err.Error(),
		})
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		wtPath := filepath.Clean(filepath.Join(worktreesDir, entry.Name()))

		if !isLinkedWorktree(wtPath) {
			_ = os.RemoveAll(wtPath)
			continue
		}
		if !activeKnown {
			continue
		}
		if active[wtPath] {
			continue
		}

		// Orphaned linked worktree: protect changes before prune.
		if HasUncommittedChanges(wtPath) {
			if err := AutoCommit(wtPath, "auto-save before prune"); err != nil {
				logger.WarnCF("git", "Skip pruning orphaned worktree: auto-commit failed", map[string]any{
					"path":  wtPath,
					"error": err.Error(),
				})
				continue
			}
			logger.InfoCF("git", "Auto-committed orphaned worktree before prune", map[string]any{"path": wtPath})
		}

		removeCmd := exec.Command("git", "worktree", "remove", "--force", wtPath)
		removeCmd.Dir = repoDir
		if out, err := removeCmd.CombinedOutput(); err != nil {
			logger.WarnCF(
				"git",
				"git worktree remove failed for orphaned worktree; removing directory directly",
				map[string]any{
					"path":  wtPath,
					"error": strings.TrimSpace(string(out)),
				},
			)
			_ = os.RemoveAll(wtPath)
			continue
		}
		logger.InfoCF("git", "Pruned orphaned worktree", map[string]any{"path": wtPath})
	}

	pruneCmd := exec.Command("git", "worktree", "prune")
	pruneCmd.Dir = repoDir
	if out, err := pruneCmd.CombinedOutput(); err != nil {
		logger.WarnCF("git", "git worktree prune failed", map[string]any{"error": strings.TrimSpace(string(out))})
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

func isSafeWorktreeName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if filepath.Base(name) != name {
		return false
	}
	if strings.ContainsAny(name, `/\\`) {
		return false
	}
	return true
}

func isLinkedWorktree(path string) bool {
	gitPath := filepath.Join(path, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	// Linked worktrees have a .git file (not a directory).
	if info.IsDir() {
		return false
	}
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = path
	return cmd.Run() == nil
}

func buildManagedWorktree(repoDir, name, wtPath string) ManagedWorktree {
	hash, subject, age := lastCommitInfo(wtPath)
	return ManagedWorktree{
		Name:              name,
		Path:              wtPath,
		Branch:            CurrentBranch(wtPath),
		LastCommitHash:    hash,
		LastCommitSubject: subject,
		LastCommitAge:     age,
		HasUncommitted:    HasUncommittedChanges(wtPath),
	}
}

func lastCommitInfo(dir string) (hash, subject, age string) {
	cmd := exec.Command("git", "log", "-1", "--pretty=format:%h\x1f%s\x1f%cr")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", "", ""
	}
	parts := strings.SplitN(strings.TrimSpace(string(out)), "\x1f", 3)
	if len(parts) > 0 {
		hash = parts[0]
	}
	if len(parts) > 1 {
		subject = parts[1]
	}
	if len(parts) > 2 {
		age = parts[2]
	}
	return hash, subject, age
}

func localBranchExists(repoDir, name string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", "refs/heads/"+name)
	cmd.Dir = repoDir
	return cmd.Run() == nil
}

func checkoutBranch(repoDir, branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout %s: %s: %w", branch, strings.TrimSpace(string(out)), err)
	}
	return nil
}

func listGitWorktreePaths(repoDir string) ([]string, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(out), "\n")
	paths := make([]string, 0)
	for _, line := range lines {
		if !strings.HasPrefix(line, "worktree ") {
			continue
		}
		p := strings.TrimSpace(strings.TrimPrefix(line, "worktree "))
		if p != "" {
			paths = append(paths, filepath.Clean(p))
		}
	}
	return paths, nil
}
