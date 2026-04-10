package skills

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestSkillsInstallFromRegistryWritesOriginMetadata(t *testing.T) {
	workspace := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v3/repos/foo/bar":
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"default_branch": "master"}))
		case "/api/v3/repos/foo/bar/contents/.agents/skills/pr-review":
			assert.Equal(t, "ref=master", r.URL.RawQuery)
			require.NoError(t, json.NewEncoder(w).Encode([]map[string]any{{
				"type":         "file",
				"name":         "SKILL.md",
				"download_url": server.URL + "/raw/foo/bar/master/.agents/skills/pr-review/SKILL.md",
			}}))
		case "/raw/foo/bar/master/.agents/skills/pr-review/SKILL.md":
			_, _ = w.Write([]byte("---\nname: pr-review\ndescription: PR review skill\n---\n# PR Review\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	githubRegistry, ok := cfg.Tools.Skills.Registries.Get("github")
	require.True(t, ok)
	githubRegistry.BaseURL = server.URL
	cfg.Tools.Skills.Registries.Set("github", githubRegistry)

	target := server.URL + "/foo/bar/tree/master/.agents/skills/pr-review"
	require.NoError(t, skillsInstallFromRegistry(cfg, "github", target))

	metaPath := filepath.Join(workspace, "skills", "pr-review", ".skill-origin.json")
	data, err := os.ReadFile(metaPath)
	require.NoError(t, err)

	var meta installedSkillOriginMeta
	require.NoError(t, json.Unmarshal(data, &meta))
	assert.Equal(t, "third_party", meta.OriginKind)
	assert.Equal(t, "github", meta.Registry)
	assert.Equal(t, "foo/bar/.agents/skills/pr-review", meta.Slug)
	assert.Equal(t, server.URL+"/foo/bar/tree/master/.agents/skills/pr-review", meta.RegistryURL)
	assert.Equal(t, "master", meta.InstalledVersion)
	assert.NotZero(t, meta.InstalledAt)
}
