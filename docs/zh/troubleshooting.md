# 🐛 疑难解答

> 返回 [README](../../README.zh.md)

## "model ... not found in model_list" 或 OpenRouter "free is not a valid model ID"

**症状：** 你看到以下任一错误：

- `Error creating provider: model "openrouter/free" not found in model_list`
- OpenRouter 返回 400：`"free is not a valid model ID"`

**原因：** `model_list` 条目中的 `model` 字段是发送给 API 的内容。对于 OpenRouter，你必须使用**完整的**模型 ID，而不是简写。

- **错误：** `"model": "free"` → OpenRouter 收到 `free` 并拒绝。
- **正确：** `"model": "openrouter/free"` → OpenRouter 收到 `openrouter/free`（自动免费层路由）。

**修复方法：** 在 `~/.picoclaw/config.json`（或你的配置路径）中：

1. **agents.defaults.model** 必须匹配 `model_list` 中的某个 `model_name`（例如 `"openrouter-free"`）。
2. 该条目的 **model** 必须是有效的 OpenRouter 模型 ID，例如：
   - `"openrouter/free"` – 自动免费层
   - `"google/gemini-2.0-flash-exp:free"`
   - `"meta-llama/llama-3.1-8b-instruct:free"`

示例片段：

```json
{
  "agents": {
    "defaults": {
      "model": "openrouter-free"
    }
  },
  "model_list": [
    {
      "model_name": "openrouter-free",
      "model": "openrouter/free",
      "api_key": "sk-or-v1-YOUR_OPENROUTER_KEY",
      "api_base": "https://openrouter.ai/api/v1"
    }
  ]
}
```

在 [OpenRouter Keys](https://openrouter.ai/keys) 获取你的密钥。
