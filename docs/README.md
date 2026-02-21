# PicoClaw Documentation

Welcome to the PicoClaw documentation. PicoClaw is an ultra-lightweight AI assistant written in Go that runs on $10 hardware with less than 10MB RAM.

## Why PicoClaw?

| Feature | OpenClaw | NanoBot | PicoClaw |
|---------|----------|---------|----------|
| **Language** | TypeScript | Python | Go |
| **RAM** | >1GB | >100MB | **<10MB** |
| **Startup** | >500s | >30s | **<1s** |
| **Cost** | Mac Mini $599 | ~$50 | **As low as $10** |

## Quick Links

- [Installation](getting-started/installation.md) - Get PicoClaw up and running
- [Quick Start](getting-started/quick-start.md) - Your first conversation in 5 minutes
- [Configuration Reference](configuration/config-file.md) - Complete config options
- [CLI Reference](user-guide/cli-reference.md) - All command-line options
- [Troubleshooting](operations/troubleshooting.md) - Common issues and solutions

## Documentation Sections

### Getting Started

New to PicoClaw? Start here.

- [Installation](getting-started/installation.md) - Install via binary, source, or Docker
- [Quick Start](getting-started/quick-start.md) - 5-minute setup guide
- [Configuration Basics](getting-started/configuration-basics.md) - Essential config concepts
- [Your First Chat](getting-started/first-chat.md) - Running your first conversation

### User Guide

Complete guide to using PicoClaw features.

- [CLI Reference](user-guide/cli-reference.md) - All commands and options
  - [onboard](user-guide/cli/onboard.md) - Initialize configuration
  - [agent](user-guide/cli/agent.md) - Chat with the agent
  - [gateway](user-guide/cli/gateway.md) - Start the gateway
  - [status](user-guide/cli/status.md) - System status
  - [auth](user-guide/cli/auth.md) - OAuth management
  - [cron](user-guide/cli/cron.md) - Scheduled tasks
  - [skills](user-guide/cli/skills.md) - Skill management
  - [migrate](user-guide/cli/migrate.md) - Migration from OpenClaw

- [Channels](user-guide/channels/README.md) - Chat platform integrations
  - [Telegram](user-guide/channels/telegram.md) - Telegram bot setup
  - [Discord](user-guide/channels/discord.md) - Discord bot setup
  - [Slack](user-guide/channels/slack.md) - Slack integration
  - [WhatsApp](user-guide/channels/whatsapp.md) - WhatsApp integration
  - [Feishu/Lark](user-guide/channels/feishu-lark.md) - Feishu setup
  - [LINE](user-guide/channels/line.md) - LINE bot setup
  - [QQ](user-guide/channels/qq.md) - QQ bot setup
  - [DingTalk](user-guide/channels/dingtalk.md) - DingTalk setup
  - [OneBot](user-guide/channels/onebot.md) - OneBot protocol
  - [MaixCam](user-guide/channels/maixcam.md) - MaixCam device

- [Providers](user-guide/providers/README.md) - LLM provider setup
  - [OpenRouter](user-guide/providers/openrouter.md) - Multi-model access (recommended)
  - [Zhipu/GLM](user-guide/providers/zhipu.md) - Chinese AI models
  - [Anthropic](user-guide/providers/anthropic.md) - Claude models
  - [OpenAI](user-guide/providers/openai.md) - GPT models
  - [Gemini](user-guide/providers/gemini.md) - Google Gemini
  - [Groq](user-guide/providers/groq.md) - Fast inference
  - [DeepSeek](user-guide/providers/deepseek.md) - DeepSeek models
  - [Ollama](user-guide/providers/ollama.md) - Local models
  - [vLLM](user-guide/providers/vllm.md) - Self-hosted models

- [IDE Setup](user-guide/ide-setup/README.md) - IDE integrations
  - [Antigravity](user-guide/ide-setup/antigravity.md) - Google Cloud Code Assist (free Claude/Gemini)

- [Workspace](user-guide/workspace/README.md) - Customizing behavior
  - [Structure](user-guide/workspace/structure.md) - Directory layout
  - [AGENT.md](user-guide/workspace/agent-md.md) - Agent behavior
  - [IDENTITY.md](user-guide/workspace/identity-md.md) - Agent identity
  - [Memory](user-guide/workspace/memory.md) - Long-term memory
  - [Heartbeat Tasks](user-guide/workspace/heartbeat-tasks.md) - Periodic tasks

- [Tools](user-guide/tools/README.md) - Available tools
  - [File System](user-guide/tools/filesystem.md) - read_file, write_file, etc.
  - [Exec](user-guide/tools/exec.md) - Shell commands
  - [Web](user-guide/tools/web.md) - web_search, web_fetch
  - [Messaging](user-guide/tools/messaging.md) - Send messages
  - [Spawn](user-guide/tools/spawn.md) - Subagents
  - [Cron](user-guide/tools/cron.md) - Scheduled tasks
  - [Hardware](user-guide/tools/hardware.md) - I2C, SPI

- [Skills](user-guide/skills/README.md) - Skills system
  - [Using Skills](user-guide/skills/using-skills.md)
  - [Installing Skills](user-guide/skills/installing-skills.md)
  - [Built-in Skills](user-guide/skills/builtin-skills.md)
  - [Creating Skills](user-guide/skills/creating-skills.md)

- [Advanced Features](user-guide/advanced/README.md)
  - [Multi-Agent System](user-guide/advanced/multi-agent.md)
  - [Message Routing](user-guide/advanced/routing.md)
  - [Session Management](user-guide/advanced/session-management.md)
  - [Model Fallbacks](user-guide/advanced/model-fallbacks.md)
  - [Security Sandbox](user-guide/advanced/security-sandbox.md)
  - [Environment Variables](user-guide/advanced/environment-variables.md)
  - [Voice Transcription](user-guide/advanced/voice-transcription.md)

### Developer Guide

Building and extending PicoClaw.

- [Architecture](developer-guide/architecture.md) - System design
- [Data Flow](developer-guide/data-flow.md) - Message bus architecture
- [Building](developer-guide/building.md) - Build from source
- [Testing](developer-guide/testing.md) - Running tests
- [Contributing](developer-guide/contributing.md) - Contribution guidelines
- [Code Style](developer-guide/code-style.md) - Code standards

#### Extending PicoClaw

- [Creating Tools](developer-guide/extending/creating-tools.md)
- [Creating Providers](developer-guide/extending/creating-providers.md)
- [Creating Channels](developer-guide/extending/creating-channels.md)
- [Creating Skills](developer-guide/extending/creating-skills.md)
- [Antigravity Implementation](developer-guide/extending/antigravity-implementation.md) - OAuth and API details

#### API Reference

- [Tool Interface](developer-guide/api/tool-interface.md)
- [Provider Interface](developer-guide/api/provider-interface.md)
- [Message Bus](developer-guide/api/message-bus.md)
- [Session API](developer-guide/api/session-api.md)

### Deployment

Deploying PicoClaw in various environments.

- [Docker](deployment/docker.md) - Docker and Docker Compose
- [Systemd](deployment/systemd.md) - Linux service
- [Termux](deployment/termux.md) - Android phones
- [Single-Board Computers](deployment/sbc/README.md)
  - [Raspberry Pi](deployment/sbc/rpi.md)
  - [LicheeRV Nano](deployment/sbc/licheerv-nano.md)
  - [MaixCAM](deployment/sbc/maixcam.md)
- [Security](deployment/security.md) - Production security

### Operations

Running and monitoring PicoClaw.

- [Health Endpoints](operations/health-endpoints.md)
- [Monitoring](operations/monitoring.md)
- [Logging](operations/logging.md)
- [Device Monitoring](operations/device-monitoring.md)
- [Troubleshooting](operations/troubleshooting.md)

### Configuration

Complete configuration reference.

- [Config File](configuration/config-file.md) - config.json reference
- [Agents Config](configuration/agents-config.md)
- [Bindings Config](configuration/bindings-config.md)
- [Channels Config](configuration/channels-config.md)
- [Providers Config](configuration/providers-config.md)
- [Tools Config](configuration/tools-config.md)
- [Gateway Config](configuration/gateway-config.md)
- [Heartbeat Config](configuration/heartbeat-config.md)
- [Devices Config](configuration/devices-config.md)

### Tutorials

Step-by-step guides.

- [Basic Assistant](tutorials/basic-assistant.md) - Beginner setup
- [Telegram Bot](tutorials/telegram-bot.md) - Telegram tutorial
- [Scheduled Tasks](tutorials/scheduled-tasks.md) - Cron and heartbeat
- [Multi-Agent Setup](tutorials/multi-agent-setup.md) - Multiple agents
- [Hardware Control](tutorials/hardware-control.md) - I2C/SPI tutorial
- [Skill Development](tutorials/skill-development.md) - Custom skills

### Community

Join the community.

- [Roadmap](community/roadmap.md) - Project roadmap
- [Support](community/support.md) - Getting help
- [Contributing Roles](community/contributing-roles.md) - Volunteer opportunities

## Getting Help

- **Discord**: [Join our server](https://discord.gg/V4sAZ9XWpN)
- **GitHub Issues**: [Report bugs](https://github.com/sipeed/picoclaw/issues)
- **GitHub Discussions**: [Feature requests](https://github.com/sipeed/picoclaw/discussions)

## Contributing

PRs are welcome! See the [Contributing Guide](developer-guide/contributing.md) for details.
