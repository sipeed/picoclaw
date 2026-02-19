# picoclaw skills

Manage skills - specialized capabilities for the agent.

## Usage

```bash
# List installed skills
picoclaw skills list

# Install from GitHub
picoclaw skills install <repo>

# Remove a skill
picoclaw skills remove <name>

# Install builtin skills
picoclaw skills install-builtin

# List builtin skills
picoclaw skills list-builtin

# Search for skills
picoclaw skills search

# Show skill details
picoclaw skills show <name>
```

## Subcommands

### skills list

List all installed skills.

```bash
picoclaw skills list
```

Output:
```
Installed Skills:
------------------
  ✓ weather (workspace)
    Get weather information for any location
  ✓ news (builtin)
    Fetch latest news headlines
  ✓ calculator (builtin)
    Perform calculations
```

### skills install

Install a skill from GitHub.

```bash
picoclaw skills install sipeed/picoclaw-skills/weather
```

### skills remove

Remove an installed skill.

```bash
picoclaw skills remove weather
```

### skills install-builtin

Copy all builtin skills to your workspace.

```bash
picoclaw skills install-builtin
```

### skills list-builtin

List available builtin skills.

```bash
picoclaw skills list-builtin
```

Output:
```
Available Builtin Skills:
-----------------------
  ✓  weather
     Get weather information
  ✓  news
     Get latest news headlines
  ✓  stock
     Check stock prices
  ✓  calculator
     Perform mathematical calculations
```

### skills search

Search for available skills online.

```bash
picoclaw skills search
```

### skills show

Show details of a specific skill.

```bash
picoclaw skills show weather
```

## Skill Locations

Skills are loaded from (in order):

1. `~/.picoclaw/workspace/skills/` - Workspace skills
2. `~/.picoclaw/skills/` - Global skills
3. `~/.picoclaw/picoclaw/skills/` - Builtin skills

## Skill Structure

A skill is a directory containing `SKILL.md`:

```
weather/
├── SKILL.md      # Skill definition
└── (optional files)
```

## Creating Skills

See [Creating Skills](../skills/creating-skills.md) for how to create custom skills.

## Examples

```bash
# Install a skill
picoclaw skills install sipeed/picoclaw-skills/weather

# List what's installed
picoclaw skills list

# Install all builtin skills
picoclaw skills install-builtin

# Show skill content
picoclaw skills show weather

# Remove a skill
picoclaw skills remove weather
```

## See Also

- [Skills Overview](../skills/README.md)
- [Creating Skills](../skills/creating-skills.md)
- [CLI Reference](../cli-reference.md)
