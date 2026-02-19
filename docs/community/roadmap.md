# Project Roadmap

This document outlines the development roadmap for PicoClaw. For the most up-to-date information, see [ROADMAP.md](../../ROADMAP.md) in the repository.

## Vision

To build the ultimate lightweight, secure, and fully autonomous AI Agent infrastructure - automate the mundane, unleash your creativity.

## Core Pillars

### 1. Core Optimization: Extreme Lightweight

Our defining characteristic. We fight software bloat to ensure PicoClaw runs smoothly on the smallest embedded devices.

**Goals:**

- Run smoothly on 64MB RAM embedded boards (e.g., low-end RISC-V SBCs)
- Core process consuming < 20MB RAM
- Startup time < 1 second

**Current Status:**

- Memory footprint significantly lower than alternatives
- Fast startup achieved
- Continuous optimization in progress

### 2. Security Hardening: Defense in Depth

Building a "Secure-by-Default" agent.

**Focus Areas:**

- **Input Defense & Permission Control**
  - Prompt injection defense
  - Tool abuse prevention
  - SSRF protection with built-in blocklists

- **Sandboxing & Isolation**
  - Filesystem sandbox for file operations
  - Context isolation between sessions
  - Privacy redaction for sensitive data

- **Authentication & Secrets**
  - Modern cryptography (ChaCha20-Poly1305)
  - OAuth 2.0 flows for providers

### 3. Connectivity: Protocol-First Architecture

Connect every model, reach every platform.

**Provider Support:**

- Architecture upgrade to protocol-based classification
- Local model integration (Ollama, vLLM, LM Studio)
- Continued support for frontier closed-source models

**Channel Support:**

- IM platforms: QQ, WeChat, DingTalk, Feishu, Telegram, Discord, WhatsApp, LINE, Slack, Email, and more
- OneBot protocol support
- Native attachment handling

**Skill Marketplace:**

- Skill discovery and installation
- Community skill sharing

### 4. Advanced Capabilities: From Chatbot to Agentic AI

Beyond conversation - focusing on action and collaboration.

**Operations:**

- MCP (Model Context Protocol) support
- Browser automation via CDP
- Mobile device control

**Multi-Agent Collaboration:**

- Basic multi-agent implementation
- Smart model routing
- Swarm mode for multi-instance collaboration
- AI-Native OS interaction paradigms

### 5. Developer Experience & Documentation

Lowering the barrier to entry so anyone can deploy in minutes.

**Goals:**

- Zero-config quick start with interactive CLI wizard
- Comprehensive platform guides (Windows, macOS, Linux, Android)
- Step-by-step tutorials for all features
- AI-assisted documentation generation

### 6. Engineering: AI-Powered Open Source

Using AI to accelerate development.

- AI-enhanced CI/CD (code review, linting, PR labeling)
- Bot noise reduction
- AI-powered issue triage

### 7. Brand & Community

**Logo Design:** We are looking for a Mantis Shrimp (Stomatopoda) logo design!

- Concept: "Small but Mighty" and "Lightning Fast Strikes"

## Release Timeline

### Short Term (Next 3 Months)

- [ ] Complete protocol-based provider architecture
- [ ] Enhanced documentation and tutorials
- [ ] Additional channel integrations
- [ ] Performance optimizations

### Medium Term (3-6 Months)

- [ ] MCP support implementation
- [ ] Multi-agent collaboration improvements
- [ ] Skill marketplace launch
- [ ] Mobile operation support

### Long Term (6-12 Months)

- [ ] Browser automation
- [ ] Swarm mode
- [ ] AI-Native OS exploration
- [ ] Expanded hardware support

## Contributing to the Roadmap

We welcome community contributions to any item on this roadmap!

### How to Contribute

1. **Comment on Issues**: Find relevant issues on [GitHub](https://github.com/sipeed/picoclaw/issues)
2. **Submit PRs**: Implement features and submit pull requests
3. **Provide Feedback**: Share your use cases and requirements
4. **Test Early Builds**: Help test new features before release

### Suggesting Features

1. Check existing issues first
2. Open a [GitHub Discussion](https://github.com/sipeed/picoclaw/discussions) for discussion
3. Provide detailed use cases
4. Be open to feedback and iteration

## Tracking Progress

- **GitHub Projects**: Track specific initiatives
- **Milestones**: See planned releases
- **Changelog**: View completed features

## Prioritization

Features are prioritized based on:

1. **Community demand**: User requests and feedback
2. **Strategic alignment**: Fit with vision and goals
3. **Resource availability**: Contributor capacity
4. **Dependencies**: Technical requirements

## Version History

| Version | Highlights |
|---------|------------|
| 1.x | Core functionality, multi-provider, multi-channel |
| Upcoming | Protocol architecture, enhanced docs, more integrations |

## Get Involved

Ready to help build the best Edge AI Agent?

- [View Open Issues](https://github.com/sipeed/picoclaw/issues)
- [Join Discord](https://discord.gg/V4sAZ9XWpN)
- [Contribute Code](../developer-guide/contributing.md)

---

*This roadmap is subject to change based on community feedback and technical requirements. Last updated: 2024.*
