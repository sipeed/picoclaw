# Config Schema Versioning Guide

## Overview

PicoClaw uses a schema versioning system for `config.json` to ensure smooth upgrades as the configuration format evolves.

## Version History

### Version 1
- **Introduction**: Initial version with version field support
- **Changes**: Added `version` field to Config struct
- **Migration**: No structural changes needed for existing configs

### Version 2
- **Introduction**: Model enable/disable support and channel config unification
- **Changes**:
  - Added `enabled` field to `ModelConfig` — allows disabling individual model entries without removing them
  - During V1→V2 migration, `enabled` is auto-inferred: models with API keys or the reserved `local-model` name are enabled; others default to disabled
  - Migrated legacy channel fields: Discord `mention_only` → `group_trigger.mention_only`, OneBot `group_trigger_prefix` → `group_trigger.prefixes`
  - V0 configs now migrate directly to CurrentVersion (V2) instead of going through V1
  - `makeBackup()` now uses date-only suffix (e.g., `config.json.20260330.bak`) and also backs up `.security.yml`

## How It Works

### Automatic Migration
When you load a config file:
1. The system first reads the `version` field from the JSON
2. Based on the detected version, it loads the appropriate config struct (`configV0`, `configV1`, etc.)
3. If the loaded version is less than the latest, migrations are applied incrementally
4. Before saving, the system automatically creates a date-stamped backup of `config.json` and `.security.yml`
5. The version number is updated automatically
6. The migrated config is automatically saved back to disk

### Version Field
The `version` field in `config.json` indicates the schema version:
- `0` or missing: Legacy config (no version field)
- `1`: Previous version (will be auto-migrated to V2 on load)
- `2`: Current version

```json
{
  "version": 2,
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
const CurrentVersion = 2  // Increment this
```

### Step 3: Add a Loader Function

```go
// loadConfigV3 loads a version 3 config
func loadConfigV3(data []byte) (*Config, error) {
    cfg := DefaultConfig()

    // Parse to ConfigV3 struct
    var v3 ConfigV3
    if err := json.Unmarshal(data, &v3); err != nil {
        return nil, err
    }

    // Convert to current Config
    cfg.Version = v3.Version
    cfg.Agents = v3.Agents
    // ... map other fields

    return cfg, nil
}
```

### Step 4: Add Migration Logic

```go
func (c *configV2) Migrate() (*Config, error) {
    // Apply V2→V3 structural changes here
    migrated := &c.Config
    migrated.Version = 3
    // Apply structural changes
    return migrated, nil
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
        cfg, err = loadConfig(data)
    case 3:
        cfg, err = loadConfigV3(data)
    default:
        return nil, fmt.Errorf("unsupported config version: %d", versionInfo.Version)
    }

    // ... migrate and validate ...
}
```

### Step 6: Test Your Migration

Create a test in `config_migration_test.go`:

```go
func TestMigrateV2ToV3(t *testing.T) {
    // Create a version 2 config
    v2Config := Config{
        Version: 2,
        // ... set up test data
    }

    // Apply migration
    migrated, err := v2Config.Migrate()
    if err != nil {
        t.Fatalf("Migration failed: %v", err)
    }

    // Verify version is updated
    if migrated.Version != 3 {
        t.Errorf("Expected version 3, got %d", migrated.Version)
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
6. **Auto-Backup**: Before saving, the system creates a date-stamped backup of `config.json` and `.security.yml`
7. **Test Thoroughly**: Test with real user config files
8. **Update Defaults**: Keep `defaults.go` in sync with the latest schema

## Example Migration

### Scenario: Adding a new field with default value

Old config (version 2):
```json
{
  "version": 2,
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4"
    }
  ]
}
```

Migration to version 3:
```go
func (c *configV2) Migrate() (*Config, error) {
    migrated := &c.Config
    migrated.Version = 3

    // Add new field with default value if not set
    // ...

    return migrated, nil
}
```

New config (version 3):
```json
{
  "version": 3,
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "new_option": true
    }
  ]
}
```

## Troubleshooting

### Config Not Upgrading
- Check that `CurrentVersion` is incremented
- Verify migration logic handles the target version
- Ensure `Migrate()` is called in `LoadConfig()`

### Migration Errors
- Check error messages for specific migration failures
- Review migration logic for edge cases
- Ensure all required fields are properly initialized
- Verify the loader function for the source version

### Data Loss After Migration
- Ensure all fields are copied during migration
- Check that the migration doesn't overwrite values with defaults unnecessarily
- Review the conversion logic in the loader functions
- Check the auto-backup files (e.g., `config.json.20260330.bak`) to recover original data

