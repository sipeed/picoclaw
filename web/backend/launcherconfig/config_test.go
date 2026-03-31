package launcherconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/web/backend/middleware"
)

func TestLoadReturnsFallbackWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "launcher-config.json")
	fallback := Config{Port: 19999, Public: true}

	got, err := Load(path, fallback)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Port != fallback.Port || got.Public != fallback.Public {
		t.Fatalf("Load() = %+v, want %+v", got, fallback)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "launcher-config.json")
	want := Config{
		Port:         18080,
		Public:       true,
		AllowedCIDRs: []string{"192.168.1.0/24", "10.0.0.0/8"},
	}

	if err := Save(path, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := Load(path, Default())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Port != want.Port || got.Public != want.Public {
		t.Fatalf("Load() = %+v, want %+v", got, want)
	}
	if len(got.AllowedCIDRs) != len(want.AllowedCIDRs) {
		t.Fatalf("allowed_cidrs len = %d, want %d", len(got.AllowedCIDRs), len(want.AllowedCIDRs))
	}
	for i := range want.AllowedCIDRs {
		if got.AllowedCIDRs[i] != want.AllowedCIDRs[i] {
			t.Fatalf("allowed_cidrs[%d] = %q, want %q", i, got.AllowedCIDRs[i], want.AllowedCIDRs[i])
		}
	}

	stat, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if perm := stat.Mode().Perm(); perm != 0o600 {
		t.Fatalf("file perm = %o, want 600", perm)
	}
}

func TestValidateRejectsInvalidPort(t *testing.T) {
	if err := Validate(Config{Port: 0, Public: false}); err == nil {
		t.Fatal("Validate() expected error for port 0")
	}
	if err := Validate(Config{Port: 65536, Public: false}); err == nil {
		t.Fatal("Validate() expected error for port 65536")
	}
}

func TestValidateRejectsInvalidCIDR(t *testing.T) {
	err := Validate(Config{
		Port:         18800,
		AllowedCIDRs: []string{"192.168.1.0/24", "not-a-cidr"},
	})
	if err == nil {
		t.Fatal("Validate() expected error for invalid CIDR")
	}
}

func TestEnsureDashboardSecrets_GeneratesEphemeral(t *testing.T) {
	t.Setenv("PICOCLAW_LAUNCHER_TOKEN", "")

	tok, key, newTok, err := EnsureDashboardSecrets()
	if err != nil {
		t.Fatalf("EnsureDashboardSecrets() error = %v", err)
	}
	if !newTok || tok == "" || len(key) != dashboardSigningKeyBytes {
		t.Fatalf("unexpected first call: newTok=%v tok=%q keyLen=%d", newTok, tok, len(key))
	}
	mac := middleware.SessionCookieValue(key, tok)
	if mac == "" {
		t.Fatal("empty session mac")
	}

	tok2, key2, newTok2, err := EnsureDashboardSecrets()
	if err != nil {
		t.Fatalf("EnsureDashboardSecrets() second error = %v", err)
	}
	if !newTok2 {
		t.Fatal("second call without env should generate another random token")
	}
	if tok2 == tok {
		t.Fatal("expected a new random dashboard token")
	}
	if string(key2) == string(key) {
		t.Fatal("expected a new signing key")
	}
}

func TestEnsureDashboardSecrets_EnvOverridesGenerated(t *testing.T) {
	t.Setenv("PICOCLAW_LAUNCHER_TOKEN", "env-only-token-override")

	tok, _, newTok, err := EnsureDashboardSecrets()
	if err != nil {
		t.Fatalf("EnsureDashboardSecrets() error = %v", err)
	}
	if tok != "env-only-token-override" {
		t.Fatalf("token = %q, want env value", tok)
	}
	if newTok {
		t.Fatal("newRandomDashboardToken should be false when env is set")
	}
}

func TestNormalizeCIDRs(t *testing.T) {
	got := NormalizeCIDRs([]string{" 192.168.1.0/24 ", "", "10.0.0.0/8", "192.168.1.0/24"})
	want := []string{"192.168.1.0/24", "10.0.0.0/8"}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
