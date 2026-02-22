package skills

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/pkg/utils"
)

func newTestRegistry(serverURL, authToken string) *ClawHubRegistry {
	return NewClawHubRegistry(ClawHubConfig{
		Enabled:   true,
		BaseURL:   serverURL,
		AuthToken: authToken,
	})
}

func TestClawHubRegistrySearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/search", r.URL.Path)
		assert.Equal(t, "github", r.URL.Query().Get("q"))

		slug := "github"
		name := "GitHub Integration"
		summary := "Interact with GitHub repos"
		version := "1.0.0"

		json.NewEncoder(w).Encode(clawhubSearchResponse{
			Results: []clawhubSearchResult{
				{Score: 0.95, Slug: &slug, DisplayName: &name, Summary: &summary, Version: &version},
			},
		})
	}))
	defer srv.Close()

	reg := newTestRegistry(srv.URL, "")
	results, err := reg.Search(context.Background(), "github", 5)

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "github", results[0].Slug)
	assert.Equal(t, "GitHub Integration", results[0].DisplayName)
	assert.InDelta(t, 0.95, results[0].Score, 0.001)
	assert.Equal(t, "clawhub", results[0].RegistryName)
}

func TestClawHubRegistryGetSkillMeta(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/skills/github", r.URL.Path)

		json.NewEncoder(w).Encode(clawhubSkillResponse{
			Slug:        "github",
			DisplayName: "GitHub Integration",
			Summary:     "Full GitHub API integration",
			LatestVersion: &clawhubVersionInfo{
				Version: "2.1.0",
			},
			Moderation: &clawhubModerationInfo{
				IsMalwareBlocked: false,
				IsSuspicious:     true,
			},
		})
	}))
	defer srv.Close()

	reg := newTestRegistry(srv.URL, "")
	meta, err := reg.GetSkillMeta(context.Background(), "github")

	require.NoError(t, err)
	assert.Equal(t, "github", meta.Slug)
	assert.Equal(t, "2.1.0", meta.LatestVersion)
	assert.False(t, meta.IsMalwareBlocked)
	assert.True(t, meta.IsSuspicious)
}

func TestClawHubRegistryGetSkillMetaUnsafeSlug(t *testing.T) {
	reg := newTestRegistry("https://example.com", "")
	_, err := reg.GetSkillMeta(context.Background(), "../etc/passwd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid slug")
}

func TestClawHubRegistryDownloadAndInstall(t *testing.T) {
	// Create a valid ZIP in memory.
	zipBuf := createTestZip(t, map[string]string{
		"SKILL.md":  "---\nname: test-skill\ndescription: A test\n---\nHello skill",
		"README.md": "# Test Skill\n",
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/skills/test-skill":
			// Metadata endpoint.
			json.NewEncoder(w).Encode(clawhubSkillResponse{
				Slug:          "test-skill",
				DisplayName:   "Test Skill",
				Summary:       "A test skill",
				LatestVersion: &clawhubVersionInfo{Version: "1.0.0"},
			})
		case "/api/v1/download":
			assert.Equal(t, "test-skill", r.URL.Query().Get("slug"))
			w.Header().Set("Content-Type", "application/zip")
			w.Write(zipBuf)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "test-skill")

	reg := newTestRegistry(srv.URL, "")
	result, err := reg.DownloadAndInstall(context.Background(), "test-skill", "1.0.0", targetDir)

	require.NoError(t, err)
	assert.Equal(t, "1.0.0", result.Version)
	assert.False(t, result.IsMalwareBlocked)

	// Verify extracted files.
	skillContent, err := os.ReadFile(filepath.Join(targetDir, "SKILL.md"))
	require.NoError(t, err)
	assert.Contains(t, string(skillContent), "Hello skill")

	readmeContent, err := os.ReadFile(filepath.Join(targetDir, "README.md"))
	require.NoError(t, err)
	assert.Contains(t, string(readmeContent), "# Test Skill")
}

func TestClawHubRegistryAuthToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		assert.Equal(t, "Bearer test-token-123", authHeader)
		json.NewEncoder(w).Encode(clawhubSearchResponse{Results: nil})
	}))
	defer srv.Close()

	reg := newTestRegistry(srv.URL, "test-token-123")
	_, _ = reg.Search(context.Background(), "test", 5)
}

func TestExtractZipPathTraversal(t *testing.T) {
	// Create a ZIP with a path traversal entry.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// Malicious entry trying to escape directory.
	w, err := zw.Create("../../etc/passwd")
	require.NoError(t, err)
	w.Write([]byte("malicious"))

	zw.Close()

	// Write to temp file for extractZipFile.
	tmpZip := filepath.Join(t.TempDir(), "bad.zip")
	require.NoError(t, os.WriteFile(tmpZip, buf.Bytes(), 0o644))

	tmpDir := t.TempDir()
	err = utils.ExtractZipFile(tmpZip, tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsafe path")
}

func TestExtractZipWithSubdirectories(t *testing.T) {
	zipBuf := createTestZip(t, map[string]string{
		"SKILL.md":           "root file",
		"scripts/helper.sh":  "#!/bin/bash\necho hello",
		"examples/demo.yaml": "key: value",
	})

	// Write to temp file for extractZipFile.
	tmpZip := filepath.Join(t.TempDir(), "test.zip")
	require.NoError(t, os.WriteFile(tmpZip, zipBuf, 0o644))

	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "my-skill")

	err := utils.ExtractZipFile(tmpZip, targetDir)
	require.NoError(t, err)

	// Verify nested file.
	data, err := os.ReadFile(filepath.Join(targetDir, "scripts", "helper.sh"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "#!/bin/bash")
}

func TestClawHubRegistrySearchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer srv.Close()

	reg := newTestRegistry(srv.URL, "")
	_, err := reg.Search(context.Background(), "test", 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestClawHubRegistrySearchNullableFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		validSlug := "valid-slug"
		validSummary := "valid summary"

		// Return results with various null/empty fields
		json.NewEncoder(w).Encode(clawhubSearchResponse{
			Results: []clawhubSearchResult{
				// Case 1: Null Slug -> Skip
				{Score: 0.1, Slug: nil, DisplayName: nil, Summary: nil, Version: nil},
				// Case 2: Valid Slug, Null Summary -> Skip
				{Score: 0.2, Slug: &validSlug, DisplayName: nil, Summary: nil, Version: nil},
				// Case 3: Valid Slug, Valid Summary, Null Name -> Keep, Name=Slug
				{Score: 0.8, Slug: &validSlug, DisplayName: nil, Summary: &validSummary, Version: nil},
			},
		})
	}))
	defer srv.Close()

	reg := newTestRegistry(srv.URL, "")
	results, err := reg.Search(context.Background(), "test", 5)

	require.NoError(t, err)
	require.Len(t, results, 1, "should only return 1 valid result")

	r := results[0]
	assert.Equal(t, "valid-slug", r.Slug)
	assert.Equal(t, "valid-slug", r.DisplayName, "should fallback name to slug")
	assert.Equal(t, "valid summary", r.Summary)
}

// --- helpers ---

func createTestZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for name, content := range files {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, zw.Close())
	return buf.Bytes()
}
