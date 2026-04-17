# Hook JSON-RPC Protocol Details

All hooks use `JSON-RPC 2.0` format, with one JSON message per line, transmitted via stdio.

---

## Basic Protocol Structure

### Request (PicoClaw → Hook)

```json
{"jsonrpc":"2.0","id":1,"method":"hook.xxx","params":{...}}
```

### Response (Hook → PicoClaw)

Success:
```json
{"jsonrpc":"2.0","id":1,"result":{...}}
```

Error:
```json
{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"error message"}}
```

---

## 1. `hook.hello` (Handshake)

Handshake must be completed at startup, otherwise the hook process will be terminated.

### Request

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "hook.hello",
  "params": {
    "name": "py_review_gate",
    "version": 1,
    "modes": ["observe", "tool", "approve"]
  }
}
```

| Field | Description |
|-------|-------------|
| `name` | hook name (from configuration) |
| `version` | protocol version, currently `1` |
| `modes` | capability modes supported by the hook |

### Response

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "ok": true,
    "name": "python-review-gate"
  }
}
```

---

## 2. `hook.before_llm`

Triggered before sending request to LLM. Can be used to inject tools.

### Request

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "hook.before_llm",
  "params": {
    "meta": {
      "AgentID": "agent-1",
      "TurnID": "turn-1",
      "ParentTurnID": "",
      "SessionKey": "session-1",
      "Iteration": 0,
      "TracePath": "runTurn",
      "Source": "turn.llm.request"
    },
    "model": "claude-sonnet",
    "messages": [
      {"role": "user", "content": "hello"}
    ],
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "echo",
          "description": "echo text",
          "parameters": {"type": "object"}
        }
      }
    ],
    "options": {
      "temperature": 0.7
    },
    "channel": "cli",
    "chat_id": "chat-1",
    "graceful_terminal": false
  }
}
```

| Field | Description |
|-------|-------------|
| `meta` | event metadata for tracing |
| `model` | requested model name |
| `messages` | conversation history |
| `tools` | list of available tool definitions |
| `options` | LLM parameters (temperature, max_tokens, etc.) |
| `channel` | request source channel |
| `chat_id` | session ID |

### Response (Tool Injection Example)

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "action": "modify",
    "request": {
      "model": "claude-sonnet",
      "messages": [{"role": "user", "content": "hello"}],
      "tools": [
        {
          "type": "function",
          "function": {
            "name": "echo",
            "description": "echo",
            "parameters": {}
          }
        },
        {
          "type": "function",
          "function": {
            "name": "my_plugin_tool",
            "description": "Plugin injected tool",
            "parameters": {
              "type": "object",
              "properties": {
                "query": {"type": "string"}
              }
            }
          }
        }
      ]
    }
  }
}
```

| Field | Description |
|-------|-------------|
| `action` | decision action (see table below) |
| `request` | modified request object |

---

## 3. `hook.after_llm`

Triggered after receiving LLM response. Can modify response content.

### Request

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "hook.after_llm",
  "params": {
    "meta": {
      "AgentID": "agent-1",
      "TurnID": "turn-1",
      "SessionKey": "session-1"
    },
    "model": "claude-sonnet",
    "response": {
      "role": "assistant",
      "content": "Hi!",
      "tool_calls": [
        {
          "id": "tc-1",
          "type": "function",
          "function": {
            "name": "echo",
            "arguments": "{\"text\":\"hi\"}"
          }
        }
      ]
    },
    "channel": "cli",
    "chat_id": "chat-1"
  }
}
```

### Response

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "action": "continue"
  }
}
```

---

## 4. `hook.before_tool`

Triggered before tool execution. Can modify tool name and arguments, deny execution, or return result directly.

### Request

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "hook.before_tool",
  "params": {
    "meta": {
      "AgentID": "agent-1",
      "TurnID": "turn-1",
      "SessionKey": "session-1"
    },
    "tool": "echo_text",
    "arguments": {
      "text": "hello"
    },
    "channel": "cli",
    "chat_id": "chat-1"
  }
}
```

| Field | Description |
|-------|-------------|
| `tool` | tool name |
| `arguments` | tool arguments |

### Response (Modify Arguments)

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "action": "modify",
    "call": {
      "tool": "echo_text",
      "arguments": {
        "text": "modified hello"
      }
    }
  }
}
```

### Response (Deny Execution)

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "action": "deny_tool",
    "reason": "Invalid arguments"
  }
}
```

### Response (Return Result Directly - respond)

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "action": "respond",
    "call": {
      "tool": "my_plugin_tool",
      "arguments": {
        "query": "hello"
      }
    },
    "result": {
      "for_llm": "Plugin tool executed successfully",
      "for_user": "",
      "silent": false,
      "is_error": false
    }
  }
}
```

The `respond` action allows hooks to return tool results directly, skipping actual tool execution. Use cases:
1. **Plugin tool injection**: External hooks can implement tools without registering in ToolRegistry
2. **Tool result caching**: Return cached results for repeated calls
3. **Tool mocking**: Return mock results during testing

| Field | Description |
|-------|-------------|
| `action` | must be `respond` |
| `call` | modified call information (optional) |
| `result` | tool result to return directly |

---

## 5. `hook.after_tool`

Triggered after tool execution completes. Can modify the result returned to LLM.

### Request

```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "hook.after_tool",
  "params": {
    "meta": {
      "AgentID": "agent-1",
      "TurnID": "turn-1",
      "SessionKey": "session-1"
    },
    "tool": "echo_text",
    "arguments": {
      "text": "hello"
    },
    "result": {
      "for_llm": "echoed: hello",
      "for_user": "",
      "silent": false,
      "is_error": false,
      "async": false,
      "media": [],
      "artifact_tags": [],
      "response_handled": false
    },
    "duration": 15000000,
    "channel": "cli",
    "chat_id": "chat-1"
  }
}
```

| Field | Description |
|-------|-------------|
| `result.for_llm` | content returned to LLM |
| `result.for_user` | content sent to user |
| `result.silent` | whether silent (not sent to user) |
| `result.is_error` | whether it's an error |
| `result.async` | whether executed asynchronously |
| `result.media` | list of media references |
| `result.artifact_tags` | local artifact path tags |
| `result.response_handled` | whether response has been handled |
| `duration` | execution time (nanoseconds) |

### Response

```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "result": {
    "action": "continue"
  }
}
```

---

## 6. `hook.approve_tool`

Approval hook for deciding whether to allow execution of sensitive tools.

### Request

```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "hook.approve_tool",
  "params": {
    "meta": {
      "AgentID": "agent-1",
      "TurnID": "turn-1",
      "SessionKey": "session-1"
    },
    "tool": "bash",
    "arguments": {
      "command": "rm -rf /"
    },
    "channel": "cli",
    "chat_id": "chat-1"
  }
}
```

### Response (Approved)

```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "result": {
    "approved": true
  }
}
```

### Response (Denied)

```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "result": {
    "approved": false,
    "reason": "Dangerous command, execution denied"
  }
}
```

---

## 7. `hook.event` (notification)

Observer event, broadcast only, no response required. `id` is `0` or absent.

```json
{
  "jsonrpc": "2.0",
  "method": "hook.event",
  "params": {
    "Kind": "tool_exec_start",
    "Meta": {
      "AgentID": "agent-1",
      "TurnID": "turn-1"
    },
    "Payload": {
      "Tool": "echo_text",
      "Arguments": {"text": "hello"}
    }
  }
}
```

Common `Kind` values:
- `turn_start` / `turn_end`
- `llm_request` / `llm_response`
- `tool_exec_start` / `tool_exec_end` / `tool_exec_skipped`
- `steering_injected`
- `interrupt_received`
- `error`

---

## Action Options

| action | Applicable hooks | Effect |
|--------|-----------------|--------|
| `continue` | All interceptor types | Pass through without modification |
| `modify` | `before_llm`, `before_tool`, `after_llm`, `after_tool` | Modify request/response and pass through |
| `respond` | `before_tool` | Return tool result directly, skip actual execution. **Note: AfterTool is NOT called (design decision - respond provides final answer).** |
| `deny_tool` | `before_tool` | Deny tool execution |
| `abort_turn` | All interceptor types | Abort current turn, return error |
| `hard_abort` | All interceptor types | Force stop entire agent loop |

---

## Complete Flow Example

```json
{"jsonrpc":"2.0","id":1,"method":"hook.hello","params":{"name":"my_hook","version":1,"modes":["tool","approve"]}}
{"jsonrpc":"2.0","id":1,"result":{"ok":true,"name":"my_hook"}}
{"jsonrpc":"2.0","id":2,"method":"hook.before_llm","params":{"model":"claude-sonnet","messages":[{"role":"user","content":"hello"}],"tools":[]}}
{"jsonrpc":"2.0","id":2,"result":{"action":"continue"}}
{"jsonrpc":"2.0","id":3,"method":"hook.before_tool","params":{"tool":"bash","arguments":{"command":"ls"}}}
{"jsonrpc":"2.0","id":3,"result":{"action":"continue"}}
{"jsonrpc":"2.0","id":4,"method":"hook.approve_tool","params":{"tool":"bash","arguments":{"command":"ls"}}}
{"jsonrpc":"2.0","id":4,"result":{"approved":true}}
{"jsonrpc":"2.0","id":5,"method":"hook.after_tool","params":{"tool":"bash","arguments":{"command":"ls"},"result":{"for_llm":"file1.txt\nfile2.txt"},"duration":5000000}}
{"jsonrpc":"2.0","id":5,"result":{"action":"continue"}}
{"jsonrpc":"2.0","id":6,"method":"hook.after_llm","params":{"model":"claude-sonnet","response":{"role":"assistant","content":"Files listed"}}}
{"jsonrpc":"2.0","id":6,"result":{"action":"continue"}}
```

---

## Plugin Tool Injection via `before_llm` and `before_tool`

Standard flow for plugin tool injection:

1. In `before_llm`, inject tool definition to let LLM know the tool is available
2. In `before_tool`, use `respond` action to return tool execution result directly

### `before_llm` Inject Tool Definition

```python
def handle_before_llm(params: dict) -> dict:
    tools = params.get("tools", [])
    
    # Add plugin tool definition
    tools.append({
        "type": "function",
        "function": {
            "name": "my_plugin_tool",
            "description": "Plugin provided tool",
            "parameters": {
                "type": "object",
                "properties": {
                    "input": {"type": "string", "description": "Input content"}
                },
                "required": ["input"]
            }
        }
    })
    
    return {
        "action": "modify",
        "request": {
            "model": params["model"],
            "messages": params["messages"],
            "tools": tools,
            "options": params.get("options", {})
        }
    }
```

### `before_tool` Return Execution Result

```python
def handle_before_tool(params: dict) -> dict:
    tool = params.get("tool", "")
    
    if tool == "my_plugin_tool":
        # Implement tool logic here
        args = params.get("arguments", {})
        input_text = args.get("input", "")
        
        # Return result directly, no need to register in ToolRegistry
        return {
            "action": "respond",
            "result": {
                "for_llm": f"Plugin tool executed successfully, input: {input_text}",
                "silent": False,
                "is_error": False
            }
        }
    
    return {"action": "continue"}
```

This way, external hooks can fully implement plugin tools without registering any tool implementation inside PicoClaw.