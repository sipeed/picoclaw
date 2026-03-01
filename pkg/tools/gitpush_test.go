package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/git"
)

// TestGitPushTool_NoWorktree verifies that git_push fails without worktree context.
func TestGitPushTool_NoWorktree(t *testing.T) {
	tool := NewGitPushTool()

	result := tool.Execute(context.Background(), map[string]any{})
	if !result.IsError {
		t.Fatal("expected error when no worktree in context")
	}
	if result.ForLLM == "" {
		t.Fatal("error message should not be empty")
	}
	// Verify helpful guidance is included
	assertContains(t, result.ForLLM, "worktree")
	assertContains(t, result.ForLLM, "heartbeat")
}

// TestGitPushTool_ProtectedBranch verifies that protected branches are blocked.
func TestGitPushTool_ProtectedBranch(t *testing.T) {
	tool := NewGitPushTool()

	protectedNames := []string{"main", "master", "develop", "release/v1.0"}

	for _, branch := range protectedNames {
		t.Run(branch, func(t *testing.T) {
			ctx := WithWorktreeInfo(context.Background(), &git.WorktreeInfo{
				Branch:     branch,
				BaseBranch: "main",
				Path:       t.TempDir(),
				RepoRoot:   t.TempDir(),
			})
			result := tool.Execute(ctx, map[string]any{})
			if !result.IsError {
				t.Fatalf("expected error for protected branch %q", branch)
			}
			assertContains(t, result.ForLLM, "protected")
			assertContains(t, result.ForLLM, branch)
		})
	}
}

// TestGitPushTool_EmptyBranch verifies that empty branch name is rejected.
func TestGitPushTool_EmptyBranch(t *testing.T) {
	tool := NewGitPushTool()

	ctx := WithWorktreeInfo(context.Background(), &git.WorktreeInfo{
		Branch:     "",
		BaseBranch: "main",
		Path:       t.TempDir(),
		RepoRoot:   t.TempDir(),
	})
	result := tool.Execute(ctx, map[string]any{})
	if !result.IsError {
		t.Fatal("expected error for empty branch")
	}
	assertContains(t, result.ForLLM, "no branch name")
}

// TestGitPushTool_AllowedBranch verifies that non-protected branches pass the branch check.
// (Push itself will fail because there's no real git repo, but it should get past validation.)
func TestGitPushTool_AllowedBranch(t *testing.T) {
	tool := NewGitPushTool()

	allowedNames := []string{"plan/add-auth", "feature/foo", "worktree/test"}

	for _, branch := range allowedNames {
		t.Run(branch, func(t *testing.T) {
			ctx := WithWorktreeInfo(context.Background(), &git.WorktreeInfo{
				Branch:     branch,
				BaseBranch: "main",
				Path:       t.TempDir(),
				RepoRoot:   t.TempDir(),
			})
			result := tool.Execute(ctx, map[string]any{})
			// Should NOT fail with "protected branch" error
			if result.IsError && strings.Contains(result.ForLLM, "protected") {
				t.Fatalf("branch %q should not be blocked as protected", branch)
			}
		})
	}
}

// TestProtectedBranchesRegex tests the regex directly.
func TestProtectedBranchesRegex(t *testing.T) {
	tests := []struct {
		branch    string
		protected bool
	}{
		{"main", true},
		{"master", true},
		{"develop", true},
		{"release/v1.0", true},
		{"release/2026-03", true},
		{"plan/add-feature", false},
		{"feature/main", false}, // "main" not at start
		{"main-backup", false},  // "main" followed by suffix
		{"hotfix/urgent", false},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			got := protectedBranches.MatchString(tt.branch)
			if got != tt.protected {
				t.Errorf("branch %q: got protected=%v, want %v", tt.branch, got, tt.protected)
			}
		})
	}
}

// TestWorktreeInfoContext verifies context round-trip.
func TestWorktreeInfoContext(t *testing.T) {
	wt := &git.WorktreeInfo{
		Branch:     "plan/test",
		BaseBranch: "main",
		Path:       "/tmp/wt",
		RepoRoot:   "/tmp/repo",
	}

	ctx := WithWorktreeInfo(context.Background(), wt)
	got := WorktreeInfoFromCtx(ctx)
	if got == nil {
		t.Fatal("expected non-nil WorktreeInfo from context")
	}
	if got.Branch != wt.Branch {
		t.Errorf("Branch: got %q, want %q", got.Branch, wt.Branch)
	}
	if got.BaseBranch != wt.BaseBranch {
		t.Errorf("BaseBranch: got %q, want %q", got.BaseBranch, wt.BaseBranch)
	}

	// Nil case
	got2 := WorktreeInfoFromCtx(context.Background())
	if got2 != nil {
		t.Errorf("expected nil WorktreeInfo from bare context, got %+v", got2)
	}
}

// TestGitPushTool_Interface verifies the tool satisfies the Tool interface.
func TestGitPushTool_Interface(t *testing.T) {
	var _ Tool = (*GitPushTool)(nil)

	tool := NewGitPushTool()
	if tool.Name() != "git_push" {
		t.Errorf("Name: got %q, want %q", tool.Name(), "git_push")
	}
	if tool.Description() == "" {
		t.Error("Description should not be empty")
	}
	params := tool.Parameters()
	if params == nil {
		t.Fatal("Parameters should not be nil")
	}
	if params["type"] != "object" {
		t.Errorf("Parameters type: got %v, want object", params["type"])
	}
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected %q to contain %q", s, substr)
	}
}
