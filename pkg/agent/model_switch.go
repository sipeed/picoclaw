package agent

import (
	"fmt"
	"sync"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// ModelSwitchManager handles dynamic model switching at runtime.
type ModelSwitchManager struct {
	config   *config.Config
	registry *AgentRegistry
	mu       sync.RWMutex
}

// NewModelSwitchManager creates a new model switch manager.
func NewModelSwitchManager(cfg *config.Config, registry *AgentRegistry) *ModelSwitchManager {
	return &ModelSwitchManager{
		config:   cfg,
		registry: registry,
	}
}

// SwitchModel switches the model for a given session.
// If sessionKey is empty, it updates the global default model.
func (m *ModelSwitchManager) SwitchModel(sessionKey, modelName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate model exists in config
	if _, err := m.config.GetModelConfig(modelName); err != nil {
		return fmt.Errorf("model %q not found. Available models: %v", modelName, m.listAvailableModels())
	}

	// Session-scoped override is not implemented yet.
	if sessionKey != "" {
		return fmt.Errorf("session-scoped model switch is not supported yet")
	}

	oldModel := m.config.Agents.Defaults.GetModelName()
	if oldModel == modelName {
		return nil
	}

	if err := m.config.SetDefaultModel(modelName); err != nil {
		return fmt.Errorf("failed to set default model: %w", err)
	}

	newProvider, _, err := providers.CreateProvider(m.config)
	if err != nil {
		// Roll back config on provider creation failure.
		_ = m.config.SetDefaultModel(oldModel)
		return fmt.Errorf("failed to create provider for model switch: %w", err)
	}

	if err := m.registry.SwitchModel(m.config, oldModel, modelName, newProvider); err != nil {
		_ = m.config.SetDefaultModel(oldModel)
		if cp, ok := newProvider.(providers.StatefulProvider); ok {
			cp.Close()
		}
		return fmt.Errorf("failed to apply hot model switch: %w", err)
	}

	return nil
}

// GetCurrentModel returns the current model for a session.
// If session-specific override exists, returns that. Otherwise returns global default.
func (m *ModelSwitchManager) GetCurrentModel(sessionKey string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// For now, always return the global default model
	// Session-scoped overrides can be added later
	return m.config.Agents.Defaults.GetModelName()
}

// ValidateModel validates if a model exists in the configuration.
func (m *ModelSwitchManager) ValidateModel(modelName string) (*config.ModelConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.config.GetModelConfig(modelName)
}

// listAvailableModels returns a slice of all available model names.
func (m *ModelSwitchManager) listAvailableModels() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	models := make([]string, 0, len(m.config.ModelList))
	for _, mc := range m.config.ModelList {
		if mc.ModelName != "" {
			models = append(models, mc.ModelName)
		}
	}
	return models
}

// GetModelInfo returns formatted information about the current model.
func (m *ModelSwitchManager) GetModelInfo(sessionKey string) (string, string) {
	currentModel := m.GetCurrentModel(sessionKey)
	if mc, err := m.config.GetModelConfig(currentModel); err == nil {
		return currentModel, mc.Model
	}

	return currentModel, "unknown"
}
