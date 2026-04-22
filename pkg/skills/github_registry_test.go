package skills

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestGitHubRegistrySearch(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v3/search/code", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "skill search filename:SKILL.md", r.URL.Query().Get("q"))
		assert.Equal(t, "2", r.URL.Query().Get("per_page"))

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(gitHubCodeSearchResponse{
			Items: []gitHubCodeSearchItem{
				{
					Path:    "skills/pr-review/SKILL.md",
					Score:   10,
					HTMLURL: server.URL + "/foo/bar/blob/main/skills/pr-review/SKILL.md",
					Repository: struct {
						FullName      string `json:"full_name"`
						Name          string `json:"name"`
						Description   string `json:"description"`
						DefaultBranch string `json:"default_branch"`
					}{
						FullName:      "foo/bar",
						Name:          "bar",
						Description:   "Review pull requests",
						DefaultBranch: "main",
					},
				},
				{
					Path:    "SKILL.md",
					Score:   5,
					HTMLURL: server.URL + "/foo/root/blob/main/SKILL.md",
					Repository: struct {
						FullName      string `json:"full_name"`
						Name          string `json:"name"`
						Description   string `json:"description"`
						DefaultBranch string `json:"default_branch"`
					}{
						FullName:      "foo/root",
						Name:          "root",
						Description:   "Root skill",
						DefaultBranch: "master",
					},
				},
			},
		}))
	}))
	defer server.Close()

	provider := GitHubRegistryConfig{
		Enabled:   true,
		BaseURL:   server.URL,
		AuthToken: "test-token",
	}
	registry := provider.BuildRegistry()
	require.NotNil(t, registry)

	results, err := registry.Search(context.Background(), "skill search", 2)
	require.NoError(t, err)
	require.Len(t, results, 2)

	assert.Equal(t, "foo/bar/skills/pr-review", results[0].Slug)
	assert.Equal(t, "pr-review", results[0].DisplayName)
	assert.Equal(t, "Review pull requests", results[0].Summary)
	assert.Equal(t, "main", results[0].Version)
	assert.Equal(t, "github", results[0].RegistryName)

	assert.Equal(t, "foo/root", results[1].Slug)
	assert.Equal(t, "root", results[1].DisplayName)
	assert.Equal(t, "master", results[1].Version)
}

func TestGitHubRegistryProviderDecodesProxyParam(t *testing.T) {
	builder := buildRegistryProvider("github", config.SkillRegistryConfig{
		Name:      "github",
		Enabled:   true,
		BaseURL:   "https://github.com",
		AuthToken: *config.NewSecureString("test-token"),
		Param: map[string]any{
			"proxy": "http://127.0.0.1:7890",
		},
	})
	require.NotNil(t, builder)

	registry := builder.BuildRegistry()
	require.NotNil(t, registry)
	ghRegistry, ok := registry.(*GitHubRegistry)
	require.True(t, ok)
	assert.Equal(t, "http://127.0.0.1:7890", ghRegistry.installer.proxy)
}

func TestGitHubRegistrySearchReturnsNoResultsOnUnauthenticatedRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"API rate limit exceeded for 1.2.3.4"}`))
	}))
	defer server.Close()

	registry := GitHubRegistryConfig{Enabled: true, BaseURL: server.URL}.BuildRegistry()
	require.NotNil(t, registry)

	results, err := registry.Search(context.Background(), "pr review", 5)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestGitHubRegistrySearchReturnsNoResultsOnUnauthenticatedAuthRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(
			`{"message":"Requires authentication","errors":[{"message":"Must be authenticated to access the code search API"}]}`,
		))
	}))
	defer server.Close()

	registry := GitHubRegistryConfig{Enabled: true, BaseURL: server.URL}.BuildRegistry()
	require.NotNil(t, registry)

	results, err := registry.Search(context.Background(), "pr review", 5)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestGitHubRegistryGetSkillMetaCanonicalizesURLSlug(t *testing.T) {
	registry := GitHubRegistryConfig{
		Enabled: true,
		BaseURL: "https://ghe.example.com/git",
	}.BuildRegistry()
	require.NotNil(t, registry)

	meta, err := registry.GetSkillMeta(
		context.Background(),
		"https://ghe.example.com/git/org/repo/tree/dev/skills/pr-review",
	)
	require.NoError(t, err)
	require.NotNil(t, meta)
	assert.Equal(t, "org/repo/skills/pr-review", meta.Slug)
	assert.Equal(t, "dev", meta.LatestVersion)
}

func TestGitHubRegistrySkillURLUsesProvidedVersionAndBasePath(t *testing.T) {
	registry := GitHubRegistryConfig{
		Enabled: true,
		BaseURL: "https://ghe.example.com/git",
	}.BuildRegistry()
	require.NotNil(t, registry)

	assert.Equal(
		t,
		"https://ghe.example.com/git/org/repo/tree/master/skills/pr-review",
		registry.SkillURL("org/repo/skills/pr-review", "master"),
	)
	assert.Equal(
		t,
		"https://ghe.example.com/git/org/repo/tree/dev/skills/pr-review",
		registry.SkillURL("https://ghe.example.com/git/org/repo/tree/dev/skills/pr-review", ""),
	)
	assert.Equal(
		t,
		"https://ghe.example.com/git/org/repo/tree/feature/skills-registry/skills/pr-review",
		registry.SkillURL("org/repo/skills/pr-review", "feature/skills-registry"),
	)
	assert.Equal(
		t,
		"https://ghe.example.com/git/org/repo/blob/main/.agents/skills/pr-review/SKILL.md",
		registry.SkillURL("https://ghe.example.com/git/org/repo/blob/main/.agents/skills/pr-review/SKILL.md", ""),
	)
	assert.Equal(
		t,
		"https://github.com/org/repo/tree/main/.agents/skills/pr-review",
		registry.SkillURL("https://github.com/org/repo/tree/main/.agents/skills/pr-review", ""),
	)
	assert.Empty(t, registry.SkillURL("org/repo/.agents/skills/pr-review", ""))
}

func TestGitHubRegistryResolveInstallDirNameSupportsFullURLs(t *testing.T) {
	registry := GitHubRegistryConfig{
		Enabled: true,
		BaseURL: "https://ghe.example.com/git",
	}.BuildRegistry()
	require.NotNil(t, registry)

	dirName, err := registry.ResolveInstallDirName("https://ghe.example.com/git/org/repo/tree/dev/skills/pr-review")
	require.NoError(t, err)
	assert.Equal(t, "pr-review", dirName)

	dirName, err = registry.ResolveInstallDirName("https://github.com/org/repo/tree/main/skills/release-checklist")
	require.NoError(t, err)
	assert.Equal(t, "release-checklist", dirName)

	dirName, err = registry.ResolveInstallDirName(
		"https://ghe.example.com/git/org/repo/blob/dev/skills/pr-review/SKILL.md",
	)
	require.NoError(t, err)
	assert.Equal(t, "pr-review", dirName)

	dirName, err = registry.ResolveInstallDirName(
		"https://ghe.example.com/git/org/repo/blob/dev/SKILL.md",
	)
	require.NoError(t, err)
	assert.Equal(t, "repo", dirName)
}
