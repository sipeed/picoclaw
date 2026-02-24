package skills

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInstallSpec(t *testing.T) {
	tests := []struct {
		spec      string
		wantRepo  string
		wantBranch string
		wantErr   bool
	}{
		{"owner/repo", "owner/repo", "", false},
		{"  owner/repo  ", "owner/repo", "", false},
		{"owner/repo@main", "owner/repo", "main", false},
		{"owner/repo@test", "owner/repo", "test", false},
		{"owner/repo@", "", "", true},
		{"", "", "", true},
		{"n slash", "", "", true},
		{"a/b@branch", "a/b", "branch", false},
	}
	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			repo, branch, err := ParseInstallSpec(tt.spec)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantRepo, repo)
			assert.Equal(t, tt.wantBranch, branch)
		})
	}
}

func TestInstallFromGitHubEx_reinstall_overwrites(t *testing.T) {
	content1 := []byte("# Skill v1")
	content2 := []byte("# Skill v2")
	var reqCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount++
		w.WriteHeader(200)
		if reqCount == 1 {
			_, _ = w.Write(content1)
		} else {
			_, _ = w.Write(content2)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	si := NewSkillInstallerWithBase(dir, server.URL)
	ctx := context.Background()

	skillName, err := si.InstallFromGitHubEx(ctx, "owner/repo", "main", "", false)
	require.NoError(t, err)
	assert.Equal(t, "repo", skillName)
	path := filepath.Join(dir, "skills", "repo", "SKILL.md")
	data, _ := os.ReadFile(path)
	assert.Equal(t, content1, data)

	// Reinstall (force overwrite)
	skillName2, err := si.InstallFromGitHubEx(ctx, "owner/repo", "main", "", true)
	require.NoError(t, err)
	assert.Equal(t, "repo", skillName2)
	data2, _ := os.ReadFile(path)
	assert.Equal(t, content2, data2)
}

func TestInstallFromGitHubEx_already_exists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("# Skill"))
	}))
	defer server.Close()

	dir := t.TempDir()
	si := NewSkillInstallerWithBase(dir, server.URL)
	ctx := context.Background()

	_, err := si.InstallFromGitHubEx(ctx, "sipeed/picoclaw-skills", "main", "", false)
	require.NoError(t, err)

	_, err = si.InstallFromGitHubEx(ctx, "sipeed/picoclaw-skills", "main", "", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
	assert.Contains(t, err.Error(), "reinstall")
}
