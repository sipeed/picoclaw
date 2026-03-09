package model

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/pkg/config"
)

var configPath = ""

func initTest(t *testing.T) {
	tmpDir := t.TempDir()
	configPath = filepath.Join(tmpDir, "config.json")
	_ = os.Setenv("PICOCLAW_CONFIG", configPath)
}

func TestNewModelCommand(t *testing.T) {
	cmd := NewModelCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "model [model_name]", cmd.Use)
	assert.Equal(t, "Show or change the default model", cmd.Short)

	assert.Len(t, cmd.Aliases, 0)

	assert.False(t, cmd.HasFlags())

	assert.Nil(t, cmd.Run)
	assert.NotNil(t, cmd.RunE)

	assert.Nil(t, cmd.PersistentPreRunE)
	assert.Nil(t, cmd.PersistentPreRun)
	assert.Nil(t, cmd.PersistentPostRun)
}

func TestShowCurrentModel_WithDefaultModel(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ModelName: "gpt-4",
			},
		},
		ModelList: []config.ModelConfig{
			{ModelName: "gpt-4", Model: "openai/gpt-4", APIKey: "test"},
			{ModelName: "claude-3", Model: "anthropic/claude-3", APIKey: "test"},
		},
	}

	showCurrentModel(cfg)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "Current default model: gpt-4")
	assert.Contains(t, output, "Available models in your config:")
	assert.Contains(t, output, "gpt-4")
	assert.Contains(t, output, "claude-3")
}

func TestShowCurrentModel_NoDefaultModel(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ModelName: "",
				Model:     "",
			},
		},
		ModelList: []config.ModelConfig{
			{ModelName: "gpt-4", Model: "openai/gpt-4", APIKey: "test"},
		},
	}

	showCurrentModel(cfg)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "No default model is currently set.")
	assert.Contains(t, output, "Available models in your config:")
}

func TestShowCurrentModel_BackwardCompatibility(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Model: "legacy-model",
			},
		},
		ModelList: []config.ModelConfig{},
	}

	showCurrentModel(cfg)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "Current default model: legacy-model")
}

func TestListAvailableModels_Empty(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := &config.Config{
		ModelList: []config.ModelConfig{},
	}

	listAvailableModels(cfg)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "No models configured in model_list")
}

func TestListAvailableModels_WithModels(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ModelName: "gpt-4",
			},
		},
		ModelList: []config.ModelConfig{
			{ModelName: "gpt-4", Model: "openai/gpt-4", APIKey: "test"},
			{ModelName: "claude-3", Model: "anthropic/claude-3", APIKey: "test"},
			{ModelName: "no-key-model", Model: "openai/test", APIKey: ""},
		},
	}

	listAvailableModels(cfg)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.NotEmpty(t, output)
	assert.Contains(t, output, "> - gpt-4 (openai/gpt-4)")
	assert.Contains(t, output, "claude-3 (anthropic/claude-3)")
	assert.NotContains(t, output, "no-key-model")
}

func TestSetDefaultModel_ValidModel(t *testing.T) {
	initTest(t)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ModelName: "old-model",
			},
		},
		ModelList: []config.ModelConfig{
			{ModelName: "new-model", Model: "openai/new-model", APIKey: "test"},
			{ModelName: "old-model", Model: "openai/old-model", APIKey: "test"},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := setDefaultModel(configPath, cfg, "new-model")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Default model changed from 'old-model' to 'new-model'")

	// Verify config was updated
	updatedCfg, err := config.LoadConfig(configPath)
	require.NoError(t, err)
	assert.Equal(t, "new-model", updatedCfg.Agents.Defaults.ModelName)
	assert.Empty(t, updatedCfg.Agents.Defaults.Model)
}

func TestSetDefaultModel_LegacyModelField(t *testing.T) {
	initTest(t)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Model: "legacy-old",
			},
		},
		ModelList: []config.ModelConfig{
			{ModelName: "new-model", Model: "openai/new-model", APIKey: "test"},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := setDefaultModel(configPath, cfg, "new-model")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Default model changed from 'legacy-old' to 'new-model'")
}

func TestSetDefaultModel_InvalidModel(t *testing.T) {
	initTest(t)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ModelName: "existing-model",
			},
		},
		ModelList: []config.ModelConfig{
			{ModelName: "existing-model", Model: "openai/existing", APIKey: "test"},
		},
	}

	assert.Error(t, setDefaultModel(configPath, cfg, "nonexistent-model"))
}

func TestSetDefaultModel_ModelWithoutAPIKey(t *testing.T) {
	initTest(t)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ModelName: "existing-model",
			},
		},
		ModelList: []config.ModelConfig{
			{ModelName: "existing-model", Model: "openai/existing", APIKey: "test"},
			{ModelName: "no-key-model", Model: "openai/nokey", APIKey: ""},
		},
	}

	assert.Error(t, setDefaultModel(configPath, cfg, "no-key-model"))
}

func TestSetDefaultModel_SaveConfigError(t *testing.T) {
	// Use an invalid path to trigger save error
	invalidPath := "/nonexistent/directory/config.json"

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ModelName: "old-model",
			},
		},
		ModelList: []config.ModelConfig{
			{ModelName: "new-model", Model: "openai/new-model", APIKey: "test"},
		},
	}

	err := setDefaultModel(invalidPath, cfg, "new-model")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save config")
}

func TestFormatModelName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", "(none)"},
		{"simple model", "gpt-4", "gpt-4"},
		{"model with version", "claude-sonnet-4.6", "claude-sonnet-4.6"},
		{"model with spaces", "my model", "my model"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatModelName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestModelCommandExecution_Show(t *testing.T) {
	initTest(t)

	// Create a test config
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ModelName: "test-model",
			},
		},
		ModelList: []config.ModelConfig{
			{ModelName: "test-model", Model: "openai/test", APIKey: "test"},
		},
	}

	err := config.SaveConfig(configPath, cfg)
	require.NoError(t, err)

	cmd := NewModelCommand()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = cmd.RunE(cmd, []string{})

	w.Close()
	os.Stdout = oldStdout

	assert.NoError(t, err)

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "Current default model: test-model")
}

func TestModelCommandExecution_Set(t *testing.T) {
	initTest(t)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ModelName: "old-model",
			},
		},
		ModelList: []config.ModelConfig{
			{ModelName: "old-model", Model: "openai/old", APIKey: "test"},
			{ModelName: "new-model", Model: "openai/new", APIKey: "test"},
		},
	}

	err := config.SaveConfig(configPath, cfg)
	require.NoError(t, err)

	cmd := NewModelCommand()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = cmd.RunE(cmd, []string{"new-model"})

	w.Close()
	os.Stdout = oldStdout

	assert.NoError(t, err)

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "Default model changed from 'old-model' to 'new-model'")
}

func TestModelCommandExecution_TooManyArgs(t *testing.T) {
	cmd := NewModelCommand()

	err := cmd.RunE(cmd, []string{"model1", "model2"})

	assert.Error(t, err)
}

func TestListAvailableModels_MarkerLogic(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ModelName: "middle-model",
			},
		},
		ModelList: []config.ModelConfig{
			{ModelName: "first-model", Model: "openai/first", APIKey: "test"},
			{ModelName: "middle-model", Model: "openai/middle", APIKey: "test"},
			{ModelName: "last-model", Model: "openai/last", APIKey: "test"},
		},
	}

	listAvailableModels(cfg)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "  - first-model (openai/first)")
	assert.Contains(t, output, "> - middle-model (openai/middle)")
	assert.Contains(t, output, "  - last-model (openai/last)")
}
