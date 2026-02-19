package aieos

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aieos.json")

	data := `{
		"version": "1.1",
		"identity": {
			"name": "TestAgent",
			"description": "A test agent",
			"purpose": "Testing"
		},
		"capabilities": [
			{"name": "search", "description": "Web search"}
		],
		"psychology": {
			"openness": 0.8,
			"conscientiousness": 0.9,
			"extraversion": 0.5,
			"agreeableness": 0.85,
			"neuroticism": 0.1
		}
	}`
	require.NoError(t, os.WriteFile(path, []byte(data), 0644))

	p, err := LoadProfile(path)
	require.NoError(t, err)

	assert.Equal(t, "1.1", p.Version)
	assert.Equal(t, "TestAgent", p.Identity.Name)
	assert.Equal(t, "A test agent", p.Identity.Description)
	assert.Equal(t, "Testing", p.Identity.Purpose)
	assert.Len(t, p.Capabilities, 1)
	assert.Equal(t, "search", p.Capabilities[0].Name)
	require.NotNil(t, p.Psychology)
	assert.Equal(t, 0.8, p.Psychology.Openness)
	assert.Equal(t, 0.1, p.Psychology.Neuroticism)
}

func TestLoadProfileInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aieos.json")

	require.NoError(t, os.WriteFile(path, []byte(`{bad json`), 0644))

	_, err := LoadProfile(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse profile")
}

func TestLoadProfileMissingVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aieos.json")

	data := `{"identity": {"name": "Agent"}}`
	require.NoError(t, os.WriteFile(path, []byte(data), 0644))

	_, err := LoadProfile(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestLoadProfileMissingName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aieos.json")

	data := `{"version": "1.1", "identity": {}}`
	require.NoError(t, os.WriteFile(path, []byte(data), 0644))

	_, err := LoadProfile(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "identity.name is required")
}

func TestLoadProfileFileNotFound(t *testing.T) {
	_, err := LoadProfile("/nonexistent/aieos.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read profile")
}

func TestProfileExists(t *testing.T) {
	dir := t.TempDir()

	assert.False(t, ProfileExists(dir))

	path := filepath.Join(dir, "aieos.json")
	require.NoError(t, os.WriteFile(path, []byte(`{}`), 0644))

	assert.True(t, ProfileExists(dir))
}

func TestDefaultProfilePath(t *testing.T) {
	got := DefaultProfilePath("/home/user/workspace")
	assert.Equal(t, "/home/user/workspace/aieos.json", got)
}
