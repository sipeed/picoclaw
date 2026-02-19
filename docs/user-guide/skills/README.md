# Skills Overview

Skills are specialized capabilities that extend PicoClaw's abilities. They provide domain-specific knowledge, commands, and workflows that the agent can use to help you accomplish tasks.

## What are Skills?

A skill is a Markdown file (`SKILL.md`) that contains instructions and examples for the agent. When you ask PicoClaw to perform a task related to a skill, the agent loads that skill's content and follows its guidance.

Skills can include:

- **Command patterns** - Pre-defined shell commands for common tasks
- **API integrations** - Instructions for working with external services
- **Workflows** - Step-by-step procedures for complex operations
- **Domain knowledge** - Specialized information about topics

## Skill Types

### Builtin Skills

PicoClaw comes with builtin skills that are always available:

| Skill | Description |
|-------|-------------|
| `weather` | Get current weather and forecasts (no API key required) |
| `news` | Fetch latest news headlines |
| `stock` | Check stock prices and market data |
| `calculator` | Perform mathematical calculations |

### Community Skills

Install additional skills from GitHub repositories:

```bash
picoclaw skills install sipeed/picoclaw-skills/github
```

### Custom Skills

Create your own skills for specialized workflows:

```
~/.picoclaw/workspace/skills/
  my-skill/
    SKILL.md
```

## Skill Loading Priority

Skills are loaded from three locations in priority order:

1. **Workspace skills** - `~/.picoclaw/workspace/skills/`
   - Project-specific skills, highest priority
   - Override global and builtin skills with the same name

2. **Global skills** - `~/.picoclaw/skills/`
   - Shared skills across all workspaces
   - Override builtin skills with the same name

3. **Builtin skills** - Packaged with PicoClaw
   - Always available as fallback
   - Can be copied to workspace for customization

## Skill Structure

A skill is a directory containing a `SKILL.md` file:

```
weather/
  SKILL.md      # Required: skill definition
  ...           # Optional: supporting files
```

The `SKILL.md` file format:

```markdown
---
name: skill-name
description: Brief description of what this skill does
metadata:
  key: value
---

# Skill Title

Instructions, commands, and examples for the agent...
```

## Using Skills

Skills are automatically activated when relevant. Just ask PicoClaw:

```
User: "What's the weather in Tokyo?"
Agent: [Uses weather skill]
       The current weather in Tokyo is...
```

For more control, see [Using Skills](using-skills.md).

## Managing Skills

### List Installed Skills

```bash
picoclaw skills list
```

### Install from GitHub

```bash
picoclaw skills install sipeed/picoclaw-skills/github
```

### Remove a Skill

```bash
picoclaw skills remove github
```

### View Skill Content

```bash
picoclaw skills show weather
```

## Documentation Sections

- [Using Skills](using-skills.md) - How to use installed skills effectively
- [Installing Skills](installing-skills.md) - Installing skills from GitHub
- [Builtin Skills](builtin-skills.md) - List of builtin skills with examples
- [Creating Skills](creating-skills.md) - How to create custom skills

## See Also

- [CLI Reference: skills](../cli/skills.md)
- [Workspace Structure](../workspace/structure.md)
