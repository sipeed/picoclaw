# Lifecycle Hooks: Plugin-Style Examples

This guide shows how to extend PicoClaw behavior with `pkg/hooks` without modifying core agent logic.

Current model:
- "Plugin-style" means registering Go handlers at startup.
- Hooks are in-process (no dynamic `.so` loading).
- If no hooks are registered, the runtime follows the normal zero-cost path.

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
agentLoop.SetHooks(buildHooks()) // Must be called before Run()
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
