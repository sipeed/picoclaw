# Backup and Restore

This document describes the PicoClaw CLI backup and restore commands (`picoclaw backup` and `picoclaw backup restore`). Other backup-related behavior (e.g. `onboard --force` creating a timestamped config backup, or `migrate` creating `.bak` files before overwriting) is mentioned only briefly; those do not have a dedicated restore command and require manual handling if you need to revert.

---

## 1. Overview

The backup feature creates a single compressed archive (tar.gz) of your PicoClaw config and workspace. Restore unpacks that archive back to the expected locations so you can recover after migration, reinstall, or accidental changes.

- **Backup**: `picoclaw backup` or `picoclaw backup create` — creates an archive.
- **List**: `picoclaw backup list` — shows which paths would be included (no archive is written).
- **Restore**: `picoclaw backup restore <archive>` — restores from an archive.

---

## 2. Why Backups Matter

- **Config and credentials**: `config.json` and `auth.json` hold model settings, channels, and authentication. Losing them means reconfiguring or logging in again.
- **Workspace assets**: Files like AGENTS.md, IDENTITY.md, SOUL.md, and USER.md define your agent’s role and preferences. The `memory`, `skills`, and `cron` directories hold long-lived content that is hard to recreate.
- **Recoverability**: Regular backups let you quickly revert after mistakes, failed upgrades, or when moving to a new machine, instead of starting from scratch.

---

## 3. Use Cases

- **Before upgrades or big changes**: Run `picoclaw backup create`, then perform the upgrade or `picoclaw onboard --force`. If something goes wrong, use `picoclaw backup restore` to roll back.
- **Moving to a new machine**: On the new machine, run `picoclaw onboard` (or create the directories), then `picoclaw backup restore --workspace <path> <archive>` to restore the archive and align config and workspace.
- **After accidental overwrite or deletion**: Restore from your latest backup with `picoclaw backup restore <archive>`. Use `--dry-run` first to see what would be restored.
- **Sharing or cloning an environment**: Copy the archive to a USB drive or cloud storage, then on another machine run restore to reproduce the same config and workspace (keep credentials secure).
- **Regular snapshots**: Use cron or a manual habit to run `picoclaw backup create -o ~/backups/picoclaw-YYYYMMDD.tar.gz` so you have dated snapshots to fall back on.

---

## 4. Default Storage Location

- **Default backup directory**: `~/.picoclaw/backups/`
- **Default filename pattern**: `picoclaw-backup-{YYYYMMDD-HHMMSS}.tar.gz` (UTC timestamp)
- You can override the path with `-o` or `--output`, e.g. `picoclaw backup create -o ~/Desktop/my-backup.tar.gz`

---

## 5. Backup Command

### Create

- **Usage**: `picoclaw backup` or `picoclaw backup create [options]`
- **Options**:
  - `-o`, `--output <path>` — Write the archive to this path (default: `~/.picoclaw/backups/picoclaw-backup-<timestamp>.tar.gz`).
  - `--with-sessions` — Include the `workspace/sessions` directory in the backup.

### List

- **Usage**: `picoclaw backup list [options]`
- Prints the local paths and their archive paths that would be included in a backup. No archive is created.

### What Gets Backed Up

Only paths that exist on disk are included:

- **Config**: `~/.picoclaw/config.json`, `~/.picoclaw/auth.json`
- **Workspace files**: AGENTS.md, HOOKS.md, IDENTITY.md, SOUL.md, TOOLS.md, USER.md
- **Workspace directories**: `memory/`, `skills/`, `cron/`
- **Optional**: `sessions/` (only with `--with-sessions`)

### Examples

```bash
# Create backup with default path
picoclaw backup create

# Create backup with custom path
picoclaw backup create -o ~/Desktop/picoclaw-backup.tar.gz

# Include sessions
picoclaw backup create --with-sessions

# See what would be backed up
picoclaw backup list
```

---

## 6. Restore Command

- **Usage**: `picoclaw backup restore <archive> [options]`
- **Options**:
  - `--dry-run` — Print what would be restored without writing files.
  - `--force` — Overwrite existing files. Without this, existing paths are skipped.
  - `--workspace <path>` — Restore workspace contents into this directory. If omitted, the workspace path from the current config is used (or you must have run `picoclaw onboard` first).

Restore maps archive paths to the current environment:

- `picoclaw/*` → `~/.picoclaw/*`
- `workspace/*` → current workspace directory (from config or `--workspace`)

If there is no config (e.g. fresh machine), you must pass `--workspace` so the command knows where to put workspace files; config files will still be restored under `~/.picoclaw/`.

### Examples

```bash
# Restore to current config locations
picoclaw backup restore ~/.picoclaw/backups/picoclaw-backup-20260101-120000.tar.gz

# Preview only
picoclaw backup restore backup.tar.gz --dry-run

# Restore workspace to a different directory and overwrite existing files
picoclaw backup restore backup.tar.gz --workspace ~/my-workspace --force
```

---

## 7. Archive Format (for scripts or debugging)

The archive is gzip-compressed tar. Paths inside use forward slashes and two top-level prefixes:

- `picoclaw/` — config and auth (e.g. `picoclaw/config.json`, `picoclaw/auth.json`)
- `workspace/` — workspace files and directories (e.g. `workspace/AGENTS.md`, `workspace/memory/`, `workspace/skills/`)

Example layout:

```
picoclaw/config.json
picoclaw/auth.json
workspace/AGENTS.md
workspace/HOOKS.md
workspace/IDENTITY.md
workspace/SOUL.md
workspace/TOOLS.md
workspace/USER.md
workspace/memory/
workspace/skills/
workspace/cron/
workspace/sessions/   (only if created with --with-sessions)
```

---

## 8. Other Backup-Related Behavior

- **`picoclaw onboard --force`**: Before overwriting, backs up the existing config to `config.json.bak.<timestamp>`. There is no CLI restore for this; copy the file back manually if needed.
- **`picoclaw migrate`**: When copying over existing files, creates a `.bak` copy first. Again, no CLI restore; restore those files manually if required.

---

## 9. FAQ

- **After restore, do I need to restart anything?** If you use the gateway or a background service, restart it so it picks up the restored config and workspace.
- **Are credentials safe?** `auth.json` can contain tokens; treat backup archives as sensitive and store them securely.
- **Restoring on a different machine?** Use `--workspace` if the workspace path differs, and ensure the restored `config.json`’s workspace path is correct for the new machine.
