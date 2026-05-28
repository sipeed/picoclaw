# Plugin Tool Injection Example

This document demonstrates how to use PicoClaw's hook system to implement external plugin tool injection, allowing LLM to call tools implemented by external hook processes.

---

## Core Principle

Through the hook system's `respond` action, external hooks can:

1. Inject tool **definitions** in `before_llm`, letting LLM know the tool is available
2. Return tool **execution results** directly in `before_tool` using `respond` action, skipping ToolRegistry

This way, external hooks can fully implement plugin tools without registering any tools inside PicoClaw.

---

## Complete Example: Weather Query Plugin

Below is a complete Python hook example implementing a weather query plugin tool.

### 1. Hook Script Implementation

Save as `/tmp/weather_plugin.py`:

```python
#!/usr/bin/env python3
"""Weather query plugin hook example"""
from __future__ import annotations

import json
import sys
import signal
from typing import Any

# Simulated weather data
WEATHER_DATA = {
    "Beijing": {"temp": 15, "weather": "Sunny", "humidity": 45},
    "Shanghai": {"temp": 18, "weather": "Cloudy", "humidity": 60},
    "Guangzhou": {"temp": 25, "weather": "Sunny", "humidity": 70},
    "Shenzhen": {"temp": 26, "weather": "Cloudy", "humidity": 75},
}


def get_weather(city: str) -> dict:
    """Get weather data (simulated)"""
    data = WEATHER_DATA.get(city)
    if data:
        return {
            "for_llm": f"{city} weather: {data['weather']}, temperature {data['temp']}°C, humidity {data['humidity']}%",
            "for_user": "",
            "silent": False,
            "is_error": False,
        }
    return {
        "for_llm": f"Weather data not found for city {city}",
        "for_user": "",
        "silent": False,
        "is_error": True,
    }


def handle_hello(params: dict) -> dict:
    return {"ok": True, "name": "weather-plugin"}


def handle_before_llm(params: dict) -> dict:
    """Inject weather query tool definition"""
    tools = params.get("tools", [])

    # Add weather query tool
    tools.append({
        "type": "function",
        "function": {
            "name": "get_weather",
            "description": "Query weather information for a specified city",
            "parameters": {
                "type": "object",
                "properties": {
                    "city": {
                        "type": "string",
                        "description": "City name, e.g.: Beijing, Shanghai, Guangzhou"
                    }
                },
                "required": ["city"]
            }
        }
    })

    return {
        "action": "modify",
        "request": {
            "model": params.get("model"),
            "messages": params.get("messages", []),
            "tools": tools,
            "options": params.get("options", {}),
        }
    }


def handle_before_tool(params: dict) -> dict:
    """Handle tool call, return result directly"""
    tool = params.get("tool", "")
    args = params.get("arguments", {})

    if tool == "get_weather":
        city = args.get("city", "")
        result = get_weather(city)

        # Use respond action to return result directly, skip ToolRegistry
        return {
            "action": "respond",
            "result": result,
        }

    # Other tools continue normal flow
    return {"action": "continue"}


def handle_request(method: str, params: dict) -> dict:
    if method == "hook.hello":
        return handle_hello(params)
    if method == "hook.before_llm":
        return handle_before_llm(params)
    if method == "hook.before_tool":
        return handle_before_tool(params)
    if method == "hook.after_llm":
        return {"action": "continue"}
    if method == "hook.after_tool":
        return {"action": "continue"}
    if method == "hook.approve_tool":
        return {"approved": True}
    raise KeyError(f"method not found: {method}")


def send_response(message_id: int, result: Any | None = None, error: str | None = None) -> None:
    payload: dict[str, Any] = {
        "jsonrpc": "2.0",
        "id": message_id,
    }
    if error is not None:
        payload["error"] = {"code": -32000, "message": error}
    else:
        payload["result"] = result if result is not None else {}

    sys.stdout.write(json.dumps(payload, ensure_ascii=True) + "\n")
    sys.stdout.flush()


def main() -> int:
    for raw_line in sys.stdin:
        line = raw_line.strip()
        if not line:
            continue

        try:
            message = json.loads(line)
        except json.JSONDecodeError:
            continue

        method = message.get("method")
        message_id = message.get("id", 0)
        params = message.get("params") or {}

        if not message_id:
            continue

        try:
            result = handle_request(str(method or ""), params)
            send_response(int(message_id), result=result)
        except KeyError as exc:
            send_response(int(message_id), error=str(exc))
        except Exception as exc:
            send_response(int(message_id), error=f"unexpected error: {exc}")

    return 0


if __name__ == "__main__":
    signal.signal(signal.SIGINT, lambda *_: raise SystemExit(0))
    signal.signal(signal.SIGTERM, lambda *_: raise SystemExit(0))
    raise SystemExit(main())
```

### 2. Configure PicoClaw

Add hook configuration in the config file:

```json
{
  "hooks": {
    "enabled": true,
    "processes": {
      "weather_plugin": {
        "enabled": true,
        "priority": 100,
        "transport": "stdio",
        "command": ["python3", "/tmp/weather_plugin.py"],
        "intercept": ["before_llm", "before_tool"]
      }
    }
  }
}
```

### 3. Test Results

When user asks "What's the weather in Beijing today?":

1. PicoClaw sends `hook.before_llm`, hook injects `get_weather` tool definition
2. LLM sees tool definition, decides to call `get_weather(city="Beijing")`
3. PicoClaw sends `hook.before_tool`, hook uses `respond` action to return weather data
4. LLM receives result, replies to user "Beijing is sunny today, temperature 15°C"

---

## Flow Diagram

```
User: "What's the weather in Beijing today?"
        ↓
    PicoClaw
        ↓
    hook.before_llm
        ↓ (inject get_weather tool definition)
    LLM request
        ↓
    LLM decides to call get_weather(city="Beijing")
        ↓
    hook.before_tool
        ↓ (respond action returns weather data)
    Return result directly to LLM
        ↓ (skip ToolRegistry)
    LLM replies: "Beijing is sunny today, temperature 15°C"
```

---

## Key Points

### `before_llm` Inject Tool Definition

Tool definition follows OpenAI function calling format:

```json
{
  "type": "function",
  "function": {
    "name": "tool_name",
    "description": "tool description",
    "parameters": {
      "type": "object",
      "properties": {
        "param_name": {
          "type": "string",
          "description": "parameter description"
        }
      },
      "required": ["list of required parameters"]
    }
  }
}
```

### `before_tool` Use respond Action

`respond` action response format:

```json
{
  "action": "respond",
  "result": {
    "for_llm": "Content returned to LLM",
    "for_user": "Optional, content sent to user",
    "silent": false,
    "is_error": false,
    "media": ["Optional, media reference list"],
    "response_handled": false
  }
}
```

| Field | Description |
|-------|-------------|
| `for_llm` | Required, LLM will see this content |
| `for_user` | Optional, sent directly to user |
| `silent` | When true, not sent to user |
| `is_error` | When true, indicates execution failure |
| `media` | Optional, media file references (images, files, etc.) |
| `response_handled` | When true, indicates user request is handled, turn will end |

---

## Media File Handling

The `respond` action supports returning media files (images, files, etc.). There are two processing modes:

### 1. Automatic Delivery (`response_handled=true`)

When `response_handled=true`, media files are automatically sent to the user and the turn ends:

```json
{
  "action": "respond",
  "result": {
    "for_llm": "Image sent to user",
    "for_user": "",
    "media": ["media://abc123"],
    "response_handled": true
  }
}
```

Use cases:
- Image generation plugin directly returning results
- File download plugin sending files to user

### 2. LLM Visible (`response_handled=false`)

When `response_handled=false`, media references are passed to the LLM, which can see the content in the next request:

```json
{
  "action": "respond",
  "result": {
    "for_llm": "Image loaded, path: /tmp/image.png [file:/tmp/image.png]",
    "media": ["media://abc123"]
  }
}
```

After seeing the content, the LLM can decide:
- Use `send_file` tool to send to user
- Analyze image content and reply to user
- Other processing approaches

### Media Reference Format

Media references use the `media://` protocol:

```
media://<store-id>
```

These references are managed by PicoClaw's MediaStore and can be:
- Sent to user via channel
- Converted to base64 in LLM vision requests

### Alternative: Use Existing Tools

If the plugin generates files, you can return the file path and let the LLM call `send_file` or similar tools:

```json
{
  "action": "respond",
  "result": {
    "for_llm": "Image generated, saved at /tmp/generated_image.png. Use send_file tool to send to user.",
    "for_user": "",
    "silent": false
  }
}
```

This approach:
- More decoupled, LLM decides when to send
- Leverages existing tool mechanisms
- Supports batch sending, delayed sending, etc.

---

## Multi-Tool Injection Example

Multiple tools can be injected simultaneously:

```python
def handle_before_llm(params: dict) -> dict:
    tools = params.get("tools", [])

    # Tool 1: Weather query
    tools.append({
        "type": "function",
        "function": {
            "name": "get_weather",
            "description": "Query city weather",
            "parameters": {
                "type": "object",
                "properties": {
                    "city": {"type": "string", "description": "City name"}
                },
                "required": ["city"]
            }
        }
    })

    # Tool 2: Calculator
    tools.append({
        "type": "function",
        "function": {
            "name": "calculate",
            "description": "Perform mathematical calculations",
            "parameters": {
                "type": "object",
                "properties": {
                    "expression": {"type": "string", "description": "Mathematical expression"}
                },
                "required": ["expression"]
            }
        }
    })

    return {
        "action": "modify",
        "request": {
            "model": params.get("model"),
            "messages": params.get("messages", []),
            "tools": tools,
            "options": params.get("options", {}),
        }
    }


def handle_before_tool(params: dict) -> dict:
    tool = params.get("tool", "")
    args = params.get("arguments", {})

    if tool == "get_weather":
        return {
            "action": "respond",
            "result": get_weather(args.get("city", "")),
        }

    if tool == "calculate":
        # Simple calculation example
        try:
            expr = args.get("expression", "")
            result = eval(expr)  # Note: needs security handling in actual use
            return {
                "action": "respond",
                "result": {
                    "for_llm": f"Calculation result: {result}",
                    "silent": False,
                    "is_error": False,
                },
            }
        except Exception as e:
            return {
                "action": "respond",
                "result": {
                    "for_llm": f"Calculation error: {e}",
                    "silent": False,
                    "is_error": True,
                },
            }

    return {"action": "continue"}
```

---

## Coexistence with Built-in Tools

Injected plugin tools coexist with PicoClaw built-in tools:

- Built-in tools (like `bash`, `read_file`) execute normally through ToolRegistry
- Plugin tools return results through hook's `respond` action
- `handle_before_tool` only handles plugin tools, other tools return `continue`

---

## Go In-Process Hook Example

If you need to implement plugin tool injection in Go code:

```go
package myhooks

import (
    "context"
    "github.com/sipeed/picoclaw/pkg/agent"
    "github.com/sipeed/picoclaw/pkg/tools"
)

type WeatherPluginHook struct{}

func (h *WeatherPluginHook) BeforeLLM(
    ctx context.Context,
    req *agent.LLMHookRequest,
) (*agent.LLMHookRequest, agent.HookDecision, error) {
    // Inject tool definition
    req.Tools = append(req.Tools, agent.ToolDefinition{
        Type: "function",
        Function: agent.FunctionDefinition{
            Name:        "get_weather",
            Description: "Query city weather",
            Parameters: map[string]any{
                "type": "object",
                "properties": map[string]any{
                    "city": map[string]any{
                        "type":        "string",
                        "description": "City name",
                    },
                },
                "required": []string{"city"},
            },
        },
    })

    return req, agent.HookDecision{Action: agent.HookActionContinue}, nil
}

func (h *WeatherPluginHook) BeforeTool(
    ctx context.Context,
    call *agent.ToolCallHookRequest,
) (*agent.ToolCallHookRequest, agent.HookDecision, error) {
    if call.Tool == "get_weather" {
        city := call.Arguments["city"].(string)

        // Set HookResult, use respond action
        next := call.Clone()
        next.HookResult = &tools.ToolResult{
            ForLLM:  getWeatherData(city),
            Silent:  false,
            IsError: false,
        }

        return next, agent.HookDecision{Action: agent.HookActionRespond}, nil
    }

    return call, agent.HookDecision{Action: agent.HookActionContinue}, nil
}

func getWeatherData(city string) string {
    // Implement weather query logic
    return fmt.Sprintf("%s weather: Sunny, temperature 20°C", city)
}
```

---

## Summary

Through the hook system's `respond` action, external processes can:

1. **Inject tool definitions**: Let LLM know new tools are available
2. **Provide tool implementation**: Return execution results directly, no need to register in ToolRegistry
3. **Coexist with built-in tools**: Does not affect normal operation of PicoClaw's original tools

This provides a flexible and elegant solution for plugin development.

---

## Security Boundaries

### Bypassing Approval Checks

**Important**: The `respond` action bypasses `ApproveTool` approval checks.

This means:
- A `before_tool` hook can return `respond` for **any tool name**, including sensitive tools (like `bash`)
- The tool won't go through the approval process, directly returning the hook-provided result
- This is designed for plugin tools but introduces security risks

### Security Recommendations

1. **Review hook configuration**: Ensure only trusted hook processes are enabled
2. **Limit hook scope**: Add your own security checks in hook implementation
3. **Use `deny_tool` for rejection**: Use `deny_tool` action instead of `respond` with error for denying execution

### Example: Hook-Internal Security Check

```python
def handle_before_tool(params: dict) -> dict:
    tool = params.get("tool", "")
    args = params.get("arguments", {})

    # Security check: only handle plugin tools
    if tool in ["get_weather", "calculate"]:
        return {
            "action": "respond",
            "result": execute_plugin_tool(tool, args),
        }

    # Other tools continue normal flow (will go through approval)
    return {"action": "continue"}
```

This ensures the hook only affects plugin tools, not system tool approval flow.