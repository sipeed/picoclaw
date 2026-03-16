# PicoClaw: Ultra-Efficient AI Assistant in Go

This repository is intended for development purposes only.

The main project is maintained at [sipeed/picoclaw](https://github.com/sipeed/picoclaw).

## Contributions

| PR | Change | Status |
|----|--------|--------|
| [#1460](https://github.com/sipeed/picoclaw/pull/1460) | fix(openai_compat): fix tool call serialization for strict OpenAI-compatible providers | Open |
| [#1479](https://github.com/sipeed/picoclaw/pull/1479) | fix(claude_cli): surface stdout in error when CLI exits non-zero | Open |
| [#1480](https://github.com/sipeed/picoclaw/pull/1480) | docs: document claude-cli and codex-cli providers in README | Open |
| [#1625](https://github.com/sipeed/picoclaw/pull/1625) | feat(channels): support multiple named Telegram bots | Open |
| [#1633](https://github.com/sipeed/picoclaw/pull/1633) | feat(providers): add gemini-cli provider | Open |
| [#1637](https://github.com/sipeed/picoclaw/pull/1637) | fix(agent): dispatch per-candidate provider in fallback chain | Open |

---

## Configuration Guide

Features in this fork that may not yet be merged upstream.

### CLI-based LLM Providers

PicoClaw supports three CLI-based providers that invoke local AI CLI tools as subprocesses.
All three read the prompt from stdin and return the response on stdout.

Add entries to `model_list` in your config:

```json
"model_list": [
    {
        "model_name": "claude-cli",
        "model": "claude-cli/claude-code",
        "request_timeout": 1200
    },
    {
        "model_name": "codex-cli",
        "model": "codex-cli/codex-cli",
        "request_timeout": 1200
    },
    {
        "model_name": "gemini-cli",
        "model": "gemini-cli/gemini-2.5-pro",
        "request_timeout": 1200
    }
]
```

The model ID after the `/` is passed as `--model` to the CLI. Sentinel values (`claude-code`, `codex-cli`, `gemini-cli`) skip the `--model` flag and let the CLI use its own default model.

| Provider | Protocol prefix | Sentinel | CLI invoked |
|----------|----------------|----------|-------------|
| Claude Code | `claude-cli/` | `claude-code` | `claude -p --output-format json --dangerously-skip-permissions --no-chrome` |
| OpenAI Codex | `codex-cli/` | `codex-cli` | `codex exec --json --dangerously-bypass-approvals-and-sandbox` |
| Gemini CLI | `gemini-cli/` | `gemini-cli` | `gemini --yolo --output-format json --prompt ""` |

**Prerequisites:** Each CLI must be installed and authenticated with `claude`, `codex`, or `gemini` available in PATH.

---

### Multiple Telegram Bots

Run multiple Telegram bots from a single picoclaw instance, each connected to a separate AI agent. Each bot entry in `telegram_bots` creates a channel named `telegram-<id>`. Use `bindings` to route each channel to its agent.

#### Config

```json
"channels": {
    "telegram_bots": [
        {
            "id": "alice",
            "enabled": true,
            "token": "ALICE_BOT_TOKEN",
            "allow_from": ["YOUR_TELEGRAM_USER_ID"],
            "typing": { "enabled": true },
            "placeholder": { "enabled": true, "text": "Thinking... 💭" }
        },
        {
            "id": "bob",
            "enabled": true,
            "token": "BOB_BOT_TOKEN",
            "allow_from": ["YOUR_TELEGRAM_USER_ID"],
            "typing": { "enabled": true },
            "placeholder": { "enabled": true, "text": "Thinking... 💭" }
        }
    ]
},
"bindings": [
    {
        "agent_id": "alice",
        "match": { "channel": "telegram-alice" }
    },
    {
        "agent_id": "bob",
        "match": { "channel": "telegram-bob" }
    }
]
```

#### Agents

Define each agent in `agents.list` with its own workspace, model, and personality files:

```json
"agents": {
    "list": [
        {
            "id": "alice",
            "name": "Alice",
            "default": true,
            "workspace": "~/.picoclaw/agents/alice",
            "model": {
                "primary": "claude-cli",
                "fallbacks": ["gemini-cli"]
            },
            "subagents": {
                "allow_agents": ["bob"]
            }
        },
        {
            "id": "bob",
            "name": "Bob",
            "default": false,
            "workspace": "~/.picoclaw/agents/bob",
            "model": {
                "primary": "claude-cli",
                "fallbacks": ["gemini-cli"]
            }
        }
    ]
}
```

Each agent's workspace can contain `IDENTITY.md` and `AGENTS.md` files to define personality and behaviour. Picoclaw creates the workspace directories automatically on first run.

**Note:** The legacy single-bot `channels.telegram` config is still supported and is automatically normalized to a `telegram-default` channel for backward compatibility.

---

## Legal

Please see LICENSE.md for copyright and other legal information.
