# picoclaw migrate

Migrate from OpenClaw to PicoClaw.

## Usage

```bash
# Detect and migrate
picoclaw migrate

# Preview changes
picoclaw migrate --dry-run

# Full options
picoclaw migrate [options]
```

## Options

| Flag | Description |
|------|-------------|
| `--dry-run` | Show what would be migrated without making changes |
| `--refresh` | Re-sync workspace files from OpenClaw |
| `--config-only` | Only migrate config, skip workspace files |
| `--workspace-only` | Only migrate workspace files, skip config |
| `--force` | Skip confirmation prompts |
| `--openclaw-home` | Override OpenClaw home directory (default: ~/.openclaw) |
| `--picoclaw-home` | Override PicoClaw home directory (default: ~/.picoclaw) |

## Description

The migrate command helps transition from OpenClaw to PicoClaw by:

1. Detecting existing OpenClaw installation
2. Converting configuration to PicoClaw format
3. Copying workspace files (AGENT.md, MEMORY.md, etc.)
4. Preserving session history

## Examples

### Basic Migration

```bash
# Run migration
picoclaw migrate

# Output shows what will be migrated
OpenClaw detected at ~/.openclaw
Migrating to ~/.picoclaw...
✓ Config migrated
✓ Workspace files copied
✓ Sessions preserved
Migration complete!
```

### Preview Changes

```bash
picoclaw migrate --dry-run
```

### Re-sync Workspace

```bash
# If you updated OpenClaw workspace, re-sync
picoclaw migrate --refresh
```

### Partial Migration

```bash
# Only migrate config
picoclaw migrate --config-only

# Only migrate workspace
picoclaw migrate --workspace-only
```

### Custom Directories

```bash
picoclaw migrate --openclaw-home /path/to/openclaw --picoclaw-home /path/to/picoclaw
```

## What Gets Migrated

| OpenClaw | PicoClaw |
|----------|----------|
| `~/.openclaw/config.json` | `~/.picoclaw/config.json` |
| `~/.openclaw/workspace/AGENT.md` | `~/.picoclaw/workspace/AGENT.md` |
| `~/.openclaw/workspace/IDENTITY.md` | `~/.picoclaw/workspace/IDENTITY.md` |
| `~/.openclaw/workspace/MEMORY.md` | `~/.picoclaw/workspace/MEMORY.md` |
| `~/.openclaw/workspace/HEARTBEAT.md` | `~/.picoclaw/workspace/HEARTBEAT.md` |
| Sessions | Sessions |

## After Migration

1. Review `~/.picoclaw/config.json`
2. Update any provider API keys
3. Test with `picoclaw agent -m "Hello"`
4. Start gateway with `picoclaw gateway`

## See Also

- [Installation](../../getting-started/installation.md)
- [Configuration](../../configuration/config-file.md)
- [OpenClaw Migration Guide](../../migration/openclaw-migration.md)
