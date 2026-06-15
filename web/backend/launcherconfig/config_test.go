package launcherconfig

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadReturnsFallbackWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "launcher-config.json")
	fallback := Default()
	fallback.Port = 19999
	fallback.Public = true

	got, err := Load(path, fallback)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Port != fallback.Port || got.Public != fallback.Public {
		t.Fatalf("Load() = %+v, want %+v", got, fallback)
	}
	if !got.AllowLocalhostBypass {
		t.Fatal("allow_localhost_bypass = false, want true")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "launcher-config.json")
	want := Config{
		Port:                  18080,
		Public:                true,
		AllowedCIDRs:          []string{"192.168.1.0/24", "10.0.0.0/8"},
		AllowLocalhostBypass:  false,
		TrustedProxyCIDRs:     []string{"172.16.0.0/12"},
		DashboardPasswordHash: "$2a$12$saved-dashboard-password-hash",
		LegacyLauncherToken:   "legacy-token-should-not-persist",
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
	if got.AllowLocalhostBypass != want.AllowLocalhostBypass {
		t.Fatalf("allow_localhost_bypass = %t, want %t", got.AllowLocalhostBypass, want.AllowLocalhostBypass)
	}
	if got.DashboardPasswordHash != want.DashboardPasswordHash {
		t.Fatalf("dashboard_password_hash = %q, want %q", got.DashboardPasswordHash, want.DashboardPasswordHash)
	}
	if got.LegacyLauncherToken != "" {
		t.Fatalf("legacy launcher_token = %q, want empty after Save", got.LegacyLauncherToken)
	}
	if len(got.AllowedCIDRs) != len(want.AllowedCIDRs) {
		t.Fatalf("allowed_cidrs len = %d, want %d", len(got.AllowedCIDRs), len(want.AllowedCIDRs))
	}
	for i := range want.AllowedCIDRs {
		if got.AllowedCIDRs[i] != want.AllowedCIDRs[i] {
			t.Fatalf("allowed_cidrs[%d] = %q, want %q", i, got.AllowedCIDRs[i], want.AllowedCIDRs[i])
		}
	}
	if len(got.TrustedProxyCIDRs) != len(want.TrustedProxyCIDRs) {
		t.Fatalf("trusted_proxy_cidrs len = %d, want %d", len(got.TrustedProxyCIDRs), len(want.TrustedProxyCIDRs))
	}
	for i := range want.TrustedProxyCIDRs {
		if got.TrustedProxyCIDRs[i] != want.TrustedProxyCIDRs[i] {
			t.Fatalf("trusted_proxy_cidrs[%d] = %q, want %q", i, got.TrustedProxyCIDRs[i], want.TrustedProxyCIDRs[i])
		}
	}

	stat, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if perm := stat.Mode().Perm(); runtime.GOOS == "windows" {
		if perm&0o200 == 0 {
			t.Fatalf("file perm = %o, want owner-writable on Windows", perm)
		}
	} else if perm != 0o600 {
		t.Fatalf("file perm = %o, want 600", perm)
	}
}

func TestLoadReadsLegacyLauncherTokenForMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "launcher-config.json")
	if err := os.WriteFile(path, []byte(`{"port":18800,"launcher_token":"legacy-token"}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := Load(path, Default())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.LegacyLauncherToken != "legacy-token" {
		t.Fatalf("legacy launcher_token = %q, want legacy-token", got.LegacyLauncherToken)
	}
}

func TestLoadDefaultsAllowLocalhostBypassForLegacyConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "launcher-config.json")
	if err := os.WriteFile(path, []byte(`{"port":18800,"allowed_cidrs":["10.0.0.0/8"]}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := Load(path, Default())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !got.AllowLocalhostBypass {
		t.Fatal("allow_localhost_bypass = false, want true for legacy config")
	}
	if got.AllowLocalhostBypassSource != BoolFieldAbsent {
		t.Fatalf("allow_localhost_bypass source = %v, want %v", got.AllowLocalhostBypassSource, BoolFieldAbsent)
	}
}

func TestLoadTracksAllowLocalhostBypassSource(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantValue  bool
		wantSource BoolFieldSource
	}{
		{
			name:       "explicit true",
			body:       `{"port":18800,"allow_localhost_bypass":true}`,
			wantValue:  true,
			wantSource: BoolFieldPresent,
		},
		{
			name:       "explicit false",
			body:       `{"port":18800,"allow_localhost_bypass":false}`,
			wantValue:  false,
			wantSource: BoolFieldPresent,
		},
		{
			name:       "explicit null keeps fallback behavior",
			body:       `{"port":18800,"allow_localhost_bypass":null}`,
			wantValue:  true,
			wantSource: BoolFieldNull,
		},
		{
			name:       "omitted uses fallback",
			body:       `{"port":18800}`,
			wantValue:  true,
			wantSource: BoolFieldAbsent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "launcher-config.json")
			if err := os.WriteFile(path, []byte(tt.body), 0o600); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			got, err := Load(path, Default())
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if got.AllowLocalhostBypass != tt.wantValue {
				t.Fatalf("allow_localhost_bypass = %t, want %t", got.AllowLocalhostBypass, tt.wantValue)
			}
			if got.AllowLocalhostBypassSource != tt.wantSource {
				t.Fatalf("allow_localhost_bypass source = %v, want %v", got.AllowLocalhostBypassSource, tt.wantSource)
			}
		})
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

func TestValidateRejectsInvalidTrustedProxyCIDR(t *testing.T) {
	err := Validate(Config{
		Port:              18800,
		TrustedProxyCIDRs: []string{"not-a-cidr"},
	})
	if err == nil {
		t.Fatal("Validate() expected error for invalid trusted proxy CIDR")
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

func TestPasswordStoreSetAndVerify(t *testing.T) {
	path := filepath.Join(t.TempDir(), "launcher-config.json")
	store := NewPasswordStore(path, Default())
	ctx := context.Background()

	initialized, err := store.IsInitialized(ctx)
	if err != nil {
		t.Fatalf("IsInitialized() error = %v", err)
	}
	if initialized {
		t.Fatal("IsInitialized() = true, want false before SetPassword")
	}

	if err = store.SetPassword(ctx, "dashboard-password"); err != nil {
		t.Fatalf("SetPassword() error = %v", err)
	}
	initialized, err = store.IsInitialized(ctx)
	if err != nil {
		t.Fatalf("IsInitialized() after SetPassword error = %v", err)
	}
	if !initialized {
		t.Fatal("IsInitialized() = false, want true after SetPassword")
	}
	ok, err := store.VerifyPassword(ctx, "dashboard-password")
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if !ok {
		t.Fatal("VerifyPassword(correct) = false, want true")
	}
	ok, err = store.VerifyPassword(ctx, "wrong-password")
	if err != nil {
		t.Fatalf("VerifyPassword(wrong) error = %v", err)
	}
	if ok {
		t.Fatal("VerifyPassword(wrong) = true, want false")
	}
}
