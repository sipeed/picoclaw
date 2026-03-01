package configcmd

import (
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

// modelConfigKeys defines allowed keys for get/set, in display order.
var modelConfigKeys = []string{
	"model_name", "model", "api_base", "api_key", "proxy",
	"auth_method", "connect_mode", "workspace",
	"token_url", "client_id", "client_secret",
	"max_tokens_field", "rpm",
}

// modelConfigKeySet is the set of allowed keys.
var modelConfigKeySet map[string]bool

func init() {
	modelConfigKeySet = make(map[string]bool, len(modelConfigKeys))
	for _, k := range modelConfigKeys {
		modelConfigKeySet[k] = true
	}
}

func isModelConfigKey(key string) bool {
	return modelConfigKeySet[key]
}

// isIntModelConfigKey returns true for keys that must be set as int (e.g. rpm).
func isIntModelConfigKey(key string) bool {
	return key == "rpm"
}

// findModelIndex returns the index of the first ModelConfig with ModelName == name.
// It returns -1 and an error if not found.
func findModelIndex(cfg *config.Config, name string) (int, error) {
	for i := range cfg.ModelList {
		if cfg.ModelList[i].ModelName == name {
			return i, nil
		}
	}
	return -1, fmt.Errorf("no model with model_name %q", name)
}

func allowedModelConfigKeysString() string {
	return strings.Join(modelConfigKeys, ", ")
}
