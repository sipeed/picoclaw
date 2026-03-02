package find_skills

import (
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/skills"
)

func GetSkillsConfig(cfg *config.Config) skills.RegistryConfig {
	tc := cfg.GetTool("skills")
	if tc == nil || !tc.Enabled {
		return skills.RegistryConfig{}
	}
	extra := tc.Extra
	if extra == nil {
		return skills.RegistryConfig{}
	}
	skillsCfg := skills.RegistryConfig{}

	if v, ok := extra["registries"]; ok {
		if m, ok := v.(map[string]any); ok {
			if clawhub, ok := m["clawhub"]; ok {
				if cm, ok := clawhub.(map[string]any); ok {
					skillsCfg.ClawHub = skills.ClawHubConfig{
						Enabled:         config.GetBoolOrDefault(cm, "enabled", false),
						BaseURL:         config.GetStringOrDefault(cm, "base_url", ""),
						SearchPath:      config.GetStringOrDefault(cm, "search_path", ""),
						SkillsPath:      config.GetStringOrDefault(cm, "skills_path", ""),
						DownloadPath:    config.GetStringOrDefault(cm, "download_path", ""),
						Timeout:         config.GetIntOrDefault(cm, "timeout", 30),
						MaxZipSize:      config.GetIntOrDefault(cm, "max_zip_size", 1024*1024*100),
						MaxResponseSize: config.GetIntOrDefault(cm, "max_response_size", 1024*1024*50),
					}
				}
			}
		}
	}
	skillsCfg.MaxConcurrentSearches = config.GetIntOrDefault(extra, "max_concurrent_searches", 2)
	return skillsCfg
}

func GetSearchCache(cfg *config.Config) *skills.SearchCache {
	maxSize, ttlSeconds := GetSearchCacheConfig(cfg)
	return skills.NewSearchCache(maxSize, time.Duration(ttlSeconds)*time.Second)
}

func GetSearchCacheConfig(cfg *config.Config) (maxSize int, ttlSeconds int) {
	tc := cfg.GetTool("skills")
	if tc == nil || !tc.Enabled {
		return 50, 300
	}
	extra := tc.Extra
	if extra == nil {
		return 50, 300
	}
	if v, ok := extra["search_cache"]; ok {
		if m, ok := v.(map[string]any); ok {
			maxSize = config.GetIntOrDefault(m, "max_size", 50)
			ttlSeconds = config.GetIntOrDefault(m, "ttl_seconds", 300)
		}
	}
	if maxSize <= 0 {
		maxSize = 50
	}
	if ttlSeconds <= 0 {
		ttlSeconds = 300
	}
	return maxSize, ttlSeconds
}
