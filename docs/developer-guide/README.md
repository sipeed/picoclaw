# PicoClaw Developer Guide

Welcome to the PicoClaw Developer Guide. This documentation is intended for developers who want to understand, extend, or contribute to PicoClaw.

## Overview

PicoClaw is an ultra-lightweight AI assistant written in Go. It follows a message bus architecture where channels (Telegram, Discord, etc.) publish inbound messages and subscribe to outbound responses.

## Documentation Sections

### Getting Started

- [Building from Source](building.md) - How to build PicoClaw from source code
- [Running Tests](testing.md) - How to run the test suite
- [Code Style Guide](code-style.md) - Coding conventions and style guidelines

### Architecture

- [System Architecture](architecture.md) - Overview of PicoClaw's architecture and components
- [Data Flow](data-flow.md) - Understanding the message bus and data flow

### Contributing

- [Contribution Guidelines](contributing.md) - How to contribute to PicoClaw

### Extending PicoClaw

The [extending/](extending/) directory contains guides for extending PicoClaw:

- [Creating Custom Tools](extending/creating-tools.md) - How to create custom tools
- [Creating LLM Providers](extending/creating-providers.md) - How to create new LLM provider implementations
- [Creating Channel Integrations](extending/creating-channels.md) - How to create new channel integrations
- [Creating Skills](extending/creating-skills.md) - How to create custom skills

### API Reference

The [api/](api/) directory contains API reference documentation:

- [Tool Interface](api/tool-interface.md) - Tool interface reference
- [Provider Interface](api/provider-interface.md) - LLMProvider interface reference
- [Message Bus API](api/message-bus.md) - Message bus API reference
- [Session API](api/session-api.md) - Session manager API reference

## Quick Start for Developers

```bash
# Clone the repository
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# Install dependencies
make deps

# Build
make build

# Run tests
make test

# Run the agent
./build/picoclaw agent -m "Hello, PicoClaw!"
```

## Key Concepts

### Message Bus Architecture

PicoClaw uses a message bus pattern for communication between components:

1. **Channels** receive messages from external platforms (Telegram, Discord, etc.)
2. Messages are published to the **inbound** bus
3. The **AgentLoop** consumes messages from the bus
4. Responses are published to the **outbound** bus
5. Channels subscribe to outbound messages and send them to their platforms

### Tool System

Tools are the primary way PicoClaw interacts with the world. Each tool:

- Implements the `Tool` interface with `Name()`, `Description()`, `Parameters()`, and `Execute()` methods
- Can optionally implement `ContextualTool` for channel context
- Can optionally implement `AsyncTool` for asynchronous operations

### LLM Provider System

Providers handle communication with LLM APIs. Each provider:

- Implements the `LLMProvider` interface with `Chat()` and `GetDefaultModel()` methods
- Handles request/response transformation for specific APIs
- Supports tool calling for agentic behavior

### Session Management

Sessions track conversation history:

- Each session has a unique key (typically "channel:chatID")
- Messages are stored in memory and persisted to disk
- Automatic summarization when history exceeds thresholds

## Project Structure

```
picoclaw/
├── cmd/picoclaw/          # CLI entry point
├── pkg/
│   ├── agent/             # Core agent logic
│   ├── bus/               # Message bus implementation
│   ├── channels/          # Platform integrations
│   ├── providers/         # LLM provider implementations
│   ├── tools/             # Tool implementations
│   ├── session/           # Session management
│   ├── config/            # Configuration handling
│   ├── skills/            # Skill loading system
│   └── ...                # Other packages
├── docs/                  # Documentation
└── Makefile               # Build commands
```

## Getting Help

- GitHub Issues: https://github.com/sipeed/picoclaw/issues
- Source Code: https://github.com/sipeed/picoclaw

## License

PicoClaw is licensed under the MIT License.
