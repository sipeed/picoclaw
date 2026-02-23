# Lifecycle Hooks: Plugin-Style Examples

This guide shows how to extend PicoClaw behavior with `pkg/hooks` without modifying core agent logic.

For future direction (beyond current hooks foundation), see [Plugin System Roadmap](plugin-system-roadmap.md).

Current model:
- "Plugin-style" means registering Go handlers at startup.
- Hooks are in-process (no dynamic `.so` loading).
- If no hooks are registered, the runtime follows the normal zero-cost path.

## How Plugin Works

PicoClaw's plugin model is a startup-time hook registry:

1. Build a registry (`hooks.NewHookRegistry()`).
2. Register one or more handlers per lifecycle hook with priority.
3. Attach once with `agentLoop.SetHooks(registry)` before `agentLoop.Run(...)` (check error).
4. Agent loop triggers hook handlers at specific lifecycle points.

Execution semantics:

- Observe-only hooks (`message_received`, `after_tool_call`, `llm_input`, `llm_output`, `session_start`, `session_end`)
  - run concurrently
  - cannot block core behavior
- Modifying hooks (`message_sending`, `before_tool_call`)
  - run sequentially by priority (lower number first)
  - may mutate event data
  - may cancel operation via `Cancel=true`

Safety model:

- Panic in one handler is recovered and logged.
- Handler errors are logged; pipeline continues unless canceled by event flag.
- With no registered hooks, agent loop behavior is unchanged.

Lifecycle map:

```text
Inbound message
  -> message_received
  -> session_start
  -> llm_input
  -> llm_output
  -> before_tool_call (cancelable)
  -> tool execute
  -> after_tool_call
  -> message_sending (cancelable)
  -> outbound publish
  -> session_end
```

Note: the map above is shown as a single pass for readability. In practice, the
agent loop may iterate up to `MaxToolIterations`, and `llm_input`, `llm_output`,
`before_tool_call`, and `after_tool_call` can fire multiple times.

## Available Hooks

| Hook | Type | Typical use |
|---|---|---|
| `message_received` | observe-only | inbound telemetry |
| `message_sending` | modifying + cancel | content filtering, safety policy |
| `before_tool_call` | modifying + cancel | tool allow/deny, arg rewriting |
| `after_tool_call` | observe-only | latency/error metrics |
| `llm_input` | observe-only | prompt size monitoring |
| `llm_output` | observe-only | response/tool-call telemetry |
| `session_start` | observe-only | session audit |
| `session_end` | observe-only | session cleanup metrics |

## Quick Start

```go
package main

import (
	"context"
	"strings"

	"github.com/sipeed/picoclaw/pkg/hooks"
)

func buildHooks() *hooks.HookRegistry {
	reg := hooks.NewHookRegistry()

	// 1) Guardrail: block shell tool globally.
	reg.OnBeforeToolCall("block-shell", 100, func(_ context.Context, e *hooks.BeforeToolCallEvent) error {
		if e.ToolName == "shell" {
			e.Cancel = true
			e.CancelReason = "shell tool is disabled by local policy"
		}
		return nil
	})

	// 2) Outbound filter: redact obvious API key patterns.
	reg.OnMessageSending("redact-secrets", 50, func(_ context.Context, e *hooks.MessageSendingEvent) error {
		e.Content = strings.ReplaceAll(e.Content, "sk-", "[redacted]-")
		return nil
	})

	// 3) Telemetry: record tool latency or errors.
	reg.OnAfterToolCall("tool-telemetry", 0, func(_ context.Context, e *hooks.AfterToolCallEvent) error {
		// Send to metrics backend / logs as needed.
		_ = e.ToolName
		_ = e.Duration
		_ = e.Result
		return nil
	})

	return reg
}
```

Attach once during startup:

```go
agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)
if err := agentLoop.SetHooks(buildHooks()); err != nil {
	panic(err) // replace with your startup error handling
}
```

## Priority and Cancellation

- Lower `priority` runs first.
- `message_sending` and `before_tool_call` are sequential and can cancel.
- Other hooks are observe-only and run concurrently.

Recommended ordering:
- `0-49`: telemetry and logging
- `50-89`: transforms (redaction, normalization)
- `90+`: hard guardrails (block/cancel)

## Safety Notes

- Hook panics are recovered internally; one bad hook does not crash the loop.
- Hook errors are logged and execution continues unless `Cancel` is set.
- Keep hook handlers fast and non-blocking to avoid latency impact.
