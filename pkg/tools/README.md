# PicoClaw Tool Development Notes

This document is a short reference for implementing tools in `pkg/tools/`.

## Core Interfaces

- `Tool`: the base synchronous tool contract. Every tool must implement `Name`, `Description`, `Parameters`, and `Execute`.
- `AsyncExecutor`: for tools that start background work and report completion later through a callback.
- `SequentialTool`: for tools that must preserve model order within a single LLM turn.
- `AdvancedMessageManager`: for tools that need synchronous send/edit callbacks from the channel manager.

## Execution Model

Tool calls from one assistant response run in parallel by default.

If a tool mutates shared state, or if one call can depend on a previous call from the same assistant message, implement `SequentialTool` and return `true` from `ExecuteSequentially()`.

Example: `tasktool` uses `SequentialTool` because `create_plan` and `update_task` can appear in the same model turn and must execute in order.

The sequential contract is honored by both the main agent loop and `RunToolLoop`, so subagent-style tool execution follows the same ordering rule.

## Context Helpers

`ExecuteWithContext` injects request-scoped metadata into the tool context:

- `ToolChannel(ctx)`
- `ToolChatID(ctx)`
- `ToolSessionKey(ctx)`

Use these helpers instead of storing per-request mutable state on tool instances.

## Guidance

- Prefer stateless tool instances. Shared mutable fields make parallel execution harder to reason about.
- Use `AsyncExecutor` only for genuinely background work. Most tools should stay synchronous.
- Use `AdvancedMessageManager` only when a tool must directly manage a user-visible message over time.
- If a tool needs ordered execution only for some calls, keep the implementation simple and still return `true`; correctness matters more than maximizing intra-turn parallelism.
