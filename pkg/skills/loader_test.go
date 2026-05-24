package skills

import (
	"os"
	"path/filepath"
	"runtime"
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

// createSkillWithFrontmatter creates a SKILL.md whose frontmatter is provided verbatim.
// Use this when the test needs to exercise extended metadata (e.g. nanobot.requires.bins)
// that the plain createSkillDir helper does not write.
func createSkillWithFrontmatter(t *testing.T, base, dirName, frontmatter, body string) {
	t.Helper()
	dir := filepath.Join(base, dirName)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	content := "---\n" + frontmatter + "\n---\n\n" + body
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644))
}

// makeFakeBin writes a stub executable named `name` into a fresh tempdir and
// returns the dir. Tests use it together with t.Setenv("PATH", dir) to control
// what exec.LookPath resolves. On Windows the file gets a .exe extension so
// LookPath honors it.
func makeFakeBin(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	binName := name
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, binName), []byte("#!/bin/sh\nexit 0\n"), 0o755))
	return dir
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

func TestSkillRootsTrimsWhitespaceAndDedups(t *testing.T) {
	tmp := t.TempDir()
	workspace := filepath.Join(tmp, "workspace")
	global := filepath.Join(tmp, "global")
	builtin := filepath.Join(tmp, "builtin")

	sl := NewSkillsLoader(workspace, "  "+global+"  ", "\t"+builtin+"\n")
	roots := sl.SkillRoots()

	assert.Equal(t, []string{
		filepath.Join(workspace, "skills"),
		global,
		builtin,
	}, roots)
}

func TestGetSkillMetadata_UsesMarkdownParagraphWhenNoFrontmatter(t *testing.T) {
	tmp := t.TempDir()
	skillDir := filepath.Join(tmp, "workspace", "skills", "plain-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	content := "# Plain Skill\n\nThis is parsed from markdown paragraph.\n"
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644))

	sl := &SkillsLoader{}
	meta := sl.getSkillMetadata(filepath.Join(skillDir, "SKILL.md"))
	require.NotNil(t, meta)
	assert.Equal(t, "plain-skill", meta.Name)
	assert.Equal(t, "This is parsed from markdown paragraph.", meta.Description)
}

func TestGetSkillMetadata_FrontmatterOverridesMarkdown(t *testing.T) {
	tmp := t.TempDir()
	skillDir := filepath.Join(tmp, "workspace", "skills", "plain-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	content := "---\nname: frontmatter-skill\ndescription: frontmatter description\n---\n\n# Plain Skill\n\nBody description.\n"
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644))

	sl := &SkillsLoader{}
	meta := sl.getSkillMetadata(filepath.Join(skillDir, "SKILL.md"))
	require.NotNil(t, meta)
	assert.Equal(t, "frontmatter-skill", meta.Name)
	assert.Equal(t, "frontmatter description", meta.Description)
}

func TestGetSkillMetadata_YAMLMultilineDescription(t *testing.T) {
	tmp := t.TempDir()
	skillDir := filepath.Join(tmp, "workspace", "skills", "plain-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	content := "---\nname: frontmatter-skill\ndescription: |\n  line 1: with colon\n  line 2\n---\n\n# Plain Skill\n\nBody description.\n"
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644))

	sl := &SkillsLoader{}
	meta := sl.getSkillMetadata(filepath.Join(skillDir, "SKILL.md"))
	require.NotNil(t, meta)
	assert.Equal(t, "frontmatter-skill", meta.Name)
	assert.Equal(t, "line 1: with colon\nline 2", meta.Description)
}

func TestGetSkillMetadata_InvalidHeadingNameFallsBackToDirName(t *testing.T) {
	tmp := t.TempDir()
	skillDir := filepath.Join(tmp, "workspace", "skills", "valid-name")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	content := "# Invalid Heading Name\n\nBody description.\n"
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644))

	sl := &SkillsLoader{}
	meta := sl.getSkillMetadata(filepath.Join(skillDir, "SKILL.md"))
	require.NotNil(t, meta)
	assert.Equal(t, "valid-name", meta.Name)
	assert.Equal(t, "Body description.", meta.Description)
}

func TestListSkillsSkipsWhenRequiredBinaryMissing(t *testing.T) {
	// Empty PATH guarantees the named bin cannot be resolved no matter
	// what's installed on the test host.
	t.Setenv("PATH", t.TempDir())

	tmp := t.TempDir()
	ws := filepath.Join(tmp, "workspace")
	frontmatter := `name: needs-tool
description: requires a missing binary
metadata: {"nanobot":{"requires":{"bins":["picoclaw-test-missing-xyz123"]}}}`
	createSkillWithFrontmatter(t, filepath.Join(ws, "skills"), "needs-tool", frontmatter, "# needs-tool")

	sl := NewSkillsLoader(ws, "", "")
	skills := sl.ListSkills()

	assert.Empty(t, skills, "skill with missing required binary should be skipped")
}

func TestListSkillsIncludesWhenRequiredBinaryPresent(t *testing.T) {
	binDir := makeFakeBin(t, "picoclaw-test-present")
	t.Setenv("PATH", binDir)

	tmp := t.TempDir()
	ws := filepath.Join(tmp, "workspace")
	frontmatter := `name: needs-tool
description: requires a present binary
metadata: {"nanobot":{"requires":{"bins":["picoclaw-test-present"]}}}`
	createSkillWithFrontmatter(t, filepath.Join(ws, "skills"), "needs-tool", frontmatter, "# needs-tool")

	sl := NewSkillsLoader(ws, "", "")
	skills := sl.ListSkills()

	require.Len(t, skills, 1)
	assert.Equal(t, "needs-tool", skills[0].Name)
}

func TestListSkillsSkipsWhenAnyRequiredBinaryMissing(t *testing.T) {
	binDir := makeFakeBin(t, "picoclaw-test-present")
	t.Setenv("PATH", binDir)

	tmp := t.TempDir()
	ws := filepath.Join(tmp, "workspace")
	frontmatter := `name: multi-bin
description: requires multiple binaries
metadata: {"nanobot":{"requires":{"bins":["picoclaw-test-present","picoclaw-test-missing-xyz123"]}}}`
	createSkillWithFrontmatter(t, filepath.Join(ws, "skills"), "multi-bin", frontmatter, "# multi-bin")

	sl := NewSkillsLoader(ws, "", "")
	skills := sl.ListSkills()

	assert.Empty(t, skills, "skill should be skipped if any required binary is missing")
}

func TestListSkillsAllowsEmptyRequiresBins(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	tmp := t.TempDir()
	ws := filepath.Join(tmp, "workspace")
	frontmatter := `name: empty-bins
description: declares an empty bins list
metadata: {"nanobot":{"requires":{"bins":[]}}}`
	createSkillWithFrontmatter(t, filepath.Join(ws, "skills"), "empty-bins", frontmatter, "# empty-bins")

	sl := NewSkillsLoader(ws, "", "")
	skills := sl.ListSkills()

	require.Len(t, skills, 1)
	assert.Equal(t, "empty-bins", skills[0].Name)
}

func TestListSkillsAllowsSkillsWithoutRequiresBins(t *testing.T) {
	// A skill that doesn't declare metadata.nanobot.requires.bins at all
	// must continue to load — this is the pre-validation behaviour and the
	// shape of the vast majority of skills.
	t.Setenv("PATH", t.TempDir())

	tmp := t.TempDir()
	ws := filepath.Join(tmp, "workspace")
	createSkillDir(t, filepath.Join(ws, "skills"), "no-meta", "no-meta", "no extended metadata")

	sl := NewSkillsLoader(ws, "", "")
	skills := sl.ListSkills()

	require.Len(t, skills, 1)
	assert.Equal(t, "no-meta", skills[0].Name)
}

func TestListSkillsMissingBinaryFallsThroughSourcePriority(t *testing.T) {
	// Workspace copy declares an unavailable bin → should be skipped.
	// Global copy of the same skill name has no bin requirement → should win.
	binDir := makeFakeBin(t, "picoclaw-test-fallthrough")
	t.Setenv("PATH", binDir)

	tmp := t.TempDir()
	ws := filepath.Join(tmp, "workspace")
	global := filepath.Join(tmp, "global")

	wsFrontmatter := `name: shared
description: workspace copy needs a missing bin
metadata: {"nanobot":{"requires":{"bins":["picoclaw-test-missing-xyz123"]}}}`
	createSkillWithFrontmatter(t, filepath.Join(ws, "skills"), "shared", wsFrontmatter, "# shared")
	createSkillDir(t, global, "shared", "shared", "global copy without bin requirement")

	sl := NewSkillsLoader(ws, global, "")
	skills := sl.ListSkills()

	require.Len(t, skills, 1)
	assert.Equal(t, "global", skills[0].Source, "global copy must replace the skipped workspace copy")
	assert.Equal(t, "global copy without bin requirement", skills[0].Description)
}

func TestGetSkillMetadata_ParsesRequiresBinsFromInlineJSON(t *testing.T) {
	tmp := t.TempDir()
	skillDir := filepath.Join(tmp, "browser-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	// Real-world frontmatter shape: YAML keys with a flow-style (JSON) metadata value.
	content := `---
name: agent-browser
description: Browser automation
metadata: {"nanobot":{"emoji":"🌐","requires":{"bins":["agent-browser"]}}}
---

# Agent Browser`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644))

	sl := &SkillsLoader{}
	meta := sl.getSkillMetadata(filepath.Join(skillDir, "SKILL.md"))
	require.NotNil(t, meta)
	assert.Equal(t, []string{"agent-browser"}, meta.RequiresBins)
}

func TestGetSkillMetadata_ParsesRequiresBinsFromPureJSON(t *testing.T) {
	tmp := t.TempDir()
	skillDir := filepath.Join(tmp, "pure-json")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	content := `---
{"name":"json-skill","description":"all JSON frontmatter","metadata":{"nanobot":{"requires":{"bins":["foo","bar"]}}}}
---

# JSON Skill`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644))

	sl := &SkillsLoader{}
	meta := sl.getSkillMetadata(filepath.Join(skillDir, "SKILL.md"))
	require.NotNil(t, meta)
	assert.Equal(t, []string{"foo", "bar"}, meta.RequiresBins)
}

func TestMissingBinaries(t *testing.T) {
	binDir := makeFakeBin(t, "picoclaw-test-real")
	t.Setenv("PATH", binDir)

	testcases := []struct {
		name string
		bins []string
		want []string
	}{
		{name: "nil input", bins: nil, want: nil},
		{name: "empty input", bins: []string{}, want: nil},
		{name: "all present", bins: []string{"picoclaw-test-real"}, want: nil},
		{
			name: "all missing",
			bins: []string{"picoclaw-missing-a", "picoclaw-missing-b"},
			want: []string{"picoclaw-missing-a", "picoclaw-missing-b"},
		},
		{
			name: "mixed",
			bins: []string{"picoclaw-test-real", "picoclaw-missing-a"},
			want: []string{"picoclaw-missing-a"},
		},
		{name: "whitespace-only entries are ignored", bins: []string{"  ", "\t", "picoclaw-test-real"}, want: nil},
		{
			name: "path-separator entry rejected (unix)",
			bins: []string{"./picoclaw-test-real"},
			want: []string{"./picoclaw-test-real"},
		},
		{
			name: "path-separator entry rejected (windows-style)",
			bins: []string{`subdir\picoclaw-test-real`},
			want: []string{`subdir\picoclaw-test-real`},
		},
		{
			name: "absolute path rejected even if file exists",
			bins: []string{"/usr/bin/sh"},
			want: []string{"/usr/bin/sh"},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, missingBinaries(tc.bins))
		})
	}
}

func TestGetSkillMetadata_IgnoresHTMLCommentBlocks(t *testing.T) {
	tmp := t.TempDir()
	skillDir := filepath.Join(tmp, "workspace", "skills", "biomed-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	content := "<!--\n# COPYRIGHT NOTICE\n# This file is part of the \"Universal Biomedical Skills\" project.\n# Copyright (c) 2026 MD BABU MIA, PhD <md.babu.mia@mssm.edu>\n# All Rights Reserved.\n#\n# This code is proprietary and confidential.\n# Unauthorized copying of this file, via any medium is strictly prohibited.\n#\n# Provenance: Authenticated by MD BABU MIA\n\n-->\n\n# Biomed Skill\n\nSummarize biomedical papers.\n"
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644))

	sl := &SkillsLoader{}
	meta := sl.getSkillMetadata(filepath.Join(skillDir, "SKILL.md"))
	require.NotNil(t, meta)
	assert.Equal(t, "biomed-skill", meta.Name)
	assert.Equal(t, "Summarize biomedical papers.", meta.Description)
}
