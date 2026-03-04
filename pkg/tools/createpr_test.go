package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/git"
)

// TestCreatePRTool_NoWorktree verifies that create_pr fails without worktree context.

func TestCreatePRTool_NoWorktree(t *testing.T) {
	tool := NewCreatePRTool()

	result := tool.Execute(context.Background(), map[string]any{
		"title": "Test PR",
	})

	if !result.IsError {
		t.Fatal("expected error when no worktree in context")
	}

	assertContains(t, result.ForLLM, "worktree")

	assertContains(t, result.ForLLM, "heartbeat")
}

// TestCreatePRTool_EmptyBranch verifies that empty branch name is rejected.

func TestCreatePRTool_EmptyBranch(t *testing.T) {
	tool := NewCreatePRTool()

	ctx := WithWorktreeInfo(context.Background(), &git.WorktreeInfo{
		Branch: "",

		BaseBranch: "main",

		Path: t.TempDir(),

		RepoRoot: t.TempDir(),
	})

	result := tool.Execute(ctx, map[string]any{
		"title": "Test PR",
	})

	if !result.IsError {
		t.Fatal("expected error for empty branch")
	}

	assertContains(t, result.ForLLM, "no branch name")
}

// TestCreatePRTool_MissingTitle verifies that missing title is rejected.

func TestCreatePRTool_MissingTitle(t *testing.T) {
	tool := NewCreatePRTool()

	ctx := WithWorktreeInfo(context.Background(), &git.WorktreeInfo{
		Branch: "plan/test",

		BaseBranch: "main",

		Path: t.TempDir(),

		RepoRoot: t.TempDir(),
	})

	tests := []struct {
		name string

		args map[string]any
	}{
		{"no title key", map[string]any{}},

		{"empty title", map[string]any{"title": ""}},

		{"whitespace title", map[string]any{"title": "   "}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.Execute(ctx, tt.args)

			if !result.IsError {
				t.Fatal("expected error for missing/empty title")
			}

			assertContains(t, result.ForLLM, "title is required")
		})
	}
}

// TestCreatePRTool_BranchNotPushed verifies the tool checks for remote branch existence.

func TestCreatePRTool_BranchNotPushed(t *testing.T) {
	tool := NewCreatePRTool()

	ctx := WithWorktreeInfo(context.Background(), &git.WorktreeInfo{
		Branch: "plan/not-pushed",

		BaseBranch: "main",

		Path: t.TempDir(),

		RepoRoot: t.TempDir(),
	})

	result := tool.Execute(ctx, map[string]any{
		"title": "Test PR",
	})

	if !result.IsError {
		t.Fatal("expected error for unpushed branch")
	}

	// Should mention git_push as the remedy

	assertContains(t, result.ForLLM, "git_push")
}

// TestCreatePRTool_DefaultBaseBranch verifies fallback to "main" when BaseBranch is empty.

func TestCreatePRTool_DefaultBaseBranch(t *testing.T) {
	tool := NewCreatePRTool()

	// With empty BaseBranch, tool should default to "main"

	ctx := WithWorktreeInfo(context.Background(), &git.WorktreeInfo{
		Branch: "plan/test",

		BaseBranch: "",

		Path: t.TempDir(),

		RepoRoot: t.TempDir(),
	})

	result := tool.Execute(ctx, map[string]any{
		"title": "Test PR",
	})

	// Will fail at ls-remote (no real repo), but should not fail at baseBranch validation

	if result.IsError && strings.Contains(result.ForLLM, "base branch") {
		t.Fatal("should not fail on base branch when defaulting to main")
	}
}

// TestCreatePRTool_Interface verifies the tool satisfies both Tool and AsyncTool interfaces.

func TestCreatePRTool_Interface(t *testing.T) {
	var _ Tool = (*CreatePRTool)(nil)

	var _ AsyncTool = (*CreatePRTool)(nil)

	tool := NewCreatePRTool()

	if tool.Name() != "create_pr" {
		t.Errorf("Name: got %q, want %q", tool.Name(), "create_pr")
	}

	if tool.Description() == "" {
		t.Error("Description should not be empty")
	}

	params := tool.Parameters()

	if params == nil {
		t.Fatal("Parameters should not be nil")
	}

	// Verify "title" is required

	required, ok := params["required"].([]string)

	if !ok {
		t.Fatal("required should be []string")
	}

	foundTitle := false

	for _, r := range required {
		if r == "title" {
			foundTitle = true
		}
	}

	if !foundTitle {
		t.Error("title should be in required parameters")
	}
}

// TestCreatePRTool_SetCallback verifies callback is stored.

func TestCreatePRTool_SetCallback(t *testing.T) {
	tool := NewCreatePRTool()

	if tool.callback != nil {
		t.Fatal("callback should be nil initially")
	}

	called := false

	tool.SetCallback(func(ctx context.Context, result *ToolResult) {
		called = true
	})

	if tool.callback == nil {
		t.Fatal("callback should be set after SetCallback")
	}

	// Verify it's callable (doesn't panic)

	tool.callback(context.Background(), NewToolResult("test"))

	if !called {
		t.Fatal("callback was not invoked")
	}
}

// TestCheckPRChecks_ParseResults tests CI status parsing logic.

func TestCheckPRChecks_ParseResults(t *testing.T) {
	// This tests the parsing logic conceptually — actual `gh` calls

	// would need integration tests. We verify the status constants exist

	// and the type is usable.

	if ciStatusPending != 0 {
		t.Error("ciStatusPending should be 0 (default)")
	}

	if ciStatusPass == ciStatusFail {
		t.Error("ciStatusPass and ciStatusFail should differ")
	}

	if ciStatusNone == ciStatusPending {
		t.Error("ciStatusNone and ciStatusPending should differ")
	}
}

// TestAllowedToolsForPreset_GitTools checks git tools are correctly assigned to presets.

func TestAllowedToolsForPreset_GitTools(t *testing.T) {
	tests := []struct {
		name string

		preset Preset

		wantGitPush bool

		wantCreatePR bool
	}{
		{"scout", PresetScout, false, false},

		{"analyst", PresetAnalyst, false, false},

		{"coder", PresetCoder, true, false},

		{"worker", PresetWorker, true, true},

		{"coordinator", PresetCoordinator, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := AllowedToolsForPreset(tt.preset)

			if got := allowed["git_push"]; got != tt.wantGitPush {
				t.Errorf("git_push: got %v, want %v", got, tt.wantGitPush)
			}

			if got := allowed["create_pr"]; got != tt.wantCreatePR {
				t.Errorf("create_pr: got %v, want %v", got, tt.wantCreatePR)
			}
		})
	}
}
