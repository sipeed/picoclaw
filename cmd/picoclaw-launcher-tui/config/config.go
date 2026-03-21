// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

// Package config provides types and I/O for ~/.picoclaw/tui.toml.
package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/sipeed/picoclaw/pkg/fileutil"
)

// DefaultConfigPath returns the default path to the tui.toml config file.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".picoclaw", "tui.toml")
}

// TUIConfig is the top-level structure of ~/.picoclaw/tui.toml.
type TUIConfig struct {
	Version  string   `toml:"version"`
	Model    Model    `toml:"model"`
	Provider Provider `toml:"provider"`
}

type Model struct {
	Type string `toml:"type"` // "provider" (default) | "manual"
}

type Provider struct {
	Schemes []Scheme        `toml:"schemes"`
	Users   []User          `toml:"users"`
	Current ProviderCurrent `toml:"current"`
}

type Scheme struct {
	Name    string `toml:"name"`    // unique key
	BaseURL string `toml:"baseURL"` // required
	Type    string `toml:"type"`    // "openai-compatible" (default) | "anthropic"
}

type User struct {
	Name   string `toml:"name"`
	Scheme string `toml:"scheme"` // references Scheme.Name; (Name+Scheme) is unique
	Type   string `toml:"type"`   // "key" (default) | "OAuth"
	Key    string `toml:"key"`
}

type ProviderCurrent struct {
	Scheme string `toml:"scheme"` // references Scheme.Name
	User   string `toml:"user"`   // references User.Name where User.Scheme == Scheme
	Model  string `toml:"model"`  // from GET <baseURL>/models
}

// DefaultConfig returns a minimal valid TUIConfig.
func DefaultConfig() *TUIConfig {
	return &TUIConfig{
		Version: "1.0",
		Model:   Model{Type: "provider"},
		Provider: Provider{
			Schemes: []Scheme{},
			Users:   []User{},
			Current: ProviderCurrent{},
		},
	}
}

// Load reads the TUI config from path. Returns a default config if the file does not exist.
func Load(path string) (*TUIConfig, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", path, err)
	}

	cfg := DefaultConfig()
	if _, err := toml.Decode(string(data), cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %q: %w", path, err)
	}

	applyDefaults(cfg)
	return cfg, nil
}

// Save writes cfg to path atomically (safe for flash / SD storage).
func Save(path string, cfg *TUIConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}
	if err := fileutil.WriteFileAtomic(path, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("failed to write config file %q: %w", path, err)
	}
	return nil
}

func applyDefaults(cfg *TUIConfig) {
	if cfg.Version == "" {
		cfg.Version = "1.0"
	}
	if cfg.Model.Type == "" {
		cfg.Model.Type = "provider"
	}
	for i := range cfg.Provider.Schemes {
		if cfg.Provider.Schemes[i].Type == "" {
			cfg.Provider.Schemes[i].Type = "openai-compatible"
		}
	}
	for i := range cfg.Provider.Users {
		if cfg.Provider.Users[i].Type == "" {
			cfg.Provider.Users[i].Type = "key"
		}
	}
}

// SchemeByName returns the first Scheme whose Name matches, or nil.
func (p *Provider) SchemeByName(name string) *Scheme {
	for i := range p.Schemes {
		if p.Schemes[i].Name == name {
			return &p.Schemes[i]
		}
	}
	return nil
}

// UsersForScheme returns all users whose Scheme field matches schemeName.
func (p *Provider) UsersForScheme(schemeName string) []User {
	var out []User
	for _, u := range p.Users {
		if u.Scheme == schemeName {
			out = append(out, u)
		}
	}
	return out
}

// SyncSelectedModelToMainConfig syncs the currently selected model to ~/.picoclaw/config.json
// Adds/replaces a "tui-prefer" model entry and sets it as the default model.
// Preserves all other existing fields in the config file unchanged.
func SyncSelectedModelToMainConfig(scheme Scheme, user User, modelID string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	mainConfigPath := filepath.Join(home, ".picoclaw", "config.json")

	var cfg map[string]any
	if data, readErr := os.ReadFile(mainConfigPath); readErr == nil {
		if unmarshalErr := json.Unmarshal(data, &cfg); unmarshalErr != nil {
			cfg = make(map[string]any)
		}
	} else {
		cfg = make(map[string]any)
	}

	if _, ok := cfg["agents"]; !ok {
		cfg["agents"] = make(map[string]any)
	}
	agents, ok := cfg["agents"].(map[string]any)
	if ok {
		if _, ok := agents["defaults"]; !ok {
			agents["defaults"] = make(map[string]any)
		}
		defaults, ok := agents["defaults"].(map[string]any)
		if ok {
			defaults["model"] = "tui-prefer"
		}
	}

	tuiModel := map[string]any{
		"model_name": "tui-prefer",
		"model":      modelID,
		"api_key":    user.Key,
		"api_base":   scheme.BaseURL,
	}

	modelList := []any{}
	if ml, ok := cfg["model_list"].([]any); ok {
		modelList = ml
	}

	found := false
	for i, m := range modelList {
		if entry, ok := m.(map[string]any); ok {
			if name, ok := entry["model_name"].(string); ok && name == "tui-prefer" {
				modelList[i] = tuiModel
				found = true
				break
			}
		}
	}
	if !found {
		modelList = append(modelList, tuiModel)
	}
	cfg["model_list"] = modelList

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(mainConfigPath), 0o700); err != nil {
		return err
	}

	return os.WriteFile(mainConfigPath, data, 0o600)
}

func (cfg *TUIConfig) CurrentModelLabel() string {
	cur := cfg.Provider.Current
	if cur.Model == "" {
		return "(not configured)"
	}
	label := cur.Scheme
	if label != "" {
		label += " / "
	}
	return label + cur.Model
}
