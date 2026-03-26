// Fork-specific provider factory extensions.

package providers

import (
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

// CreateProviderByName looks up a model in the config's model_list by provider
// name (case-insensitive) and creates an LLM provider from it.
// This is used as a legacy fallback when resolving providers for fallback models.
func CreateProviderByName(cfg *config.Config, providerName string) (LLMProvider, error) {
	providerLower := strings.ToLower(providerName)

	// Search model_list for a matching provider name (model_name or protocol prefix)
	for _, mc := range cfg.ModelList {
		// Match by model_name
		if strings.ToLower(mc.ModelName) == providerLower {
			p, _, err := CreateProviderFromConfig(mc)
			if err != nil {
				return nil, fmt.Errorf("failed to create provider %q: %w", providerName, err)
			}
			return p, nil
		}

		// Match by protocol prefix in Model field (e.g., "openai/gpt-4o" matches "openai")
		if parts := strings.SplitN(mc.Model, "/", 2); len(parts) == 2 {
			if strings.ToLower(parts[0]) == providerLower {
				p, _, err := CreateProviderFromConfig(mc)
				if err != nil {
					return nil, fmt.Errorf("failed to create provider %q: %w", providerName, err)
				}
				return p, nil
			}
		}
	}

	return nil, fmt.Errorf("provider %q not found in model_list", providerName)
}
