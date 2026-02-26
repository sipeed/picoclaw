package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillsLoaderListSkillsEmpty(t *testing.T) {
	workspace := t.TempDir()
	globalSkills := t.TempDir()
	builtinSkills := t.TempDir()

	loader := NewSkillsLoader(workspace, globalSkills, builtinSkills)
	skills := loader.ListSkills()
	assert.Empty(t, skills)
}

func TestSkillsLoaderListSkillsWorkspace(t *testing.T) {
	workspace := t.TempDir()
	globalSkills := t.TempDir()
	builtinSkills := t.TempDir()

	// Create workspace skill (name must be alphanumeric with hyphens only)
	skillDir := filepath.Join(workspace, "skills", "test-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	skillFile := filepath.Join(skillDir, "SKILL.md")
	content := `---
name: test-skill
description: A test skill for unit testing
---

# Test Skill Content
This is the skill content.
`
	require.NoError(t, os.WriteFile(skillFile, []byte(content), 0o644))

	loader := NewSkillsLoader(workspace, globalSkills, builtinSkills)
	skills := loader.ListSkills()

	assert.Len(t, skills, 1)
	assert.Equal(t, "test-skill", skills[0].Name)
	assert.Equal(t, "A test skill for unit testing", skills[0].Description)
	assert.Equal(t, "workspace", skills[0].Source)
}

func TestSkillsLoaderListSkillsGlobal(t *testing.T) {
	workspace := t.TempDir()
	globalSkills := t.TempDir()
	builtinSkills := t.TempDir()

	// Create global skill
	skillDir := filepath.Join(globalSkills, "global-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	skillFile := filepath.Join(skillDir, "SKILL.md")
	content := `---
name: global-skill
description: A global skill
---

# Global Skill
`
	require.NoError(t, os.WriteFile(skillFile, []byte(content), 0o644))

	loader := NewSkillsLoader(workspace, globalSkills, builtinSkills)
	skills := loader.ListSkills()

	assert.Len(t, skills, 1)
	assert.Equal(t, "global-skill", skills[0].Name)
	assert.Equal(t, "global", skills[0].Source)
}

func TestSkillsLoaderListSkillsBuiltin(t *testing.T) {
	workspace := t.TempDir()
	globalSkills := t.TempDir()
	builtinSkills := t.TempDir()

	// Create builtin skill
	skillDir := filepath.Join(builtinSkills, "builtin-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	skillFile := filepath.Join(skillDir, "SKILL.md")
	content := `---
name: builtin-skill
description: A builtin skill
---

# Builtin Skill
`
	require.NoError(t, os.WriteFile(skillFile, []byte(content), 0o644))

	loader := NewSkillsLoader(workspace, globalSkills, builtinSkills)
	skills := loader.ListSkills()

	assert.Len(t, skills, 1)
	assert.Equal(t, "builtin-skill", skills[0].Name)
	assert.Equal(t, "builtin", skills[0].Source)
}

func TestSkillsLoaderPriority(t *testing.T) {
	workspace := t.TempDir()
	globalSkills := t.TempDir()
	builtinSkills := t.TempDir()

	// Create same skill in all three locations
	createSkill := func(basePath, name string) {
		skillDir := filepath.Join(basePath, "skills", name)
		if basePath == globalSkills || basePath == builtinSkills {
			skillDir = filepath.Join(basePath, name)
		}
		require.NoError(t, os.MkdirAll(skillDir, 0o755))

		skillFile := filepath.Join(skillDir, "SKILL.md")
		content := `---
name: ` + name + `
description: ` + name + ` description
---

# ` + name
		require.NoError(t, os.WriteFile(skillFile, []byte(content), 0o644))
	}

	createSkill(workspace, "override-skill")
	createSkill(globalSkills, "override-skill")
	createSkill(globalSkills, "global-only")
	createSkill(builtinSkills, "override-skill")
	createSkill(builtinSkills, "builtin-only")

	loader := NewSkillsLoader(workspace, globalSkills, builtinSkills)
	skills := loader.ListSkills()

	// Should have 3 skills: override-skill (workspace), global-only, builtin-only
	assert.Len(t, skills, 3)

	// Find override-skill, should be from workspace
	var overrideSkill SkillInfo
	for _, s := range skills {
		if s.Name == "override-skill" {
			overrideSkill = s
			break
		}
	}
	assert.Equal(t, "workspace", overrideSkill.Source)
}

func TestSkillsLoaderLoadSkill(t *testing.T) {
	workspace := t.TempDir()
	globalSkills := t.TempDir()
	builtinSkills := t.TempDir()

	// Create workspace skill
	skillDir := filepath.Join(workspace, "skills", "loadable-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	skillFile := filepath.Join(skillDir, "SKILL.md")
	content := `---
name: loadable-skill
description: Can be loaded
---

# Skill Content
This is the actual skill content.
`
	require.NoError(t, os.WriteFile(skillFile, []byte(content), 0o644))

	loader := NewSkillsLoader(workspace, globalSkills, builtinSkills)

	// Load the skill
	skillContent, ok := loader.LoadSkill("loadable-skill")
	assert.True(t, ok)
	assert.Contains(t, skillContent, "# Skill Content")
	assert.NotContains(t, skillContent, "---") // frontmatter stripped
}

func TestSkillsLoaderLoadSkillNotFound(t *testing.T) {
	workspace := t.TempDir()
	globalSkills := t.TempDir()
	builtinSkills := t.TempDir()

	loader := NewSkillsLoader(workspace, globalSkills, builtinSkills)

	content, ok := loader.LoadSkill("nonexistent")
	assert.False(t, ok)
	assert.Empty(t, content)
}

func TestSkillsLoaderBuildSkillsSummary(t *testing.T) {
	workspace := t.TempDir()
	globalSkills := t.TempDir()
	builtinSkills := t.TempDir()

	// Create multiple skills
	createSkill := func(basePath, name, desc string) {
		skillDir := filepath.Join(basePath, "skills", name)
		require.NoError(t, os.MkdirAll(skillDir, 0o755))

		skillFile := filepath.Join(skillDir, "SKILL.md")
		content := `---
name: ` + name + `
description: ` + desc + `
---

# ` + name
		require.NoError(t, os.WriteFile(skillFile, []byte(content), 0o644))
	}

	createSkill(workspace, "skill1", "First skill")
	createSkill(workspace, "skill2", "Second skill")

	loader := NewSkillsLoader(workspace, globalSkills, builtinSkills)
	summary := loader.BuildSkillsSummary()

	assert.Contains(t, summary, "<skills>")
	assert.Contains(t, summary, "</skills>")
	assert.Contains(t, summary, "skill1")
	assert.Contains(t, summary, "First skill")
	assert.Contains(t, summary, "skill2")
	assert.Contains(t, summary, "Second skill")
}

func TestSkillsLoaderBuildSkillsSummaryEmpty(t *testing.T) {
	workspace := t.TempDir()
	globalSkills := t.TempDir()
	builtinSkills := t.TempDir()

	loader := NewSkillsLoader(workspace, globalSkills, builtinSkills)
	summary := loader.BuildSkillsSummary()

	assert.Empty(t, summary)
}

func TestSkillsLoaderLoadSkillsForContext(t *testing.T) {
	workspace := t.TempDir()
	globalSkills := t.TempDir()
	builtinSkills := t.TempDir()

	// Create skills
	createSkill := func(basePath, name, content string) {
		skillDir := filepath.Join(basePath, "skills", name)
		require.NoError(t, os.MkdirAll(skillDir, 0o755))

		skillFile := filepath.Join(skillDir, "SKILL.md")
		fullContent := `---
name: ` + name + `
description: Desc
---

` + content
		require.NoError(t, os.WriteFile(skillFile, []byte(fullContent), 0o644))
	}

	createSkill(workspace, "ctx-skill1", "# Content 1")
	createSkill(workspace, "ctx-skill2", "# Content 2")

	loader := NewSkillsLoader(workspace, globalSkills, builtinSkills)

	context := loader.LoadSkillsForContext([]string{"ctx-skill1", "ctx-skill2"})
	assert.Contains(t, context, "### Skill: ctx-skill1")
	assert.Contains(t, context, "# Content 1")
	assert.Contains(t, context, "### Skill: ctx-skill2")
	assert.Contains(t, context, "# Content 2")
}

func TestSkillsLoaderLoadSkillsForContextEmpty(t *testing.T) {
	workspace := t.TempDir()
	globalSkills := t.TempDir()
	builtinSkills := t.TempDir()

	loader := NewSkillsLoader(workspace, globalSkills, builtinSkills)
	context := loader.LoadSkillsForContext([]string{})
	assert.Empty(t, context)
}

func TestSkillsLoaderValidateSkill(t *testing.T) {
	tests := []struct {
		name    string
		info    SkillInfo
		wantErr bool
	}{
		{
			name: "valid",
			info: SkillInfo{
				Name:        "valid-skill",
				Description: "A valid skill",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			info: SkillInfo{
				Description: "Missing name",
			},
			wantErr: true,
		},
		{
			name: "missing description",
			info: SkillInfo{
				Name: "no-desc",
			},
			wantErr: true,
		},
		{
			name: "invalid name format",
			info: SkillInfo{
				Name:        "invalid_name",
				Description: "Has underscore",
			},
			wantErr: true,
		},
		{
			name: "name too long",
			info: SkillInfo{
				Name:        string(make([]byte, 100)),
				Description: "Too long name",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.info.validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSkillsLoaderExtractFrontmatter(t *testing.T) {
	workspace := t.TempDir()
	loader := NewSkillsLoader(workspace, "", "")

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "with frontmatter",
			content: `---
name: Test
description: Desc
---

Content`,
			expected: "name: Test\ndescription: Desc",
		},
		{
			name: "without frontmatter",
			content: `# Just content
No frontmatter here`,
			expected: "",
		},
		{
			name:     "windows line endings",
			content:  "---\r\nname: Test\r\ndescription: Desc\r\n---\r\n\r\nContent",
			expected: "name: Test\r\ndescription: Desc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := loader.extractFrontmatter(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSkillsLoaderStripFrontmatter(t *testing.T) {
	workspace := t.TempDir()
	loader := NewSkillsLoader(workspace, "", "")

	content := `---
name: Test
description: Desc
---

# Actual Content
This should remain.`

	stripped := loader.stripFrontmatter(content)
	assert.Contains(t, stripped, "# Actual Content")
	assert.NotContains(t, stripped, "---")
	assert.NotContains(t, stripped, "name: Test")
}

func TestSkillsLoaderParseSimpleYAML(t *testing.T) {
	workspace := t.TempDir()
	loader := NewSkillsLoader(workspace, "", "")

	tests := []struct {
		name     string
		content  string
		expected map[string]string
	}{
		{
			name: "simple key value",
			content: `name: Test
description: A test skill`,
			expected: map[string]string{
				"name":        "Test",
				"description": "A test skill",
			},
		},
		{
			name: "with quotes",
			content: `name: "Quoted Name"
description: 'Single quoted'`,
			expected: map[string]string{
				"name":        "Quoted Name",
				"description": "Single quoted",
			},
		},
		{
			name: "with comments",
			content: `# This is a comment
name: Test
# Another comment
description: Test skill`,
			expected: map[string]string{
				"name":        "Test",
				"description": "Test skill",
			},
		},
		{
			name:    "windows line endings",
			content: "name: Test\r\ndescription: Windows",
			expected: map[string]string{
				"name":        "Test",
				"description": "Windows",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := loader.parseSimpleYAML(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSkillsLoaderGetSkillMetadata(t *testing.T) {
	workspace := t.TempDir()

	// Create skill file
	skillDir := filepath.Join(workspace, "skills", "meta-test")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	skillFile := filepath.Join(skillDir, "SKILL.md")
	content := `---
name: Meta Test Skill
description: Testing metadata extraction
---

# Content`

	require.NoError(t, os.WriteFile(skillFile, []byte(content), 0o644))

	loader := NewSkillsLoader(workspace, "", "")
	metadata := loader.getSkillMetadata(skillFile)

	assert.NotNil(t, metadata)
	assert.Equal(t, "Meta Test Skill", metadata.Name)
	assert.Equal(t, "Testing metadata extraction", metadata.Description)
}

func TestEscapeXML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal text", "normal text"},
		{"text & more", "text &amp; more"},
		{"text < tag", "text &lt; tag"},
		{"text > tag", "text &gt; tag"},
		{"all & < >", "all &amp; &lt; &gt;"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeXML(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
