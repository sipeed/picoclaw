// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/sipeed/picoclaw/pkg/config"
)

// ModelRegistry manages model configurations with thread-safe round-robin load balancing.
// It allows multiple configurations for the same model_name to distribute load across endpoints.
type ModelRegistry struct {
	configs  map[string][]config.ModelConfig // model_name -> []ModelConfig
	counters map[string]*atomic.Uint64       // model_name -> round-robin counter
	mu       sync.RWMutex
}

// NewModelRegistry creates a new ModelRegistry from a slice of ModelConfig.
func NewModelRegistry(modelList []config.ModelConfig) *ModelRegistry {
	r := &ModelRegistry{
		configs:  make(map[string][]config.ModelConfig),
		counters: make(map[string]*atomic.Uint64),
	}

	for _, cfg := range modelList {
		r.configs[cfg.ModelName] = append(r.configs[cfg.ModelName], cfg)
	}

	// Initialize counters for models with multiple configs
	for name, cfgs := range r.configs {
		if len(cfgs) > 1 {
			r.counters[name] = &atomic.Uint64{}
		}
	}

	return r
}

// GetModelConfig returns a ModelConfig for the given model name.
// If multiple configs exist for the same model_name, it uses round-robin selection.
// Returns an error if the model is not found.
func (r *ModelRegistry) GetModelConfig(modelName string) (*config.ModelConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	configs, ok := r.configs[modelName]
	if !ok || len(configs) == 0 {
		return nil, fmt.Errorf("model %q not found", modelName)
	}

	// Single config - return directly
	if len(configs) == 1 {
		return &configs[0], nil
	}

	// Multiple configs - use round-robin for load balancing
	counter, ok := r.counters[modelName]
	if !ok {
		// Should not happen, but handle gracefully
		return &configs[0], nil
	}

	idx := counter.Add(1) % uint64(len(configs))
	return &configs[idx], nil
}

// AddConfig adds a new ModelConfig to the registry.
func (r *ModelRegistry) AddConfig(cfg config.ModelConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.configs[cfg.ModelName] = append(r.configs[cfg.ModelName], cfg)

	// Initialize counter if we now have multiple configs
	if len(r.configs[cfg.ModelName]) > 1 && r.counters[cfg.ModelName] == nil {
		r.counters[cfg.ModelName] = &atomic.Uint64{}
	}
}

// RemoveConfig removes all configs with the given model_name.
func (r *ModelRegistry) RemoveConfig(modelName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.configs, modelName)
	delete(r.counters, modelName)
}

// ListModels returns all unique model names in the registry.
func (r *ModelRegistry) ListModels() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.configs))
	for name := range r.configs {
		names = append(names, name)
	}
	return names
}

// ConfigCount returns the number of configurations for a given model name.
func (r *ModelRegistry) ConfigCount(modelName string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.configs[modelName])
}
