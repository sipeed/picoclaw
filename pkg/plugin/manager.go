// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package plugin

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/hooks"
)

// APIVersion identifies the compile-time plugin contract version.
const APIVersion = "v1alpha1"

// Plugin is the Phase-1 compile-time contract for PicoClaw extensions.
type Plugin interface {
	Name() string
	APIVersion() string
	Register(*hooks.HookRegistry) error
}

// Manager owns a shared hook registry and loaded plugin metadata.
type Manager struct {
	mu       sync.RWMutex
	registry *hooks.HookRegistry
	names    []string
	seen     map[string]struct{}
}

// NewManager creates an empty plugin manager with a fresh hook registry.
func NewManager() *Manager {
	return &Manager{
		registry: hooks.NewHookRegistry(),
		seen:     make(map[string]struct{}),
	}
}

// HookRegistry returns the shared registry where plugins register hooks.
func (m *Manager) HookRegistry() *hooks.HookRegistry {
	return m.registry
}

// Names returns loaded plugin names in registration order.
func (m *Manager) Names() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return slices.Clone(m.names)
}

// Register loads one plugin into the shared hook registry.
func (m *Manager) Register(p Plugin) error {
	if p == nil {
		return errors.New("plugin is nil")
	}
	name := strings.TrimSpace(p.Name())
	if name == "" {
		return errors.New("plugin name is required")
	}
	if got := strings.TrimSpace(p.APIVersion()); got != APIVersion {
		if got == "" {
			got = "<empty>"
		}
		return fmt.Errorf(
			"plugin %q api version mismatch: got %s, want %s",
			name,
			got,
			APIVersion,
		)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.seen[name]; exists {
		return fmt.Errorf("plugin %q already registered", name)
	}
	if err := p.Register(m.registry); err != nil {
		return fmt.Errorf("register plugin %q: %w", name, err)
	}
	m.seen[name] = struct{}{}
	m.names = append(m.names, name)
	return nil
}

// RegisterAll loads plugins sequentially.
func (m *Manager) RegisterAll(plugins ...Plugin) error {
	for _, p := range plugins {
		if err := m.Register(p); err != nil {
			return err
		}
	}
	return nil
}
