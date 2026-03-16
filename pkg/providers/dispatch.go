// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"fmt"
	"sync"

	"github.com/sipeed/picoclaw/pkg/config"
)

// ProviderDispatcher creates and caches per-model LLMProvider instances.
// This fixes the bug where all agents share a single provider instance: when a
// fallback chain selects a different protocol/model pair than the agent's default
// provider, the dispatcher creates and caches a dedicated provider for that pair.
//
// Cache key is "protocol/modelID" which matches the Model field in ModelConfig.
// Thread-safe: uses sync.RWMutex with read-locking for cache hits.
type ProviderDispatcher struct {
	mu    sync.RWMutex
	cache map[string]LLMProvider
	cfg   *config.Config
}

// NewProviderDispatcher creates a new dispatcher with the given config.
func NewProviderDispatcher(cfg *config.Config) *ProviderDispatcher {
	return &ProviderDispatcher{
		cache: make(map[string]LLMProvider),
		cfg:   cfg,
	}
}

// Get returns a cached or newly created provider for the given protocol+modelID pair.
// It finds the ModelConfig by iterating cfg.ModelList and matching where
// ModelConfig.Model == protocol+"/"+modelID.
// Returns an error if no matching ModelConfig is found or provider creation fails.
func (d *ProviderDispatcher) Get(protocol, modelID string) (LLMProvider, error) {
	key := protocol + "/" + modelID

	// Fast path: read-lock to check cache.
	d.mu.RLock()
	if p, ok := d.cache[key]; ok {
		d.mu.RUnlock()
		return p, nil
	}
	d.mu.RUnlock()

	// Slow path: find config and create provider under write-lock.
	d.mu.Lock()
	defer d.mu.Unlock()

	// Double-check after acquiring write-lock.
	if p, ok := d.cache[key]; ok {
		return p, nil
	}

	// Find the matching ModelConfig entry.
	var matched *config.ModelConfig
	for i := range d.cfg.ModelList {
		if d.cfg.ModelList[i].Model == key {
			matched = &d.cfg.ModelList[i]
			break
		}
	}
	if matched == nil {
		return nil, fmt.Errorf("dispatcher: no model_list entry with model=%q", key)
	}

	provider, _, err := CreateProviderFromConfig(matched)
	if err != nil {
		return nil, fmt.Errorf("dispatcher: creating provider for %q: %w", key, err)
	}

	d.cache[key] = provider
	return provider, nil
}

// Flush clears the provider cache and updates the config reference.
// Call this after a config reload so the dispatcher picks up new settings.
func (d *ProviderDispatcher) Flush(cfg *config.Config) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cache = make(map[string]LLMProvider)
	d.cfg = cfg
}
