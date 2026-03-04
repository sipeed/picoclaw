package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsGitURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		// SSH format
		{"SSH GitHub", "git@github.com:user/repo.git", true},
		{"SSH GitLab", "git@gitlab.com:group/project.git", true},
		{"SSH custom host", "git@git.mycompany.com:team/skill.git", true},
		{"SSH nested path", "git@gitlab.com:group/subgroup/project.git", true},

		// SSH URL format
		{"SSH URL format", "ssh://git@github.com/user/repo.git", true},
		{"SSH URL custom port", "ssh://git@gitlab.com:2222/user/repo.git", true},

		// HTTPS format
		{"HTTPS GitHub", "https://github.com/user/repo.git", true},
		{"HTTPS GitLab", "https://gitlab.com/user/repo.git", true},
		{"HTTPS with port", "https://git.mycompany.com:8443/team/skill.git", true},

		// HTTP format
		{"HTTP simple", "http://git.local/user/repo.git", true},

		// Invalid cases
		{"No .git suffix", "https://github.com/user/repo", false},
		{"GitHub path format", "sipeed/picoclaw-skills/weather", false},
		{"Plain text", "my-skill", false},
		{"Empty string", "", false},
		{"Just .git", ".git", false},
		{"SSH without .git", "git@github.com:user/repo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGitURL(tt.url)
			assert.Equal(t, tt.expected, result, "IsGitURL(%q) should be %v", tt.url, tt.expected)
		})
	}
}

func TestExtractSkillNameFromGitURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		// SSH format
		{"SSH simple", "git@github.com:user/my-skill.git", "my-skill"},
		{"SSH nested", "git@gitlab.com:group/subgroup/project.git", "project"},

		// HTTPS format
		{"HTTPS simple", "https://github.com/user/weather-skill.git", "weather-skill"},
		{"HTTPS nested", "https://gitlab.com/company/team/skill.git", "skill"},

		// SSH URL format
		{"SSH URL format", "ssh://git@github.com/user/repo.git", "repo"},

		// HTTP format
		{"HTTP simple", "http://git.local/user/local-skill.git", "local-skill"},

		// Without .git suffix (edge cases)
		{"No .git suffix", "https://github.com/user/repo", "repo"},

		// Edge cases
		{"Empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSkillNameFromGitURL(tt.url)
			assert.Equal(t, tt.expected, result, "extractSkillNameFromGitURL(%q) should be %q", tt.url, tt.expected)
		})
	}
}

func TestExtractFirstLine(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{"Simple description field", "description: This is a description", "This is a description"},
		{"Description with quotes", "description: \"This is a description\"", "This is a description"},
		{"Description after header", "# Title\ndescription: This is a description", "This is a description"},
		{"Description case insensitive", "Description: This is a description", "This is a description"},
		{"Empty content", "", ""},
		{"No description field", "# Title\n## Subtitle", ""},
		{"Long description truncated", "description: This is a very long description that exceeds one hundred characters and should be truncated with ellipsis at the end", "This is a very long description that exceeds one hundred characters and should be truncated with ell..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFirstLine(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDiscoverSkillsInDir(t *testing.T) {
	// Create temp directory structure.
	tempDir := t.TempDir()

	// Create skill directories.
	skill1Dir := filepath.Join(tempDir, "weather")
	skill2Dir := filepath.Join(tempDir, "news")
	nonSkillDir := filepath.Join(tempDir, "not-a-skill")
	hiddenDir := filepath.Join(tempDir, ".hidden")

	assert.NoError(t, os.MkdirAll(skill1Dir, 0o755))
	assert.NoError(t, os.MkdirAll(skill2Dir, 0o755))
	assert.NoError(t, os.MkdirAll(nonSkillDir, 0o755))
	assert.NoError(t, os.MkdirAll(hiddenDir, 0o755))

	// Create SKILL.md files.
	assert.NoError(t, os.WriteFile(filepath.Join(skill1Dir, "SKILL.md"), []byte("# Weather\nGet weather information"), 0o644))
	assert.NoError(t, os.WriteFile(filepath.Join(skill2Dir, "SKILL.md"), []byte("# News\nGet latest news"), 0o644))
	assert.NoError(t, os.WriteFile(filepath.Join(hiddenDir, "SKILL.md"), []byte("# Hidden\nShould be ignored"), 0o644))
	// nonSkillDir has no SKILL.md

	installer := NewSkillInstaller(tempDir)
	skills, err := installer.discoverSkillsInDir(tempDir)

	assert.NoError(t, err)
	assert.Len(t, skills, 2)

	// Check discovered skills.
	skillNames := make(map[string]bool)
	for _, s := range skills {
		skillNames[s.Name] = true
	}
	assert.True(t, skillNames["weather"])
	assert.True(t, skillNames["news"])
	assert.False(t, skillNames["not-a-skill"])
	assert.False(t, skillNames[".hidden"])
}
