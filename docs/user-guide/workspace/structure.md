# Workspace Structure

Detailed breakdown of the workspace directory structure.

## Directory Overview

```
~/.picoclaw/workspace/
├── sessions/          # Conversation sessions
├── memory/           # Long-term memory
├── state/            # Runtime state
├── cron/             # Scheduled jobs
├── skills/           # Custom skills
├── AGENT.md          # Agent behavior
├── IDENTITY.md       # Agent identity
├── SOUL.md           # Agent soul
├── USER.md           # User info
├── TOOLS.md          # Tool descriptions
└── HEARTBEAT.md      # Periodic tasks
```

## Sessions Directory

**Path:** `sessions/`

Stores conversation history as JSON files.

```
sessions/
├── cli:default.json
├── telegram:123456789:user:123456789.json
└── discord:987654321:user:987654321.json
```

**Session file format:**

```json
{
  "messages": [
    {"role": "user", "content": "Hello"},
    {"role": "assistant", "content": "Hi there!"}
  ],
  "summary": "",
  "last_access": "2025-02-20T10:00:00Z"
}
```

## Memory Directory

**Path:** `memory/`

Contains MEMORY.md for long-term memory.

```markdown
# Long-term Memory

## User Information
- Name: Alice
- Location: Beijing
- Interests: AI, Programming

## Important Notes
- Prefers concise responses
- Works in tech industry
```

## State Directory

**Path:** `state/`

Stores runtime state that persists across restarts.

```
state/
└── state.json
```

Contains:
- Last active channel
- Last active user
- Other runtime info

## Cron Directory

**Path:** `cron/`

Stores scheduled job database.

```
cron/
└── jobs.json
```

## Skills Directory

**Path:** `skills/`

Contains installed custom skills.

```
skills/
├── weather/
│   └── SKILL.md
├── news/
│   └── SKILL.md
└── custom-skill/
    └── SKILL.md
```

## Markdown Files

### AGENT.md

Agent behavior instructions. The agent reads this to understand how to behave.

### IDENTITY.md

Agent identity - who the agent is, its purpose and background.

### SOUL.md

Agent personality and values.

### USER.md

User preferences and information about the user.

### TOOLS.md

Custom tool descriptions and usage guides.

### HEARTBEAT.md

Tasks to execute periodically (every 30 minutes by default).

## File Permissions

The workspace should be readable and writable by the user running PicoClaw:

```bash
chmod -R 755 ~/.picoclaw/workspace
```

Sensitive files (credentials, sessions) should be restricted:

```bash
chmod 600 ~/.picoclaw/workspace/sessions/*.json
```

## Backup

To backup your workspace:

```bash
tar -czf picoclaw-backup.tar.gz ~/.picoclaw/workspace
```

## See Also

- [AGENT.md Guide](agent-md.md)
- [Memory Guide](memory.md)
- [Heartbeat Tasks](heartbeat-tasks.md)
