package sandbox

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var blockedHostPaths = []string{
	"/etc",
	"/private/etc",
	"/proc",
	"/sys",
	"/dev",
	"/root",
	"/boot",
	"/run",
	"/var/run",
	"/private/var/run",
	"/var/run/docker.sock",
	"/private/var/run/docker.sock",
	"/run/docker.sock",
}

var blockedEnvVarPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^ANTHROPIC_API_KEY$`),
	regexp.MustCompile(`(?i)^OPENAI_API_KEY$`),
	regexp.MustCompile(`(?i)^GEMINI_API_KEY$`),
	regexp.MustCompile(`(?i)^OPENROUTER_API_KEY$`),
	regexp.MustCompile(`(?i)^MINIMAX_API_KEY$`),
	regexp.MustCompile(`(?i)^ELEVENLABS_API_KEY$`),
	regexp.MustCompile(`(?i)^SYNTHETIC_API_KEY$`),
	regexp.MustCompile(`(?i)^TELEGRAM_BOT_TOKEN$`),
	regexp.MustCompile(`(?i)^DISCORD_BOT_TOKEN$`),
	regexp.MustCompile(`(?i)^SLACK_(BOT|APP)_TOKEN$`),
	regexp.MustCompile(`(?i)^LINE_CHANNEL_SECRET$`),
	regexp.MustCompile(`(?i)^LINE_CHANNEL_ACCESS_TOKEN$`),
	regexp.MustCompile(`(?i)^OPENCLAW_GATEWAY_(TOKEN|PASSWORD)$`),
	regexp.MustCompile(`(?i)^AWS_(SECRET_ACCESS_KEY|SECRET_KEY|SESSION_TOKEN)$`),
	regexp.MustCompile(`(?i)^(GH|GITHUB)_TOKEN$`),
	regexp.MustCompile(`(?i)^(AZURE|AZURE_OPENAI|COHERE|AI_GATEWAY|OPENROUTER)_API_KEY$`),
	regexp.MustCompile(`(?i)_?(API_KEY|TOKEN|PASSWORD|PRIVATE_KEY|SECRET)$`),
}

func validateSandboxSecurity(cfg ContainerSandboxConfig) error {
	if err := validateBindMounts(cfg.Binds); err != nil {
		return err
	}
	if err := validateNetworkMode(cfg.Network); err != nil {
		return err
	}
	if err := validateSeccompProfile(cfg.SeccompProfile); err != nil {
		return err
	}
	if err := validateApparmorProfile(cfg.ApparmorProfile); err != nil {
		return err
	}
	return nil
}

func validateBindMounts(binds []string) error {
	for _, raw := range binds {
		bind := strings.TrimSpace(raw)
		if bind == "" {
			continue
		}
		source := parseBindSourcePath(bind)
		if !strings.HasPrefix(source, "/") {
			return fmt.Errorf("sandbox security: bind mount %q uses a non-absolute source path %q", bind, source)
		}
		normalized := normalizeHostPath(source)
		if err := validateBindSourcePath(bind, normalized); err != nil {
			return err
		}
		if real := tryRealpathAbsolute(normalized); real != normalized {
			if err := validateBindSourcePath(bind, real); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateBindSourcePath(bind, source string) error {
	if source == "/" {
		return fmt.Errorf("sandbox security: bind mount %q covers blocked path %q", bind, "/")
	}
	for _, blocked := range blockedHostPaths {
		if source == blocked || strings.HasPrefix(source, blocked+"/") {
			return fmt.Errorf("sandbox security: bind mount %q targets blocked path %q", bind, blocked)
		}
	}
	return nil
}

func parseBindSourcePath(bind string) string {
	trimmed := strings.TrimSpace(bind)
	idx := strings.Index(trimmed, ":")
	if idx <= 0 {
		return trimmed
	}
	return trimmed[:idx]
}

func normalizeHostPath(raw string) string {
	normalized := path.Clean(strings.TrimSpace(raw))
	if normalized == "." || normalized == "" {
		return "/"
	}
	if normalized != "/" {
		normalized = strings.TrimRight(normalized, "/")
		if normalized == "" {
			return "/"
		}
	}
	return normalized
}

func tryRealpathAbsolute(p string) string {
	if !strings.HasPrefix(p, "/") {
		return p
	}
	if _, err := os.Stat(p); err != nil {
		return p
	}
	resolved, err := filepathEvalSymlinks(p)
	if err != nil {
		return p
	}
	return normalizeHostPath(resolved)
}

func validateNetworkMode(network string) error {
	if strings.EqualFold(strings.TrimSpace(network), "host") {
		return fmt.Errorf("sandbox security: network mode %q is blocked", network)
	}
	return nil
}

func validateSeccompProfile(profile string) error {
	if strings.EqualFold(strings.TrimSpace(profile), "unconfined") {
		return fmt.Errorf("sandbox security: seccomp profile %q is blocked", profile)
	}
	return nil
}

func validateApparmorProfile(profile string) error {
	if strings.EqualFold(strings.TrimSpace(profile), "unconfined") {
		return fmt.Errorf("sandbox security: apparmor profile %q is blocked", profile)
	}
	return nil
}

func sanitizeEnvVars(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for rawKey, value := range in {
		key := strings.TrimSpace(rawKey)
		if key == "" {
			continue
		}
		if isBlockedEnvVarKey(key) {
			continue
		}
		if strings.Contains(value, "\x00") {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func isBlockedEnvVarKey(key string) bool {
	for _, pattern := range blockedEnvVarPatterns {
		if pattern.MatchString(key) {
			return true
		}
	}
	return false
}

var filepathEvalSymlinks = func(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}
