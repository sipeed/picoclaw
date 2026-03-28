# Config Schema Versioning Guide

## Overview

PicoClaw uses a schema versioning system for `config.json` to ensure smooth upgrades as the configuration format evolves.

## Version History

### Version 1
- **Introduction**: Initial version with version field support
- **Changes**: Added `version` field to Config struct
- **Migration**: No structural changes needed for existing configs

## How It Works

### Automatic Migration
When you load a config file:
1. The system first reads the `version` field from the JSON
2. Based on the detected version, it loads the appropriate config struct (`ConfigV0`, `ConfigV1`, etc.)
3. If the loaded version is less than the latest, migrations are applied incrementally
4. The version number is updated automatically
5. The migrated config is automatically saved back to disk

### Version Field
The `version` field in `config.json` indicates the schema version:
- `0` or missing: Legacy config (no version field)
- `1`: Current version with versioning support

```json
{
  "version": 1,
  "agents": {...},
  ...
}
```

## Adding a New Migration

When making breaking changes to the config schema:

### Step 1: Define the New Version Struct

Create a new struct for the new version if the structure changes significantly:

```go
// ConfigV2 represents version 2 config structure
type ConfigV2 struct {
    Version   int             `json:"version"`
    Agents    AgentsConfig    `json:"agents"`
    // ... other fields with new structure
}
```

### Step 2: Update Current Config Version

```go
const CurrentConfigVersion = 2  // Increment this
```

### Step 3: Add a Loader Function

```go
// loadConfigV2 loads a version 2 config
func loadConfigV2(data []byte) (*Config, error) {
    cfg := DefaultConfig()

    // Parse to ConfigV2 struct
    var v2 ConfigV2
    if err := json.Unmarshal(data, &v2); err != nil {
        return nil, err
    }

    // Convert to current Config
    cfg.Version = v2.Version
    cfg.Agents = v2.Agents
    // ... map other fields

    return cfg, nil
}
```

### Step 4: Add Migration Logic

```go
// applyMigration applies a single migration step from fromVersion to toVersion
func applyMigration(cfg *Config, fromVersion, toVersion int) (*Config, error) {
    switch toVersion {
    case 1:
        // Migration from version 0 to 1
        return &Config{
            Version: 1,
            Agents:  cfg.Agents,
            // ... copy all fields
        }, nil
    case 2:
        // Migration from version 1 to 2
        // Example: Move or rename fields
        migrated := *cfg
        migrated.Version = 2
        // Apply structural changes
        if cfg.SomeOldField != "" {
            migrated.SomeNewField = cfg.SomeOldField
        }
        return &migrated, nil
    default:
        return nil, fmt.Errorf("unsupported migration target version: %d", toVersion)
    }
}
```

### Step 5: Update LoadConfig Switch

```go
func LoadConfig(path string) (*Config, error) {
    // ... read file ...

    switch versionInfo.Version {
    case 0:
        cfg, err = loadConfigV0(data)
    case 1:
        cfg, err = loadConfigV1(data)
    case 2:
        cfg, err = loadConfigV2(data)
    default:
        return nil, fmt.Errorf("unsupported config version: %d", versionInfo.Version)
    }

    // ... migrate and validate ...
}
```

### Step 6: Test Your Migration

Create a test in `config_migration_test.go`:

```go
func TestMigrateV1ToV2(t *testing.T) {
    // Create a version 1 config
    v1Config := Config{
        Version: 1,
        // ... set up test data
    }

    // Apply migration
    migrated, err := applyMigration(&v1Config, 1, 2)
    if err != nil {
        t.Fatalf("Migration failed: %v", err)
    }

    // Verify version is updated
    if migrated.Version != 2 {
        t.Errorf("Expected version 2, got %d", migrated.Version)
    }

    // Verify data is preserved/transformed correctly
    // ...
}
```

## Migration Best Practices

1. **Version-Specific Structs**: Define a separate struct for each version that has structural changes
2. **Backward Compatibility**: Ensure old configs can still be loaded with their specific structs
3. **No Data Loss**: Migrations should preserve all user settings
4. **Idempotent**: Running the same migration multiple times should be safe
5. **Auto-Save**: Migrated configs are automatically saved to update the user's file
6. **Test Thoroughly**: Test with real user config files
7. **Update Defaults**: Keep `defaults.go` in sync with the latest schema

## Example Migration

### Scenario: Adding a new field with default value

Old config (version 1):
```json
{
  "version": 1,
  "agents": {
    "defaults": {
      "max_tokens": 32768
    }
  }
}
```

Migration to version 2:
```go
case 2:
    migrated := *cfg
    migrated.Version = 2

    // Add new field with default value if not set
    if migrated.Agents.Defaults.NewFeatureEnabled == false {
        // Use default value
    }

    return &migrated, nil
```

New config (version 2):
```json
{
  "version": 2,
  "agents": {
    "defaults": {
      "max_tokens": 32768,
      "new_feature_enabled": false
    }
  }
}
```

## Troubleshooting

### Config Not Upgrading
- Check that `CurrentConfigVersion` is incremented
- Verify migration logic in `applyMigration()` handles the target version
- Ensure `migrateConfig()` is called in `LoadConfig()`

### Migration Errors
- Check error messages for specific migration failures
- Review migration logic for edge cases
- Ensure all required fields are properly initialized
- Verify the loader function for the source version

### Data Loss After Migration
- Ensure all fields are copied during migration
- Check that the migration doesn't overwrite values with defaults unnecessarily
- Review the conversion logic in the loader functions

