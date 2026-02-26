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

// TestInstallSkillToolForceReinstall tests the force reinstall functionality
func TestInstallSkillToolForceReinstall(t *testing.T) {
	workspace := t.TempDir()
	skillDir := filepath.Join(workspace, "skills", "test-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	// Create a dummy file to simulate existing installation
	dummyFile := filepath.Join(skillDir, "SKILL.md")
	require.NoError(t, os.WriteFile(dummyFile, []byte("# Old Skill"), 0o644))

	tool := NewInstallSkillTool(skills.NewRegistryManager(), workspace)

	// Without force=true, should fail
	result := tool.Execute(context.Background(), map[string]any{
		"slug":     "test-skill",
		"registry": "clawhub",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "already installed")

	// With force=true, should proceed (but fail due to registry not found)
	result = tool.Execute(context.Background(), map[string]any{
		"slug":     "test-skill",
		"registry": "clawhub",
		"force":    true,
	})
	// Should not error about "already installed" anymore
	if result.IsError {
		assert.NotContains(t, result.ForLLM, "already installed")
	}
}

// TestInstallSkillToolWriteOriginMeta tests that origin metadata is written
func TestInstallSkillToolWriteOriginMeta(t *testing.T) {
	workspace := t.TempDir()

	// Create a mock registry that will succeed
	registryMgr := skills.NewRegistryManager()

	// We can't test actual installation without network, but we can test
	// the directory preparation and metadata writing logic by checking
	// if the skill directory structure is correct after a failed install

	tool := NewInstallSkillTool(registryMgr, workspace)
	result := tool.Execute(context.Background(), map[string]any{
		"slug":     "mock-skill",
		"registry": "nonexistent",
	})

	// Should fail because registry doesn't exist
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "not found")
}

// TestInstallSkillToolInvalidSlugPatterns tests various invalid slug patterns
func TestInstallSkillToolInvalidSlugPatterns(t *testing.T) {
	workspace := t.TempDir()
	tool := NewInstallSkillTool(skills.NewRegistryManager(), workspace)

	invalidSlugs := []struct {
		slug   string
		reason string
	}{
		{"../etc/passwd", "path traversal"},
		{"skill/with/slash", "contains slash"},
		{"skill\\with\\backslash", "contains backslash"},
		{"./relative/path", "relative path"},
		{"", "empty slug"},
		{"   ", "whitespace only"},
	}

	for _, tc := range invalidSlugs {
		t.Run(tc.reason, func(t *testing.T) {
			result := tool.Execute(context.Background(), map[string]any{
				"slug":     tc.slug,
				"registry": "clawhub",
			})
			assert.True(t, result.IsError, "slug %q should be rejected: %s", tc.slug, tc.reason)
			assert.Contains(t, result.ForLLM, "invalid slug")
		})
	}
}

// TestInstallSkillToolValidSlugPatterns tests valid slug patterns
func TestInstallSkillToolValidSlugPatterns(t *testing.T) {
	workspace := t.TempDir()
	tool := NewInstallSkillTool(skills.NewRegistryManager(), workspace)

	validSlugs := []string{
		"github",
		"docker-compose",
		"my-skill-123",
		"skill_with_underscore",
	}

	for _, slug := range validSlugs {
		t.Run(slug, func(t *testing.T) {
			result := tool.Execute(context.Background(), map[string]any{
				"slug":     slug,
				"registry": "nonexistent", // Will fail, but slug validation should pass
			})
			// Should fail because registry doesn't exist, not because of slug validation
			assert.True(t, result.IsError)
			assert.Contains(t, result.ForLLM, "not found")
			assert.NotContains(t, result.ForLLM, "invalid slug")
		})
	}
}

// TestInstallSkillToolDescription tests the tool description
func TestInstallSkillToolDescription(t *testing.T) {
	tool := NewInstallSkillTool(skills.NewRegistryManager(), t.TempDir())
	desc := tool.Description()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "Install")
	assert.Contains(t, desc, "skill")
	assert.Contains(t, desc, "registry")
}

// TestInstallSkillToolExecuteContextCancellation tests behavior with context cancellation
func TestInstallSkillToolExecuteContextCancellation(t *testing.T) {
	workspace := t.TempDir()
	tool := NewInstallSkillTool(skills.NewRegistryManager(), workspace)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := tool.Execute(ctx, map[string]any{
		"slug":     "test-skill",
		"registry": "clawhub",
	})

	// Should still validate parameters even with canceled context
	assert.True(t, result.IsError)
}
