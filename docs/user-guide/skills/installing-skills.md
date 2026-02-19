# Installing Skills

Learn how to install and manage skills in PicoClaw.

## Installation Methods

### From GitHub

Install skills directly from GitHub repositories:

```bash
picoclaw skills install <owner>/<repo>/<skill-name>
```

Example:
```bash
picoclaw skills install sipeed/picoclaw-skills/github
```

The installer fetches the `SKILL.md` file from the repository and saves it to your workspace.

### Builtin Skills

Install all builtin skills to your workspace for customization:

```bash
picoclaw skills install-builtin
```

This copies the builtin skills (weather, news, stock, calculator) to `~/.picoclaw/workspace/skills/`, allowing you to modify them.

### Manual Installation

Create a skill manually by creating a directory and `SKILL.md` file:

```bash
mkdir -p ~/.picoclaw/workspace/skills/my-skill
```

Then create `~/.picoclaw/workspace/skills/my-skill/SKILL.md` with your skill content.

## Managing Installed Skills

### List Installed Skills

View all installed skills:

```bash
picoclaw skills list
```

Output shows the skill name, source (workspace/global/builtin), and description:
```
Installed Skills:
------------------
  weather (workspace)
    Get current weather and forecasts
  github (global)
    Interact with GitHub using the gh CLI
  calculator (builtin)
    Perform mathematical calculations
```

### View Skill Details

Display the full content of a skill:

```bash
picoclaw skills show weather
```

Output:
```
Skill: weather
----------------------
# Weather

Two free services, no API keys needed.

## wttr.in (primary)

Quick one-liner:
curl -s "wttr.in/London?format=3"
...
```

### Remove a Skill

Uninstall a skill from your workspace:

```bash
picoclaw skills remove <skill-name>
```

Example:
```bash
picoclaw skills remove github
```

## Searching for Skills

Search the online skill registry for available skills:

```bash
picoclaw skills search
```

Output:
```
Available Skills (5):
--------------------
  github
     Interact with GitHub using the gh CLI
     Repo: sipeed/picoclaw-skills/github
     Author: picoclaw
     Tags: [git, github, ci]

  slack
     Send messages to Slack channels
     Repo: sipeed/picoclaw-skills/slack
     Author: picoclaw
     Tags: [slack, messaging]
...
```

## Skill Installation Locations

Skills are installed to different locations based on scope:

| Location | Path | Description |
|----------|------|-------------|
| Workspace | `~/.picoclaw/workspace/skills/` | Project-specific, highest priority |
| Global | `~/.picoclaw/skills/` | Shared across workspaces |
| Builtin | Packaged with PicoClaw | Always available as fallback |

### Choosing Installation Scope

- **Workspace skills** - For skills specific to a project or workflow
- **Global skills** - For skills you want available everywhere
- **Builtin skills** - Ready to use, can be copied for customization

## Installation Examples

### Install from Official Registry

```bash
# Install GitHub integration skill
picoclaw skills install sipeed/picoclaw-skills/github

# Install tmux session management skill
picoclaw skills install sipeed/picoclaw-skills/tmux
```

### Install Custom Skill from Your Repo

```bash
# From your personal GitHub
picoclaw skills install yourname/picoclaw-skills/my-custom-skill
```

### Install All Builtin Skills

```bash
# Copy builtin skills to workspace for customization
picoclaw skills install-builtin

# Now you can edit them
vim ~/.picoclaw/workspace/skills/weather/SKILL.md
```

## Updating Skills

Skills don't auto-update. To update a skill:

```bash
# Remove old version
picoclaw skills remove weather

# Reinstall from source
picoclaw skills install sipeed/picoclaw-skills/weather
```

## GitHub Repository Structure

Skill repositories should have this structure:

```
picoclaw-skills/
  weather/
    SKILL.md
  github/
    SKILL.md
  skills.json       # Optional: registry file
```

The `SKILL.md` file must be at the root of the skill directory.

## Troubleshooting

### Installation Fails

Common causes:

1. **Network error** - Check your internet connection
2. **Repository not found** - Verify the repository path
3. **Skill already exists** - Remove the existing skill first

```bash
# Remove and reinstall
picoclaw skills remove <name>
picoclaw skills install <repo>
```

### Skill Not Loading After Install

1. Verify installation: `picoclaw skills list`
2. Check file exists: `ls ~/.picoclaw/workspace/skills/<name>/SKILL.md`
3. Validate skill format: `picoclaw skills show <name>`

### Permission Denied

Ensure you have write permissions to the skills directory:

```bash
ls -la ~/.picoclaw/workspace/skills/
```

## See Also

- [Skills Overview](README.md)
- [Using Skills](using-skills.md)
- [Builtin Skills](builtin-skills.md)
- [Creating Skills](creating-skills.md)
- [CLI Reference: skills](../cli/skills.md)
