# 🔌 提供商与模型配置

> 返回 [README](../project/README.zh.md)

### 提供商 (Providers)

> [!NOTE]
> 语音转录现在可以通过 `voice.model_name` 指定的多模态模型完成；如果未配置语音模型，Groq Whisper 仍可作为回退方案。

| 提供商               | 用途                         | 获取 API Key                                                         |
| -------------------- | ---------------------------- | -------------------------------------------------------------------- |
| `gemini`             | LLM (Gemini 直连)            | [aistudio.google.com](https://aistudio.google.com)                   |
| `zhipu`              | LLM (智谱直连)               | [bigmodel.cn](https://bigmodel.cn)                                   |
| `volcengine`         | LLM (火山引擎直连)           | [volcengine.com](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) |
| `openrouter`         | LLM (推荐，可访问所有模型)   | [openrouter.ai](https://openrouter.ai)                               |
| `anthropic`          | LLM (Claude 直连)            | [console.anthropic.com](https://console.anthropic.com)               |
| `openai`             | LLM (GPT 直连)               | [platform.openai.com](https://platform.openai.com)                   |
| `venice`             | LLM (Venice AI 直连)         | [venice.ai](https://venice.ai)                                       |
| `deepseek`           | LLM (DeepSeek 直连)          | [platform.deepseek.com](https://platform.deepseek.com)               |
| `qwen`               | LLM (通义千问)               | [dashscope.console.aliyun.com](https://dashscope.console.aliyun.com) |
| `groq`               | LLM + **语音转录** (Whisper) | [console.groq.com](https://console.groq.com)                         |
| `cerebras`           | LLM (Cerebras 直连)          | [cerebras.ai](https://cerebras.ai)                                   |
| `vivgrid`            | LLM (Vivgrid 直连)           | [vivgrid.com](https://vivgrid.com)                                   |
| `moonshot`           | LLM (Kimi/Moonshot 直连)     | [platform.moonshot.cn](https://platform.moonshot.cn)                 |
| `minimax`            | LLM (Minimax 直连)           | [platform.minimaxi.com](https://platform.minimaxi.com)              |
| `avian`              | LLM (Avian 直连)             | [avian.io](https://avian.io)                                         |
| `mistral`            | LLM (Mistral 直连)           | [console.mistral.ai](https://console.mistral.ai)                    |
| `longcat`            | LLM (Longcat 直连)           | [longcat.ai](https://longcat.ai)                                     |
| `modelscope`         | LLM (ModelScope 直连)        | [modelscope.cn](https://modelscope.cn)                               |
| `mimo`               | LLM (小米 MiMo 直连)         | [platform.xiaomimimo.com](https://platform.xiaomimimo.com)           |

<a id="模型配置-model_list"></a>
### 模型配置 (model_list)

> **新功能！** PicoClaw 现在采用**以模型为中心**的配置方式。只需使用 `厂商/模型` 格式（如 `zhipu/glm-4.7`）即可添加新的 provider——**无需修改任何代码！**

如果你想看 agent 分发和轻量模型路由的完整示例，请看 [路由使用指南](routing-guide.zh.md)。

该设计同时支持**多 Agent 场景**，提供灵活的 Provider 选择：

- **不同 Agent 使用不同 Provider**：每个 Agent 可以使用自己的 LLM provider
- **模型回退（Fallback）**：配置主模型和备用模型，提高可靠性
- **负载均衡**：在多个 API 端点之间分配请求
- **集中化配置**：在一个地方管理所有 provider

#### 📋 所有支持的厂商

| 厂商                | `model` 前缀      | 默认 API Base                                       | 协议      | 获取 API Key                                                      |
| ------------------- | ----------------- | --------------------------------------------------- | --------- | ----------------------------------------------------------------- |
| **OpenAI**          | `openai/`         | `https://api.openai.com/v1`                         | OpenAI    | [获取密钥](https://platform.openai.com)                           |
| **Venice AI**       | `venice/`         | `https://api.venice.ai/api/v1`                      | OpenAI    | [获取密钥](https://venice.ai)                                     |
| **Anthropic**       | `anthropic/`      | `https://api.anthropic.com/v1`                      | Anthropic | [获取密钥](https://console.anthropic.com)                         |
| **智谱 AI (GLM)**   | `zhipu/`          | `https://open.bigmodel.cn/api/paas/v4`              | OpenAI    | [获取密钥](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) |
| **DeepSeek**        | `deepseek/`       | `https://api.deepseek.com/v1`                       | OpenAI    | [获取密钥](https://platform.deepseek.com)                         |
| **Google Gemini**   | `gemini/`         | `https://generativelanguage.googleapis.com/v1beta`  | Gemini    | [获取密钥](https://aistudio.google.com/api-keys)                  |
| **Groq**            | `groq/`           | `https://api.groq.com/openai/v1`                    | OpenAI    | [获取密钥](https://console.groq.com)                              |
| **Moonshot**        | `moonshot/`       | `https://api.moonshot.cn/v1`                        | OpenAI    | [获取密钥](https://platform.moonshot.cn)                          |
| **通义千问 (Qwen)** | `qwen/`           | `https://dashscope.aliyuncs.com/compatible-mode/v1` | OpenAI    | [获取密钥](https://dashscope.console.aliyun.com)                  |
| **NVIDIA**          | `nvidia/`         | `https://integrate.api.nvidia.com/v1`               | OpenAI    | [获取密钥](https://build.nvidia.com)                              |
| **Ollama**          | `ollama/`         | `http://localhost:11434/v1`                         | OpenAI    | 本地（无需密钥）                                                  |
| **LM Studio**       | `lmstudio/`       | `http://localhost:1234/v1`                          | OpenAI    | 可选（本地默认无需密钥）                                          |
| **OpenRouter**      | `openrouter/`     | `https://openrouter.ai/api/v1`                      | OpenAI    | [获取密钥](https://openrouter.ai/keys)                            |
| **LiteLLM Proxy**   | `litellm/`        | `http://localhost:4000/v1`                          | OpenAI    | 你的 LiteLLM 代理密钥                                             |
| **VLLM**            | `vllm/`           | `http://localhost:8000/v1`                          | OpenAI    | 本地                                                              |
| **Cerebras**        | `cerebras/`       | `https://api.cerebras.ai/v1`                        | OpenAI    | [获取密钥](https://cerebras.ai)                                   |
| **火山引擎（Doubao）** | `volcengine/`  | `https://ark.cn-beijing.volces.com/api/v3`          | OpenAI    | [获取密钥](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) |
| **神算云**          | `shengsuanyun/`   | `https://router.shengsuanyun.com/api/v1`            | OpenAI    | -                                                                 |
| **BytePlus**        | `byteplus/`       | `https://ark.ap-southeast.bytepluses.com/api/v3`    | OpenAI    | [获取密钥](https://www.byteplus.com)                              |
| **Vivgrid**         | `vivgrid/`        | `https://api.vivgrid.com/v1`                        | OpenAI    | [获取密钥](https://vivgrid.com)                                   |
| **LongCat**         | `longcat/`        | `https://api.longcat.chat/openai`                   | OpenAI    | [获取密钥](https://longcat.chat/platform)                         |
| **ModelScope (魔搭)**| `modelscope/`    | `https://api-inference.modelscope.cn/v1`            | OpenAI    | [获取 Token](https://modelscope.cn/my/tokens)                    |
| **小米 MiMo**       | `mimo/`           | `https://api.xiaomimimo.com/v1`                     | OpenAI    | [获取密钥](https://platform.xiaomimimo.com)                      |
| **Antigravity**     | `antigravity/`    | Google Cloud                                        | 自定义    | 仅 OAuth                                                          |
| **GitHub Copilot**  | `github-copilot/` | `localhost:4321`                                    | gRPC      | -                                                                 |

#### 基础配置示例

```json
{
  "model_list": [
    {
      "model_name": "ark-code-latest",
      "model": "volcengine/ark-code-latest",
      "api_keys": ["sk-your-api-key"]
    },
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_keys": ["sk-your-openai-key"]
    },
    {
      "model_name": "claude-sonnet-4.6",
      "model": "anthropic/claude-sonnet-4.6",
      "api_keys": ["sk-ant-your-key"]
    },
    {
      "model_name": "glm-4.7",
      "model": "zhipu/glm-4.7",
      "api_keys": ["your-zhipu-key"]
    }
  ],
  "agents": {
    "defaults": {
      "model_name": "gpt-5.4"
    }
  }
}
```

#### `model_list` 条目字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model_name` | string | 是 | 在 agent 配置中引用此模型的唯一名称 |
| `model` | string | 是 | 厂商/模型标识符（如 `openai/gpt-5.4`、`azure/gpt-5.4`、`anthropic/claude-sonnet-4.6`） |
| `api_keys` | string[] | 是* | 认证密钥。多个密钥可按请求轮换。本地 provider（Ollama、LM Studio、VLLM）不需要 |
| `api_base` | string | 否 | 覆盖默认的 API 端点 URL |
| `proxy` | string | 否 | 此模型条目的 HTTP 代理 URL |
| `user_agent` | string | 否 | 自定义 `User-Agent` 请求头（支持 OpenAI 兼容、Gemini、Anthropic 和 Azure provider） |
| `request_timeout` | int | 否 | 请求超时时间（秒），默认值因 provider 而异 |
| `max_tokens_field` | string | 否 | 覆盖请求体中 max tokens 的字段名（如 o1 模型使用 `max_completion_tokens`） |
| `thinking_level` | string | 否 | 扩展思考级别：`off`、`low`、`medium`、`high`、`xhigh` 或 `adaptive` |
| `extra_body` | object | 否 | 注入到每个请求体中的额外字段 |
| `custom_headers` | object | 否 | 注入到每个请求中的额外 HTTP 请求头（例如 `{"X-Source":"coding-plan"}`）。若键名与内置请求头同名，会覆盖内置值（如 `Authorization`、`User-Agent`、`Content-Type`、`Accept`）。 |
| `rpm` | int | 否 | 每分钟请求速率限制 |
| `fallbacks` | string[] | 否 | 自动故障转移的备用模型名称 |
| `enabled` | bool | 否 | 是否启用此模型条目（默认：`true`） |

#### 语音转录

你可以通过 `voice.model_name` 为语音转录指定一个专用模型。这样可以直接复用已经配置好的、支持音频输入的多模态 provider，而不必只依赖 Groq。

如果没有配置 `voice.model_name`，且存在 Groq API Key，PicoClaw 会继续回退到 Groq 转录。

```json
{
  "model_list": [
    {
      "model_name": "voice-gemini",
      "model": "gemini/gemini-2.5-flash",
      "api_keys": ["your-gemini-key"]
    }
  ],
  "voice": {
    "model_name": "voice-gemini",
    "echo_transcription": false
  },
  "providers": {
    "groq": {
      "api_key": "gsk_xxx"
    }
  }
}
```

#### 各厂商配置示例

**OpenAI**

```json
{
  "model_name": "gpt-5.4",
  "model": "openai/gpt-5.4",
  "api_keys": ["sk-..."]
}
```

**火山引擎（Doubao）**

```json
{
  "model_name": "ark-code-latest",
  "model": "volcengine/ark-code-latest",
  "api_keys": ["sk-..."]
}
```

**智谱 AI (GLM)**

```json
{
  "model_name": "glm-4.7",
  "model": "zhipu/glm-4.7",
  "api_keys": ["your-key"]
}
```

**DeepSeek**

```json
{
  "model_name": "deepseek-chat",
  "model": "deepseek/deepseek-chat",
  "api_keys": ["sk-..."]
}
```

**Anthropic (使用 OAuth)**

```json
{
  "model_name": "claude-sonnet-4.6",
  "model": "anthropic/claude-sonnet-4.6",
  "auth_method": "oauth"
}
```

> 运行 `picoclaw auth login --provider anthropic` 来设置 OAuth 凭证。

**Anthropic Messages API（原生格式）**

用于直接访问 Anthropic API 或仅支持 Anthropic 原生消息格式的自定义端点：

```json
{
  "model_name": "claude-opus-4-6",
  "model": "anthropic-messages/claude-opus-4-6",
  "api_keys": ["sk-ant-your-key"],
  "api_base": "https://api.anthropic.com"
}
```

> 使用 `anthropic-messages` 协议的场景：
> - 使用仅支持 Anthropic 原生 `/v1/messages` 端点的第三方代理（不支持 OpenAI 兼容的 `/v1/chat/completions`）
> - 连接到 MiniMax、Synthetic 等需要 Anthropic 原生消息格式的服务
> - 现有的 `anthropic` 协议返回 404 错误（说明端点不支持 OpenAI 兼容格式）
>
> **注意：** `anthropic` 协议使用 OpenAI 兼容格式（`/v1/chat/completions`），而 `anthropic-messages` 使用 Anthropic 原生格式（`/v1/messages`）。请根据端点支持的格式选择。

**Ollama (本地)**

```json
{
  "model_name": "llama3",
  "model": "ollama/llama3"
}
```

**LM Studio（本地）**

```json
{
  "model_name": "lmstudio-local",
  "model": "lmstudio/openai/gpt-oss-20b"
}
```

`api_base` 默认是 `http://localhost:1234/v1`。除非你在 LM Studio 侧启用了认证，否则不需要配置 API Key。
PicoClaw 向 LM Studio 的 OpenAI 兼容终结点发送请求，且将移除首个 `lmstudio/` 前缀，因此 `lmstudio/openai/gpt-oss-20b` 会发送 `openai/gpt-oss-20b`。

**自定义代理/API**

```json
{
  "model_name": "my-custom-model",
  "model": "openai/custom-model",
  "api_base": "https://my-proxy.com/v1",
  "api_keys": ["sk-..."],
  "user_agent": "MyApp/1.0",
  "request_timeout": 300
}
```

**LiteLLM Proxy**

```json
{
  "model_name": "lite-gpt4",
  "model": "litellm/lite-gpt4",
  "api_base": "http://localhost:4000/v1",
  "api_keys": ["sk-..."]
}
```

PicoClaw 在发送请求前仅去除外层 `litellm/` 前缀，因此 `litellm/lite-gpt4` 会发送 `lite-gpt4`，而 `litellm/openai/gpt-4o` 会发送 `openai/gpt-4o`。

#### 负载均衡

为同一个模型名称配置多个端点——PicoClaw 会自动在它们之间轮询：

```json
{
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_base": "https://api1.example.com/v1",
      "api_keys": ["sk-key1"]
    },
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_base": "https://api2.example.com/v1",
      "api_keys": ["sk-key2"]
    }
  ]
}
```

#### 自动模型失败切换（Cascade）

当你在 Agent 的模型设置里配置 `primary` + `fallbacks` 时，PicoClaw 已经支持自动失败切换。
运行时 fallback 链会在可重试错误时切到下一个候选（例如 HTTP `429`、配额/限流错误、超时错误）。
同时会对每个候选应用 cooldown，避免对刚失败的目标立即重试。

```json
{
  "model_list": [
    {
      "model_name": "qwen-main",
      "model": "openai/qwen3.5:cloud",
      "api_base": "https://api.example.com/v1",
      "api_keys": ["sk-main"]
    },
    {
      "model_name": "deepseek-backup",
      "model": "deepseek/deepseek-chat",
      "api_keys": ["sk-backup-1"]
    },
    {
      "model_name": "gemini-backup",
      "model": "gemini/gemini-2.5-flash",
      "api_keys": ["sk-backup-2"]
    }
  ],
  "agents": {
    "defaults": {
      "model": {
        "primary": "qwen-main",
        "fallbacks": ["deepseek-backup", "gemini-backup"]
      }
    }
  }
}
```

如果你在同一模型上启用了 key 级失败切换，PicoClaw 会先在该模型的多 key 候选间切换，再继续切到跨模型备选。

#### 从旧的 `providers` 配置迁移

旧的 `providers` 配置格式**已弃用**，V2 中已移除。现有 V0/V1 配置会自动迁移。

**旧配置（已弃用）：**

```json
{
  "providers": {
    "zhipu": {
      "api_key": "your-key",
      "api_base": "https://open.bigmodel.cn/api/paas/v4"
    }
  },
  "agents": {
    "defaults": {
      "provider": "zhipu",
      "model": "glm-4.7"
    }
  }
}
```

**新配置（推荐）：**

```json
{
  "version": 3,
  "model_list": [
    {
      "model_name": "glm-4.7",
      "model": "zhipu/glm-4.7",
      "api_keys": ["your-key"]
    }
  ],
  "agents": {
    "defaults": {
      "model_name": "glm-4.7"
    }
  }
}
```

详细的迁移指南请参考 [docs/migration/model-list-migration.md](../migration/model-list-migration.md)。

### Provider 架构

PicoClaw 按协议族路由 Provider：

- OpenAI 兼容协议：OpenRouter、OpenAI 兼容网关、Groq、智谱、vLLM 风格端点。
- Gemini 原生协议：Google Gemini 通过原生 `models/*:generateContent` 和 `models/*:streamGenerateContent` 端点接入。
- Anthropic 协议：Claude 原生 API 行为。
- Codex/OAuth 路径：OpenAI OAuth/Token 认证路由。

这使得运行时保持轻量，同时让新的 OpenAI 兼容后端基本只需配置操作（`api_base` + `api_keys`）。

<details>
<summary><b>智谱 (Zhipu) 配置示例</b></summary>

**1. 获取 API key 和 base URL**

- 获取 [API key](https://bigmodel.cn/usercenter/proj-mgmt/apikeys)

**2. 配置**

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model_name": "glm-4.7",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "providers": {
    "zhipu": {
      "api_key": "Your API Key",
      "api_base": "https://open.bigmodel.cn/api/paas/v4"
    }
  }
}
```

**3. 运行**

```bash
picoclaw agent -m "你好"
```

</details>

<details>
<summary><b>完整配置示例</b></summary>

```json
{
  "agents": {
    "defaults": {
      "model_name": "anthropic/claude-opus-4-5"
    }
  },
  "session": {
    "dm_scope": "per-channel-peer"
  },
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-xxx"
    },
    "groq": {
      "api_key": "gsk_xxx"
    }
  },
  "voice": {
    "model_name": "voice-gemini",
    "echo_transcription": false
  },
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "token": "123456:ABC...",
      "allow_from": ["123456789"]
    },
    "discord": {
      "enabled": true,
      "type": "discord",
      "token": "",
      "allow_from": [""]
    },
    "whatsapp": {
      "enabled": false,
      "type": "whatsapp",
      "bridge_url": "ws://localhost:3001",
      "use_native": false,
      "session_store_path": "",
      "allow_from": []
    },
    "feishu": {
      "enabled": false,
      "type": "feishu",
      "app_id": "cli_xxx",
      "app_secret": "xxx",
      "encrypt_key": "",
      "verification_token": "",
      "allow_from": []
    },
    "qq": {
      "enabled": false,
      "type": "qq",
      "app_id": "",
      "app_secret": "",
      "allow_from": []
    }
  },
  "tools": {
    "web": {
      "brave": {
        "enabled": false,
        "api_key": "BSA...",
        "max_results": 5
      },
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      },
      "perplexity": {
        "enabled": false,
        "api_key": "",
        "max_results": 5
      },
      "searxng": {
        "enabled": false,
        "base_url": "http://localhost:8888",
        "max_results": 5
      }
    },
    "cron": {
      "exec_timeout_minutes": 5
    }
  },
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

</details>

---

## 📝 API Key 对比

| 服务 | 价格 | 适用场景 |
| --- | --- | --- |
| **OpenRouter** | 免费: 200K tokens/月 | 多模型聚合 (Claude, GPT-4 等) |
| **火山引擎 CodingPlan** | ¥9.9/首月 | 最适合国内用户，多种 SOTA 模型（豆包、DeepSeek 等） |
| **智谱 (Zhipu)** | 免费: 200K tokens/月 | 适合中国用户 |
| **Brave Search** | $5/1000 次查询 | 网络搜索功能 |
| **SearXNG** | 免费（自建） | 隐私优先的元搜索引擎（70+ 搜索引擎） |
| **Groq** | 免费额度可用 | 极速推理 (Llama, Mixtral) |
| **Cerebras** | 免费额度可用 | 极速推理 (Llama, Qwen 等) |
| **LongCat** | 免费: 最多 5M tokens/天 | 极速推理 |
| **ModelScope (魔搭)** | 免费: 2000 次请求/天 | 推理 (Qwen, GLM, DeepSeek 等) |
