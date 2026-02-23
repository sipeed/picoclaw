package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillsInfoValidate(t *testing.T) {
	testcases := []struct {
		name        string
		skillName   string
		description string
		wantErr     bool
		errContains []string
	}{
		{
			name:        "valid-skill",
			skillName:   "valid-skill",
			description: "a valid skill description",
			wantErr:     false,
		},
		{
			name:        "empty-name",
			skillName:   "",
			description: "description without name",
			wantErr:     true,
			errContains: []string{"name is required"},
		},
		{
			name:        "empty-description",
			skillName:   "skill-without-description",
			description: "",
			wantErr:     true,
			errContains: []string{"description is required"},
		},
		{
			name:        "empty-both",
			skillName:   "",
			description: "",
			wantErr:     true,
			errContains: []string{"name is required", "description is required"},
		},
		{
			name:        "name-with-spaces",
			skillName:   "skill with spaces",
			description: "invalid name with spaces",
			wantErr:     true,
			errContains: []string{"name must be alphanumeric with hyphens"},
		},
		{
			name:        "name-with-underscore",
			skillName:   "skill_underscore",
			description: "invalid name with underscore",
			wantErr:     true,
			errContains: []string{"name must be alphanumeric with hyphens"},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			info := SkillInfo{
				Name:        tc.skillName,
				Description: tc.description,
			}
			err := info.validate()
			if tc.wantErr {
				assert.Error(t, err)
				for _, msg := range tc.errContains {
					assert.ErrorContains(t, err, msg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExtractFrontmatter(t *testing.T) {
	sl := &SkillsLoader{}

	testcases := []struct {
		name           string
		content        string
		expectedName   string
		expectedDesc   string
		lineEndingType string
	}{
		{
			name:           "unix-line-endings",
			lineEndingType: "Unix (\\n)",
			content:        "---\nname: test-skill\ndescription: A test skill\n---\n\n# Skill Content",
			expectedName:   "test-skill",
			expectedDesc:   "A test skill",
		},
		{
			name:           "windows-line-endings",
			lineEndingType: "Windows (\\r\\n)",
			content:        "---\r\nname: test-skill\r\ndescription: A test skill\r\n---\r\n\r\n# Skill Content",
			expectedName:   "test-skill",
			expectedDesc:   "A test skill",
		},
		{
			name:           "classic-mac-line-endings",
			lineEndingType: "Classic Mac (\\r)",
			content:        "---\rname: test-skill\rdescription: A test skill\r---\r\r# Skill Content",
			expectedName:   "test-skill",
			expectedDesc:   "A test skill",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// Extract frontmatter
			frontmatter := sl.extractFrontmatter(tc.content)
			assert.NotEmpty(t, frontmatter, "Frontmatter should be extracted for %s line endings", tc.lineEndingType)

			// Parse YAML to get name and description (parseSimpleYAML now handles all line ending types)
			yamlMeta := sl.parseSimpleYAML(frontmatter)
			assert.Equal(
				t,
				tc.expectedName,
				yamlMeta["name"],
				"Name should be correctly parsed from frontmatter with %s line endings",
				tc.lineEndingType,
			)
			assert.Equal(
				t,
				tc.expectedDesc,
				yamlMeta["description"],
				"Description should be correctly parsed from frontmatter with %s line endings",
				tc.lineEndingType,
			)
		})
	}
}

// createSkillDir creates a skill directory with a SKILL.md file containing the given frontmatter.
func createSkillDir(t *testing.T, base, dirName, name, description string) {
	t.Helper()
	dir := filepath.Join(base, dirName)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\n\n# " + name
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644))
}

func TestListSkillsWorkspaceOverridesGlobal(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "workspace")
	global := filepath.Join(tmp, "global")

	createSkillDir(t, filepath.Join(ws, "skills"), "my-skill", "my-skill", "workspace version")
	createSkillDir(t, global, "my-skill", "my-skill", "global version")

	sl := NewSkillsLoader(ws, global, "")
	skills := sl.ListSkills()

	assert.Len(t, skills, 1)
	assert.Equal(t, "workspace", skills[0].Source)
	assert.Equal(t, "workspace version", skills[0].Description)
}

func TestListSkillsGlobalOverridesBuiltin(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "workspace")
	global := filepath.Join(tmp, "global")
	builtin := filepath.Join(tmp, "builtin")

	createSkillDir(t, global, "my-skill", "my-skill", "global version")
	createSkillDir(t, builtin, "my-skill", "my-skill", "builtin version")

	sl := NewSkillsLoader(ws, global, builtin)
	skills := sl.ListSkills()

	assert.Len(t, skills, 1)
	assert.Equal(t, "global", skills[0].Source)
	assert.Equal(t, "global version", skills[0].Description)
}

func TestListSkillsMetadataNameDedup(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "workspace")
	global := filepath.Join(tmp, "global")

	// Different directory names but same metadata name
	createSkillDir(t, filepath.Join(ws, "skills"), "dir-a", "shared-name", "workspace version")
	createSkillDir(t, global, "dir-b", "shared-name", "global version")

	sl := NewSkillsLoader(ws, global, "")
	skills := sl.ListSkills()

	assert.Len(t, skills, 1)
	assert.Equal(t, "shared-name", skills[0].Name)
	assert.Equal(t, "workspace", skills[0].Source)
}

func TestListSkillsMultipleDistinctSkills(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "workspace")
	global := filepath.Join(tmp, "global")
	builtin := filepath.Join(tmp, "builtin")

	createSkillDir(t, filepath.Join(ws, "skills"), "skill-a", "skill-a", "desc a")
	createSkillDir(t, global, "skill-b", "skill-b", "desc b")
	createSkillDir(t, builtin, "skill-c", "skill-c", "desc c")

	sl := NewSkillsLoader(ws, global, builtin)
	skills := sl.ListSkills()

	assert.Len(t, skills, 3)
	names := map[string]string{}
	for _, s := range skills {
		names[s.Name] = s.Source
	}
	assert.Equal(t, "workspace", names["skill-a"])
	assert.Equal(t, "global", names["skill-b"])
	assert.Equal(t, "builtin", names["skill-c"])
}

func TestListSkillsInvalidSkillSkipped(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "workspace")
	global := filepath.Join(tmp, "global")

	// Invalid name (underscore)
	createSkillDir(t, filepath.Join(ws, "skills"), "bad_skill", "bad_skill", "desc")
	// Valid skill
	createSkillDir(t, global, "good-skill", "good-skill", "desc")

	sl := NewSkillsLoader(ws, global, "")
	skills := sl.ListSkills()

	assert.Len(t, skills, 1)
	assert.Equal(t, "good-skill", skills[0].Name)
}

func TestListSkillsEmptyAndNonexistentDirs(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "workspace")
	emptyDir := filepath.Join(tmp, "empty")
	require.NoError(t, os.MkdirAll(emptyDir, 0o755))

	sl := NewSkillsLoader(ws, emptyDir, filepath.Join(tmp, "nonexistent"))
	skills := sl.ListSkills()

	assert.Empty(t, skills)
}

func TestListSkillsDirWithoutSkillMD(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "workspace")
	global := filepath.Join(tmp, "global")

	// Directory exists but has no SKILL.md
	require.NoError(t, os.MkdirAll(filepath.Join(global, "no-skillmd"), 0o755))
	// Valid skill alongside
	createSkillDir(t, global, "real-skill", "real-skill", "desc")

	sl := NewSkillsLoader(ws, global, "")
	skills := sl.ListSkills()

	assert.Len(t, skills, 1)
	assert.Equal(t, "real-skill", skills[0].Name)
}

func TestStripFrontmatter(t *testing.T) {
	sl := &SkillsLoader{}

	testcases := []struct {
		name            string
		content         string
		expectedContent string
		lineEndingType  string
	}{
		{
			name:            "unix-line-endings",
			lineEndingType:  "Unix (\\n)",
			content:         "---\nname: test-skill\ndescription: A test skill\n---\n\n# Skill Content",
			expectedContent: "# Skill Content",
		},
		{
			name:            "windows-line-endings",
			lineEndingType:  "Windows (\\r\\n)",
			content:         "---\r\nname: test-skill\r\ndescription: A test skill\r\n---\r\n\r\n# Skill Content",
			expectedContent: "# Skill Content",
		},
		{
			name:            "classic-mac-line-endings",
			lineEndingType:  "Classic Mac (\\r)",
			content:         "---\rname: test-skill\rdescription: A test skill\r---\r\r# Skill Content",
			expectedContent: "# Skill Content",
		},
		{
			name:            "unix-line-endings-without-trailing-newline",
			lineEndingType:  "Unix (\\n) without trailing newline",
			content:         "---\nname: test-skill\ndescription: A test skill\n---\n# Skill Content",
			expectedContent: "# Skill Content",
		},
		{
			name:            "windows-line-endings-without-trailing-newline",
			lineEndingType:  "Windows (\\r\\n) without trailing newline",
			content:         "---\r\nname: test-skill\r\ndescription: A test skill\r\n---\r\n# Skill Content",
			expectedContent: "# Skill Content",
		},
		{
			name:            "no-frontmatter",
			lineEndingType:  "No frontmatter",
			content:         "# Skill Content\n\nSome content here.",
			expectedContent: "# Skill Content\n\nSome content here.",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			result := sl.stripFrontmatter(tc.content)
			assert.Equal(
				t,
				tc.expectedContent,
				result,
				"Frontmatter should be stripped correctly for %s",
				tc.lineEndingType,
			)
		})
	}
}
