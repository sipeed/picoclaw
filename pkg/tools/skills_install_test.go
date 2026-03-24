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

func TestInstallSkillToolName(t *testing.T) {
	tool := NewInstallSkillTool(skills.NewRegistryManager(), t.TempDir(), nil, false)
	assert.Equal(t, "install_skill", tool.Name())
}

func TestInstallSkillToolMissingSlug(t *testing.T) {
	tool := NewInstallSkillTool(skills.NewRegistryManager(), t.TempDir(), nil, false)
	result := tool.Execute(context.Background(), map[string]any{})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "identifier is required and must be a non-empty string")
}

func TestInstallSkillToolEmptySlug(t *testing.T) {
	tool := NewInstallSkillTool(skills.NewRegistryManager(), t.TempDir(), nil, false)
	result := tool.Execute(context.Background(), map[string]any{
		"slug": "   ",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "identifier is required and must be a non-empty string")
}

func TestInstallSkillToolUnsafeSlug(t *testing.T) {
	tool := NewInstallSkillTool(skills.NewRegistryManager(), t.TempDir(), nil, false)

	cases := []string{
		"../etc/passwd",
		"path/traversal",
		"path\\traversal",
	}

	for _, slug := range cases {
		result := tool.Execute(context.Background(), map[string]any{
			"slug": slug,
		})
		assert.True(t, result.IsError, "slug %q should be rejected", slug)
		assert.Contains(t, result.ForLLM, "invalid slug")
	}
}

func TestInstallSkillToolAlreadyExists(t *testing.T) {
	workspace := t.TempDir()
	skillDir := filepath.Join(workspace, "skills", "existing-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	tool := NewInstallSkillTool(skills.NewRegistryManager(), workspace, nil, false)
	result := tool.Execute(context.Background(), map[string]any{
		"slug":     "existing-skill",
		"registry": "clawhub",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "already installed")
}

func TestInstallSkillToolRegistryNotFound(t *testing.T) {
	workspace := t.TempDir()
	tool := NewInstallSkillTool(skills.NewRegistryManager(), workspace, nil, false)
	result := tool.Execute(context.Background(), map[string]any{
		"slug":     "some-skill",
		"registry": "nonexistent",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "registry")
	assert.Contains(t, result.ForLLM, "not found")
}

func TestInstallSkillToolParameters(t *testing.T) {
	tool := NewInstallSkillTool(skills.NewRegistryManager(), t.TempDir(), nil, false)
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
	assert.Contains(t, required, "registry")
}

func TestInstallSkillToolMissingRegistry(t *testing.T) {
	tool := NewInstallSkillTool(skills.NewRegistryManager(), t.TempDir(), nil, false)
	result := tool.Execute(context.Background(), map[string]any{
		"slug": "some-skill",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "invalid registry")
}

func TestInstallSkillToolWhitelist(t *testing.T) {
	workspace := t.TempDir()
	rm := skills.NewRegistryManager()

	t.Run("blocked-by-whitelist", func(t *testing.T) {
		tool := NewInstallSkillTool(rm, workspace, []string{"allowed-skill"}, true)
		result := tool.Execute(context.Background(), map[string]any{
			"slug":     "blocked-skill",
			"registry": "clawhub",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, result.ForLLM, "not in whitelist")
	})

	t.Run("allowed-by-whitelist", func(t *testing.T) {
		// This will still fail because registry is not found, but it should pass the whitelist check
		tool := NewInstallSkillTool(rm, workspace, []string{"allowed-skill"}, true)
		result := tool.Execute(context.Background(), map[string]any{
			"slug":     "allowed-skill",
			"registry": "clawhub",
		})
		assert.True(t, result.IsError)
		assert.NotContains(t, result.ForLLM, "not in whitelist")
	})

	t.Run("empty-whitelist-allows-all", func(t *testing.T) {
		tool := NewInstallSkillTool(rm, workspace, []string{}, false)
		result := tool.Execute(context.Background(), map[string]any{
			"slug":     "any-skill",
			"registry": "clawhub",
		})
		assert.True(t, result.IsError)
		assert.NotContains(t, result.ForLLM, "not in whitelist")
	})

	t.Run("nil-whitelist-allows-all", func(t *testing.T) {
		tool := NewInstallSkillTool(rm, workspace, nil, false)
		result := tool.Execute(context.Background(), map[string]any{
			"slug":     "any-skill",
			"registry": "clawhub",
		})
		assert.True(t, result.IsError)
		assert.NotContains(t, result.ForLLM, "not in whitelist")
	})
}
