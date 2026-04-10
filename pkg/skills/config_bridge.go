package skills

import "github.com/sipeed/picoclaw/pkg/config"

const defaultGitHubRegistryBaseURL = "https://github.com"

func effectiveRegistryConfigsFromToolsConfig(cfg config.SkillsToolsConfig) []config.SkillRegistryConfig {
	effective := make([]config.SkillRegistryConfig, 0, len(cfg.Registries)+1)
	seen := map[string]struct{}{}

	for _, registryCfg := range cfg.Registries {
		if registryCfg == nil || registryCfg.Name == "" {
			continue
		}
		resolved := *registryCfg
		if resolved.Name == "github" {
			resolved = applyLegacyGithubRegistryCompatibility(cfg, resolved)
		}
		effective = append(effective, resolved)
		seen[resolved.Name] = struct{}{}
	}

	if _, ok := seen["github"]; ok {
		return effective
	}

	legacyGithubConfigured := cfg.Github.BaseURL != "" || cfg.Github.Token.String() != "" || cfg.Github.Proxy != ""
	if !legacyGithubConfigured {
		return effective
	}

	effective = append(effective, applyLegacyGithubRegistryCompatibility(cfg, config.SkillRegistryConfig{
		Name:    "github",
		Enabled: true,
	}))
	return effective
}

func applyLegacyGithubRegistryCompatibility(
	cfg config.SkillsToolsConfig,
	registryCfg config.SkillRegistryConfig,
) config.SkillRegistryConfig {
	if registryCfg.Name != "github" {
		return registryCfg
	}
	if registryCfg.Param == nil {
		registryCfg.Param = map[string]any{}
	}
	if registryCfg.BaseURL == "" ||
		(registryCfg.BaseURL == defaultGitHubRegistryBaseURL &&
			cfg.Github.BaseURL != "" &&
			cfg.Github.BaseURL != defaultGitHubRegistryBaseURL) {
		registryCfg.BaseURL = cfg.Github.BaseURL
	}
	if registryCfg.AuthToken.String() == "" {
		registryCfg.AuthToken = cfg.Github.Token
	}
	if _, ok := registryCfg.Param["proxy"]; !ok && cfg.Github.Proxy != "" {
		registryCfg.Param["proxy"] = cfg.Github.Proxy
	}
	return registryCfg
}

func registryProvidersFromToolsConfig(cfg config.SkillsToolsConfig) []RegistryProvider {
	registryConfigs := effectiveRegistryConfigsFromToolsConfig(cfg)
	providers := make([]RegistryProvider, 0, len(registryConfigs))
	for _, registryCfg := range registryConfigs {
		provider := buildRegistryProvider(registryCfg.Name, registryCfg)
		if provider == nil {
			continue
		}
		providers = append(providers, provider)
	}
	return providers
}

func NewRegistryManagerFromToolsConfig(cfg config.SkillsToolsConfig) *RegistryManager {
	return NewRegistryManagerFromConfig(RegistryConfig{
		Providers:             registryProvidersFromToolsConfig(cfg),
		MaxConcurrentSearches: cfg.MaxConcurrentSearches,
	})
}

func LookupRegistryFromToolsConfig(cfg config.SkillsToolsConfig, name string) SkillRegistry {
	for _, provider := range registryProvidersFromToolsConfig(cfg) {
		if provider == nil {
			continue
		}
		registry := provider.BuildRegistry()
		if registry == nil || registry.Name() != name {
			continue
		}
		return registry
	}
	return nil
}

func NormalizeInstallTargetForRegistry(cfg config.SkillsToolsConfig, registryName, target string) string {
	if registryName == "" || target == "" {
		return target
	}
	registry := LookupRegistryFromToolsConfig(cfg, registryName)
	if registry == nil {
		return target
	}
	ghRegistry, ok := registry.(*GitHubRegistry)
	if !ok {
		return target
	}
	normalized, err := canonicalGitHubRegistrySlugWithBaseURL(target, ghRegistry.webBase)
	if err != nil || normalized == "" {
		return target
	}
	return normalized
}
