// Fork-specific config extensions for picoclaw.
// Adds OCRConfig, FindModelConfigByRef, and other helpers
// that are not present in the upstream config package.

package config

import "strings"

// OCRConfig holds configuration for PDF OCR processing.
type OCRConfig struct {
	Command      string            `json:"command"                 env:"PICOCLAW_OCR_COMMAND"`
	Args         []string          `json:"args,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	ReadingOrder string            `json:"reading_order,omitempty" env:"PICOCLAW_OCR_READING_ORDER"`
	Timeout      int               `json:"timeout,omitempty"       env:"PICOCLAW_OCR_TIMEOUT"` // seconds
}

// GetOCRTimeout returns the OCR timeout in seconds, defaulting to 300 (5 minutes).
func (c *OCRConfig) GetOCRTimeout() int {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return 300
}

// FindModelConfigByRef searches model_list for a ModelConfig matching the
// given provider/model reference. The providerName is matched against the
// "protocol/" prefix of ModelConfig.Model, and modelName is matched against
// either ModelConfig.ModelName or the model portion after the slash.
// Returns nil if no match is found.
func (c *Config) FindModelConfigByRef(providerName, modelName string) *ModelConfig {
	providerName = strings.ToLower(providerName)
	modelName = strings.ToLower(modelName)

	for i := range c.ModelList {
		mc := c.ModelList[i]

		// Match by model_name (user-facing alias)
		if strings.ToLower(mc.ModelName) == modelName {
			return mc
		}

		// Match by "provider/model" in the Model field
		parts := strings.SplitN(mc.Model, "/", 2)
		if len(parts) == 2 {
			mcProvider := strings.ToLower(parts[0])
			mcModel := strings.ToLower(parts[1])
			if mcProvider == providerName && mcModel == modelName {
				return mc
			}
		}

		// Match by full "provider/model" as modelName
		if strings.ToLower(mc.Model) == providerName+"/"+modelName {
			return mc
		}
	}
	return nil
}
