# PicoClaw Multi-Agent Architecture

This directory contains C4 model diagrams (rendered with Mermaid) documenting the multi-agent collaboration framework for PicoClaw.

## Reference Implementation

**OpenClaw (moltbot)** — the state-of-the-art personal AI gateway whose founder was hired by OpenAI. picoclaw ports and improves upon OpenClaw's validated patterns in a lightweight Go single-binary.

- Reference code: `/home/leeaandrob/Projects/Personal/llm/auto-agents/moltbot`
- See [PRP](../prp/multi-agent-hardening.md) for detailed implementation plan

## Documents

| Document | Scope | Description |
|----------|-------|-------------|
| [C1 - System Context](./c1-system-context.md) | Highest level | PicoClaw in its ecosystem: users, channels, LLM providers |
| [C2 - Container](./c2-container.md) | Runtime containers | Gateway, Agent Loop, Provider Layer, Channels |
| [C3 - Component](./c3-component-multi-agent.md) | Multi-agent internals | Current + planned components across 4 hardening phases |
| [C4 - Code](./c4-code-detail.md) | Key structs/interfaces | Go interfaces, data flow, tool execution |
| [Sequence Diagrams](./sequences.md) | Runtime flows | Handoff, Blackboard sync, Fallback chain |
| [Roadmap](./roadmap.md) | Phased plan | 4-phase hardening based on OpenClaw gap analysis |

## Related

- **PRP**: [Multi-Agent Hardening](../prp/multi-agent-hardening.md) — Full implementation plan with acceptance criteria
- **Issue**: [#294 - Base Multi-agent Collaboration Framework](https://github.com/sipeed/picoclaw/issues/294)
- **Issue**: [#283 - Refactor Provider Architecture](https://github.com/sipeed/picoclaw/issues/283)
- **Discussion**: [#122 - Provider Architecture Proposal](https://github.com/sipeed/picoclaw/discussions/122)

## Status

| Phase | Status | PR |
|-------|--------|----|
| Provider Protocol Refactor | Merged | [#213](https://github.com/sipeed/picoclaw/pull/213) |
| Model Fallback + Routing | Merged | [#131](https://github.com/sipeed/picoclaw/pull/131) |
| Blackboard + Handoff + Discovery | WIP | [#423](https://github.com/sipeed/picoclaw/pull/423) |
| Phase 1: Foundation Fix | Planned | PR #423 |
| Phase 2: Tool Policy | Planned | PR #423 |
| Phase 3: Resilience | Planned | TBD |
| Phase 4: Async Multi-Agent | Planned | TBD |
| SOUL.md Bootstrap | In Progress (other dev) | TBD |

## picoclaw Advantages Over OpenClaw

| Area | picoclaw | OpenClaw |
|------|----------|----------|
| Shared agent state | Blackboard (real-time) | None (announce-only) |
| Runtime | Go single binary | Node.js |
| Memory footprint | ~10x smaller | Node.js overhead |
| Deployment | Copy binary and run | npm install + config |
| Concurrency | goroutines (native) | async/await |
| Type safety | Compile-time | Runtime |
