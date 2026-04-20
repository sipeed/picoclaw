package launcherconfig

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

const (
	// FileName is the launcher-specific settings file name.
	FileName = "launcher-config.json"
	// DefaultPort is the default port for the web launcher.
	DefaultPort = 18800
	// EnvLauncherToken overrides launcher dashboard token.
	EnvLauncherToken = "PICOCLAW_LAUNCHER_TOKEN"
	// EnvLauncherHost overrides launcher listen host.
	EnvLauncherHost = "PICOCLAW_LAUNCHER_HOST"

	// dashboardSigningKeyBytes is the HMAC-SHA256 key size (256 bits).
	dashboardSigningKeyBytes = 32
	// dashboardTokenEntropyBytes is CSPRNG length before base64 for the per-run dashboard token (256 bits).
	dashboardTokenEntropyBytes = 32
)

type DashboardTokenSource string

const (
	DashboardTokenSourceEnv    DashboardTokenSource = "env"
	DashboardTokenSourceConfig DashboardTokenSource = "config"
	DashboardTokenSourceRandom DashboardTokenSource = "random"
)

// Config stores launch parameters for the web backend service.
type Config struct {
	Port          int      `json:"port"`
	Public        bool     `json:"public"`
	AllowedCIDRs  []string `json:"allowed_cidrs,omitempty"`
	LauncherToken string   `json:"launcher_token,omitempty"`
}

// Default returns default launcher settings.
func Default() Config {
	return Config{Port: DefaultPort, Public: false}
}

// Validate checks if launcher settings are valid.
func Validate(cfg Config) error {
	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("port %d is out of range (1-65535)", cfg.Port)
	}
	for _, cidr := range cfg.AllowedCIDRs {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return fmt.Errorf("invalid CIDR %q", cidr)
		}
	}
	return nil
}

// EnsureDashboardSecrets returns signing key bytes and the effective dashboard token for this
// process. The signing key is freshly random each call; the token comes from
// EnvLauncherToken when set, otherwise launcher-config.json launcher_token,
// otherwise a new random token.
func EnsureDashboardSecrets(
	cfg Config,
) (effectiveToken string, signingKey []byte, source DashboardTokenSource, err error) {
	signingKey = make([]byte, dashboardSigningKeyBytes)
	if _, err = rand.Read(signingKey); err != nil {
		return "", nil, "", err
	}

	effectiveToken = strings.TrimSpace(os.Getenv(EnvLauncherToken))
	if effectiveToken != "" {
		return effectiveToken, signingKey, DashboardTokenSourceEnv, nil
	}
	effectiveToken = strings.TrimSpace(cfg.LauncherToken)
	if effectiveToken != "" {
		return effectiveToken, signingKey, DashboardTokenSourceConfig, nil
	}
	tok, genErr := randomDashboardToken()
	if genErr != nil {
		return "", nil, "", genErr
	}
	return tok, signingKey, DashboardTokenSourceRandom, nil
}

func randomDashboardToken() (string, error) {
	buf := make([]byte, dashboardTokenEntropyBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// NormalizeCIDRs trims entries, removes empty values, and deduplicates CIDRs.
func NormalizeCIDRs(cidrs []string) []string {
	if len(cidrs) == 0 {
		return nil
	}
	out := make([]string, 0, len(cidrs))
	seen := make(map[string]struct{}, len(cidrs))
	for _, raw := range cidrs {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// PathForAppConfig returns launcher-config path near the app config file.
func PathForAppConfig(appConfigPath string) string {
	dir := filepath.Dir(appConfigPath)
	if dir == "" || dir == "." {
		dir = "."
	}
	return filepath.Join(dir, FileName)
}

// Load reads launcher settings; fallback is returned when file does not exist.
func Load(path string, fallback Config) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fallback, nil
		}
		return Config{}, err
	}

	cfg := fallback
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	cfg.AllowedCIDRs = NormalizeCIDRs(cfg.AllowedCIDRs)
	cfg.LauncherToken = strings.TrimSpace(cfg.LauncherToken)
	if err := Validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Save writes launcher settings to disk.
func Save(path string, cfg Config) error {
	cfg.AllowedCIDRs = NormalizeCIDRs(cfg.AllowedCIDRs)
	cfg.LauncherToken = strings.TrimSpace(cfg.LauncherToken)
	if err := Validate(cfg); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}
