package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/pkg/skills"
)

type mockInstallRegistry struct{}

func (m *mockInstallRegistry) Name() string { return "clawhub" }

func (m *mockInstallRegistry) ResolveInstallDirName(target string) (string, error) {
	return target, nil
}

func (m *mockInstallRegistry) SkillURL(slug, _ string) string { return slug }

func (m *mockInstallRegistry) Search(context.Context, string, int) ([]skills.SearchResult, error) {
	return nil, nil
}

func (m *mockInstallRegistry) GetSkillMeta(context.Context, string) (*skills.SkillMeta, error) {
	return nil, nil
}

func (m *mockInstallRegistry) DownloadAndInstall(
	context.Context,
	string,
	string,
	string,
) (*skills.InstallResult, error) {
	return &skills.InstallResult{Version: "test"}, nil
}

type mockGitHubInstallRegistry struct{}

func (m *mockGitHubInstallRegistry) Name() string { return "github" }

func (m *mockGitHubInstallRegistry) ResolveInstallDirName(target string) (string, error) {
	return "pr-review", nil
}

func (m *mockGitHubInstallRegistry) SkillURL(slug, _ string) string { return slug }

func (m *mockGitHubInstallRegistry) Search(context.Context, string, int) ([]skills.SearchResult, error) {
	return nil, nil
}

func (m *mockGitHubInstallRegistry) GetSkillMeta(context.Context, string) (*skills.SkillMeta, error) {
	return nil, nil
}

func (m *mockGitHubInstallRegistry) DownloadAndInstall(
	context.Context,
	string,
	string,
	string,
) (*skills.InstallResult, error) {
	return &skills.InstallResult{Version: "main"}, nil
}

func TestInstallSkillToolName(t *testing.T) {
	tool := NewInstallSkillTool(skills.NewRegistryManager(), t.TempDir())
	assert.Equal(t, "install_skill", tool.Name())
}

func TestInstallSkillToolMissingSlug(t *testing.T) {
	tool := NewInstallSkillTool(skills.NewRegistryManager(), t.TempDir())
	result := tool.Execute(context.Background(), map[string]any{})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "identifier is required and must be a non-empty string")
}

func TestInstallSkillToolEmptySlug(t *testing.T) {
	tool := NewInstallSkillTool(skills.NewRegistryManager(), t.TempDir())
	result := tool.Execute(context.Background(), map[string]any{
		"slug": "   ",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "identifier is required and must be a non-empty string")
}

func TestInstallSkillToolUnsafeSlug(t *testing.T) {
	registryMgr := skills.NewRegistryManager()
	registryMgr.AddRegistry(skills.NewClawHubRegistry(skills.ClawHubConfig{Enabled: true}))
	tool := NewInstallSkillTool(registryMgr, t.TempDir())

	cases := []string{
		"../etc/passwd",
		"path/traversal",
		"path\\traversal",
	}

	for _, slug := range cases {
		result := tool.Execute(context.Background(), map[string]any{
			"slug":     slug,
			"registry": "clawhub",
		})
		assert.True(t, result.IsError, "slug %q should be rejected", slug)
		assert.Contains(t, result.ForLLM, "invalid slug")
	}
}

func TestInstallSkillToolAlreadyExists(t *testing.T) {
	workspace := t.TempDir()
	skillDir := filepath.Join(workspace, "skills", "existing-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	registryMgr := skills.NewRegistryManager()
	registryMgr.AddRegistry(&mockInstallRegistry{})
	tool := NewInstallSkillTool(registryMgr, workspace)
	result := tool.Execute(context.Background(), map[string]any{
		"slug":     "existing-skill",
		"registry": "clawhub",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "already installed")
}

func TestInstallSkillToolRegistryNotFound(t *testing.T) {
	workspace := t.TempDir()
	tool := NewInstallSkillTool(skills.NewRegistryManager(), workspace)
	result := tool.Execute(context.Background(), map[string]any{
		"slug":     "some-skill",
		"registry": "nonexistent",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "registry")
	assert.Contains(t, result.ForLLM, "not found")
}

func TestInstallSkillToolParameters(t *testing.T) {
	tool := NewInstallSkillTool(skills.NewRegistryManager(), t.TempDir())
	params := tool.Parameters()

	props, ok := params["properties"].(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, props, "slug")
	assert.Contains(t, props, "version")
	assert.Contains(t, props, "registry")
	assert.Contains(t, props, "force")

	required, ok := params["required"].([]string)
	assert.True(t, ok)
	assert.Contains(t, required, "slug")
	assert.NotContains(t, required, "registry")
}

func TestInstallSkillToolMissingRegistry(t *testing.T) {
	registryMgr := skills.NewRegistryManager()
	registryMgr.AddRegistry(&mockGitHubInstallRegistry{})
	tool := NewInstallSkillTool(registryMgr, t.TempDir())
	result := tool.Execute(context.Background(), map[string]any{
		"slug": "some-skill",
	})
	assert.False(t, result.IsError)
	assert.Contains(t, result.ForLLM, `Successfully installed skill`)
}

func TestInstallSkillToolAllowsGitHubURLSlug(t *testing.T) {
	registryMgr := skills.NewRegistryManager()
	registryMgr.AddRegistry(&mockGitHubInstallRegistry{})
	tool := NewInstallSkillTool(registryMgr, t.TempDir())

	result := tool.Execute(context.Background(), map[string]any{
		"slug":     "https://github.com/synthetic-lab/octofriend/tree/main/.agents/skills/pr-review",
		"registry": "github",
	})

	assert.False(t, result.IsError)
	assert.Contains(t, result.ForLLM, `Successfully installed skill`)
}
