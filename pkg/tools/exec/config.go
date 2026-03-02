package exec

import (
	"github.com/sipeed/picoclaw/pkg/config"
)

type ExecToolConfig struct {
	Enabled            bool
	EnableDenyPatterns bool
	CustomDenyPatterns []string
}

func GetExecConfig(cfg *config.Config) ExecToolConfig {
	tc := cfg.GetTool("exec")
	if tc == nil {
		return ExecToolConfig{}
	}
	return ParseExecConfig(tc)
}

func ParseExecConfig(tc *config.ToolConfig) ExecToolConfig {
	if tc == nil || !tc.Enabled {
		return ExecToolConfig{}
	}
	extra := tc.Extra
	if extra == nil {
		return ExecToolConfig{Enabled: true}
	}
	return ExecToolConfig{
		Enabled:            true,
		EnableDenyPatterns: config.GetBoolOrDefault(extra, "enable_deny_patterns", true),
		CustomDenyPatterns: config.GetStringSlice(extra, "custom_deny_patterns"),
	}
}
