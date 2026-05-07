package launcherconfig

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type stubMigrationPasswordStore struct {
	initialized bool
	password    string
}

func (s *stubMigrationPasswordStore) IsInitialized(context.Context) (bool, error) {
	return s.initialized, nil
}

func (s *stubMigrationPasswordStore) SetPassword(_ context.Context, plain string) error {
	s.password = plain
	s.initialized = true
	return nil
}

func TestMigrateLegacyLauncherToken(t *testing.T) {
	dir := t.TempDir()
	launcherPath := filepath.Join(dir, FileName)
	cfg := Config{
		Port:                DefaultPort,
		LegacyLauncherToken: "legacy-password",
	}
	if err := os.WriteFile(
		launcherPath,
		[]byte("{\n  \"port\": 18800,\n  \"launcher_token\": \"legacy-password\"\n}\n"),
		0o600,
	); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := NewPasswordStore(launcherPath, Default())
	result, err := MigrateLegacyLauncherToken(context.Background(), store, launcherPath, cfg)
	if err != nil {
		t.Fatalf("MigrateLegacyLauncherToken() error = %v", err)
	}
	if !result.Migrated {
		t.Fatal("MigrateLegacyLauncherToken().Migrated = false, want true")
	}
	if result.CleanupErr != nil {
		t.Fatalf("MigrateLegacyLauncherToken().CleanupErr = %v, want nil", result.CleanupErr)
	}

	loaded, err := Load(launcherPath, Default())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.LegacyLauncherToken != "" {
		t.Fatalf("legacy launcher token = %q, want empty", loaded.LegacyLauncherToken)
	}
	if loaded.DashboardPasswordHash == "" {
		t.Fatal("dashboard password hash should be set after migration")
	}
	ok, err := store.VerifyPassword(context.Background(), "legacy-password")
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if !ok {
		t.Fatal("VerifyPassword() = false, want true")
	}
}

func TestMigrateLegacyLauncherTokenCleanupFailureIsNonFatal(t *testing.T) {
	dir := t.TempDir()
	launcherPath := filepath.Join(dir, FileName)
	cfg := Config{
		Port:                DefaultPort,
		LegacyLauncherToken: "legacy-password",
	}
	if err := os.WriteFile(
		launcherPath,
		[]byte("{\n  \"port\": 18800,\n  \"launcher_token\": \"legacy-password\"\n}\n"),
		0o600,
	); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := &stubMigrationPasswordStore{}
	origSave := saveConfigForMigration
	saveConfigForMigration = func(string, Config) error {
		return errors.New("write launcher config")
	}
	t.Cleanup(func() {
		saveConfigForMigration = origSave
	})

	result, err := MigrateLegacyLauncherToken(context.Background(), store, launcherPath, cfg)
	if err != nil {
		t.Fatalf("MigrateLegacyLauncherToken() error = %v, want nil", err)
	}
	if !result.Migrated {
		t.Fatal("MigrateLegacyLauncherToken().Migrated = false, want true")
	}
	if result.CleanupErr == nil {
		t.Fatal("MigrateLegacyLauncherToken().CleanupErr = nil, want non-nil")
	}
	if store.password != "legacy-password" {
		t.Fatalf("password = %q, want legacy-password", store.password)
	}

	loaded, err := Load(launcherPath, Default())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.LegacyLauncherToken != "legacy-password" {
		t.Fatalf(
			"legacy launcher token = %q, want legacy-password after cleanup failure",
			loaded.LegacyLauncherToken,
		)
	}
}

func TestMigrateLegacyLauncherTokenNoopWithoutToken(t *testing.T) {
	launcherPath := filepath.Join(t.TempDir(), FileName)
	store := NewPasswordStore(launcherPath, Default())
	result, err := MigrateLegacyLauncherToken(context.Background(), store, launcherPath, Default())
	if err != nil {
		t.Fatalf("MigrateLegacyLauncherToken() error = %v", err)
	}
	if result.Migrated {
		t.Fatal("MigrateLegacyLauncherToken().Migrated = true, want false")
	}
	if result.CleanupErr != nil {
		t.Fatalf("MigrateLegacyLauncherToken().CleanupErr = %v, want nil", result.CleanupErr)
	}
}
