package providers

import (
	"github.com/KarakuriAgent/clawdroid/pkg/config"
)

// CreateProvider is the single entry point for constructing an LLMProvider.
// When replacing the underlying LLM library, modify only this function
// and the adapter it delegates to (currently AnyLLMAdapter).
func CreateProvider(cfg *config.Config) (LLMProvider, error) {
	return NewAnyLLMAdapter(cfg.LLM.Model, cfg.LLM.APIKey, cfg.LLM.BaseURL)
}
