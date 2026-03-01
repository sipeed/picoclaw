package openclaw

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/migrate/internal"
)

var providerMapping = map[string]string{
	"anthropic":  "anthropic",
	"claude":     "anthropic",
	"openai":     "openai",
	"gpt":        "openai",
	"groq":       "groq",
	"ollama":     "ollama",
	"openrouter": "openrouter",
	"deepseek":   "deepseek",
	"together":   "together",
	"mistral":    "mistral",
	"fireworks":  "fireworks",
	"google":     "google",
	"gemini":     "google",
	"xai":        "xai",
	"grok":       "xai",
	"cerebras":   "cerebras",
	"sambanova":  "sambanova",
}

type OpenclawHandler struct {
	opts             Options
	sourceConfigFile string
	sourceWorkspace  string
}

type (
	Options   = internal.Options
	Action    = internal.Action
	Result    = internal.Result
	Operation = internal.Operation
)

func NewOpenclawHandler(opts Options) (Operation, error) {
	home, err := resolveSourceHome(opts.SourceHome)
	if err != nil {
		return nil, err
	}
	opts.SourceHome = home

	configFile, err := findSourceConfig(home)
	if err != nil {
		return nil, err
	}
	return &OpenclawHandler{
		opts:             opts,
		sourceWorkspace:  filepath.Join(opts.SourceHome, "workspace"),
		sourceConfigFile: configFile,
	}, nil
}

func (o *OpenclawHandler) GetSourceName() string {
	return "openclaw"
}

func (o *OpenclawHandler) GetSourceHome() (string, error) {
	return o.opts.SourceHome, nil
}

func (o *OpenclawHandler) GetSourceWorkspace() (string, error) {
	return o.sourceWorkspace, nil
}

func (o *OpenclawHandler) GetSourceConfigFile() (string, error) {
	return o.sourceConfigFile, nil
}

func (o *OpenclawHandler) GetMigrateableFiles() []string {
	return migrateableFiles
}

func (o *OpenclawHandler) GetMigrateableDirs() []string {
	return migrateableDirs
}

func (o *OpenclawHandler) ExecuteConfigMigration(srcConfigPath, dstConfigPath string) error {
	openclawCfg, err := LoadOpenClawConfig(srcConfigPath)
	if err != nil {
		return err
	}

	picoCfg, warnings, err := openclawCfg.ConvertToPicoClaw(o.opts.SourceHome)
	if err != nil {
		return err
	}

	for _, w := range warnings {
		fmt.Printf("  Warning: %s\n", w)
	}

	incoming := picoCfg.ToStandardConfig()
	if err := os.MkdirAll(filepath.Dir(dstConfigPath), 0o755); err != nil {
		return err
	}

	return config.SaveConfig(dstConfigPath, incoming)
}

func resolveSourceHome(override string) (string, error) {
	if override != "" {
		return internal.ExpandHome(override), nil
	}
	if envHome := os.Getenv("OPENCLAW_HOME"); envHome != "" {
		return internal.ExpandHome(envHome), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".openclaw"), nil
}

func findSourceConfig(sourceHome string) (string, error) {
	candidates := []string{
		filepath.Join(sourceHome, "openclaw.json"),
		filepath.Join(sourceHome, "config.json"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("no config file found in %s (tried openclaw.json, config.json)", sourceHome)
}

func rewriteWorkspacePath(path string) string {
	path = strings.Replace(path, ".openclaw", ".picoclaw", 1)
	return path
}

func mapProvider(provider string) string {
	if mapped, ok := providerMapping[strings.ToLower(provider)]; ok {
		return mapped
	}
	return strings.ToLower(provider)
}
