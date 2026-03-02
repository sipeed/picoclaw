package cron

import (
	"github.com/sipeed/picoclaw/pkg/config"
)

type CronToolConfig struct {
	Enabled            bool
	ExecTimeoutMinutes int
}

func GetCronConfig(cfg *config.Config) CronToolConfig {
	tc := cfg.GetTool("cron")
	if tc == nil {
		return CronToolConfig{}
	}
	return ParseCronConfig(tc)
}

func ParseCronConfig(tc *config.ToolConfig) CronToolConfig {
	if tc == nil || !tc.Enabled {
		return CronToolConfig{}
	}
	extra := tc.Extra
	if extra == nil {
		return CronToolConfig{Enabled: true}
	}
	return CronToolConfig{
		Enabled:            true,
		ExecTimeoutMinutes: config.GetIntOrDefault(extra, "exec_timeout_minutes", 5),
	}
}
