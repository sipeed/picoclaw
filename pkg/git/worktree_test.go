package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Add auth module", "plan/add-auth-module"},
		{"", "plan/worktree"},
		{"  spaces  ", "plan/spaces"},
		{"UPPER-case_Mix", "plan/upper-case-mix"},
		{"a/b/c", "plan/a-b-c"},
		{
			"very long task name that exceeds the forty character limit for safety",
			"plan/very-long-task-name-that-exceeds-the-for",
		},
		{"---leading-trailing---", "plan/leading-trailing"},
		{"special!@#$%chars", "plan/special-chars"},
	}

	for _, tt := range tests {
		got := SanitizeBranchName(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeBranchName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBranchBaseName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"plan/add-auth", "plan-add-auth"},
		{"heartbeat/20260224", "heartbeat-20260224"},
		{"main", "main"},
		{"", "worktree"},
	}

	for _, tt := range tests {
		got := BranchBaseName(tt.input)
		if got != tt.want {
			t.Errorf("BranchBaseName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// initTestRepo creates a temporary git repo with an initial commit.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git init: %s: %v", out, err)
		}
	}

	// Create initial commit
	f := filepath.Join(dir, "README.md")
	os.WriteFile(f, []byte("# Test\n"), 0o644)
	add := exec.Command("git", "add", "-A")
	add.Dir = dir
	add.Run()
	commit := exec.Command("git", "commit", "-m", "initial")
	commit.Dir = dir
	commit.Run()

	return dir
}

func TestFindRepoRoot(t *testing.T) {
	dir := initTestRepo(t)
	root := FindRepoRoot(dir)
	if root == "" {
		t.Fatal("FindRepoRoot returned empty for valid repo")
	}

	// Non-repo should return ""
	tmpDir := t.TempDir()
	if got := FindRepoRoot(tmpDir); got != "" {
		t.Errorf("FindRepoRoot(non-repo) = %q, want empty", got)
	}
}

func TestCurrentBranch(t *testing.T) {
	dir := initTestRepo(t)
	branch := CurrentBranch(dir)
	// Should be "main" or "master" depending on git config
	if branch == "" {
		t.Fatal("CurrentBranch returned empty for valid repo")
	}
}

func TestCreateWorktreeAndDispose(t *testing.T) {
	dir := initTestRepo(t)
	wtPath := filepath.Join(dir, ".picoclaw", "worktrees", "test-wt")

	wt, err := CreateWorktree(dir, wtPath, "plan/test-feature")
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}

	if wt.Path != wtPath {
		t.Errorf("Path = %q, want %q", wt.Path, wtPath)
	}
	if wt.Branch != "plan/test-feature" {
		t.Errorf("Branch = %q, want %q", wt.Branch, "plan/test-feature")
	}

	// Verify worktree exists
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Fatal("worktree dir was not created")
	}

	// SafeDispose with no changes — should delete branch
	result := SafeDispose(dir, wt)
	if result.AutoCommitted {
		t.Error("AutoCommitted should be false with no changes")
	}
	if result.CommitsAhead != 0 {
		t.Errorf("CommitsAhead = %d, want 0", result.CommitsAhead)
	}
	if !result.BranchDeleted {
		t.Error("BranchDeleted should be true when no unique commits")
	}
}

func TestCreateWorktreeWithChangesAndDispose(t *testing.T) {
	dir := initTestRepo(t)
	wtPath := filepath.Join(dir, ".picoclaw", "worktrees", "test-changes")

	wt, err := CreateWorktree(dir, wtPath, "plan/with-changes")
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}

	// Make a change in the worktree
	os.WriteFile(filepath.Join(wtPath, "new-file.txt"), []byte("hello"), 0o644)

	if !HasUncommittedChanges(wtPath) {
		t.Fatal("HasUncommittedChanges should be true after adding file")
	}

	// SafeDispose should auto-commit
	result := SafeDispose(dir, wt)
	if !result.AutoCommitted {
		t.Error("AutoCommitted should be true")
	}
	if result.CommitsAhead != 1 {
		t.Errorf("CommitsAhead = %d, want 1", result.CommitsAhead)
	}
	if result.BranchDeleted {
		t.Error("BranchDeleted should be false when branch has commits")
	}
}

func TestHasUncommittedChanges(t *testing.T) {
	dir := initTestRepo(t)

	if HasUncommittedChanges(dir) {
		t.Fatal("clean repo should have no uncommitted changes")
	}

	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("data"), 0o644)
	if !HasUncommittedChanges(dir) {
		t.Fatal("should detect uncommitted changes after adding file")
	}
}

func TestCommitsAhead(t *testing.T) {
	dir := initTestRepo(t)
	base := CurrentBranch(dir)

	// Create a branch with a commit
	exec.Command("git", "checkout", "-b", "test-ahead").Run()
	branchCmd := exec.Command("git", "checkout", "-b", "test-ahead")
	branchCmd.Dir = dir
	branchCmd.Run()

	os.WriteFile(filepath.Join(dir, "extra.txt"), []byte("data"), 0o644)
	AutoCommit(dir, "extra commit")

	n := CommitsAhead(dir, base, "test-ahead")
	if n != 1 {
		t.Errorf("CommitsAhead = %d, want 1", n)
	}
}

func TestPruneOrphaned(t *testing.T) {
	dir := initTestRepo(t)

	// Use a separate temp dir for worktrees (outside the repo) to avoid
	// git rev-parse finding the parent repo's .git.
	worktreesDir := filepath.Join(t.TempDir(), "worktrees")
	os.MkdirAll(worktreesDir, 0o755)

	// Create a fake dir that's not a worktree
	orphanDir := filepath.Join(worktreesDir, "orphan")
	os.MkdirAll(orphanDir, 0o755)

	PruneOrphaned(dir, worktreesDir)

	if _, err := os.Stat(orphanDir); !os.IsNotExist(err) {
		t.Error("orphaned dir should have been removed")
	}
}
