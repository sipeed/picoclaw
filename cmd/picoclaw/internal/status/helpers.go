package status

import (
	"fmt"
	"os"
	"strings"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/cmd/picoclaw/internal/cliui"
	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
)

func statusCmd() {
	cfg, err := internal.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	configPath := internal.GetConfigPath()
	build, _ := config.FormatBuildInfo()

	_, configStatErr := os.Stat(configPath)
	configOK := configStatErr == nil

	workspace := cfg.WorkspacePath()
	_, wsErr := os.Stat(workspace)
	wsOK := wsErr == nil

	report := cliui.StatusReport{
		Logo:          internal.Logo,
		Version:       config.FormatVersion(),
		Build:         build,
		ConfigPath:    configPath,
		ConfigOK:      configOK,
		WorkspacePath: workspace,
		WorkspaceOK:   wsOK,
		Model:         cfg.Agents.Defaults.GetModelName(),
	}

	if configOK {
		// PicoClaw moved to a model-centric configuration (model_list). Status should
		// not depend on a legacy cfg.Providers field (which may not exist under some
		// build tags). We infer provider availability from model_list entries.
		hasProtocolKey := func(protocol string) bool {
			prefix := protocol + "/"
			for _, m := range cfg.ModelList {
				if m == nil {
					continue
				}
				if strings.HasPrefix(m.Model, prefix) && m.APIKey() != "" {
					return true
				}
			}
			return false
		}
		findLocalModelBase := func(modelName string) (string, bool) {
			for _, m := range cfg.ModelList {
				if m == nil {
					continue
				}
				if m.ModelName == modelName && m.APIBase != "" {
					return m.APIBase, true
				}
			}
			return "", false
		}
		findProtocolBase := func(protocol string) (string, bool) {
			prefix := protocol + "/"
			for _, m := range cfg.ModelList {
				if m == nil {
					continue
				}
				if strings.HasPrefix(m.Model, prefix) && m.APIBase != "" {
					return m.APIBase, true
				}
			}
			return "", false
		}

		hasOpenRouter := hasProtocolKey("openrouter")
		hasAnthropic := hasProtocolKey("anthropic")
		hasOpenAI := hasProtocolKey("openai")
		hasGemini := hasProtocolKey("gemini")
		hasZhipu := hasProtocolKey("zhipu")
		hasQwen := hasProtocolKey("qwen")
		hasGroq := hasProtocolKey("groq")
		hasMoonshot := hasProtocolKey("moonshot")
		hasDeepSeek := hasProtocolKey("deepseek")
		hasVolcEngine := hasProtocolKey("volcengine")
		hasNvidia := hasProtocolKey("nvidia")

		// Local endpoints: allow both the special reserved name and protocol-based entries.
		vllmBase, hasVLLM := findLocalModelBase("local-model")
		if !hasVLLM {
			vllmBase, hasVLLM = findProtocolBase("vllm")
		}
		ollamaBase, hasOllama := findProtocolBase("ollama")

		val := func(enabled bool, extra ...string) string {
			if enabled {
				if len(extra) > 0 && extra[0] != "" {
					return "✓ " + extra[0]
				}
				return "✓"
			}
			return "not set"
		}

		report.Providers = []cliui.ProviderRow{
			{"OpenRouter API", val(hasOpenRouter)},
			{"Anthropic API", val(hasAnthropic)},
			{"OpenAI API", val(hasOpenAI)},
			{"Gemini API", val(hasGemini)},
			{"Zhipu API", val(hasZhipu)},
			{"Qwen API", val(hasQwen)},
			{"Groq API", val(hasGroq)},
			{"Moonshot API", val(hasMoonshot)},
			{"DeepSeek API", val(hasDeepSeek)},
			{"VolcEngine API", val(hasVolcEngine)},
			{"Nvidia API", val(hasNvidia)},
			{"vLLM / local", val(hasVLLM, vllmBase)},
			{"Ollama", val(hasOllama, ollamaBase)},
		}

		store, _ := auth.LoadStore()
		if store != nil && len(store.Credentials) > 0 {
			for provider, cred := range store.Credentials {
				st := "authenticated"
				if cred.IsExpired() {
					st = "expired"
				} else if cred.NeedsRefresh() {
					st = "needs refresh"
				}
				report.OAuthLines = append(report.OAuthLines,
					fmt.Sprintf("%s (%s): %s", provider, cred.AuthMethod, st))
			}
		}
	}

	cliui.PrintStatus(report)
}
