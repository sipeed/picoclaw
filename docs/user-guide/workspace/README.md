# Workspace Overview

The workspace is where PicoClaw stores data and customization files.

## Workspace Location

Default: `~/.picoclaw/workspace/`

Configure in `~/.picoclaw/config.json`:

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace"
    }
  }
}
```

## Directory Structure

```
~/.picoclaw/workspace/
├── sessions/          # Conversation history
│   └── *.json        # Session files
├── memory/           # Long-term memory
│   └── MEMORY.md     # Persistent memory
├── state/            # Persistent state
│   └── state.json    # Last channel, etc.
├── cron/             # Scheduled jobs
│   └── jobs.json     # Job database
├── skills/           # Custom skills
│   └── skill-name/   # Skill directories
├── AGENT.md          # Agent behavior guide
├── IDENTITY.md       # Agent identity
├── SOUL.md           # Agent personality
├── USER.md           # User preferences
├── TOOLS.md          # Tool descriptions
└── HEARTBEAT.md      # Periodic tasks
```

## Customization Files

### AGENT.md

Defines how the agent behaves. Include:
- Role and capabilities
- Response style
- Constraints and preferences

### IDENTITY.md

Defines who the agent is:
- Name and purpose
- Background
- Personality traits

### USER.md

User preferences and information:
- Name and preferences
- Communication style
- Personal context

### MEMORY.md

Long-term memory storage:
- Important facts
- User preferences
- Persistent context

### HEARTBEAT.md

Periodic task prompts:
- Scheduled reminders
- Regular checks
- Automated tasks

## Security

When `restrict_to_workspace` is `true` (default):
- File operations limited to workspace
- Shell commands must run within workspace
- Symlink attacks prevented

Disable only in trusted environments:

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": false
    }
  }
}
```

## Multi-Agent Workspaces

Each agent can have its own workspace:

```json
{
  "agents": {
    "list": [
      {
        "id": "personal",
        "workspace": "~/.picoclaw/workspace/personal"
      },
      {
        "id": "work",
        "workspace": "~/.picoclaw/workspace/work"
      }
    ]
  }
}
```

## Guides

- [Directory Structure](structure.md)
- [AGENT.md Customization](agent-md.md)
- [IDENTITY.md Customization](identity-md.md)
- [Long-term Memory](memory.md)
- [Heartbeat Tasks](heartbeat-tasks.md)
