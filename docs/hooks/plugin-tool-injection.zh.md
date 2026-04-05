# 插件工具注入示例

本文档展示如何利用 PicoClaw 的 hook 系统实现外部插件工具注入，让 LLM 能调用由外部 hook 进程实现的工具。

---

## 核心原理

通过 hook 系统的 `respond` action，外部 hook 可以：

1. 在 `before_llm` 中注入工具**定义**，让 LLM 知道有这个工具可用
2. 在 `before_tool` 中使用 `respond` action 直接返回工具**执行结果**，跳过 ToolRegistry

这样，外部 hook 可以完全实现插件工具，无需在 PicoClaw 内部注册任何工具。

---

## 完整示例：天气查询插件

下面是一个完整的 Python hook 示例，实现一个天气查询插件工具。

### 1. Hook 脚本实现

保存为 `/tmp/weather_plugin.py`：

```python
#!/usr/bin/env python3
"""天气查询插件 hook 示例"""
from __future__ import annotations

import json
import sys
import signal
from typing import Any

# 模拟天气数据
WEATHER_DATA = {
    "北京": {"temp": 15, "weather": "晴", "humidity": 45},
    "上海": {"temp": 18, "weather": "多云", "humidity": 60},
    "广州": {"temp": 25, "weather": "晴", "humidity": 70},
    "深圳": {"temp": 26, "weather": "多云", "humidity": 75},
}


def get_weather(city: str) -> dict:
    """获取天气数据（模拟）"""
    data = WEATHER_DATA.get(city)
    if data:
        return {
            "for_llm": f"{city}天气：{data['weather']}，温度{data['temp']}°C，湿度{data['humidity']}%",
            "for_user": "",
            "silent": False,
            "is_error": False,
        }
    return {
        "for_llm": f"未找到城市 {city} 的天气数据",
        "for_user": "",
        "silent": False,
        "is_error": True,
    }


def handle_hello(params: dict) -> dict:
    return {"ok": True, "name": "weather-plugin"}


def handle_before_llm(params: dict) -> dict:
    """注入天气查询工具定义"""
    tools = params.get("tools", [])
    
    # 添加天气查询工具
    tools.append({
        "type": "function",
        "function": {
            "name": "get_weather",
            "description": "查询指定城市的天气信息",
            "parameters": {
                "type": "object",
                "properties": {
                    "city": {
                        "type": "string",
                        "description": "城市名称，如：北京、上海、广州"
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
    """处理工具调用，直接返回结果"""
    tool = params.get("tool", "")
    args = params.get("arguments", {})
    
    if tool == "get_weather":
        city = args.get("city", "")
        result = get_weather(city)
        
        # 使用 respond action 直接返回结果，跳过 ToolRegistry
        return {
            "action": "respond",
            "result": result,
        }
    
    # 其他工具继续正常流程
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

### 2. 配置 PicoClaw

在配置文件中添加 hook 配置：

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

### 3. 测试效果

当用户问"北京今天天气怎么样？"时：

1. PicoClaw 发送 `hook.before_llm`，hook 注入 `get_weather` 工具定义
2. LLM 看到工具定义，决定调用 `get_weather(city="北京")`
3. PicoClaw 发送 `hook.before_tool`，hook 使用 `respond` action 返回天气数据
4. LLM 收到结果，回复用户"北京今天晴天，温度15°C"

---

## 流程图解

```
用户: "北京今天天气怎么样？"
        ↓
    PicoClaw
        ↓
    hook.before_llm
        ↓ (注入 get_weather 工具定义)
    LLM 请求
        ↓
    LLM 决定调用 get_weather(city="北京")
        ↓
    hook.before_tool
        ↓ (respond action 返回天气数据)
    直接返回结果给 LLM
        ↓ (跳过 ToolRegistry)
    LLM 回复: "北京今天晴天，温度15°C"
```

---

## 关键点说明

### `before_llm` 注入工具定义

工具定义遵循 OpenAI function calling 格式：

```json
{
  "type": "function",
  "function": {
    "name": "工具名称",
    "description": "工具描述",
    "parameters": {
      "type": "object",
      "properties": {
        "参数名": {
          "type": "string",
          "description": "参数描述"
        }
      },
      "required": ["必需参数列表"]
    }
  }
}
```

### `before_tool` 使用 respond action

`respond` action 的响应格式：

```json
{
  "action": "respond",
  "result": {
    "for_llm": "返回给 LLM 的内容",
    "for_user": "可选，发送给用户的内容",
    "silent": false,
    "is_error": false,
    "media": ["可选，媒体引用列表"],
    "response_handled": false
  }
}
```

| 字段 | 说明 |
|------|------|
| `for_llm` | 必须，LLM 会看到这个内容 |
| `for_user` | 可选，直接发送给用户 |
| `silent` | 为 true 时不发送给用户 |
| `is_error` | 为 true 时表示执行失败 |
| `media` | 可选，媒体文件引用列表（如图片、文件） |
| `response_handled` | 为 true 时表示已处理用户请求，轮次将结束 |

---

## 媒体文件处理

`respond` action 支持返回媒体文件（图片、文件等）。有两种处理方式：

### 1. 自动发送（`response_handled=true`）

当 `response_handled=true` 时，媒体文件会自动发送给用户，轮次结束：

```json
{
  "action": "respond",
  "result": {
    "for_llm": "图片已发送给用户",
    "for_user": "",
    "media": ["media://abc123"],
    "response_handled": true
  }
}
```

适用场景：
- 图像生成插件直接返回结果
- 文件下载插件发送文件给用户

### 2. LLM 可见（`response_handled=false`）

当 `response_handled=false` 时，媒体引用会传递给 LLM，LLM 可以在下一轮请求中看到内容：

```json
{
  "action": "respond",
  "result": {
    "for_llm": "图片已加载，路径：/tmp/image.png [file:/tmp/image.png]",
    "media": ["media://abc123"]
  }
}
```

LLM 看到内容后，可以自主决定：
- 使用 `send_file` 工具发送给用户
- 分析图片内容并回复用户
- 其他处理方式

### 媒体引用格式

媒体引用使用 `media://` 协议：

```
media://<store-id>
```

这些引用由 PicoClaw 的 MediaStore 管理，可以：
- 通过 channel 发送给用户
- 在 LLM vision 请求中转换为 base64

### 替代方案：使用现有工具

如果插件生成文件，可以返回文件路径让 LLM 调用 `send_file` 等工具：

```json
{
  "action": "respond",
  "result": {
    "for_llm": "图片已生成，保存在 /tmp/generated_image.png。使用 send_file 工具发送给用户。",
    "for_user": "",
    "silent": false
  }
}
```

这种方式：
- 更解耦，LLM 自主决策发送时机
- 利用现有工具机制
- 支持批量发送、延迟发送等场景

---

## 多工具注入示例

可以同时注入多个工具：

```python
def handle_before_llm(params: dict) -> dict:
    tools = params.get("tools", [])
    
    # 工具1：天气查询
    tools.append({
        "type": "function",
        "function": {
            "name": "get_weather",
            "description": "查询城市天气",
            "parameters": {
                "type": "object",
                "properties": {
                    "city": {"type": "string", "description": "城市名称"}
                },
                "required": ["city"]
            }
        }
    })
    
    # 工具2：计算器
    tools.append({
        "type": "function",
        "function": {
            "name": "calculate",
            "description": "执行数学计算",
            "parameters": {
                "type": "object",
                "properties": {
                    "expression": {"type": "string", "description": "数学表达式"}
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
        # 简单计算示例
        try:
            expr = args.get("expression", "")
            result = eval(expr)  # 注意：实际使用时需要安全处理
            return {
                "action": "respond",
                "result": {
                    "for_llm": f"计算结果: {result}",
                    "silent": False,
                    "is_error": False,
                },
            }
        except Exception as e:
            return {
                "action": "respond",
                "result": {
                    "for_llm": f"计算错误: {e}",
                    "silent": False,
                    "is_error": True,
                },
            }
    
    return {"action": "continue"}
```

---

## 与内置工具共存

注入的插件工具与 PicoClaw 内置工具共存：

- 内置工具（如 `bash`、`read_file`）正常通过 ToolRegistry 执行
- 插件工具通过 hook 的 `respond` action 返回结果
- `handle_before_tool` 中只处理插件工具，其他工具返回 `continue`

---

## Go 进程内 Hook 示例

如果需要在 Go 代码中实现插件工具注入：

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
    // 注入工具定义
    req.Tools = append(req.Tools, agent.ToolDefinition{
        Type: "function",
        Function: agent.FunctionDefinition{
            Name:        "get_weather",
            Description: "查询城市天气",
            Parameters: map[string]any{
                "type": "object",
                "properties": map[string]any{
                    "city": map[string]any{
                        "type":        "string",
                        "description": "城市名称",
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
        
        // 设置 HookResult，使用 respond action
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
    // 实现天气查询逻辑
    return fmt.Sprintf("%s天气：晴，温度20°C", city)
}
```

---

## 总结

通过 hook 系统的 `respond` action，外部进程可以：

1. **注入工具定义**：让 LLM 知道有新工具可用
2. **提供工具实现**：直接返回执行结果，无需注册到 ToolRegistry
3. **与内置工具共存**：不影响 PicoClaw 原有工具的正常运行

这为插件开发提供了灵活、优雅的解决方案。

---

## 安全边界说明

### 绕过审批检查

**重要**：`respond` action 会绕过 `ApproveTool` 审批检查。

这意味着：
- `before_tool` hook 可以为**任何工具名称**返回 `respond`，包括敏感工具（如 `bash`）
- 工具不会经过审批流程，直接返回 hook 提供的结果
- 这是为了支持插件工具而设计，但也带来了安全风险

### 安全建议

1. **审查 hook 配置**：确保只有可信的 hook 进程被启用
2. **限制 hook 权限**：在 hook 实现中添加自己的安全检查
3. **优先使用 `deny_tool`**：对于拒绝执行，使用 `deny_tool` action 而非 `respond` 返回错误

### 示例：hook 内置安全检查

```python
def handle_before_tool(params: dict) -> dict:
    tool = params.get("tool", "")
    args = params.get("arguments", {})
    
    # 安全检查：只处理插件工具
    if tool in ["get_weather", "calculate"]:
        return {
            "action": "respond",
            "result": execute_plugin_tool(tool, args),
        }
    
    # 其他工具继续正常流程（会经过审批）
    return {"action": "continue"}
```

这样可以确保 hook 只影响插件工具，不影响系统工具的审批流程。