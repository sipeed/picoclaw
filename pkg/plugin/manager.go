// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package plugin

import (
	"errors"
	"fmt"
	"sort"
	"slices"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/hooks"
)

// APIVersion identifies the compile-time plugin contract version.
const APIVersion = "v1alpha1"

// SelectionInput controls plugin enable/disable resolution.
type SelectionInput struct {
	DefaultEnabled bool
	Enabled        []string
	Disabled       []string
}

// SelectionResult is the normalized output of plugin enable/disable resolution.
type SelectionResult struct {
	EnabledNames    []string
	DisabledNames   []string
	UnknownEnabled  []string
	UnknownDisabled []string
	Warnings        []string
}

// Plugin is the Phase-1 compile-time contract for PicoClaw extensions.
type Plugin interface {
	Name() string
	APIVersion() string
	Register(registry *hooks.HookRegistry) error
}

// PluginInfo describes plugin metadata for introspection APIs.
type PluginInfo struct {
	Name       string `json:"name"`
	APIVersion string `json:"api_version"`
	Status     string `json:"status"`
}

// PluginDescriptor optionally provides richer plugin metadata.
type PluginDescriptor interface {
	Info() PluginInfo
}

// NormalizePluginName normalizes plugin names for deterministic matching.
func NormalizePluginName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// ResolveSelection resolves final enabled/disabled plugin names deterministically.
func ResolveSelection(available []string, in SelectionInput) (SelectionResult, error) {
	result := SelectionResult{}

	availableSet := make(map[string]struct{}, len(available))
	for _, name := range available {
		normalized := NormalizePluginName(name)
		if normalized == "" {
			continue
		}
		availableSet[normalized] = struct{}{}
	}

	enabledSet := make(map[string]struct{}, len(in.Enabled))
	for _, name := range in.Enabled {
		normalized := NormalizePluginName(name)
		if _, exists := enabledSet[normalized]; exists {
			result.Warnings = append(result.Warnings, fmt.Sprintf("duplicate enabled plugin %q ignored", normalized))
			continue
		}
		enabledSet[normalized] = struct{}{}
	}

	disabledSet := make(map[string]struct{}, len(in.Disabled))
	for _, name := range in.Disabled {
		normalized := NormalizePluginName(name)
		if _, exists := disabledSet[normalized]; exists {
			result.Warnings = append(result.Warnings, fmt.Sprintf("duplicate disabled plugin %q ignored", normalized))
			continue
		}
		disabledSet[normalized] = struct{}{}
	}

	for name := range enabledSet {
		if _, ok := availableSet[name]; !ok {
			result.UnknownEnabled = append(result.UnknownEnabled, name)
		}
	}
	sort.Strings(result.UnknownEnabled)

	for name := range disabledSet {
		if _, ok := availableSet[name]; !ok {
			result.UnknownDisabled = append(result.UnknownDisabled, name)
		}
	}
	sort.Strings(result.UnknownDisabled)
	for _, name := range result.UnknownDisabled {
		result.Warnings = append(result.Warnings, fmt.Sprintf("unknown disabled plugin %q ignored", name))
	}

	resolvedEnabled := make(map[string]struct{}, len(availableSet))
	if len(enabledSet) > 0 {
		for name := range enabledSet {
			if _, ok := availableSet[name]; !ok {
				continue
			}
			if _, disabled := disabledSet[name]; disabled {
				continue
			}
			resolvedEnabled[name] = struct{}{}
		}
	} else if in.DefaultEnabled {
		for name := range availableSet {
			if _, disabled := disabledSet[name]; disabled {
				continue
			}
			resolvedEnabled[name] = struct{}{}
		}
	}

	for name := range resolvedEnabled {
		result.EnabledNames = append(result.EnabledNames, name)
	}
	sort.Strings(result.EnabledNames)

	for name := range availableSet {
		if _, enabled := resolvedEnabled[name]; enabled {
			continue
		}
		result.DisabledNames = append(result.DisabledNames, name)
	}
	sort.Strings(result.DisabledNames)

	if len(result.UnknownEnabled) > 0 {
		return result, fmt.Errorf("unknown enabled plugins: %s", strings.Join(result.UnknownEnabled, ", "))
	}
	return result, nil
}

// Manager owns a shared hook registry and loaded plugin metadata.
type Manager struct {
	mu       sync.RWMutex
	registry *hooks.HookRegistry
	names    []string
	plugins  []Plugin
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

// DescribeAll returns plugin metadata in registration order.
func (m *Manager) DescribeAll() []PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]PluginInfo, 0, len(m.plugins))
	for i, p := range m.plugins {
		fallbackName := ""
		if i < len(m.names) {
			fallbackName = m.names[i]
		}
		infos = append(infos, normalizePluginInfo(p, fallbackName))
	}
	return infos
}

// DescribeEnabled returns metadata for currently enabled plugins.
func (m *Manager) DescribeEnabled() []PluginInfo {
	return m.DescribeAll()
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
	m.plugins = append(m.plugins, p)
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

func normalizePluginInfo(p Plugin, fallbackName string) PluginInfo {
	info := PluginInfo{
		Name:       strings.TrimSpace(fallbackName),
		APIVersion: strings.TrimSpace(p.APIVersion()),
		Status:     "enabled",
	}
	if descriptor, ok := p.(PluginDescriptor); ok {
		described := descriptor.Info()
		if name := strings.TrimSpace(described.Name); name != "" {
			info.Name = name
		}
		if version := strings.TrimSpace(described.APIVersion); version != "" {
			info.APIVersion = version
		}
		if status := strings.TrimSpace(described.Status); status != "" {
			info.Status = status
		}
	}
	if info.Name == "" {
		info.Name = strings.TrimSpace(p.Name())
	}
	if info.APIVersion == "" {
		info.APIVersion = APIVersion
	}
	if info.Status == "" {
		info.Status = "enabled"
	}
	return info
}
