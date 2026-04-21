package status

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = oldStdout
	defer r.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("io.Copy() error = %v", err)
	}
	return buf.String()
}

func TestStatusCmd_RecognizesProviderFieldWithoutModelPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	workspace := filepath.Join(tmpDir, "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	t.Setenv(config.EnvConfig, configPath)
	t.Setenv(config.EnvHome, tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ModelName:   "gpt-5.4",
				Workspace:   workspace,
				Provider:    "openai",
				MaxTokens:   65536,
				Temperature: nil,
			},
		},
		ModelList: []*config.ModelConfig{
			{
				ModelName: "gpt-5.4",
				Provider:  "openai",
				Model:     "gpt-5.4",
				APIBase:   "https://api.openai.com/v1",
				APIKeys:   config.SimpleSecureStrings("test-key"),
				Enabled:   true,
			},
			{
				ModelName: "qwen-plus",
				Provider:  "qwen",
				Model:     "qwen-plus",
				APIBase:   "https://dashscope.aliyuncs.com/compatible-mode/v1",
				APIKeys:   config.SimpleSecureStrings("test-key"),
				Enabled:   true,
			},
		},
	}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("config.SaveConfig() error = %v", err)
	}

	output := captureStdout(t, statusCmd)

	if !strings.Contains(output, "OpenAI API: \u2713") {
		t.Fatalf("status output missing OpenAI provider: %s", output)
	}
	if !strings.Contains(output, "Qwen API: \u2713") {
		t.Fatalf("status output missing Qwen provider: %s", output)
	}
}
