# Skills CLI Reference

The `picoclaw skills` command manages local skills: install, list, remove, search, and view. Installed skills are written under your **workspace** at `skills/{skillName}/`. At runtime, the [SkillsLoader](pkg/skills/loader.go) discovers skills by scanning these directories; a directory is treated as a valid skill only if it contains a `SKILL.md` file.

Some features (e.g. `reinstall`, `repo@branch`, optional `subpath`) may require a recent PicoClaw version. If a command or option is not recognized, upgrade to the latest release.

---

## Subcommands Overview

| Subcommand | Usage | Description |
|------------|--------|-------------|
| list | `picoclaw skills list` | List installed skills |
| install | `picoclaw skills install <repo> [subpath]`, `install <path-or-url> [name]`, or `install --registry <name> <slug>` | Install from GitHub, from a .zip/.tar.gz/.tgz file or URL, or from a registry |
| reinstall | `picoclaw skills reinstall <repo> [subpath]` | Overwrite install (remove then install) |
| install-builtin | `picoclaw skills install-builtin` | Copy built-in skills into the workspace |
| list-builtin | `picoclaw skills list-builtin` | List available built-in skills |
| remove | `picoclaw skills remove <name>` | Uninstall a skill by name |
| search | `picoclaw skills search` | Search list of installable skills (e.g. picoclaw-skills) |
| show | `picoclaw skills show <name>` | Show the content of an installed skill |

---

## Install from GitHub (install / reinstall)

### Repo format

- **`owner/repo`** — Uses the GitHub API to resolve the repository’s default branch; if that fails, falls back to `main`.
- **`owner/repo@branch`** — Use a specific branch (e.g. `owner/repo@v1`).

### Optional subpath

For a monorepo, you can pass a **subpath** (e.g. `skills/k8s-report`). The **skill name** is the last segment of `subpath` if given; otherwise it is the repo name.

### Behavior

- **install** — If `workspace/skills/{skillName}` already exists, the command fails and suggests using `reinstall`.
- **reinstall** — Removes the existing skill directory, then performs the same download and write as `install` (overwrite update).

### What gets installed (production)

The installer uses the GitHub Trees API to list all blobs under the given branch (and optionally under `subpath`). The tree must include `SKILL.md` (at the repo root or under `subpath`). Files are downloaded from `https://raw.githubusercontent.com/{repo}/{branch}/{path}` and written under `workspace/skills/{skillName}/` with the correct relative paths (subpath prefix stripped).

### Examples

```bash
# Install from repo root (default branch)
picoclaw skills install sipeed/picoclaw-skills

# Install from a subpath (skill name = k8s-report)
picoclaw skills install sipeed/picoclaw-skills k8s-report

# Install from a specific branch
picoclaw skills install owner/repo@v1

# Overwrite an existing install
picoclaw skills reinstall sipeed/picoclaw-skills k8s-report
```

---

## Install from archive (zip / tar.gz / tgz)

You can install a skill from a **local file** or **HTTP(S) URL** that points to a `.zip`, `.tar.gz`, or `.tgz` archive. The archive must contain a `SKILL.md` at the root (or as the only file under a single top-level directory, which is normalized automatically).

**Usage:** `picoclaw skills install <path-or-url> [name]`

- **path-or-url** — A local path (e.g. `./skill.zip`, `/tmp/skill.tar.gz`) or a URL (e.g. `https://example.com/skill.zip`). To avoid conflicting with GitHub repo names like `owner/repo.zip`, the installer treats an argument as an archive only when it **looks like a path or URL** (starts with `./`, `/`, `\`, or `http://`/`https://`) **and** has an archive extension.
- **name** — Optional. The skill name under `workspace/skills/`. If omitted, it is derived from the file or URL name (e.g. `skill.zip` → `skill`).

**Supported formats:** `.zip`, `.tar.gz`, `.tgz`.

**Reinstall:** Use `picoclaw skills reinstall <path-or-url> [name]` to overwrite an existing skill installed from an archive.

**Examples:**

```bash
# Install from a local zip file (skill name = my-skill from filename)
picoclaw skills install ./my-skill.zip

# Install from a URL and set skill name explicitly
picoclaw skills install https://example.com/skill.tar.gz my-skill

# Overwrite existing install
picoclaw skills reinstall ./skill.zip
```

---

## Install from Registry

Use a configured registry (e.g. ClawHub) to install by slug.

**Usage:** `picoclaw skills install --registry <registry_name> <slug>`

The registry must be enabled under `tools.skills` in your config. See [Tools Configuration – Skills Tool](#related-configuration) for registry settings. If the skill is already installed, the command fails. Currently only the GitHub path supports overwriting via `reinstall`; registry install does not have a reinstall/force option yet.

**Example:**

```bash
picoclaw skills install --registry clawhub github
```

---

## Built-in Skills (install-builtin / list-builtin)

- **install-builtin** — Copies a predefined set of built-in skills from the PicoClaw install directory into the current workspace’s `skills/` directory.
- **list-builtin** — Lists available built-in skill names and descriptions (read-only; does not install).

---

## Other Subcommands

- **remove** / **uninstall** — Deletes `workspace/skills/<name>`. Fails if the skill is not installed.
- **search** — Fetches the remote skills list (e.g. from sipeed/picoclaw-skills `skills.json`) and prints name, description, repository, author, and tags.
- **show** — Reads and prints the installed skill’s `SKILL.md` content (as used by the SkillsLoader).

---

## Install Directory and Discovery

- **Install directory:** `{workspace}/skills/{skillName}/`
  - For GitHub: `skillName` is the last segment of `subpath` if provided, otherwise the repo name.
  - For registry: `skillName` is the install slug.
- **Discovery:** The SkillsLoader scans workspace, global, and built-in skills directories. A directory is considered a valid skill only if it contains `SKILL.md`.

---

## Errors and Tips

| Situation | What to do |
|-----------|------------|
| Skill already exists | Use `reinstall` to overwrite (GitHub path only). |
| No `SKILL.md` (GitHub install) | Ensure the repo (and subpath, if used) contains a `SKILL.md` file. |
| Registry not found or disabled | Check `tools.skills.registries` in your config and see [Tools Configuration](tools_configuration.md#skills-tool). |
| Skill not found for `remove` / `show` | Confirm the skill name (e.g. from `picoclaw skills list`) and that it is installed in the current workspace. |

---

## Related Configuration

Registry enablement and API paths (e.g. ClawHub base URL, skills path, download path) are configured under the Skills Tool. See [Tools Configuration – Skills Tool](tools_configuration.md#skills-tool).
