# Hook 系统使用说明

这份文档对应当前仓库里已经实现的 hook 系统，而不是设计草案。

当前实现支持两类挂载方式：

1. 进程内 hook
2. 进程外 process hook（`JSON-RPC over stdio`）

当前仓库不再内置示例代码文件。下面的 Go / Python 示例都直接写在本文档里；如果你要使用它们，需要先复制到你自己的文件路径。

## 支持的 hook 类型

| 类型 | 接口 | 作用阶段 | 能否改写 |
| --- | --- | --- | --- |
| 观察型 | `EventObserver` | EventBus 广播事件时 | 否 |
| LLM 拦截型 | `LLMInterceptor` | `before_llm` / `after_llm` | 是 |
| Tool 拦截型 | `ToolInterceptor` | `before_tool` / `after_tool` | 是 |
| Tool 审批型 | `ToolApprover` | `approve_tool` | 否，返回批准/拒绝 |

当前公开的同步点位只有：

- `before_llm`
- `after_llm`
- `before_tool`
- `after_tool`
- `approve_tool`

其余 lifecycle 通过事件形式只读暴露。

## Hook Actions

Hook 可以返回不同的 action 来控制流程：

| Action | 适用阶段 | 效果 |
| --- | --- | --- |
| `continue` | 所有拦截型 | 放行，不做修改 |
| `modify` | `before_llm`, `after_llm`, `before_tool`, `after_tool` | 改写请求/响应后放行 |
| `respond` | `before_tool` | 直接返回工具结果，跳过实际工具执行 |
| `deny_tool` | `before_tool` | 拒绝工具执行，返回错误信息 |
| `abort_turn` | 所有拦截型 | 中止当前 turn |
| `hard_abort` | 所有拦截型 | 强制终止整个 agent loop |

### `respond` Action

`respond` action 是特殊的：它允许 `before_tool` hook 直接提供工具结果，跳过实际工具执行。适用于：

1. **插件工具注入**：外部 hook 可以实现工具，无需在 ToolRegistry 注册
2. **工具结果缓存**：对重复调用返回缓存结果
3. **工具模拟**：测试时返回模拟结果

当 hook 返回 `respond` 并携带 `HookResult` 时，agent loop 会：
1. 跳过实际工具执行
2. 使用提供的结果作为工具执行结果
3. 正常继续 turn 流程

示例（Go 进程内 hook）：

```go
func (h *MyHook) BeforeTool(
    ctx context.Context,
    call *agent.ToolCallHookRequest,
) (*agent.ToolCallHookRequest, agent.HookDecision, error) {
    if call.Tool == "my_plugin_tool" {
        next := call.Clone()
        next.HookResult = &tools.ToolResult{
            ForLLM:  "Plugin tool executed successfully",
            Silent:  false,
            IsError: false,
        }
        return next, agent.HookDecision{Action: agent.HookActionRespond}, nil
    }
    return call, agent.HookDecision{Action: agent.HookActionContinue}, nil
}
```

示例（Python process hook）：

```python
def handle_before_tool(params: dict) -> dict:
    tool = params.get("tool", "")
    if tool == "my_plugin_tool":
        return {
            "action": "respond",
            "result": {
                "for_llm": "Plugin tool executed successfully",
                "silent": False,
                "is_error": False
            }
        }
    return {"action": "continue"}
```

## 执行顺序

HookManager 的排序规则是：

1. 先执行进程内 hook
2. 再执行 process hook
3. 同一来源内按 `priority` 从小到大
4. 若 `priority` 相同，再按名字排序

## 超时

当前配置在 `hooks.defaults` 中统一设置：

- `observer_timeout_ms`
- `interceptor_timeout_ms`
- `approval_timeout_ms`

注意：当前实现还没有单个 process hook 自己的 `timeout_ms` 字段，超时配置是全局默认值。

## 快速开始

如果你的目标只是先把当前 hook 流程跑通并观察到实际请求，最省事的是先用下面的 Python process hook 示例：

1. 打开 `hooks.enabled`
2. 把下面文档里的 Python 示例保存到本地文件，例如 `/tmp/review_gate.py`
3. 给它配置 `PICOCLAW_HOOK_LOG_FILE`
4. 重启 gateway
5. 用 `tail -f` 观察日志文件

例如：

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

观察方式：

```bash
tail -f /tmp/picoclaw-hook-review-gate.log
```

如果你是在开发 PicoClaw 本体，而不是只想验证协议，那么再看后面的 Go in-process 示例。

## 两个示例的定位

- Go in-process 示例
  适合验证宿主内的 hook 链路、理解 `MountHook()` 和各个同步点位
- Python process 示例
  适合理解 `JSON-RPC over stdio` 协议、确认宿主和外部进程之间的消息来回是否正常

这两个示例都刻意保持为“只记录、不改写、不拒绝”的安全模式。它们的目的不是提供策略能力，而是帮你观察当前 hook 系统。

## Go 进程内示例

下面这段代码是一个最小的“记录型” in-process hook。它实现了：

1. `EventObserver`
2. `LLMInterceptor`
3. `ToolInterceptor`
4. `ToolApprover`

它只记录，不改写请求，也不拒绝工具。

你可以把它保存成你自己的 Go 文件，例如 `pkg/myhooks/example_logger.go`：

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

### 如何挂载

如果你只需要代码挂载，直接在 `AgentLoop` 初始化后调用：

```go
hook := myhooks.NewExampleLoggerHook(myhooks.ExampleLoggerHookOptions{
    LogFile:   "/tmp/picoclaw-hook-example-logger.log",
    LogEvents: true,
})

if err := al.MountHook(agent.NamedHook("example-logger", hook)); err != nil {
    panic(err)
}
```

### 如果你还想用配置挂载

当前 hook 系统支持 builtin hook，但这要求你自己把 factory 编进二进制。也就是说，下面这段注册代码需要和上面的 hook 定义一起放进你的工程里：

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

只有在你自己注册了 builtin 之后，下面的配置才会生效：

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

### 如何观察它是否生效

- 如果设置了 `log_file`，它会把每次 hook 调用按 JSON Lines 写入文件
- 如果没有设置 `log_file`，它仍然会把摘要写到 gateway 日志
- 普通只走 LLM 的请求，通常会看到 `before_llm` 和 `after_llm`
- 触发工具调用的请求，通常还会看到 `before_tool`、`approve_tool`、`after_tool`
- 如果 `log_events=true`，还会额外看到 `event`

典型日志：

```json
{"ts":"2026-03-21T14:10:00Z","stage":"before_tool","meta":{"session_key":"session-1"},"payload":{"tool":"echo_text","arguments":{"text":"hello"}},"decision":{"action":"continue"}}
{"ts":"2026-03-21T14:10:00Z","stage":"approve_tool","meta":{"session_key":"session-1"},"payload":{"tool":"echo_text","arguments":{"text":"hello"}},"decision":{"approved":true}}
```

如果你只看到了 `before_llm` / `after_llm`，没有看到 tool 相关阶段，通常不是 hook 没挂上，而是这次请求本身没有触发工具调用。

## Python process hook 示例

下面这段脚本是一个最小的 `process hook` 示例。它只使用 Python 标准库，支持：

1. `hook.hello`
2. `hook.event`
3. `hook.before_tool`
4. `hook.approve_tool`

它默认只记录，不改写，也不拒绝。

你可以把它保存到任意本地路径，例如 `/tmp/review_gate.py`：

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

### 如何配置

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

### 环境变量

- `PICOCLAW_HOOK_LOG_EVENTS`
  是否把 `hook.event` 写到 `stderr`，默认开启
- `PICOCLAW_HOOK_LOG_FILE`
  外部日志文件路径。设置后，脚本会把收到的 hook 请求、notification 和返回结果按 JSON Lines 追加到该文件

注意：`PICOCLAW_HOOK_LOG_FILE` 没有默认值。不设置时，脚本不会自动落盘日志。

### 如何确认它收到了 hook

推荐同时看两个地方：

- gateway 日志
  用来观察宿主是否成功启动了外部进程，以及脚本写到 `stderr` 的事件摘要
- `PICOCLAW_HOOK_LOG_FILE`
  用来观察脚本实际收到了什么请求、返回了什么响应

典型判断方式：

- 只看到 `hook.hello`
  说明进程启动并完成握手了，但还没有新的业务 hook 请求真正打进来
- 看到 `hook.event`
  说明 `observe` 配置生效了
- 看到 `hook.before_tool`
  说明 `intercept: ["before_tool", ...]` 生效了
- 看到 `hook.approve_tool`
  说明审批 hook 生效了

这份示例脚本不会改写任何参数，也不会拒绝工具，所以你应该看到的典型返回是：

```json
{"direction":"out","id":7,"response":{"action":"continue"},"error":null}
{"direction":"out","id":8,"response":{"approved":true},"error":null}
```

一组完整样例：

```json
{"ts":"2026-03-21T14:12:00+00:00","direction":"in","id":1,"method":"hook.hello","params":{"name":"py_review_gate","version":1,"modes":["observe","tool","approve"]},"notification":false}
{"ts":"2026-03-21T14:12:00+00:00","direction":"out","id":1,"response":{"ok":true,"name":"python-review-gate"},"error":null}
{"ts":"2026-03-21T14:12:05+00:00","direction":"in","id":0,"method":"hook.event","params":{"Kind":"tool_exec_start"},"notification":true}
{"ts":"2026-03-21T14:12:05+00:00","direction":"in","id":7,"method":"hook.before_tool","params":{"tool":"echo_text","arguments":{"text":"hello"}},"notification":false}
{"ts":"2026-03-21T14:12:05+00:00","direction":"out","id":7,"response":{"action":"continue"},"error":null}
```

补充说明：

- 时间戳是 UTC，不是本地时区
- `notification=true` 表示这是 `hook.event` 这类不需要响应的通知
- `id` 会随着当前进程内的请求递增；如果 hook 进程重启，计数会重新开始

## Process Hook 协议约定

当前 process hook 使用 `JSON-RPC over stdio`：

- PicoClaw 启动外部进程
- 请求和响应都按“一行一个 JSON 消息”传输
- `hook.event` 是 notification，不需要响应
- `hook.before_llm` / `hook.after_llm` / `hook.before_tool` / `hook.after_tool` / `hook.approve_tool` 是 request/response

当前宿主不会接受 process hook 主动发起的新 RPC。也就是说，外部 hook 现在只能“响应 PicoClaw 的调用”，不能反向调用宿主去发送 channel 消息。

## 配置字段

### `hooks.builtins.<name>`

- `enabled`
- `priority`
- `config`

### `hooks.processes.<name>`

- `enabled`
- `priority`
- `transport`
  当前只支持 `stdio`
- `command`
- `dir`
- `env`
- `observe`
- `intercept`

## 排查建议

当你觉得“hook 没触发”时，优先按这个顺序排查：

1. `hooks.enabled` 是否为 `true`
2. 对应的 builtin/process hook 是否 `enabled`
3. process hook 的 `command` 路径是否正确
4. 你看的是否是正确的日志文件
5. 当前请求是否真的走到了对应阶段
6. `observe` / `intercept` 是否包含了你想看的点位

一个很实用的最小排查组合是：

- 先用文档里的 Python process 示例确认外部协议没问题
- 再用文档里的 Go in-process 示例确认宿主内的 hook 链路没问题

如果前者有 `hook.hello` 但没有业务请求，通常不是协议挂了，而是当前这次请求没有真正触发对应的 hook 点位。

## 适用边界

当前 hook 系统最适合做这些事：

- LLM 请求改写
- 工具参数规范化
- 工具执行前审批
- 审计和观测

当前还不适合直接承载这些需求：

- 外部 hook 主动发 channel 消息
- 挂起 turn 并等待人工审批回复
- inbound/outbound 全链路消息拦截

如果你要做人审流转，推荐把 hook 作为审批入口，把审批状态机和 channel 交互放到独立的 `ApprovalManager`。
