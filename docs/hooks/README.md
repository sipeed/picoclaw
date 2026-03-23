# Hook System Guide

This document describes the hook system that is implemented in the current repository, not the older design draft.

The current implementation supports two mounting modes:

1. In-process hooks
2. Out-of-process process hooks (`JSON-RPC over stdio`)

The repository no longer ships standalone example source files. The Go and Python examples below are embedded directly in this document. If you want to use them, copy them into your own local files first.

## Supported Hook Types

| Type | Interface | Stage | Can modify data |
| --- | --- | --- | --- |
| Observer | `EventObserver` | EventBus broadcast | No |
| LLM interceptor | `LLMInterceptor` | `before_llm` / `after_llm` | Yes |
| Tool interceptor | `ToolInterceptor` | `before_tool` / `after_tool` | Yes |
| Tool approver | `ToolApprover` | `approve_tool` | No, returns allow/deny |

The currently exposed synchronous hook points are:

- `before_llm`
- `after_llm`
- `before_tool`
- `after_tool`
- `approve_tool`

Everything else is exposed as read-only events.

## Execution Order

`HookManager` sorts hooks like this:

1. In-process hooks first
2. Process hooks second
3. Lower `priority` first within the same source
4. Name order as the final tie-breaker

## Timeouts

Global defaults live under `hooks.defaults`:

- `observer_timeout_ms`
- `interceptor_timeout_ms`
- `approval_timeout_ms`

Note: the current implementation does not support per-process-hook `timeout_ms`. Timeouts are global defaults.

## Quick Start

If your first goal is simply to prove that the hook flow works and observe real requests, the easiest path is the Python process-hook example below:

1. Enable `hooks.enabled`
2. Save the Python example from this document to a local file, for example `/tmp/review_gate.py`
3. Set `PICOCLAW_HOOK_LOG_FILE`
4. Restart the gateway
5. Watch the log file with `tail -f`

Example:

```json
{
  "hooks": {
    "enabled": true,
    "processes": {
      "py_review_gate": {
        "enabled": true,
        "priority": 100,
        "transport": "stdio",
        "command": [
          "python3",
          "/tmp/review_gate.py"
        ],
        "observe": [
          "tool_exec_start",
          "tool_exec_end",
          "tool_exec_skipped"
        ],
        "intercept": [
          "before_tool",
          "approve_tool"
        ],
        "env": {
          "PICOCLAW_HOOK_LOG_FILE": "/tmp/picoclaw-hook-review-gate.log"
        }
      }
    }
  }
}
```

Watch it with:

```bash
tail -f /tmp/picoclaw-hook-review-gate.log
```

If you are developing PicoClaw itself rather than only validating the protocol, continue with the Go in-process example as well.

## What The Two Examples Are For

- Go in-process example
  Best for validating the host-side hook chain and understanding `MountHook()` plus the synchronous stages
- Python process example
  Best for understanding the `JSON-RPC over stdio` protocol and verifying the message flow between PicoClaw and an external process

Both examples are intentionally safe: they only log, never rewrite, and never deny.

## Go In-Process Example

The following is a minimal logging hook for in-process use. It implements:

1. `EventObserver`
2. `LLMInterceptor`
3. `ToolInterceptor`
4. `ToolApprover`

It only records activity. It does not rewrite requests or reject tools.

You can save it as your own Go file, for example `pkg/myhooks/example_logger.go`:

```go
package myhooks

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type ExampleLoggerHookOptions struct {
	LogFile   string `json:"log_file,omitempty"`
	LogEvents bool   `json:"log_events,omitempty"`
}

type ExampleLoggerHook struct {
	logFile   string
	logEvents bool
	mu        sync.Mutex
}

func NewExampleLoggerHook(opts ExampleLoggerHookOptions) *ExampleLoggerHook {
	return &ExampleLoggerHook{
		logFile:   strings.TrimSpace(opts.LogFile),
		logEvents: opts.LogEvents,
	}
}

func (h *ExampleLoggerHook) OnEvent(ctx context.Context, evt agent.Event) error {
	_ = ctx
	if h == nil || !h.logEvents {
		return nil
	}
	h.record("event", evt.Meta, map[string]any{
		"event":   evt.Kind.String(),
		"payload": evt.Payload,
	}, nil)
	return nil
}

func (h *ExampleLoggerHook) BeforeLLM(
	ctx context.Context,
	req *agent.LLMHookRequest,
) (*agent.LLMHookRequest, agent.HookDecision, error) {
	_ = ctx
	h.record("before_llm", req.Meta, req, agent.HookDecision{Action: agent.HookActionContinue})
	return req, agent.HookDecision{Action: agent.HookActionContinue}, nil
}

func (h *ExampleLoggerHook) AfterLLM(
	ctx context.Context,
	resp *agent.LLMHookResponse,
) (*agent.LLMHookResponse, agent.HookDecision, error) {
	_ = ctx
	h.record("after_llm", resp.Meta, resp, agent.HookDecision{Action: agent.HookActionContinue})
	return resp, agent.HookDecision{Action: agent.HookActionContinue}, nil
}

func (h *ExampleLoggerHook) BeforeTool(
	ctx context.Context,
	call *agent.ToolCallHookRequest,
) (*agent.ToolCallHookRequest, agent.HookDecision, error) {
	_ = ctx
	h.record("before_tool", call.Meta, call, agent.HookDecision{Action: agent.HookActionContinue})
	return call, agent.HookDecision{Action: agent.HookActionContinue}, nil
}

func (h *ExampleLoggerHook) AfterTool(
	ctx context.Context,
	result *agent.ToolResultHookResponse,
) (*agent.ToolResultHookResponse, agent.HookDecision, error) {
	_ = ctx
	h.record("after_tool", result.Meta, result, agent.HookDecision{Action: agent.HookActionContinue})
	return result, agent.HookDecision{Action: agent.HookActionContinue}, nil
}

func (h *ExampleLoggerHook) ApproveTool(
	ctx context.Context,
	req *agent.ToolApprovalRequest,
) (agent.ApprovalDecision, error) {
	_ = ctx
	decision := agent.ApprovalDecision{Approved: true}
	h.record("approve_tool", req.Meta, req, decision)
	return decision, nil
}

func (h *ExampleLoggerHook) record(stage string, meta agent.EventMeta, payload any, decision any) {
	logger.InfoCF("hooks", "Example hook observed", map[string]any{
		"stage": stage,
	})
	if h == nil || h.logFile == "" {
		return
	}

	entry := map[string]any{
		"ts":       time.Now().UTC(),
		"stage":    stage,
		"meta":     meta,
		"payload":  payload,
		"decision": decision,
	}

	body, err := json.Marshal(entry)
	if err != nil {
		logger.WarnCF("hooks", "Example hook log encode failed", map[string]any{
			"stage": stage,
			"error": err.Error(),
		})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if dir := filepath.Dir(h.logFile); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			logger.WarnCF("hooks", "Example hook log mkdir failed", map[string]any{
				"stage": stage,
				"path":  h.logFile,
				"error": err.Error(),
			})
			return
		}
	}

	file, err := os.OpenFile(h.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		logger.WarnCF("hooks", "Example hook log open failed", map[string]any{
			"stage": stage,
			"path":  h.logFile,
			"error": err.Error(),
		})
		return
	}
	defer func() { _ = file.Close() }()

	if _, err := file.Write(append(body, '\n')); err != nil {
		logger.WarnCF("hooks", "Example hook log write failed", map[string]any{
			"stage": stage,
			"path":  h.logFile,
			"error": err.Error(),
		})
	}
}
```

### Mounting It In Code

If code mounting is enough, call this after `AgentLoop` is initialized:

```go
hook := myhooks.NewExampleLoggerHook(myhooks.ExampleLoggerHookOptions{
    LogFile:   "/tmp/picoclaw-hook-example-logger.log",
    LogEvents: true,
})

if err := al.MountHook(agent.NamedHook("example-logger", hook)); err != nil {
    panic(err)
}
```

### If You Also Want Config Mounting

The hook system supports builtin hooks, but that requires you to compile the factory into your binary. In practice, that means you need registration code like this alongside the hook definition above:

```go
package myhooks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	if err := agent.RegisterBuiltinHook("example_logger", func(
		ctx context.Context,
		spec config.BuiltinHookConfig,
	) (any, error) {
		_ = ctx

		var opts ExampleLoggerHookOptions
		if len(spec.Config) > 0 {
			if err := json.Unmarshal(spec.Config, &opts); err != nil {
				return nil, fmt.Errorf("decode example_logger config: %w", err)
			}
		}
		return NewExampleLoggerHook(opts), nil
	}); err != nil {
		panic(err)
	}
}
```

Only after you register that builtin will the following config work:

```json
{
  "hooks": {
    "enabled": true,
    "builtins": {
      "example_logger": {
        "enabled": true,
        "priority": 10,
        "config": {
          "log_file": "/tmp/picoclaw-hook-example-logger.log",
          "log_events": true
        }
      }
    }
  }
}
```

### How To Observe It

- If `log_file` is set, each hook call is appended as JSON Lines
- If `log_file` is not set, the hook still writes summaries to the gateway log
- Requests that only hit the LLM path usually show `before_llm` and `after_llm`
- Requests that trigger tools usually also show `before_tool`, `approve_tool`, and `after_tool`
- If `log_events=true`, you will also see `event`

Typical log lines:

```json
{"ts":"2026-03-21T14:10:00Z","stage":"before_tool","meta":{"session_key":"session-1"},"payload":{"tool":"echo_text","arguments":{"text":"hello"}},"decision":{"action":"continue"}}
{"ts":"2026-03-21T14:10:00Z","stage":"approve_tool","meta":{"session_key":"session-1"},"payload":{"tool":"echo_text","arguments":{"text":"hello"}},"decision":{"approved":true}}
```

If you only see `before_llm` and `after_llm`, that usually means the request did not trigger any tool call, not that the hook failed to mount.

## Python Process-Hook Example

The following script is a minimal process-hook example. It uses only the Python standard library and supports:

1. `hook.hello`
2. `hook.event`
3. `hook.before_tool`
4. `hook.approve_tool`

It only records activity. It does not rewrite or deny anything.

Save it to any local path, for example `/tmp/review_gate.py`:

```python
#!/usr/bin/env python3
from __future__ import annotations

import json
import os
import signal
import sys
from datetime import datetime, timezone
from typing import Any

LOG_EVENTS = os.getenv("PICOCLAW_HOOK_LOG_EVENTS", "1").lower() not in {"0", "false", "no"}
LOG_FILE = os.getenv("PICOCLAW_HOOK_LOG_FILE", "").strip()


def append_log(entry: dict[str, Any]) -> None:
    if not LOG_FILE:
        return

    payload = {
        "ts": datetime.now(timezone.utc).isoformat(),
        **entry,
    }
    try:
        log_dir = os.path.dirname(LOG_FILE)
        if log_dir:
            os.makedirs(log_dir, exist_ok=True)
        with open(LOG_FILE, "a", encoding="utf-8") as handle:
            handle.write(json.dumps(payload, ensure_ascii=True) + "\n")
    except OSError as exc:
        log_stderr(f"failed to write hook log file {LOG_FILE}: {exc}")


def send_response(message_id: int, result: Any | None = None, error: str | None = None) -> None:
    payload: dict[str, Any] = {
        "jsonrpc": "2.0",
        "id": message_id,
    }
    if error is not None:
        payload["error"] = {"code": -32000, "message": error}
    else:
        payload["result"] = result if result is not None else {}

    append_log({
        "direction": "out",
        "id": message_id,
        "response": payload.get("result"),
        "error": payload.get("error"),
    })

    try:
        sys.stdout.write(json.dumps(payload, ensure_ascii=True) + "\n")
        sys.stdout.flush()
    except BrokenPipeError:
        raise SystemExit(0) from None


def log_stderr(message: str) -> None:
    try:
        sys.stderr.write(message + "\n")
        sys.stderr.flush()
    except BrokenPipeError:
        raise SystemExit(0) from None


def handle_shutdown_signal(signum: int, _frame: Any) -> None:
    raise KeyboardInterrupt(f"received signal {signum}")


def handle_before_tool(params: dict[str, Any]) -> dict[str, Any]:
    _ = params
    return {"action": "continue"}


def handle_approve_tool(params: dict[str, Any]) -> dict[str, Any]:
    _ = params
    return {"approved": True}


def handle_request(method: str, params: dict[str, Any]) -> dict[str, Any]:
    if method == "hook.hello":
        return {"ok": True, "name": "python-review-gate"}
    if method == "hook.before_tool":
        return handle_before_tool(params)
    if method == "hook.approve_tool":
        return handle_approve_tool(params)
    if method == "hook.before_llm":
        return {"action": "continue"}
    if method == "hook.after_llm":
        return {"action": "continue"}
    if method == "hook.after_tool":
        return {"action": "continue"}
    raise KeyError(f"method not found: {method}")


def main() -> int:
    try:
        for raw_line in sys.stdin:
            line = raw_line.strip()
            if not line:
                continue

            try:
                message = json.loads(line)
            except json.JSONDecodeError as exc:
                log_stderr(f"failed to decode request: {exc}")
                append_log({
                    "direction": "in",
                    "decode_error": str(exc),
                    "raw": line,
                })
                continue

            method = message.get("method")
            message_id = message.get("id", 0)
            params = message.get("params") or {}
            if not isinstance(params, dict):
                params = {}

            append_log({
                "direction": "in",
                "id": message_id,
                "method": method,
                "params": params,
                "notification": not bool(message_id),
            })

            if not message_id:
                if method == "hook.event" and LOG_EVENTS:
                    log_stderr(f"observed event: {params.get('Kind')}")
                continue

            try:
                result = handle_request(str(method or ""), params)
            except KeyError as exc:
                send_response(int(message_id), error=str(exc))
                continue
            except Exception as exc:
                send_response(int(message_id), error=f"unexpected error: {exc}")
                continue

            send_response(int(message_id), result=result)
    except KeyboardInterrupt:
        return 0

    return 0


if __name__ == "__main__":
    signal.signal(signal.SIGINT, handle_shutdown_signal)
    signal.signal(signal.SIGTERM, handle_shutdown_signal)
    raise SystemExit(main())
```

### Configuration

```json
{
  "hooks": {
    "enabled": true,
    "processes": {
      "py_review_gate": {
        "enabled": true,
        "priority": 100,
        "transport": "stdio",
        "command": [
          "python3",
          "/abs/path/to/review_gate.py"
        ],
        "observe": [
          "tool_exec_start",
          "tool_exec_end",
          "tool_exec_skipped"
        ],
        "intercept": [
          "before_tool",
          "approve_tool"
        ],
        "env": {
          "PICOCLAW_HOOK_LOG_FILE": "/tmp/picoclaw-hook-review-gate.log"
        }
      }
    }
  }
}
```

### Environment Variables

- `PICOCLAW_HOOK_LOG_EVENTS`
  Whether to write `hook.event` summaries to `stderr`, enabled by default
- `PICOCLAW_HOOK_LOG_FILE`
  Path to an external log file. When set, the script appends inbound hook requests, notifications, and outbound responses as JSON Lines

Note: `PICOCLAW_HOOK_LOG_FILE` has no default. If you do not set it, the script does not write any file logs.

### How To Confirm It Received Hooks

Watch two places:

- Gateway logs
  Useful for confirming that the host successfully started the process and for seeing event summaries written to `stderr`
- `PICOCLAW_HOOK_LOG_FILE`
  Useful for seeing the exact requests the script received and the exact responses it returned

Typical interpretation:

- Only `hook.hello`
  The process started and completed the handshake, but no business hook request has arrived yet
- `hook.event`
  The `observe` configuration is working
- `hook.before_tool`
  The `intercept: ["before_tool", ...]` configuration is working
- `hook.approve_tool`
  The approval hook path is working

Because this example never rewrites or denies, the expected responses look like:

```json
{"direction":"out","id":7,"response":{"action":"continue"},"error":null}
{"direction":"out","id":8,"response":{"approved":true},"error":null}
```

A complete sample:

```json
{"ts":"2026-03-21T14:12:00+00:00","direction":"in","id":1,"method":"hook.hello","params":{"name":"py_review_gate","version":1,"modes":["observe","tool","approve"]},"notification":false}
{"ts":"2026-03-21T14:12:00+00:00","direction":"out","id":1,"response":{"ok":true,"name":"python-review-gate"},"error":null}
{"ts":"2026-03-21T14:12:05+00:00","direction":"in","id":0,"method":"hook.event","params":{"Kind":"tool_exec_start"},"notification":true}
{"ts":"2026-03-21T14:12:05+00:00","direction":"in","id":7,"method":"hook.before_tool","params":{"tool":"echo_text","arguments":{"text":"hello"}},"notification":false}
{"ts":"2026-03-21T14:12:05+00:00","direction":"out","id":7,"response":{"action":"continue"},"error":null}
```

Additional notes:

- Timestamps are UTC
- `notification=true` means it was a notification such as `hook.event`, which does not expect a response
- `id` increases within a single hook process; if the process restarts, the counter starts over

## Process-Hook Protocol

Current process hooks use `JSON-RPC over stdio`:

- PicoClaw starts the external process
- Requests and responses are exchanged as one JSON message per line
- `hook.event` is a notification and does not need a response
- `hook.before_llm`, `hook.after_llm`, `hook.before_tool`, `hook.after_tool`, and `hook.approve_tool` are request/response calls

The host does not currently accept new RPCs initiated by the process hook. In practice, that means an external hook can only respond to PicoClaw calls; it cannot call back into the host to send channel messages.

## Configuration Fields

### `hooks.builtins.<name>`

- `enabled`
- `priority`
- `config`

### `hooks.processes.<name>`

- `enabled`
- `priority`
- `transport`
  Currently only `stdio` is supported
- `command`
- `dir`
- `env`
- `observe`
- `intercept`

## Troubleshooting

If a hook looks like it is not firing, check these in order:

1. `hooks.enabled`
2. Whether the target builtin or process hook is `enabled`
3. Whether the process-hook `command` path is correct
4. Whether you are watching the correct log file
5. Whether the current request actually reached the stage you care about
6. Whether `observe` or `intercept` contains the hook point you want

A practical minimal troubleshooting pair is:

- Use the Python process-hook example from this document to validate the external protocol
- Use the Go in-process example from this document to validate the host-side chain

If the Python side shows `hook.hello` but no business hook requests, the protocol is usually fine; the current request simply did not trigger the stage you expected.

## Scope And Limits

The current hook system is best suited for:

- LLM request rewriting
- Tool argument normalization
- Pre-execution tool approval
- Auditing and observability

It is not yet well suited for:

- External hooks actively sending channel messages
- Suspending a turn and waiting for human approval replies
- Full inbound/outbound message interception across the whole platform

If you want a real human approval workflow, use hooks as the approval entry point and keep the state machine plus channel interaction in a separate `ApprovalManager`.
