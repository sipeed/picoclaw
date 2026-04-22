package integrationtools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/pkg/skills"
)

type mockInstallRegistry struct{}

const validSkillMarkdown = "---\nname: pr-review\ndescription: Review pull requests\n---\n# PR Review\n"

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
	_ context.Context,
	_ string,
	_ string,
	targetDir string,
) (*skills.InstallResult, error) {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(targetDir, "SKILL.md"), []byte(validSkillMarkdown), 0o600); err != nil {
		return nil, err
	}
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
	_ context.Context,
	_ string,
	_ string,
	targetDir string,
) (*skills.InstallResult, error) {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(targetDir, "SKILL.md"), []byte(validSkillMarkdown), 0o600); err != nil {
		return nil, err
	}
	return &skills.InstallResult{Version: "main"}, nil
}

type stubGitHubInstallRegistry struct {
	*skills.GitHubRegistry
}

func (m *stubGitHubInstallRegistry) DownloadAndInstall(
	_ context.Context,
	_ string,
	_ string,
	targetDir string,
) (*skills.InstallResult, error) {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(targetDir, "SKILL.md"), []byte(validSkillMarkdown), 0o600); err != nil {
		return nil, err
	}
	return &skills.InstallResult{Version: "main"}, nil
}

type mockInvalidInstallRegistry struct{}

type mockFailingInstallRegistry struct{}

func (m *mockInvalidInstallRegistry) Name() string { return "clawhub" }

func (m *mockInvalidInstallRegistry) ResolveInstallDirName(target string) (string, error) {
	return target, nil
}

func (m *mockInvalidInstallRegistry) SkillURL(slug, _ string) string { return slug }

func (m *mockInvalidInstallRegistry) Search(context.Context, string, int) ([]skills.SearchResult, error) {
	return nil, nil
}

func (m *mockInvalidInstallRegistry) GetSkillMeta(context.Context, string) (*skills.SkillMeta, error) {
	return nil, nil
}

func (m *mockInvalidInstallRegistry) DownloadAndInstall(
	_ context.Context,
	_ string,
	_ string,
	targetDir string,
) (*skills.InstallResult, error) {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(
		filepath.Join(targetDir, "SKILL.md"),
		[]byte("---\nname: bad_skill\ndescription: invalid name\n---\n# Invalid\n"),
		0o600,
	); err != nil {
		return nil, err
	}
	return &skills.InstallResult{Version: "test"}, nil
}

func (m *mockFailingInstallRegistry) Name() string { return "clawhub" }

func (m *mockFailingInstallRegistry) ResolveInstallDirName(target string) (string, error) {
	return target, nil
}

func (m *mockFailingInstallRegistry) SkillURL(slug, _ string) string { return slug }

func (m *mockFailingInstallRegistry) Search(context.Context, string, int) ([]skills.SearchResult, error) {
	return nil, nil
}

func (m *mockFailingInstallRegistry) GetSkillMeta(context.Context, string) (*skills.SkillMeta, error) {
	return nil, nil
}

func (m *mockFailingInstallRegistry) DownloadAndInstall(
	_ context.Context,
	_ string,
	_ string,
	_ string,
) (*skills.InstallResult, error) {
	return nil, assert.AnError
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
	registry := skills.GitHubRegistryConfig{Enabled: true, BaseURL: "https://github.com"}.BuildRegistry()
	githubRegistry, ok := registry.(*skills.GitHubRegistry)
	require.True(t, ok)

	registryMgr := skills.NewRegistryManager()
	registryMgr.AddRegistry(&stubGitHubInstallRegistry{GitHubRegistry: githubRegistry})
	workspace := t.TempDir()
	tool := NewInstallSkillTool(registryMgr, workspace)

	slug := "https://github.com/synthetic-lab/octofriend/tree/main/.agents/skills/pr-review"
	result := tool.Execute(context.Background(), map[string]any{
		"slug":     slug,
		"registry": "github",
	})

	assert.False(t, result.IsError)
	assert.Contains(t, result.ForLLM, `Successfully installed skill`)

	data, err := os.ReadFile(filepath.Join(workspace, "skills", "pr-review", ".skill-origin.json"))
	require.NoError(t, err)

	var meta originMeta
	require.NoError(t, json.Unmarshal(data, &meta))
	assert.Equal(t, "third_party", meta.OriginKind)
	assert.Equal(t, "github", meta.Registry)
	assert.Equal(t, "synthetic-lab/octofriend/.agents/skills/pr-review", meta.Slug)
	assert.Equal(t, slug, meta.RegistryURL)
	assert.Equal(t, "main", meta.InstalledVersion)
	assert.NotZero(t, meta.InstalledAt)
}

func TestInstallSkillToolPreservesGitHubSourceURLWithEnterpriseRegistry(t *testing.T) {
	registry := skills.GitHubRegistryConfig{Enabled: true, BaseURL: "https://ghe.example.com/git"}.BuildRegistry()
	githubRegistry, ok := registry.(*skills.GitHubRegistry)
	require.True(t, ok)

	registryMgr := skills.NewRegistryManager()
	registryMgr.AddRegistry(&stubGitHubInstallRegistry{GitHubRegistry: githubRegistry})
	workspace := t.TempDir()
	tool := NewInstallSkillTool(registryMgr, workspace)

	slug := "https://github.com/synthetic-lab/octofriend/tree/main/.agents/skills/pr-review"
	result := tool.Execute(context.Background(), map[string]any{
		"slug":     slug,
		"registry": "github",
	})

	assert.False(t, result.IsError)

	data, err := os.ReadFile(filepath.Join(workspace, "skills", "pr-review", ".skill-origin.json"))
	require.NoError(t, err)

	var meta originMeta
	require.NoError(t, json.Unmarshal(data, &meta))
	assert.Equal(t, "synthetic-lab/octofriend/.agents/skills/pr-review", meta.Slug)
	assert.Equal(t, slug, meta.RegistryURL)
	assert.Equal(t, "main", meta.InstalledVersion)
}

func TestInstallSkillToolRejectsInvalidInstalledSkill(t *testing.T) {
	workspace := t.TempDir()
	registryMgr := skills.NewRegistryManager()
	registryMgr.AddRegistry(&mockInvalidInstallRegistry{})
	tool := NewInstallSkillTool(registryMgr, workspace)

	result := tool.Execute(context.Background(), map[string]any{
		"slug":     "broken-skill",
		"registry": "clawhub",
	})

	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "not a valid skill")
	_, err := os.Stat(filepath.Join(workspace, "skills", "broken-skill"))
	assert.True(t, os.IsNotExist(err))
}

func TestInstallSkillToolRollsBackOnOriginMetadataWriteFailure(t *testing.T) {
	workspace := t.TempDir()
	registryMgr := skills.NewRegistryManager()
	registryMgr.AddRegistry(&mockInstallRegistry{})
	tool := NewInstallSkillTool(registryMgr, workspace)

	previousPersist := persistInstalledSkillOriginMeta
	persistInstalledSkillOriginMeta = func(string, skills.SkillRegistry, string, string) error {
		return assert.AnError
	}
	defer func() {
		persistInstalledSkillOriginMeta = previousPersist
	}()

	result := tool.Execute(context.Background(), map[string]any{
		"slug":     "rollback-skill",
		"registry": "clawhub",
	})

	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "failed to persist skill metadata")
	_, err := os.Stat(filepath.Join(workspace, "skills", "rollback-skill"))
	assert.True(t, os.IsNotExist(err))
}

func TestInstallSkillToolForceReinstallRestoresPreviousSkillAfterDownloadFailure(t *testing.T) {
	workspace := t.TempDir()
	skillDir := filepath.Join(workspace, "skills", "existing-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	oldContent := []byte("---\nname: existing-skill\ndescription: Existing skill\n---\n# Existing\n")
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), oldContent, 0o600))

	registryMgr := skills.NewRegistryManager()
	registryMgr.AddRegistry(&mockFailingInstallRegistry{})
	tool := NewInstallSkillTool(registryMgr, workspace)

	result := tool.Execute(context.Background(), map[string]any{
		"slug":     "existing-skill",
		"registry": "clawhub",
		"force":    true,
	})

	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "failed to install")

	gotContent, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	require.NoError(t, err)
	assert.Equal(t, oldContent, gotContent)
}

func TestInstallSkillToolForceReinstallRestoresPreviousSkillAfterMetadataFailure(t *testing.T) {
	workspace := t.TempDir()
	skillDir := filepath.Join(workspace, "skills", "existing-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	oldContent := []byte("---\nname: existing-skill\ndescription: Existing skill\n---\n# Existing\n")
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), oldContent, 0o600))

	registryMgr := skills.NewRegistryManager()
	registryMgr.AddRegistry(&mockInstallRegistry{})
	tool := NewInstallSkillTool(registryMgr, workspace)

	previousPersist := persistInstalledSkillOriginMeta
	persistInstalledSkillOriginMeta = func(string, skills.SkillRegistry, string, string) error {
		return assert.AnError
	}
	defer func() {
		persistInstalledSkillOriginMeta = previousPersist
	}()

	result := tool.Execute(context.Background(), map[string]any{
		"slug":     "existing-skill",
		"registry": "clawhub",
		"force":    true,
	})

	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "failed to persist skill metadata")

	gotContent, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	require.NoError(t, err)
	assert.Equal(t, oldContent, gotContent)
}
