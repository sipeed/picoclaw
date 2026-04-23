package launcherconfig

import (
	"context"
	"strings"
)

var (
	loadConfigForMigration = Load
	saveConfigForMigration = Save
)

type dashboardPasswordStore interface {
	IsInitialized(ctx context.Context) (bool, error)
	SetPassword(ctx context.Context, plain string) error
}

// LegacyLauncherTokenMigrationResult reports the outcome of converting a
// removed launcher_token value into the current password-based auth flow.
type LegacyLauncherTokenMigrationResult struct {
	Migrated bool
	// CleanupErr is non-nil when password migration succeeded (or was already in
	// place) but removing launcher_token from launcher-config.json failed.
	CleanupErr error
}

// MigrateLegacyLauncherToken converts the removed launcher_token setting into
// the current password-login store, then removes launcher_token from config.
func MigrateLegacyLauncherToken(
	ctx context.Context,
	store dashboardPasswordStore,
	launcherPath string,
	fallback Config,
) (LegacyLauncherTokenMigrationResult, error) {
	legacyToken := strings.TrimSpace(fallback.LegacyLauncherToken)
	if legacyToken == "" || store == nil {
		return LegacyLauncherTokenMigrationResult{}, nil
	}

	result := LegacyLauncherTokenMigrationResult{}
	initialized, err := store.IsInitialized(ctx)
	if err != nil {
		return result, err
	}
	if !initialized {
		if err = store.SetPassword(ctx, legacyToken); err != nil {
			return result, err
		}
		result.Migrated = true
	}
	result.CleanupErr = cleanupLegacyLauncherTokenConfig(launcherPath, fallback)
	return result, nil
}

func cleanupLegacyLauncherTokenConfig(launcherPath string, fallback Config) error {
	cfg, err := loadConfigForMigration(launcherPath, fallback)
	if err != nil {
		return err
	}
	cfg.LegacyLauncherToken = ""
	return saveConfigForMigration(launcherPath, cfg)
}
