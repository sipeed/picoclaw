# PicoClaw Multi-Agent Architecture

This directory contains C4 model diagrams (rendered with Mermaid) documenting the multi-agent collaboration framework for PicoClaw.

## Documents

| Document | Scope | Description |
|----------|-------|-------------|
| [C1 - System Context](./c1-system-context.md) | Highest level | PicoClaw in its ecosystem: users, channels, LLM providers |
| [C2 - Container](./c2-container.md) | Runtime containers | Gateway, Agent Loop, Provider Layer, Channels |
| [C3 - Component](./c3-component-multi-agent.md) | Multi-agent internals | Blackboard, Handoff, Routing, Registry, Fallback |
| [C4 - Code](./c4-code-detail.md) | Key structs/interfaces | Go interfaces, data flow, tool execution |
| [Sequence Diagrams](./sequences.md) | Runtime flows | Handoff, Blackboard sync, Fallback chain |
| [Roadmap](./roadmap.md) | Phased plan | What's done, what's next, dependencies |

## Related Issues

- [#294 - Base Multi-agent Collaboration Framework & Shared Context](https://github.com/sipeed/picoclaw/issues/294)
- [#283 - Refactor Provider Architecture: By Protocol Instead of By Vendor](https://github.com/sipeed/picoclaw/issues/283)
- [Discussion #122 - Provider Architecture Proposal](https://github.com/sipeed/picoclaw/discussions/122)

## Status

| Phase | Status | PR |
|-------|--------|----|
| Provider Protocol Refactor | Merged | [#213](https://github.com/sipeed/picoclaw/pull/213) |
| Model Fallback + Multi-agent Routing | Merged | [#131](https://github.com/sipeed/picoclaw/pull/131) |
| Multi-agent Collaboration Framework | WIP | [#423](https://github.com/sipeed/picoclaw/pull/423) |
