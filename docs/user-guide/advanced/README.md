# Advanced Features

This section covers advanced configuration and features of PicoClaw for power users and complex deployment scenarios.

## Overview

PicoClaw provides several advanced features that enable sophisticated multi-agent architectures, robust model fallback chains, and fine-grained control over security and routing.

## Topics

### Multi-Agent System

- **[Multi-Agent Configuration](multi-agent.md)** - Configure multiple specialized agents with isolated workspaces and different models

### Routing and Sessions

- **[Message Routing](routing.md)** - Route messages from channels to specific agents using bindings
- **[Session Management](session-management.md)** - Handle conversation persistence, identity linking, and session scoping

### Reliability

- **[Model Fallbacks](model-fallbacks.md)** - Configure automatic fallback chains for high availability

### Security

- **[Security Sandbox](security-sandbox.md)** - Restrict file and command access to protect your system

### Configuration

- **[Environment Variables](environment-variables.md)** - Complete reference for all environment variables

### Voice Features

- **[Voice Transcription](voice-transcription.md)** - Enable voice message transcription using Groq Whisper

## Architecture Overview

PicoClaw follows a message bus architecture where:

1. **Channels** (Telegram, Discord, Slack, etc.) receive messages from users
2. **Routing** determines which agent handles each message based on bindings
3. **Agents** process messages using LLM providers and tools
4. **Sessions** maintain conversation state and history

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Telegram   │     │   Discord   │     │    Slack    │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       └───────────────────┼───────────────────┘
                           │
                    ┌──────▼──────┐
                    │ Message Bus │
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │   Routing   │
                    └──────┬──────┘
                           │
       ┌───────────────────┼───────────────────┐
       │                   │                   │
┌──────▼──────┐     ┌──────▼──────┐     ┌──────▼──────┐
│   Agent A   │     │   Agent B   │     │   Agent C   │
│  (Research) │     │  (Coding)   │     │ (Assistant) │
└─────────────┘     └─────────────┘     └─────────────┘
```

## Configuration Hierarchy

PicoClaw uses a hierarchical configuration system:

1. **Defaults** - Default values built into the application
2. **Config File** - Settings from `~/.picoclaw/config.json`
3. **Environment Variables** - Override config file settings
4. **Command-line Flags** - Override all other settings

## Quick Reference

| Feature | Config Section | Key Setting |
|---------|---------------|-------------|
| Multi-agent | `agents.list` | `id`, `workspace`, `model` |
| Routing | `bindings` | `agent_id`, `match` |
| Sessions | `session` | `dm_scope`, `identity_links` |
| Fallbacks | `agents.defaults` | `model_fallbacks` |
| Sandbox | `agents.defaults` | `restrict_to_workspace` |
| Voice | `providers.groq` | `api_key` |

## Related Documentation

- [Configuration File Reference](../../configuration/config-file.md)
- [CLI Reference](../cli-reference.md)
- [Workspace Management](../workspace/README.md)
