package sandbox

import (
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

var defaultSandboxAllow = []string{"exec", "read_file", "write_file", "list_dir", "edit_file", "append_file"}

func IsToolSandboxEnabled(cfg *config.Config, tool string) bool {
	name := strings.ToLower(strings.TrimSpace(tool))
	if name == "" {
		return false
	}

	allow := defaultSandboxAllow
	deny := []string{}
	if cfg != nil {
		allow = cfg.Tools.Sandbox.Tools.Allow
		deny = cfg.Tools.Sandbox.Tools.Deny
	}

	if containsTool(deny, name) {
		return false
	}
	if len(allow) == 0 {
		// Empty allow list falls back to built-in defaults.
		allow = defaultSandboxAllow
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
