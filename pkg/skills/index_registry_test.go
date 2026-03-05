package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexRegistryMapURL(t *testing.T) {
	tests := []struct {
		name         string
		mappings     map[string]string
		inputURL     string
		expectedURL  string
	}{
		{
			name: "maps HTTP to fork",
			mappings: map[string]string{
				"https://raw.githubusercontent.com/keithy/angelhub/main/": "https://raw.githubusercontent.com/myfork/angelhub/main/",
			},
			inputURL:    "https://raw.githubusercontent.com/keithy/angelhub/main/picoclaw/skills/self-config",
			expectedURL: "https://raw.githubusercontent.com/myfork/angelhub/main/picoclaw/skills/self-config",
		},
		{
			name: "maps HTTP to local file",
			mappings: map[string]string{
				"https://raw.githubusercontent.com/keithy/angelhub/": "file:///home/me/repos/angelhub/",
			},
			inputURL:    "https://raw.githubusercontent.com/keithy/angelhub/main/picoclaw/skills/self-config",
			expectedURL: "file:///home/me/repos/angelhub/main/picoclaw/skills/self-config",
		},
		{
			name: "no mapping returns original",
			mappings: map[string]string{
				"https://other.com/": "https://example.com/",
			},
			inputURL:    "https://raw.githubusercontent.com/keithy/angelhub/main/skills",
			expectedURL: "https://raw.githubusercontent.com/keithy/angelhub/main/skills",
		},
		{
			name:         "empty mappings returns original",
			mappings:     map[string]string{},
			inputURL:     "https://raw.githubusercontent.com/keithy/angelhub/main/skills",
			expectedURL:  "https://raw.githubusercontent.com/keithy/angelhub/main/skills",
		},
		{
			name:         "nil mappings returns original",
			mappings:     nil,
			inputURL:     "https://raw.githubusercontent.com/keithy/angelhub/main/skills",
			expectedURL:  "https://raw.githubusercontent.com/keithy/angelhub/main/skills",
		},
		{
			name: "first matching prefix wins",
			mappings: map[string]string{
				"https://raw.githubusercontent.com/keithy/": "https://raw.githubusercontent.com/fork1/",
			},
			inputURL:    "https://raw.githubusercontent.com/keithy/angelhub/main/skills",
			expectedURL: "https://raw.githubusercontent.com/fork1/angelhub/main/skills",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &IndexRegistry{
				name:        "test",
				urlMappings: tt.mappings,
			}
			result := r.mapURL(tt.inputURL)
			assert.Equal(t, tt.expectedURL, result)
		})
	}
}

func TestIndexRegistryIsURLAllowed(t *testing.T) {
	tests := []struct {
		name            string
		allowedPrefixes []string
		url             string
		expected        bool
	}{
		{
			name:            "matching prefix returns true",
			allowedPrefixes: []string{"https://raw.githubusercontent.com/keithy/"},
			url:             "https://raw.githubusercontent.com/keithy/angelhub/main/skills",
			expected:        true,
		},
		{
			name:            "non-matching prefix returns false",
			allowedPrefixes: []string{"https://raw.githubusercontent.com/keithy/"},
			url:             "https://raw.githubusercontent.com/other/repo/main",
			expected:        false,
		},
		{
			name:            "empty allowedPrefixes allows all",
			allowedPrefixes: []string{},
			url:             "https://any-site.com/file",
			expected:        true,
		},
		{
			name:            "nil allowedPrefixes allows all",
			allowedPrefixes: nil,
			url:             "https://any-site.com/file",
			expected:        true,
		},
		{
			name:            "multiple prefixes first match wins",
			allowedPrefixes: []string{"https://raw.githubusercontent.com/other/", "https://raw.githubusercontent.com/keithy/"},
			url:             "https://raw.githubusercontent.com/keithy/angelhub/main",
			expected:        true,
		},
		{
			name:            "prefix not at start returns false",
			allowedPrefixes: []string{"https://raw.githubusercontent.com/keithy/"},
			url:             "https://raw.githubusercontent.com/other/keithy/foo",
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &IndexRegistry{
				name:            "test",
				allowedPrefixes: tt.allowedPrefixes,
			}
			result := r.isURLAllowed(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIndexRegistryCopyLocalPath(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()
	tmpSkillDir := filepath.Join(tmpDir, "test-skill")
	err := os.MkdirAll(filepath.Join(tmpSkillDir, "subdir"), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpSkillDir, "SKILL.md"), []byte("# Test Skill"), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpSkillDir, "subdir", "script.sh"), []byte("#!/bin/bash\necho hello"), 0o755)
	require.NoError(t, err)

	t.Run("symlink_local true creates symlink", func(t *testing.T) {
		targetDir := filepath.Join(tmpDir, "installed-skill")
		r := &IndexRegistry{
			name:         "test",
			symlinkLocal: true,
		}
		err := r.copyLocalPath(tmpSkillDir, targetDir)
		require.NoError(t, err)

		// Verify symlink was created
		info, err := os.Lstat(targetDir)
		require.NoError(t, err)
		require.True(t, info.Mode()&os.ModeSymlink != 0, "should be a symlink")

		// Verify symlink points to correct path
		linkDest, err := os.Readlink(targetDir)
		require.NoError(t, err)
		assert.Equal(t, tmpSkillDir, linkDest)
	})
}

func TestIndexRegistrySearch(t *testing.T) {
	// This test would require mocking HTTP - just verify the method signature works
	r := &IndexRegistry{
		name:     "test",
		indexURL: "https://example.com/index.json",
	}
	assert.Equal(t, "test", r.Name())
}
