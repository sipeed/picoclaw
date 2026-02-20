package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallSkillToolName(t *testing.T) {
	tool := NewInstallSkillTool(skills.NewRegistryManager(), t.TempDir())
	assert.Equal(t, "install_skill", tool.Name())
}

func TestInstallSkillToolMissingSlug(t *testing.T) {
	tool := NewInstallSkillTool(skills.NewRegistryManager(), t.TempDir())
	result := tool.Execute(context.Background(), map[string]interface{}{})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "identifier is required and must be a non-empty string")
}

func TestInstallSkillToolEmptySlug(t *testing.T) {
	tool := NewInstallSkillTool(skills.NewRegistryManager(), t.TempDir())
	result := tool.Execute(context.Background(), map[string]interface{}{
		"slug": "   ",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "identifier is required and must be a non-empty string")
}

func TestInstallSkillToolUnsafeSlug(t *testing.T) {
	tool := NewInstallSkillTool(skills.NewRegistryManager(), t.TempDir())

	cases := []string{
		"../etc/passwd",
		"path/traversal",
		"path\\traversal",
	}

	for _, slug := range cases {
		result := tool.Execute(context.Background(), map[string]interface{}{
			"slug": slug,
		})
		assert.True(t, result.IsError, "slug %q should be rejected", slug)
		assert.Contains(t, result.ForLLM, "invalid slug")
	}
}

func TestInstallSkillToolAlreadyExists(t *testing.T) {
	workspace := t.TempDir()
	skillDir := filepath.Join(workspace, "skills", "existing-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))

	tool := NewInstallSkillTool(skills.NewRegistryManager(), workspace)
	result := tool.Execute(context.Background(), map[string]interface{}{
		"slug":     "existing-skill",
		"registry": "clawhub",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "already installed")
}

func TestInstallSkillToolRegistryNotFound(t *testing.T) {
	workspace := t.TempDir()
	tool := NewInstallSkillTool(skills.NewRegistryManager(), workspace)
	result := tool.Execute(context.Background(), map[string]interface{}{
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

	props, ok := params["properties"].(map[string]interface{})
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
	tool := NewInstallSkillTool(skills.NewRegistryManager(), t.TempDir())
	result := tool.Execute(context.Background(), map[string]interface{}{
		"slug": "some-skill",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "invalid registry")
}
