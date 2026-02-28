package openclaw

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOpenclawHandler(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "openclaw.json")
	err := os.WriteFile(configPath, []byte("{}"), 0o644)
	require.NoError(t, err)

	handler, err := NewOpenclawHandler(Options{
		SourceHome: tmpDir,
	})
	require.NoError(t, err)
	require.NotNil(t, handler)
}

func TestNewOpenclawHandlerNoConfig(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := NewOpenclawHandler(Options{
		SourceHome: tmpDir,
	})
	require.Error(t, err)
}

func TestOpenclawHandlerGetSourceName(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "openclaw.json")
	err := os.WriteFile(configPath, []byte("{}"), 0o644)
	require.NoError(t, err)

	handler, err := NewOpenclawHandler(Options{
		SourceHome: tmpDir,
	})
	require.NoError(t, err)

	assert.Equal(t, "openclaw", handler.GetSourceName())
}

func TestOpenclawHandlerGetSourceHome(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "openclaw.json")
	err := os.WriteFile(configPath, []byte("{}"), 0o644)
	require.NoError(t, err)

	handler, err := NewOpenclawHandler(Options{
		SourceHome: tmpDir,
	})
	require.NoError(t, err)

	home, err := handler.GetSourceHome()
	require.NoError(t, err)
	assert.Equal(t, tmpDir, home)
}

func TestOpenclawHandlerGetSourceWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "openclaw.json")
	err := os.WriteFile(configPath, []byte("{}"), 0o644)
	require.NoError(t, err)

	handler, err := NewOpenclawHandler(Options{
		SourceHome: tmpDir,
	})
	require.NoError(t, err)

	workspace, err := handler.GetSourceWorkspace()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpDir, "workspace"), workspace)
}

func TestOpenclawHandlerGetSourceConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "openclaw.json")
	err := os.WriteFile(configPath, []byte("{}"), 0o644)
	require.NoError(t, err)

	handler, err := NewOpenclawHandler(Options{
		SourceHome: tmpDir,
	})
	require.NoError(t, err)

	configFile, err := handler.GetSourceConfigFile()
	require.NoError(t, err)
	assert.Equal(t, configPath, configFile)
}

func TestOpenclawHandlerGetSourceConfigFileWithConfigJson(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	err := os.WriteFile(configPath, []byte("{}"), 0o644)
	require.NoError(t, err)

	handler, err := NewOpenclawHandler(Options{
		SourceHome: tmpDir,
	})
	require.NoError(t, err)

	configFile, err := handler.GetSourceConfigFile()
	require.NoError(t, err)
	assert.Equal(t, configPath, configFile)
}

func TestOpenclawHandlerGetMigrateableFiles(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "openclaw.json")
	err := os.WriteFile(configPath, []byte("{}"), 0o644)
	require.NoError(t, err)

	handler, err := NewOpenclawHandler(Options{
		SourceHome: tmpDir,
	})
	require.NoError(t, err)

	files := handler.GetMigrateableFiles()
	assert.NotEmpty(t, files)
	assert.Contains(t, files, "AGENTS.md")
	assert.Contains(t, files, "SOUL.md")
	assert.Contains(t, files, "USER.md")
}

func TestOpenclawHandlerGetMigrateableDirs(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "openclaw.json")
	err := os.WriteFile(configPath, []byte("{}"), 0o644)
	require.NoError(t, err)

	handler, err := NewOpenclawHandler(Options{
		SourceHome: tmpDir,
	})
	require.NoError(t, err)

	dirs := handler.GetMigrateableDirs()
	assert.NotEmpty(t, dirs)
	assert.Contains(t, dirs, "memory")
	assert.Contains(t, dirs, "skills")
}

func TestResolveSourceHome(t *testing.T) {
	result, err := resolveSourceHome("/custom/path")
	require.NoError(t, err)
	assert.Equal(t, "/custom/path", result)
}

func TestResolveSourceHomeWithEnvVar(t *testing.T) {
	t.Setenv("OPENCLAW_HOME", "/env/path")

	result, err := resolveSourceHome("")
	require.NoError(t, err)
	assert.Equal(t, "/env/path", result)
}

func TestResolveSourceHomeWithTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	result, err := resolveSourceHome("~/openclaw")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, "openclaw"), result)
}

func TestFindSourceConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "openclaw.json")
	err := os.WriteFile(configPath, []byte("{}"), 0o644)
	require.NoError(t, err)

	result, err := findSourceConfig(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, configPath, result)
}

func TestFindSourceConfigWithConfigJson(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	err := os.WriteFile(configPath, []byte("{}"), 0o644)
	require.NoError(t, err)

	result, err := findSourceConfig(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, configPath, result)
}

func TestFindSourceConfigNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := findSourceConfig(tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no config file found")
}

func TestMapProvider(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"anthropic", "anthropic"},
		{"claude", "anthropic"},
		{"openai", "openai"},
		{"gpt", "openai"},
		{"groq", "groq"},
		{"ollama", "ollama"},
		{"openrouter", "openrouter"},
		{"deepseek", "deepseek"},
		{"together", "together"},
		{"mistral", "mistral"},
		{"fireworks", "fireworks"},
		{"google", "google"},
		{"gemini", "google"},
		{"xai", "xai"},
		{"grok", "xai"},
		{"cerebras", "cerebras"},
		{"sambanova", "sambanova"},
		{"unknown", "unknown"},
		{"", ""},
	}

	for _, tt := range tests {
		result := mapProvider(tt.input)
		assert.Equal(t, tt.expected, result, "mapProvider(%q)", tt.input)
	}
}

func TestRewriteWorkspacePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"~/.openclaw/workspace", "~/.picoclaw/workspace"},
		{"/home/user/.openclaw/workspace", "/home/user/.picoclaw/workspace"},
		{"/path/without/openclaw/change", "/path/without/openclaw/change"},
		{"", ""},
	}

	for _, tt := range tests {
		result := rewriteWorkspacePath(tt.input)
		assert.Equal(t, tt.expected, result, "rewriteWorkspacePath(%q)", tt.input)
	}
}
