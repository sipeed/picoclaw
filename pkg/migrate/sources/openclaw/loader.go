package openclaw

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func LoadOpenClawConfig(path string) (*OpenClawConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config OpenClawConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &config, nil
}

func LoadOpenClawConfigFromDir(dir string) (*OpenClawConfig, error) {
	candidates := []string{
		filepath.Join(dir, "openclaw.json"),
		filepath.Join(dir, "config.json"),
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return LoadOpenClawConfig(p)
		}
	}

	return nil, fmt.Errorf("no config file found in %s", dir)
}

func GetProviderConfig(models *OpenClawModels) map[string]OpenClawProviderConfig {
	result := make(map[string]OpenClawProviderConfig)
	if models == nil || models.Providers == nil {
		return result
	}

	for name, raw := range models.Providers {
		var prov OpenClawProviderConfig
		if err := json.Unmarshal(raw, &prov); err != nil {
			continue
		}
		mappedName := mapProvider(name)
		result[mappedName] = prov
	}

	return result
}

func GetProviderConfigFromDir(dir string) map[string]ProviderConfig {
	result := make(map[string]ProviderConfig)
	p := filepath.Join(dir, "agents", "main", "agent", "models.json")

	if _, err := os.Stat(p); err != nil {
		return result
	}

	data, err := os.ReadFile(p)
	if err != nil {
		return result
	}
	var models OpenClawModels
	if err := json.Unmarshal(data, &models); err != nil {
		return result
	}

	for name, raw := range models.Providers {
		var prov ProviderConfig
		if err := json.Unmarshal(raw, &prov); err != nil {
			continue
		}
		mappedName := mapProvider(name)
		result[mappedName] = prov
	}
	return result
}

func GetChannelAllowFrom(ch any) []string {
	switch c := ch.(type) {
	case *OpenClawTelegramConfig:
		if c == nil {
			return nil
		}
		return c.AllowFrom
	case *OpenClawDiscordConfig:
		if c == nil {
			return nil
		}
		return c.AllowFrom
	case *OpenClawSlackConfig:
		if c == nil {
			return nil
		}
		return c.AllowFrom
	case *OpenClawMatrixConfig:
		if c == nil {
			return nil
		}
		return c.AllowFrom
	case *OpenClawWhatsAppConfig:
		if c == nil {
			return nil
		}
		return c.AllowFrom
	default:
		return nil
	}
}
