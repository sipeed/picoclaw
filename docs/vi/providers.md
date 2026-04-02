# 🔌 Nhà Cung Cấp và Cấu Hình Mô Hình

> Quay lại [README](../../README.vi.md)

### Nhà Cung Cấp

> [!NOTE]
> Groq cung cấp chuyển đổi giọng nói miễn phí qua Whisper. Nếu được cấu hình, tin nhắn âm thanh từ bất kỳ kênh nào sẽ được tự động chuyển đổi ở cấp agent.

| Provider     | Purpose                                 | Get API Key                                                  |
| ------------ | --------------------------------------- | ------------------------------------------------------------ |
| `gemini`     | LLM (Gemini direct)                     | [aistudio.google.com](https://aistudio.google.com)           |
| `zhipu`      | LLM (Zhipu direct)                      | [bigmodel.cn](https://bigmodel.cn)                           |
| `volcengine` | LLM(Volcengine direct)                  | [volcengine.com](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw)                 |
| `openrouter` | LLM (recommended, access to all models) | [openrouter.ai](https://openrouter.ai)                       |
| `anthropic`  | LLM (Claude direct)                     | [console.anthropic.com](https://console.anthropic.com)       |
| `openai`     | LLM (GPT direct)                        | [platform.openai.com](https://platform.openai.com)           |
| `deepseek`   | LLM (DeepSeek direct)                   | [platform.deepseek.com](https://platform.deepseek.com)       |
| `qwen`       | LLM (Qwen direct)                       | [dashscope.console.aliyun.com](https://dashscope.console.aliyun.com) |
| `groq`       | LLM + **Voice transcription** (Whisper) | [console.groq.com](https://console.groq.com)                 |
| `cerebras`   | LLM (Cerebras direct)                   | [cerebras.ai](https://cerebras.ai)                           |
| `vivgrid`    | LLM (Vivgrid direct)                    | [vivgrid.com](https://vivgrid.com)                           |
| `moonshot`   | LLM (Kimi/Moonshot direct)              | [platform.moonshot.cn](https://platform.moonshot.cn)         |
| `minimax`    | LLM (Minimax direct)                    | [platform.minimaxi.com](https://platform.minimaxi.com)      |
| `avian`      | LLM (Avian direct)                      | [avian.io](https://avian.io)                                 |
| `mistral`    | LLM (Mistral direct)                    | [console.mistral.ai](https://console.mistral.ai)            |
| `longcat`    | LLM (Longcat direct)                    | [longcat.ai](https://longcat.ai)                             |
| `modelscope` | LLM (ModelScope direct)                 | [modelscope.cn](https://modelscope.cn)                       |

### Cấu Hình Mô Hình (model_list)

> **Có gì mới?** PicoClaw hiện sử dụng cách tiếp cận cấu hình **tập trung vào mô hình**. Chỉ cần chỉ định định dạng `vendor/model` (ví dụ: `zhipu/glm-4.7`) để thêm provider mới — **không cần thay đổi code!**

Thiết kế này cũng cho phép **hỗ trợ đa agent** với lựa chọn provider linh hoạt:

- **Agent khác nhau, provider khác nhau**: Mỗi agent có thể sử dụng provider LLM riêng
- **Fallback mô hình**: Cấu hình mô hình chính và dự phòng cho khả năng phục hồi
- **Cân bằng tải**: Phân phối yêu cầu qua nhiều endpoint
- **Cấu hình tập trung**: Quản lý tất cả provider tại một nơi

#### 📋 Tất Cả Vendor Được Hỗ Trợ

| Vendor              | `model` Prefix    | Default API Base                                    | Protocol  | API Key                                                          |
| ------------------- | ----------------- |-----------------------------------------------------| --------- | ---------------------------------------------------------------- |
| **OpenAI**          | `openai/`         | `https://api.openai.com/v1`                         | OpenAI    | [Get Key](https://platform.openai.com)                           |
| **Anthropic**       | `anthropic/`      | `https://api.anthropic.com/v1`                      | Anthropic | [Get Key](https://console.anthropic.com)                         |
| **智谱 AI (GLM)**   | `zhipu/`          | `https://open.bigmodel.cn/api/paas/v4`              | OpenAI    | [Get Key](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) |
| **DeepSeek**        | `deepseek/`       | `https://api.deepseek.com/v1`                       | OpenAI    | [Get Key](https://platform.deepseek.com)                         |
| **Google Gemini**   | `gemini/`         | `https://generativelanguage.googleapis.com/v1beta`  | OpenAI    | [Get Key](https://aistudio.google.com/api-keys)                  |
| **Groq**            | `groq/`           | `https://api.groq.com/openai/v1`                    | OpenAI    | [Get Key](https://console.groq.com)                              |
| **Moonshot**        | `moonshot/`       | `https://api.moonshot.cn/v1`                        | OpenAI    | [Get Key](https://platform.moonshot.cn)                          |
| **通义千问 (Qwen)** | `qwen/`           | `https://dashscope.aliyuncs.com/compatible-mode/v1` | OpenAI    | [Get Key](https://dashscope.console.aliyun.com)                  |
| **NVIDIA**          | `nvidia/`         | `https://integrate.api.nvidia.com/v1`               | OpenAI    | [Get Key](https://build.nvidia.com)                              |
| **Ollama**          | `ollama/`         | `http://localhost:11434/v1`                         | OpenAI    | Local (no key needed)                                            |
| **OpenRouter**      | `openrouter/`     | `https://openrouter.ai/api/v1`                      | OpenAI    | [Get Key](https://openrouter.ai/keys)                            |
| **LiteLLM Proxy**   | `litellm/`        | `http://localhost:4000/v1`                          | OpenAI    | Your LiteLLM proxy key                                            |
| **VLLM**            | `vllm/`           | `http://localhost:8000/v1`                          | OpenAI    | Local                                                            |
| **Cerebras**        | `cerebras/`       | `https://api.cerebras.ai/v1`                        | OpenAI    | [Get Key](https://cerebras.ai)                                   |
| **VolcEngine (Doubao)** | `volcengine/`     | `https://ark.cn-beijing.volces.com/api/v3`          | OpenAI    | [Get Key](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw)                        |
| **神算云**          | `shengsuanyun/`   | `https://router.shengsuanyun.com/api/v1`            | OpenAI    | -                                                                |
| **BytePlus**        | `byteplus/`       | `https://ark.ap-southeast.bytepluses.com/api/v3`    | OpenAI    | [Get Key](https://www.byteplus.com)                        |
| **Vivgrid**         | `vivgrid/`        | `https://api.vivgrid.com/v1`                        | OpenAI    | [Get Key](https://vivgrid.com)                                   |
| **LongCat**         | `longcat/`        | `https://api.longcat.chat/openai`                   | OpenAI    | [Get Key](https://longcat.chat/platform)                         |
| **ModelScope (魔搭)**| `modelscope/`    | `https://api-inference.modelscope.cn/v1`            | OpenAI    | [Get Token](https://modelscope.cn/my/tokens)                     |
| **Antigravity**     | `antigravity/`    | Google Cloud                                        | Custom    | OAuth only                                                       |
| **GitHub Copilot**  | `github-copilot/` | `localhost:4321`                                    | gRPC      | -                                                                |

#### Cấu Hình Cơ Bản

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

#### Các trường entry `model_list`

| Trường | Kiểu | Bắt buộc | Mô tả |
|--------|------|----------|------|
| `model_name` | string | Có | Tên duy nhất để tham chiếu model này trong cấu hình agent |
| `model` | string | Có | Định danh nhà cung cấp/model (ví dụ: `openai/gpt-5.4`, `azure/gpt-5.4`, `anthropic/claude-sonnet-4.6`) |
| `api_keys` | string[] | Có* | Khóa API xác thực. Nhiều khóa cho phép xoay vòng theo yêu cầu. Không cần thiết cho provider nội bộ (Ollama, LM Studio, VLLM) |
| `api_base` | string | Không | Ghi đè URL endpoint API mặc định |
| `proxy` | string | Không | URL proxy HTTP cho entry model này |
| `user_agent` | string | Không | Header `User-Agent` tùy chỉnh gửi với yêu cầu API (được hỗ trợ bởi provider OpenAI-compatible, Anthropic và Azure) |
| `request_timeout` | int | Không | Timeout yêu cầu tính bằng giây (mặc định khác nhau tùy provider) |
| `max_tokens_field` | string | Không | Ghi đè tên trường max tokens trong request body (ví dụ: `max_completion_tokens` cho model o1) |
| `thinking_level` | string | Không | Mức độ tư duy mở rộng: `off`, `low`, `medium`, `high`, `xhigh` hoặc `adaptive` |
| `extra_body` | object | Không | Các trường bổ sung để chèn vào mỗi request body |
| `rpm` | int | Không | Giới hạn tốc độ yêu cầu mỗi phút |
| `fallbacks` | string[] | Không | Tên model dự phòng cho failover tự động |
| `enabled` | bool | Không | Kích hoạt hay vô hiệu hóa entry model này (mặc định: `true`) |

#### Ví Dụ Theo Vendor

**OpenAI**

```json
{
  "model_name": "gpt-5.4",
  "model": "openai/gpt-5.4",
  "api_keys": ["sk-..."]
}
```

**VolcEngine (Doubao)**

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

**Anthropic (với API key)**

```json
{
  "model_name": "claude-sonnet-4.6",
  "model": "anthropic/claude-sonnet-4.6",
  "api_keys": ["sk-ant-your-key"]
}
```

> Chạy `picoclaw auth login --provider anthropic` để dán API token.

**Anthropic Messages API (định dạng native)**

Để truy cập trực tiếp API Anthropic hoặc endpoint tùy chỉnh chỉ hỗ trợ định dạng message native của Anthropic:

```json
{
  "model_name": "claude-opus-4-6",
  "model": "anthropic-messages/claude-opus-4-6",
  "api_keys": ["sk-ant-your-key"],
  "api_base": "https://api.anthropic.com"
}
```

> Sử dụng giao thức `anthropic-messages` khi:
> - Sử dụng proxy bên thứ ba chỉ hỗ trợ endpoint native `/v1/messages` của Anthropic (không tương thích OpenAI `/v1/chat/completions`)
> - Kết nối đến dịch vụ như MiniMax, Synthetic yêu cầu định dạng message native của Anthropic
> - Giao thức `anthropic` hiện tại trả về lỗi 404 (cho thấy endpoint không hỗ trợ định dạng tương thích OpenAI)
>
> **Lưu ý:** Giao thức `anthropic` sử dụng định dạng tương thích OpenAI (`/v1/chat/completions`), trong khi `anthropic-messages` sử dụng định dạng native của Anthropic (`/v1/messages`). Chọn dựa trên định dạng endpoint hỗ trợ.

**Ollama (local)**

```json
{
  "model_name": "llama3",
  "model": "ollama/llama3"
}
```

**Proxy/API Tùy Chỉnh**

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

PicoClaw chỉ loại bỏ tiền tố ngoài `litellm/` trước khi gửi yêu cầu, nên alias proxy như `litellm/lite-gpt4` gửi `lite-gpt4`, trong khi `litellm/openai/gpt-4o` gửi `openai/gpt-4o`.

#### Cân Bằng Tải

Cấu hình nhiều endpoint cho cùng tên mô hình — PicoClaw sẽ tự động round-robin giữa chúng:

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

#### Di Chuyển Từ Cấu Hình Legacy `providers`

Cấu hình `providers` cũ đã **bị deprecated** và đã được loại bỏ trong V2. Các cấu hình V0/V1 hiện có sẽ được tự động migrate.

**Cấu hình cũ (ngừng hỗ trợ):**

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

**Cấu hình mới (khuyến nghị):**

```json
{
  "version": 2,
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

Để xem hướng dẫn di chuyển chi tiết, xem [migration/model-list-migration.md](../migration/model-list-migration.md).

### Kiến Trúc Provider

PicoClaw định tuyến provider theo họ giao thức:

- Giao thức tương thích OpenAI: OpenRouter, gateway tương thích OpenAI, Groq, Zhipu, và endpoint kiểu vLLM.
- Giao thức Anthropic: Hành vi API native của Claude.
- Đường dẫn Codex/OAuth: Tuyến xác thực OAuth/token của OpenAI.

Điều này giữ runtime nhẹ trong khi làm cho backend tương thích OpenAI mới chủ yếu là thao tác cấu hình (`api_base` + `api_keys`).

<details>
<summary><b>Zhipu</b></summary>

**1. Lấy API key và URL base**

* Lấy [API key](https://bigmodel.cn/usercenter/proj-mgmt/apikeys)

**2. Cấu hình**

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

**3. Chạy**

```bash
picoclaw agent -m "Hello"
```

</details>

<details>
<summary><b>Ví dụ cấu hình đầy đủ</b></summary>

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
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "123456:ABC...",
      "allow_from": ["123456789"]
    },
    "discord": {
      "enabled": true,
      "token": "",
      "allow_from": [""]
    },
    "whatsapp": {
      "enabled": false,
      "bridge_url": "ws://localhost:3001",
      "use_native": false,
      "session_store_path": "",
      "allow_from": []
    },
    "feishu": {
      "enabled": false,
      "app_id": "cli_xxx",
      "app_secret": "xxx",
      "encrypt_key": "",
      "verification_token": "",
      "allow_from": []
    },
    "qq": {
      "enabled": false,
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

## 📝 So Sánh API Key

| Service          | Pricing                  | Use Case                              |
| ---------------- | ------------------------ | ------------------------------------- |
| **OpenRouter**   | Free: 200K tokens/month  | Multiple models (Claude, GPT-4, etc.) |
| **Volcengine CodingPlan** | ¥9.9/first month | Best for Chinese users, multiple SOTA models (Doubao, DeepSeek, etc.) |
| **Zhipu**        | Free: 200K tokens/month  | Suitable for Chinese users                |
| **Brave Search** | $5/1000 queries          | Web search functionality              |
| **SearXNG**      | Free (self-hosted)       | Privacy-focused metasearch (70+ engines) |
| **Groq**         | Free tier available      | Fast inference (Llama, Mixtral)       |
| **Cerebras**     | Free tier available      | Fast inference (Llama, Qwen, etc.)    |
| **LongCat**      | Free: up to 5M tokens/day | Fast inference                       |
| **ModelScope**   | Free: 2000 requests/day  | Inference (Qwen, GLM, DeepSeek, etc.) |

---

<div align="center">
  <img src="assets/logo.jpg" alt="PicoClaw Meme" width="512">
</div>
