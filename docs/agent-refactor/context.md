# Context

## What this document covers

This document makes explicit the boundaries of context management in the agent loop:

- what fills the context window and how space is divided
- what is stored in session history vs. built at request time
- when and how context compression happens
- how token budgets are estimated

These are existing concepts. This document clarifies their boundaries rather than introducing new ones.

---

## Context window regions

The context window is the model's total input capacity. Four regions fill it:

| Region | Assembled by | Stored in session? |
|---|---|---|
| System prompt | `BuildMessages()` — static + dynamic parts | No |
| Summary | `SetSummary()` stores it; `BuildMessages()` injects it | Separate from history |
| Session history | User / assistant / tool messages | Yes |
| Tool definitions | Provider adapter injects at call time | No |

`MaxTokens` (the output generation limit) must also be reserved from the total budget.

The available space for history is therefore:

```
history_budget = ContextWindow - system_prompt - summary - tool_definitions - MaxTokens
```

---

## ContextWindow vs MaxTokens

These serve different purposes:

- **MaxTokens** — maximum tokens the LLM may generate in one response. Sent as the `max_tokens` request parameter.
- **ContextWindow** — the model's total input context capacity.

These were previously set to the same value, which caused the summarization threshold to fire either far too early (at the default 32K) or not at all (when a user raised `max_tokens`).

Current default when not explicitly configured: `ContextWindow = MaxTokens * 4`.

---

## Session history

Session history stores only conversation messages:

- `user` — user input
- `assistant` — LLM response (may include `ToolCalls`)
- `tool` — tool execution results

Session history does **not** contain:

- System prompts — assembled at request time by `BuildMessages`
- Summary content — stored separately via `SetSummary`, injected by `BuildMessages`

This distinction matters: any code that operates on session history — compression, boundary detection, token estimation — must not assume a system message is present.

---

## Turn

A **Turn** is one complete cycle:

> user message -> LLM iterations (possibly including tool calls) -> final assistant response

This definition comes from the agent loop design (#1316). In session history, Turn boundaries are identified by `user`-role messages.

Turn is the atomic unit for compression. Cutting inside a Turn can orphan tool-call sequences — an assistant message with `ToolCalls` separated from its corresponding `tool` results. Compressing at Turn boundaries avoids this by construction.

`parseTurnBoundaries(history)` returns the starting index of each Turn.
`findSafeBoundary(history, targetIndex)` snaps a target cut point to the nearest Turn boundary.

---

## Compression paths

Three compression paths exist, in order of preference:

### 1. Async summarization

`maybeSummarize` runs after each Turn completes.

Triggers when message count exceeds a threshold, or when estimated history tokens exceed a percentage of `ContextWindow`. If triggered, a background goroutine calls the LLM to produce a summary of the oldest messages. The summary is stored via `SetSummary`; `BuildMessages` injects it into the system prompt on the next call.

Cut point uses `findSafeBoundary` so no Turn is split.

### 2. Proactive budget check

`isOverContextBudget` runs before each LLM call.

Uses the full budget formula: `message_tokens + tool_def_tokens + MaxTokens > ContextWindow`. If over budget, triggers `forceCompression` and rebuilds messages before calling the LLM.

This prevents wasted (and billed) LLM calls that would otherwise fail with a context-window error.

### 3. Emergency compression (reactive)

`forceCompression` runs when the LLM returns a context-window error despite the proactive check.

Drops the oldest ~50% of Turns. Stores a compression note in the session summary (not in history messages) so `BuildMessages` can include it in the next system prompt.

This is the fallback for when the token estimate undershoots reality.

---

## Token estimation

Estimation uses a heuristic of ~2.5 characters per token (`chars * 2 / 5`).

`estimateMessageTokens` counts:

- `Content` (rune count, for multibyte correctness)
- `ReasoningContent` (extended thinking / chain-of-thought)
- `ToolCalls` — ID, type, function name, arguments
- `ToolCallID` (tool result metadata)
- Per-message overhead (role label, JSON structure)
- `Media` items — flat per-item token estimate, added directly to the final count (not through the character heuristic, since actual cost depends on resolution and provider-specific image tokenization)

`estimateToolDefsTokens` counts tool definition overhead: name, description, JSON schema of parameters.

These are deliberately heuristic. The proactive check handles the common case; the reactive path catches estimation errors.

---

## Interface boundaries

Context budget functions (`parseTurnBoundaries`, `findSafeBoundary`, `estimateMessageTokens`, `isOverContextBudget`) are **pure functions**. They take `[]providers.Message` and integer parameters. They have no dependency on `AgentLoop` or any other runtime struct.

`BuildMessages` is the sole assembler of the final message array sent to the LLM. Budget functions inform compression decisions but do not construct messages.

`forceCompression` and `summarizeSession` mutate session state (history and summary). `BuildMessages` reads that state to construct context. The flow is:

```
budget check --> compression decision --> mutate session --> BuildMessages reads session --> LLM call
```

---

## Known gaps

These are recognized limitations in the current implementation, documented here for visibility:

- **Summarization trigger does not use the full budget formula.** `maybeSummarize` compares estimated history tokens against a percentage of `ContextWindow`. It does not account for system prompt size, tool definition overhead, or `MaxTokens` reserve. The proactive check covers the critical path (preventing 400 errors), but the summarization trigger could be aligned with the same budget model for more accurate early compression.

- **Token estimation is heuristic.** It does not account for provider-specific tokenization, exact system prompt size (assembled separately), or variable image token costs. The two-path design (proactive + reactive) is intended to tolerate this imprecision.

- **Reactive retry does not preserve media.** When the reactive path rebuilds context after compression, it currently passes empty values for media references. This is a pre-existing issue in the main loop, not introduced by the budget system.

---

## What this document does not cover

- How `AGENT.md` frontmatter configures context parameters — that is part of the Agent definition work
- How the context builder assembles context in the new architecture — that is upcoming work
- How compression events surface through the event system — that is part of the event model (#1316)
- Subagent context isolation — that is a separate track
