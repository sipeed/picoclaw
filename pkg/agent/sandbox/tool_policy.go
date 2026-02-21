package sandbox

import (
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

var defaultSandboxAllow = []string{"exec", "read_file", "write_file"}

func IsToolSandboxEnabled(cfg *config.Config, tool string) bool {
	name := strings.ToLower(strings.TrimSpace(tool))
	if name == "" {
		return false
	}

	allow, hasAllow := defaultSandboxAllow, false
	deny := []string{}
	if cfg != nil {
		allow = cfg.Tools.Sandbox.Tools.Allow
		deny = cfg.Tools.Sandbox.Tools.Deny
		hasAllow = cfg.Tools.Sandbox.Tools.Allow != nil
	}

	if containsTool(deny, name) {
		return false
	}
	if !hasAllow {
		return containsTool(defaultSandboxAllow, name)
	}
	if len(allow) == 0 {
		return true
	}
	return containsTool(allow, name)
}

func containsTool(list []string, tool string) bool {
	for _, v := range list {
		if strings.EqualFold(strings.TrimSpace(v), tool) {
			return true
		}
	}
	return false
}
