# picoclaw onboard

Initialize PicoClaw configuration and workspace.

## Usage

```bash
picoclaw onboard
```

## Description

The `onboard` command sets up PicoClaw for first-time use by:

1. Creating the configuration directory at `~/.picoclaw/`
2. Generating `~/.picoclaw/config.json` with default settings
3. Creating the workspace directory at `~/.picoclaw/workspace/`
4. Copying default templates (AGENT.md, IDENTITY.md, etc.)

## What It Creates

```
~/.picoclaw/
├── config.json           # Main configuration file
└── workspace/
    ├── sessions/         # Conversation history
    ├── memory/           # Long-term memory
    ├── cron/             # Scheduled jobs
    ├── skills/           # Custom skills
    ├── AGENT.md          # Agent behavior guide
    ├── IDENTITY.md       # Agent identity
    ├── SOUL.md           # Agent personality
    ├── USER.md           # User preferences
    └── HEARTBEAT.md      # Periodic tasks
```

## Example

```bash
$ picoclaw onboard

Config already exists at /home/user/.picoclaw/config.json
Overwrite? (y/n): y

🦞 picoclaw is ready!

Next steps:
  1. Add your API key to /home/user/.picoclaw/config.json
     Get one at: https://openrouter.ai/keys
  2. Chat: picoclaw agent -m "Hello!"
```

## Next Steps

After running onboard:

1. Edit `~/.picoclaw/config.json` and add your API key
2. Run `picoclaw agent -m "Hello!"` to test

## See Also

- [Quick Start](../../getting-started/quick-start.md)
- [Configuration Reference](../../configuration/config-file.md)
