package migrate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "apiKey", "api_key"},
		{"two words", "apiBase", "api_base"},
		{"three words", "maxToolIterations", "max_tool_iterations"},
		{"already snake", "api_key", "api_key"},
		{"single word", "enabled", "enabled"},
		{"all lower", "model", "model"},
		{"consecutive caps", "apiURL", "api_url"},
		{"starts upper", "Model", "model"},
		{"bridge url", "bridgeUrl", "bridge_url"},
		{"client id", "clientId", "client_id"},
		{"app secret", "appSecret", "app_secret"},
		{"verification token", "verificationToken", "verification_token"},
		{"allow from", "allowFrom", "allow_from"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := camelToSnake(tt.input)
			if got != tt.want {
				t.Errorf("camelToSnake(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestConvertKeysToSnake(t *testing.T) {
	input := map[string]interface{}{
		"apiKey":  "test-key",
		"apiBase": "https://example.com",
		"nested": map[string]interface{}{
			"maxTokens": float64(8192),
			"allowFrom": []interface{}{"user1", "user2"},
			"deeperLevel": map[string]interface{}{
				"clientId": "abc",
			},
		},
	}

	result := convertKeysToSnake(input)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map[string]interface{}")
	}

	if _, ok := m["api_key"]; !ok {
		t.Error("expected key 'api_key' after conversion")
	}
	if _, ok := m["api_base"]; !ok {
		t.Error("expected key 'api_base' after conversion")
	}

	nested, ok := m["nested"].(map[string]interface{})
	if !ok {
		t.Fatal("expected nested map")
	}
	if _, ok := nested["max_tokens"]; !ok {
		t.Error("expected key 'max_tokens' in nested map")
	}
	if _, ok := nested["allow_from"]; !ok {
		t.Error("expected key 'allow_from' in nested map")
	}

	deeper, ok := nested["deeper_level"].(map[string]interface{})
	if !ok {
		t.Fatal("expected deeper_level map")
	}
	if _, ok := deeper["client_id"]; !ok {
		t.Error("expected key 'client_id' in deeper level")
	}
}

func TestLoadOpenClawConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "openclaw.json")

	openclawConfig := map[string]interface{}{
		"providers": map[string]interface{}{
			"anthropic": map[string]interface{}{
				"apiKey":  "sk-ant-test123",
				"apiBase": "https://api.anthropic.com",
			},
		},
		"agents": map[string]interface{}{
			"defaults": map[string]interface{}{
				"maxTokens": float64(4096),
				"model":     "claude-3-opus",
			},
		},
	}

	data, err := json.Marshal(openclawConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadOpenClawConfig(configPath)
	if err != nil {
		t.Fatalf("LoadOpenClawConfig: %v", err)
	}

	providers, ok := result["providers"].(map[string]interface{})
	if !ok {
		t.Fatal("expected providers map")
	}
	anthropic, ok := providers["anthropic"].(map[string]interface{})
	if !ok {
		t.Fatal("expected anthropic map")
	}
	if anthropic["api_key"] != "sk-ant-test123" {
		t.Errorf("api_key = %v, want sk-ant-test123", anthropic["api_key"])
	}

	agents, ok := result["agents"].(map[string]interface{})
	if !ok {
		t.Fatal("expected agents map")
	}
	defaults, ok := agents["defaults"].(map[string]interface{})
	if !ok {
		t.Fatal("expected defaults map")
	}
	if defaults["max_tokens"] != float64(4096) {
		t.Errorf("max_tokens = %v, want 4096", defaults["max_tokens"])
	}
}

func TestConvertConfig(t *testing.T) {
	t.Run("providers mapping", func(t *testing.T) {
		data := map[string]interface{}{
			"providers": map[string]interface{}{
				"anthropic": map[string]interface{}{
					"api_key":  "sk-ant-test",
					"api_base": "https://api.anthropic.com",
				},
				"openrouter": map[string]interface{}{
					"api_key": "sk-or-test",
				},
				"groq": map[string]interface{}{
					"api_key": "gsk-test",
				},
			},
		}

		cfg, warnings, err := ConvertConfig(data)
		if err != nil {
			t.Fatalf("ConvertConfig: %v", err)
		}
		if len(warnings) != 0 {
			t.Errorf("expected no warnings, got %v", warnings)
		}
		if cfg.Providers.Anthropic.APIKey != "sk-ant-test" {
			t.Errorf("Anthropic.APIKey = %q, want %q", cfg.Providers.Anthropic.APIKey, "sk-ant-test")
		}
		if cfg.Providers.OpenRouter.APIKey != "sk-or-test" {
			t.Errorf("OpenRouter.APIKey = %q, want %q", cfg.Providers.OpenRouter.APIKey, "sk-or-test")
		}
		if cfg.Providers.Groq.APIKey != "gsk-test" {
			t.Errorf("Groq.APIKey = %q, want %q", cfg.Providers.Groq.APIKey, "gsk-test")
		}
	})

	t.Run("unsupported provider warning", func(t *testing.T) {
		data := map[string]interface{}{
			"providers": map[string]interface{}{
				"unknown_provider": map[string]interface{}{
					"api_key": "sk-test",
				},
			},
		}

		_, warnings, err := ConvertConfig(data)
		if err != nil {
			t.Fatalf("ConvertConfig: %v", err)
		}
		if len(warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(warnings))
		}
		if warnings[0] != "Provider 'unknown_provider' not supported in PicoClaw, skipping" {
			t.Errorf("unexpected warning: %s", warnings[0])
		}
	})

	t.Run("channels mapping", func(t *testing.T) {
		data := map[string]interface{}{
			"channels": map[string]interface{}{
				"telegram": map[string]interface{}{
					"enabled":    true,
					"token":      "tg-token-123",
					"allow_from": []interface{}{"user1"},
				},
				"discord": map[string]interface{}{
					"enabled": true,
					"token":   "disc-token-456",
				},
			},
		}

		cfg, _, err := ConvertConfig(data)
		if err != nil {
			t.Fatalf("ConvertConfig: %v", err)
		}
		if !cfg.Channels.Telegram.Enabled {
			t.Error("Telegram should be enabled")
		}
		if cfg.Channels.Telegram.Token != "tg-token-123" {
			t.Errorf("Telegram.Token = %q, want %q", cfg.Channels.Telegram.Token, "tg-token-123")
		}
		if len(cfg.Channels.Telegram.AllowFrom) != 1 || cfg.Channels.Telegram.AllowFrom[0] != "user1" {
			t.Errorf("Telegram.AllowFrom = %v, want [user1]", cfg.Channels.Telegram.AllowFrom)
		}
		if !cfg.Channels.Discord.Enabled {
			t.Error("Discord should be enabled")
		}
	})

	t.Run("unsupported channel warning", func(t *testing.T) {
		data := map[string]interface{}{
			"channels": map[string]interface{}{
				"email": map[string]interface{}{
					"enabled": true,
				},
			},
		}

		_, warnings, err := ConvertConfig(data)
		if err != nil {
			t.Fatalf("ConvertConfig: %v", err)
		}
		if len(warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(warnings))
		}
		if warnings[0] != "Channel 'email' not supported in PicoClaw, skipping" {
			t.Errorf("unexpected warning: %s", warnings[0])
		}
	})

	t.Run("agent defaults", func(t *testing.T) {
		data := map[string]interface{}{
			"agents": map[string]interface{}{
				"defaults": map[string]interface{}{
					"model":               "claude-3-opus",
					"max_tokens":          float64(4096),
					"temperature":         0.5,
					"max_tool_iterations": float64(10),
					"workspace":           "~/.openclaw/workspace",
				},
			},
		}

		cfg, _, err := ConvertConfig(data)
		if err != nil {
			t.Fatalf("ConvertConfig: %v", err)
		}
		if cfg.Agents.Defaults.Model != "claude-3-opus" {
			t.Errorf("Model = %q, want %q", cfg.Agents.Defaults.Model, "claude-3-opus")
		}
		if cfg.Agents.Defaults.MaxTokens != 4096 {
			t.Errorf("MaxTokens = %d, want %d", cfg.Agents.Defaults.MaxTokens, 4096)
		}
		if cfg.Agents.Defaults.Temperature == nil {
			t.Fatalf("Temperature is nil, want %f", 0.5)
		}
		if *cfg.Agents.Defaults.Temperature != 0.5 {
			t.Errorf("Temperature = %f, want %f", *cfg.Agents.Defaults.Temperature, 0.5)
		}
		if cfg.Agents.Defaults.Workspace != "~/.picoclaw/workspace" {
			t.Errorf("Workspace = %q, want %q", cfg.Agents.Defaults.Workspace, "~/.picoclaw/workspace")
		}
	})

	t.Run("empty config", func(t *testing.T) {
		data := map[string]interface{}{}

		cfg, warnings, err := ConvertConfig(data)
		if err != nil {
			t.Fatalf("ConvertConfig: %v", err)
		}
		if len(warnings) != 0 {
			t.Errorf("expected no warnings, got %v", warnings)
		}
		if cfg.Agents.Defaults.Model != "glm-4.7" {
			t.Errorf("default model should be glm-4.7, got %q", cfg.Agents.Defaults.Model)
		}
	})
}

func TestSupportedProvidersCompatibility(t *testing.T) {
	expected := []string{
		"anthropic",
		"openai",
		"openrouter",
		"groq",
		"zhipu",
		"vllm",
		"gemini",
	}

	for _, provider := range expected {
		if !supportedProviders[provider] {
			t.Fatalf("supportedProviders missing expected key %q", provider)
		}
	}
}

func TestMergeConfig(t *testing.T) {
	t.Run("fills empty fields", func(t *testing.T) {
		existing := config.DefaultConfig()
		incoming := config.DefaultConfig()
		incoming.Providers.Anthropic.APIKey = "sk-ant-incoming"
		incoming.Providers.OpenRouter.APIKey = "sk-or-incoming"

		result := MergeConfig(existing, incoming)
		if result.Providers.Anthropic.APIKey != "sk-ant-incoming" {
			t.Errorf("Anthropic.APIKey = %q, want %q", result.Providers.Anthropic.APIKey, "sk-ant-incoming")
		}
		if result.Providers.OpenRouter.APIKey != "sk-or-incoming" {
			t.Errorf("OpenRouter.APIKey = %q, want %q", result.Providers.OpenRouter.APIKey, "sk-or-incoming")
		}
	})

	t.Run("preserves existing non-empty fields", func(t *testing.T) {
		existing := config.DefaultConfig()
		existing.Providers.Anthropic.APIKey = "sk-ant-existing"

		incoming := config.DefaultConfig()
		incoming.Providers.Anthropic.APIKey = "sk-ant-incoming"
		incoming.Providers.OpenAI.APIKey = "sk-oai-incoming"

		result := MergeConfig(existing, incoming)
		if result.Providers.Anthropic.APIKey != "sk-ant-existing" {
			t.Errorf("Anthropic.APIKey should be preserved, got %q", result.Providers.Anthropic.APIKey)
		}
		if result.Providers.OpenAI.APIKey != "sk-oai-incoming" {
			t.Errorf("OpenAI.APIKey should be filled, got %q", result.Providers.OpenAI.APIKey)
		}
	})

	t.Run("merges enabled channels", func(t *testing.T) {
		existing := config.DefaultConfig()
		incoming := config.DefaultConfig()
		incoming.Channels.Telegram.Enabled = true
		incoming.Channels.Telegram.Token = "tg-token"

		result := MergeConfig(existing, incoming)
		if !result.Channels.Telegram.Enabled {
			t.Error("Telegram should be enabled after merge")
		}
		if result.Channels.Telegram.Token != "tg-token" {
			t.Errorf("Telegram.Token = %q, want %q", result.Channels.Telegram.Token, "tg-token")
		}
	})

	t.Run("preserves existing enabled channels", func(t *testing.T) {
		existing := config.DefaultConfig()
		existing.Channels.Telegram.Enabled = true
		existing.Channels.Telegram.Token = "existing-token"

		incoming := config.DefaultConfig()
		incoming.Channels.Telegram.Enabled = true
		incoming.Channels.Telegram.Token = "incoming-token"

		result := MergeConfig(existing, incoming)
		if result.Channels.Telegram.Token != "existing-token" {
			t.Errorf("Telegram.Token should be preserved, got %q", result.Channels.Telegram.Token)
		}
	})
}

func TestPlanWorkspaceMigration(t *testing.T) {
	t.Run("copies available files", func(t *testing.T) {
		srcDir := t.TempDir()
		dstDir := t.TempDir()

		os.WriteFile(filepath.Join(srcDir, "AGENTS.md"), []byte("# Agents"), 0644)
		os.WriteFile(filepath.Join(srcDir, "SOUL.md"), []byte("# Soul"), 0644)
		os.WriteFile(filepath.Join(srcDir, "USER.md"), []byte("# User"), 0644)

		actions, err := PlanWorkspaceMigration(srcDir, dstDir, false)
		if err != nil {
			t.Fatalf("PlanWorkspaceMigration: %v", err)
		}

		copyCount := 0
		skipCount := 0
		for _, a := range actions {
			if a.Type == ActionCopy {
				copyCount++
			}
			if a.Type == ActionSkip {
				skipCount++
			}
		}
		if copyCount != 3 {
			t.Errorf("expected 3 copies, got %d", copyCount)
		}
		if skipCount != 2 {
			t.Errorf("expected 2 skips (TOOLS.md, HEARTBEAT.md), got %d", skipCount)
		}
	})

	t.Run("plans backup for existing destination files", func(t *testing.T) {
		srcDir := t.TempDir()
		dstDir := t.TempDir()

		os.WriteFile(filepath.Join(srcDir, "AGENTS.md"), []byte("# Agents from OpenClaw"), 0644)
		os.WriteFile(filepath.Join(dstDir, "AGENTS.md"), []byte("# Existing Agents"), 0644)

		actions, err := PlanWorkspaceMigration(srcDir, dstDir, false)
		if err != nil {
			t.Fatalf("PlanWorkspaceMigration: %v", err)
		}

		backupCount := 0
		for _, a := range actions {
			if a.Type == ActionBackup && filepath.Base(a.Destination) == "AGENTS.md" {
				backupCount++
			}
		}
		if backupCount != 1 {
			t.Errorf("expected 1 backup action for AGENTS.md, got %d", backupCount)
		}
	})

	t.Run("force skips backup", func(t *testing.T) {
		srcDir := t.TempDir()
		dstDir := t.TempDir()

		os.WriteFile(filepath.Join(srcDir, "AGENTS.md"), []byte("# Agents"), 0644)
		os.WriteFile(filepath.Join(dstDir, "AGENTS.md"), []byte("# Existing"), 0644)

		actions, err := PlanWorkspaceMigration(srcDir, dstDir, true)
		if err != nil {
			t.Fatalf("PlanWorkspaceMigration: %v", err)
		}

		for _, a := range actions {
			if a.Type == ActionBackup {
				t.Error("expected no backup actions with force=true")
			}
		}
	})

	t.Run("handles memory directory", func(t *testing.T) {
		srcDir := t.TempDir()
		dstDir := t.TempDir()

		memDir := filepath.Join(srcDir, "memory")
		os.MkdirAll(memDir, 0755)
		os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte("# Memory"), 0644)

		actions, err := PlanWorkspaceMigration(srcDir, dstDir, false)
		if err != nil {
			t.Fatalf("PlanWorkspaceMigration: %v", err)
		}

		hasCopy := false
		hasDir := false
		for _, a := range actions {
			if a.Type == ActionCopy && filepath.Base(a.Source) == "MEMORY.md" {
				hasCopy = true
			}
			if a.Type == ActionCreateDir {
				hasDir = true
			}
		}
		if !hasCopy {
			t.Error("expected copy action for memory/MEMORY.md")
		}
		if !hasDir {
			t.Error("expected create dir action for memory/")
		}
	})

	t.Run("handles skills directory", func(t *testing.T) {
		srcDir := t.TempDir()
		dstDir := t.TempDir()

		skillDir := filepath.Join(srcDir, "skills", "weather")
		os.MkdirAll(skillDir, 0755)
		os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Weather"), 0644)

		actions, err := PlanWorkspaceMigration(srcDir, dstDir, false)
		if err != nil {
			t.Fatalf("PlanWorkspaceMigration: %v", err)
		}

		hasCopy := false
		for _, a := range actions {
			if a.Type == ActionCopy && filepath.Base(a.Source) == "SKILL.md" {
				hasCopy = true
			}
		}
		if !hasCopy {
			t.Error("expected copy action for skills/weather/SKILL.md")
		}
	})
}

func TestFindOpenClawConfig(t *testing.T) {
	t.Run("finds openclaw.json", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "openclaw.json")
		os.WriteFile(configPath, []byte("{}"), 0644)

		found, err := findOpenClawConfig(tmpDir)
		if err != nil {
			t.Fatalf("findOpenClawConfig: %v", err)
		}
		if found != configPath {
			t.Errorf("found %q, want %q", found, configPath)
		}
	})

	t.Run("falls back to config.json", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")
		os.WriteFile(configPath, []byte("{}"), 0644)

		found, err := findOpenClawConfig(tmpDir)
		if err != nil {
			t.Fatalf("findOpenClawConfig: %v", err)
		}
		if found != configPath {
			t.Errorf("found %q, want %q", found, configPath)
		}
	})

	t.Run("prefers openclaw.json over config.json", func(t *testing.T) {
		tmpDir := t.TempDir()
		openclawPath := filepath.Join(tmpDir, "openclaw.json")
		os.WriteFile(openclawPath, []byte("{}"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte("{}"), 0644)

		found, err := findOpenClawConfig(tmpDir)
		if err != nil {
			t.Fatalf("findOpenClawConfig: %v", err)
		}
		if found != openclawPath {
			t.Errorf("should prefer openclaw.json, got %q", found)
		}
	})

	t.Run("error when no config found", func(t *testing.T) {
		tmpDir := t.TempDir()

		_, err := findOpenClawConfig(tmpDir)
		if err == nil {
			t.Fatal("expected error when no config found")
		}
	})
}

func TestRewriteWorkspacePath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"default path", "~/.openclaw/workspace", "~/.picoclaw/workspace"},
		{"custom path", "/custom/path", "/custom/path"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewriteWorkspacePath(tt.input)
			if got != tt.want {
				t.Errorf("rewriteWorkspacePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRunDryRun(t *testing.T) {
	openclawHome := t.TempDir()
	picoClawHome := t.TempDir()

	wsDir := filepath.Join(openclawHome, "workspace")
	os.MkdirAll(wsDir, 0755)
	os.WriteFile(filepath.Join(wsDir, "SOUL.md"), []byte("# Soul"), 0644)
	os.WriteFile(filepath.Join(wsDir, "AGENTS.md"), []byte("# Agents"), 0644)

	configData := map[string]interface{}{
		"providers": map[string]interface{}{
			"anthropic": map[string]interface{}{
				"apiKey": "test-key",
			},
		},
	}
	data, _ := json.Marshal(configData)
	os.WriteFile(filepath.Join(openclawHome, "openclaw.json"), data, 0644)

	opts := Options{
		DryRun:       true,
		OpenClawHome: openclawHome,
		PicoClawHome: picoClawHome,
	}

	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	picoWs := filepath.Join(picoClawHome, "workspace")
	if _, err := os.Stat(filepath.Join(picoWs, "SOUL.md")); !os.IsNotExist(err) {
		t.Error("dry run should not create files")
	}
	if _, err := os.Stat(filepath.Join(picoClawHome, "config.json")); !os.IsNotExist(err) {
		t.Error("dry run should not create config")
	}

	_ = result
}

func TestRunFullMigration(t *testing.T) {
	openclawHome := t.TempDir()
	picoClawHome := t.TempDir()

	wsDir := filepath.Join(openclawHome, "workspace")
	os.MkdirAll(wsDir, 0755)
	os.WriteFile(filepath.Join(wsDir, "SOUL.md"), []byte("# Soul from OpenClaw"), 0644)
	os.WriteFile(filepath.Join(wsDir, "AGENTS.md"), []byte("# Agents from OpenClaw"), 0644)
	os.WriteFile(filepath.Join(wsDir, "USER.md"), []byte("# User from OpenClaw"), 0644)

	memDir := filepath.Join(wsDir, "memory")
	os.MkdirAll(memDir, 0755)
	os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte("# Memory notes"), 0644)

	configData := map[string]interface{}{
		"providers": map[string]interface{}{
			"anthropic": map[string]interface{}{
				"apiKey": "sk-ant-migrate-test",
			},
			"openrouter": map[string]interface{}{
				"apiKey": "sk-or-migrate-test",
			},
		},
		"channels": map[string]interface{}{
			"telegram": map[string]interface{}{
				"enabled": true,
				"token":   "tg-migrate-test",
			},
		},
	}
	data, _ := json.Marshal(configData)
	os.WriteFile(filepath.Join(openclawHome, "openclaw.json"), data, 0644)

	opts := Options{
		Force:        true,
		OpenClawHome: openclawHome,
		PicoClawHome: picoClawHome,
	}

	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	picoWs := filepath.Join(picoClawHome, "workspace")

	soulData, err := os.ReadFile(filepath.Join(picoWs, "SOUL.md"))
	if err != nil {
		t.Fatalf("reading SOUL.md: %v", err)
	}
	if string(soulData) != "# Soul from OpenClaw" {
		t.Errorf("SOUL.md content = %q, want %q", string(soulData), "# Soul from OpenClaw")
	}

	agentsData, err := os.ReadFile(filepath.Join(picoWs, "AGENTS.md"))
	if err != nil {
		t.Fatalf("reading AGENTS.md: %v", err)
	}
	if string(agentsData) != "# Agents from OpenClaw" {
		t.Errorf("AGENTS.md content = %q", string(agentsData))
	}

	memData, err := os.ReadFile(filepath.Join(picoWs, "memory", "MEMORY.md"))
	if err != nil {
		t.Fatalf("reading memory/MEMORY.md: %v", err)
	}
	if string(memData) != "# Memory notes" {
		t.Errorf("MEMORY.md content = %q", string(memData))
	}

	picoConfig, err := config.LoadConfig(filepath.Join(picoClawHome, "config.json"))
	if err != nil {
		t.Fatalf("loading PicoClaw config: %v", err)
	}
	if picoConfig.Providers.Anthropic.APIKey != "sk-ant-migrate-test" {
		t.Errorf("Anthropic.APIKey = %q, want %q", picoConfig.Providers.Anthropic.APIKey, "sk-ant-migrate-test")
	}
	if picoConfig.Providers.OpenRouter.APIKey != "sk-or-migrate-test" {
		t.Errorf("OpenRouter.APIKey = %q, want %q", picoConfig.Providers.OpenRouter.APIKey, "sk-or-migrate-test")
	}
	if !picoConfig.Channels.Telegram.Enabled {
		t.Error("Telegram should be enabled")
	}
	if picoConfig.Channels.Telegram.Token != "tg-migrate-test" {
		t.Errorf("Telegram.Token = %q, want %q", picoConfig.Channels.Telegram.Token, "tg-migrate-test")
	}

	if result.FilesCopied < 3 {
		t.Errorf("expected at least 3 files copied, got %d", result.FilesCopied)
	}
	if !result.ConfigMigrated {
		t.Error("config should have been migrated")
	}
	if len(result.Errors) > 0 {
		t.Errorf("expected no errors, got %v", result.Errors)
	}
}

func TestRunOpenClawNotFound(t *testing.T) {
	opts := Options{
		OpenClawHome: "/nonexistent/path/to/openclaw",
		PicoClawHome: t.TempDir(),
	}

	_, err := Run(opts)
	if err == nil {
		t.Fatal("expected error when OpenClaw not found")
	}
}

func TestRunMutuallyExclusiveFlags(t *testing.T) {
	opts := Options{
		ConfigOnly:    true,
		WorkspaceOnly: true,
	}

	_, err := Run(opts)
	if err == nil {
		t.Fatal("expected error for mutually exclusive flags")
	}
}

func TestBackupFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")
	os.WriteFile(filePath, []byte("original content"), 0644)

	if err := backupFile(filePath); err != nil {
		t.Fatalf("backupFile: %v", err)
	}

	bakPath := filePath + ".bak"
	bakData, err := os.ReadFile(bakPath)
	if err != nil {
		t.Fatalf("reading backup: %v", err)
	}
	if string(bakData) != "original content" {
		t.Errorf("backup content = %q, want %q", string(bakData), "original content")
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "src.md")
	dstPath := filepath.Join(tmpDir, "dst.md")

	os.WriteFile(srcPath, []byte("file content"), 0644)

	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	data, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("reading copy: %v", err)
	}
	if string(data) != "file content" {
		t.Errorf("copy content = %q, want %q", string(data), "file content")
	}
}

func TestRunConfigOnly(t *testing.T) {
	openclawHome := t.TempDir()
	picoClawHome := t.TempDir()

	wsDir := filepath.Join(openclawHome, "workspace")
	os.MkdirAll(wsDir, 0755)
	os.WriteFile(filepath.Join(wsDir, "SOUL.md"), []byte("# Soul"), 0644)

	configData := map[string]interface{}{
		"providers": map[string]interface{}{
			"anthropic": map[string]interface{}{
				"apiKey": "sk-config-only",
			},
		},
	}
	data, _ := json.Marshal(configData)
	os.WriteFile(filepath.Join(openclawHome, "openclaw.json"), data, 0644)

	opts := Options{
		Force:        true,
		ConfigOnly:   true,
		OpenClawHome: openclawHome,
		PicoClawHome: picoClawHome,
	}

	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !result.ConfigMigrated {
		t.Error("config should have been migrated")
	}

	picoWs := filepath.Join(picoClawHome, "workspace")
	if _, err := os.Stat(filepath.Join(picoWs, "SOUL.md")); !os.IsNotExist(err) {
		t.Error("config-only should not copy workspace files")
	}
}

func TestRunWorkspaceOnly(t *testing.T) {
	openclawHome := t.TempDir()
	picoClawHome := t.TempDir()

	wsDir := filepath.Join(openclawHome, "workspace")
	os.MkdirAll(wsDir, 0755)
	os.WriteFile(filepath.Join(wsDir, "SOUL.md"), []byte("# Soul"), 0644)

	configData := map[string]interface{}{
		"providers": map[string]interface{}{
			"anthropic": map[string]interface{}{
				"apiKey": "sk-ws-only",
			},
		},
	}
	data, _ := json.Marshal(configData)
	os.WriteFile(filepath.Join(openclawHome, "openclaw.json"), data, 0644)

	opts := Options{
		Force:         true,
		WorkspaceOnly: true,
		OpenClawHome:  openclawHome,
		PicoClawHome:  picoClawHome,
	}

	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.ConfigMigrated {
		t.Error("workspace-only should not migrate config")
	}

	picoWs := filepath.Join(picoClawHome, "workspace")
	soulData, err := os.ReadFile(filepath.Join(picoWs, "SOUL.md"))
	if err != nil {
		t.Fatalf("reading SOUL.md: %v", err)
	}
	if string(soulData) != "# Soul" {
		t.Errorf("SOUL.md content = %q", string(soulData))
	}
}
