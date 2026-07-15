# 🔌 服務供應商與模型設定

> 回到 [README](../project/README.zh-tw.md)

### 服務供應商

> [!NOTE]
> 語音轉錄可以使用 `voice.model_name` 指定已設定的多模態模型。未設定語音模型時，仍可使用 Groq Whisper 作為備援。

| 服務供應商   | 用途                                    | 取得 API 金鑰                                               |
| ------------ | --------------------------------------- | ------------------------------------------------------------ |
| `gemini`     | LLM（直接使用 Gemini）                  | [aistudio.google.com](https://aistudio.google.com)           |
| `zhipu`      | LLM（直接使用 Zhipu）                   | [bigmodel.cn](https://bigmodel.cn)                           |
| `zai-coding` | LLM（Z.AI Coding Plan）                 | [z.ai](https://z.ai/manage-apikey/apikey-list)               |
| `volcengine` | LLM（直接使用 VolcEngine）              | [volcengine.com](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) |
| `openrouter` | LLM（建議使用，可存取所有模型）         | [openrouter.ai](https://openrouter.ai)                       |
| `anthropic`  | LLM（直接使用 Claude）                  | [console.anthropic.com](https://console.anthropic.com)       |
| `openai`     | LLM（直接使用 GPT）                     | [platform.openai.com](https://platform.openai.com)           |
| `venice`     | LLM（直接使用 Venice AI）               | [venice.ai](https://venice.ai)                               |
| `nearai`     | LLM（NEAR AI Cloud TEE 推論）           | [near.ai](https://near.ai)                                   |
| `deepseek`   | LLM（直接使用 DeepSeek）                | [platform.deepseek.com](https://platform.deepseek.com)       |
| `qwen`       | LLM（直接使用 Qwen）                    | [dashscope.console.aliyun.com](https://dashscope.console.aliyun.com) |
| `groq`       | LLM + **語音轉錄**（Whisper）           | [console.groq.com](https://console.groq.com)                 |
| `cerebras`   | LLM（直接使用 Cerebras）                | [cerebras.ai](https://cerebras.ai)                           |
| `vivgrid`    | LLM（直接使用 Vivgrid）                 | [vivgrid.com](https://vivgrid.com)                           |
| `nvidia`     | LLM（NVIDIA NIM）                       | [build.nvidia.com](https://build.nvidia.com)                 |
| `moonshot`   | LLM（直接使用 Kimi／Moonshot）          | [platform.moonshot.cn](https://platform.moonshot.cn)         |
| `minimax`    | LLM（直接使用 Minimax）                 | [platform.minimaxi.com](https://platform.minimaxi.com)       |
| `avian`      | LLM（直接使用 Avian）                   | [avian.io](https://avian.io)                                 |
| `mistral`    | LLM（直接使用 Mistral）                 | [console.mistral.ai](https://console.mistral.ai)             |
| `longcat`    | LLM（直接使用 Longcat）                 | [longcat.ai](https://longcat.ai)                             |
| `modelscope` | LLM（直接使用 ModelScope）              | [modelscope.cn](https://modelscope.cn)                       |
| `mimo`       | LLM（直接使用 Xiaomi MiMo）             | [platform.xiaomimimo.com](https://platform.xiaomimimo.com)   |

### 模型設定（model_list）

> **新增內容：** PicoClaw 現在優先採用明確的 `provider` 加原生 `model` 設定（例如 `"provider": "zhipu", "model": "glm-4.7"`）。省略 `provider` 時，為了維持相容性，仍支援舊版單一欄位 `provider/model` 格式。

代理人分派及輕量模型路由範例，請參閱 [路由指南](routing-guide.md)。

這項設計也能彈性選擇服務供應商，提供**多代理人功能**：

- **不同代理人使用不同服務供應商**：每個代理人都能使用各自的 LLM 服務供應商
- **模型備援**：設定主要與備援模型，提高容錯能力
- **負載平衡**：將要求分配至多個端點
- **集中設定**：在同一處管理所有服務供應商

#### 📋 所有支援的廠商

| 廠商                | `provider` 值     | 預設 API 基底網址                                    | 通訊協定  | API 金鑰                                                          |
| ------------------- | ----------------- |-----------------------------------------------------| --------- | ---------------------------------------------------------------- |
| **OpenAI**          | `openai`          | `https://api.openai.com/v1`                         | OpenAI    | [取得金鑰](https://platform.openai.com)                           |
| **Venice AI**       | `venice`          | `https://api.venice.ai/api/v1`                      | OpenAI    | [取得金鑰](https://venice.ai)                                     |
| **NEAR AI Cloud**   | `nearai`          | `https://cloud-api.near.ai/v1`                      | OpenAI    | [取得金鑰](https://near.ai)                                       |
| **Anthropic**       | `anthropic`       | `https://api.anthropic.com/v1`                      | Anthropic | [取得金鑰](https://console.anthropic.com)                         |
| **智譜 AI（GLM）**  | `zhipu`           | `https://open.bigmodel.cn/api/paas/v4`              | OpenAI    | [取得金鑰](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) |
| **Z.AI Coding Plan** | `openai`         | `https://api.z.ai/api/coding/paas/v4`               | OpenAI    | [取得金鑰](https://z.ai/manage-apikey/apikey-list)                |
| **DeepSeek**        | `deepseek`        | `https://api.deepseek.com/v1`                       | OpenAI    | [取得金鑰](https://platform.deepseek.com)                         |
| **Google Gemini**   | `gemini`          | `https://generativelanguage.googleapis.com/v1beta`  | Gemini    | [取得金鑰](https://aistudio.google.com/api-keys)                  |
| **Groq**            | `groq`            | `https://api.groq.com/openai/v1`                    | OpenAI    | [取得金鑰](https://console.groq.com)                              |
| **Moonshot**        | `moonshot`        | `https://api.moonshot.cn/v1`                        | OpenAI    | [取得金鑰](https://platform.moonshot.cn)                          |
| **通義千問（Qwen）** | `qwen`           | `https://dashscope.aliyuncs.com/compatible-mode/v1` | OpenAI    | [取得金鑰](https://dashscope.console.aliyun.com)                  |
| **NVIDIA**          | `nvidia`          | `https://integrate.api.nvidia.com/v1`               | OpenAI    | [取得金鑰](https://build.nvidia.com)                              |
| **Ollama**          | `ollama`          | `http://localhost:11434/v1`                         | OpenAI    | 本機（不需要金鑰）                                                |
| **LM Studio**       | `lmstudio`        | `http://localhost:1234/v1`                          | OpenAI    | 選用（本機預設不需要金鑰）                                       |
| **OpenRouter**      | `openrouter`      | `https://openrouter.ai/api/v1`                      | OpenAI    | [取得金鑰](https://openrouter.ai/keys)                            |
| **LiteLLM Proxy**   | `litellm`         | `http://localhost:4000/v1`                          | OpenAI    | LiteLLM Proxy 金鑰                                                |
| **VLLM**            | `vllm`            | `http://localhost:8000/v1`                          | OpenAI    | 本機                                                             |
| **Cerebras**        | `cerebras`        | `https://api.cerebras.ai/v1`                        | OpenAI    | [取得金鑰](https://cerebras.ai)                                   |
| **VolcEngine（Doubao）** | `volcengine` | `https://ark.cn-beijing.volces.com/api/v3`          | OpenAI    | [取得金鑰](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) |
| **神算雲**          | `shengsuanyun`    | `https://router.shengsuanyun.com/api/v1`            | OpenAI    | -                                                                |
| **BytePlus**        | `byteplus`        | `https://ark.ap-southeast.bytepluses.com/api/v3`    | OpenAI    | [取得金鑰](https://www.byteplus.com)                              |
| **Vivgrid**         | `vivgrid`         | `https://api.vivgrid.com/v1`                        | OpenAI    | [取得金鑰](https://vivgrid.com)                                   |
| **LongCat**         | `longcat`         | `https://api.longcat.chat/openai`                   | OpenAI    | [取得金鑰](https://longcat.chat/platform)                         |
| **ModelScope（魔搭）** | `modelscope`   | `https://api-inference.modelscope.cn/v1`            | OpenAI    | [取得 Token](https://modelscope.cn/my/tokens)                     |
| **Xiaomi MiMo**     | `mimo`            | `https://api.xiaomimimo.com/v1`                     | OpenAI    | [取得金鑰](https://platform.xiaomimimo.com)                       |
| **Azure OpenAI**    | `azure`           | `https://{resource}.openai.azure.com`               | Azure     | [取得金鑰](https://portal.azure.com)                              |
| **Antigravity**     | `antigravity`     | Google Cloud                                        | 自訂      | 僅限 OAuth                                                       |
| **GitHub Copilot**  | `github-copilot`  | `localhost:4321`                                    | gRPC      | -                                                                |

#### 基本設定

```json
{
  "model_list": [
    {
      "model_name": "ark-code-latest",
      "provider": "volcengine",
      "model": "ark-code-latest",
      "api_keys": ["sk-your-api-key"]
    },
    {
      "model_name": "gpt-5.4",
      "provider": "openai",
      "model": "gpt-5.4",
      "api_keys": ["sk-your-openai-key"]
    },
    {
      "model_name": "claude-sonnet-4.6",
      "provider": "anthropic",
      "model": "claude-sonnet-4.6",
      "api_keys": ["sk-ant-your-key"]
    },
    {
      "model_name": "glm-4.7",
      "provider": "zhipu",
      "model": "glm-4.7",
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

#### `model_list` 設定欄位

| 欄位 | 型別 | 必填 | 說明 |
|------|------|------|------|
| `model_name` | string | 是 | 在代理人設定中參照此模型時使用的唯一名稱 |
| `provider` | string | 否 | 建議使用的服務供應商識別碼。設定後，PicoClaw 會將 `model` 原樣傳送給該服務供應商 |
| `model` | string | 是 | 設定 `provider` 時使用的原生模型 ID。省略 `provider` 時，仍支援舊版 `provider/model` 格式 |
| `api_keys` | string[] | 是* | 用於驗證的一或多組 API 金鑰。提供多組金鑰即可逐次輪替；本機服務供應商（Ollama、LM Studio、VLLM）不需要 |
| `api_base` | string | 否 | 覆寫預設 API 端點 URL |
| `proxy` | string | 否 | 此模型設定使用的 HTTP Proxy URL |
| `user_agent` | string | 否 | 隨 API 要求傳送的自訂 `User-Agent` 標頭（OpenAI 相容、Gemini、Anthropic 及 Azure 服務供應商皆支援） |
| `request_timeout` | int | 否 | 要求逾時秒數（預設值依服務供應商而異） |
| `max_tokens_field` | string | 否 | 覆寫要求本文中的 Token 上限欄位名稱（例如 o1 模型使用的 `max_completion_tokens`） |
| `thinking_level` | string | 否 | 延伸思考等級：`off`、`low`、`medium`、`high`、`xhigh` 或 `adaptive` |
| `tool_schema_transform` | string | 否 | 選用的工具參數結構描述相容性轉換。預設為停用；支援的值：`simple` |
| `extra_body` | object | 否 | 注入每個要求本文的其他欄位 |
| `custom_headers` | object | 否 | 注入每個要求的其他 HTTP 標頭（例如 `{"X-Source":"coding-plan"}`）。若鍵名與內建標頭相同，自訂值會覆寫內建值（例如 `Authorization`、`User-Agent`、`Content-Type`、`Accept`） |
| `streaming.enabled` | bool | 否 | 為此模型設定啟用服務供應商的串流功能。預設為 `false`，而且作用中頻道的 `settings.streaming.enabled` 也必須設為 `true` |
| `rpm` | int | 否 | 每分鐘要求速率限制 |
| `fallbacks` | string[] | 否 | 自動容錯移轉使用的備援模型名稱 |
| `enabled` | bool | 否 | 是否啟用此模型設定（預設：`true`） |

停用串流時，請省略 `streaming` 區塊。自動產生或手動撰寫設定時，不需要另外寫入 `"streaming": {"enabled": false}`。

`extra_body` 特別適合在 OpenAI 相容的語音路由中設定模型專用 TTS 欄位，例如自訂 `voice` 名稱或 `response_format: "mp3"`。

#### 工具結構描述相容性

PicoClaw 預設會原樣轉送工具的 JSON Schema。

部分服務供應商不接受工具宣告中的進階 JSON Schema 功能，例如 `$ref`、`$defs`、`anyOf`、`oneOf`、`allOf`、`pattern`，或數值／字串限制。針對這類模型，可以在個別模型設定中使用 `tool_schema_transform`，選擇套用相容性轉換。

上游服務供應商只接受較保守的函式結構描述子集時，請使用 `simple`：

```json
{
  "model_name": "gemini-2.5-flash-safe-tools",
  "provider": "gemini",
  "model": "gemini-2.5-flash",
  "api_keys": ["your-gemini-key"],
  "tool_schema_transform": "simple"
}
```

注意事項：

- 預設停用此功能。省略 `tool_schema_transform` 時，PicoClaw 會傳送原始工具結構描述。
- 此設定以單一模型為單位，因此只需為有需要的服務供應商啟用。

#### 服務供應商／模型解析方式

PicoClaw 使用下列規則解析 `provider` 及執行階段模型 ID：

- 如果設定 `provider`，則會原樣使用 `model`。
- 如果省略 `provider`，PicoClaw 會將 `model` 中第一個 `/` 之前的部分視為服務供應商，其餘部分則視為執行階段模型 ID。

範例：

| 設定 | 解析後的服務供應商 | 傳送至上游的模型 |
| --- | --- | --- |
| `"provider": "openai", "model": "gpt-5.4"` | `openai` | `gpt-5.4` |
| `"model": "openai/gpt-5.4"` | `openai` | `gpt-5.4` |
| `"provider": "openrouter", "model": "openai/gpt-5.4"` | `openrouter` | `openai/gpt-5.4` |
| `"model": "openrouter/openai/gpt-5.4"` | `openrouter` | `openai/gpt-5.4` |

#### 語音轉錄

可以使用 `voice.model_name` 設定專用的音訊轉錄模型。如此便能重複使用支援音訊輸入的既有多模態服務供應商，不必只仰賴 Groq。

如果未設定 `voice.model_name`，但有可用的 Groq API 金鑰，PicoClaw 仍會改用 Groq 進行轉錄。

```json
{
  "model_list": [
    {
      "model_name": "voice-gemini",
      "provider": "gemini",
      "model": "gemini-2.5-flash",
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

#### 語音合成

可以使用 `voice.tts_model_name` 設定專用的文字轉語音模型。
如果服務供應商需要模型專用的 TTS 要求欄位，請將這些欄位放入
`model_list[].extra_body`。

以下範例使用 OpenRouter `microsoft/mai-voice-2`：

```json
{
  "model_list": [
    {
      "model_name": "mai-voice-2",
      "provider": "openrouter",
      "model": "microsoft/mai-voice-2",
      "api_base": "https://openrouter.ai/api/v1",
      "extra_body": {
        "voice": "en-US-Harper:MAI-Voice-2",
        "response_format": "mp3"
      },
      "api_keys": ["sk-or-your-openrouter-key"]
    }
  ],
  "voice": {
    "tts_model_name": "mai-voice-2"
  }
}
```

重要注意事項：

- 此設定仍使用與 OpenAI 相容的 `/audio/speech` 路由。
- 預設 TTS 要求使用 `voice: alloy` 及 `response_format: opus`。
- 如果選用的 TTS 模型需要不同值，請使用 `extra_body` 覆寫這些預設值。

#### 各廠商設定範例

**OpenAI**

```json
{
  "model_name": "gpt-5.4",
  "provider": "openai",
  "model": "gpt-5.4",
  "api_keys": ["sk-..."]
}
```

**NEAR AI Cloud**

```json
{
  "model_name": "nearai-glm",
  "provider": "nearai",
  "model": "zai-org/GLM-5.1-FP8",
  "api_keys": ["your-nearai-api-key"]
}
```

**VolcEngine（Doubao）**

```json
{
  "model_name": "ark-code-latest",
  "provider": "volcengine",
  "model": "ark-code-latest",
  "api_keys": ["sk-..."]
}
```

**智譜 AI（GLM）**

```json
{
  "model_name": "glm-4.7",
  "provider": "zhipu",
  "model": "glm-4.7",
  "api_keys": ["your-key"]
}
```

**Z.AI Coding Plan（GLM）**
> Z.AI 與智譜 AI 是同一服務供應商旗下的兩個品牌。使用 Z.AI Coding Plan 時，請依下列方式使用 `openai` 模型鍵與 API 基底網址，不要使用 zhipu 設定。
```json
{
  "model_name": "glm-4.7",
  "provider": "openai",
  "model": "glm-4.7",
  "api_keys": ["your-z.ai-key"],
  "api_base": "https://api.z.ai/api/coding/paas/v4"
}
```

**DeepSeek**

```json
{
  "model_name": "deepseek-chat",
  "provider": "deepseek",
  "model": "deepseek-chat",
  "api_keys": ["sk-..."]
}
```

**Anthropic（使用 API 金鑰）**

```json
{
  "model_name": "claude-sonnet-4.6",
  "provider": "anthropic",
  "model": "claude-sonnet-4.6",
  "api_keys": ["sk-ant-your-key"]
}
```

> 執行 `picoclaw auth login --provider anthropic`，再貼上 API Token。

**Anthropic Messages API（原生格式）**

如需直接存取 Anthropic API，或使用只支援 Anthropic 原生訊息格式的自訂端點：

```json
{
  "model_name": "claude-opus-4-6",
  "provider": "anthropic-messages",
  "model": "claude-opus-4-6",
  "api_keys": ["sk-ant-your-key"],
  "api_base": "https://api.anthropic.com"
}
```

> 下列情況請使用 `anthropic-messages` 通訊協定：
> - 使用只支援 Anthropic 原生 `/v1/messages` 端點的第三方 Proxy（不支援 OpenAI 相容的 `/v1/chat/completions`）
> - 連線至 MiniMax、Synthetic 等需要 Anthropic 原生訊息格式的服務
> - 既有 `anthropic` 通訊協定傳回 404 錯誤（表示該端點不支援 OpenAI 相容格式）
>
> **注意：** `anthropic` 通訊協定使用 OpenAI 相容格式（`/v1/chat/completions`），`anthropic-messages` 則使用 Anthropic 原生格式（`/v1/messages`）。請依端點支援的格式選擇。

**Ollama（本機）**

```json
{
  "model_name": "llama3",
  "provider": "ollama",
  "model": "llama3"
}
```

**LM Studio（本機）**

```json
{
  "model_name": "lmstudio-local",
  "provider": "lmstudio",
  "model": "openai/gpt-oss-20b"
}
```

`api_base` 預設為 `http://localhost:1234/v1`。除非 LM Studio 伺服器啟用驗證，否則 API 金鑰為選填。<br/>
明確設定 `provider` 時，PicoClaw 會將 `openai/gpt-oss-20b` 原樣傳送至 LM Studio 伺服器。省略 `provider` 時，舊版相容格式 `"model": "lmstudio/openai/gpt-oss-20b"` 仍會解析為相同的上游模型 ID。

**自訂 Proxy／API**

```json
{
  "model_name": "my-custom-model",
  "provider": "openai",
  "model": "custom-model",
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
  "provider": "litellm",
  "model": "lite-gpt4",
  "api_base": "http://localhost:4000/v1",
  "api_keys": ["sk-..."]
}
```

明確設定 `provider` 時，PicoClaw 會原樣傳送 `model`。也就是說，`"provider": "litellm", "model": "lite-gpt4"` 會傳送 `lite-gpt4`，而 `"provider": "litellm", "model": "openai/gpt-4o"` 會傳送 `openai/gpt-4o`。省略 `provider` 時，舊版相容格式 `litellm/lite-gpt4` 及 `litellm/openai/gpt-4o` 仍會以相同方式解析。

**Z.AI Coding Plan**

如果標準 Zhipu 端點（`https://open.bigmodel.cn/api/paas/v4`）傳回 429（錯誤碼 1113：餘額不足），請改用 Z.AI Coding Plan 端點：

```json
{
  "model_name": "glm-4.7",
  "provider": "openai",
  "model": "glm-4.7",
  "api_keys": ["your-zhipu-api-key"],
  "api_base": "https://api.z.ai/api/coding/paas/v4"
}
```

**注意：** Z.AI Coding Plan 端點與標準 Zhipu 端點使用相同的 API 金鑰格式，但會分開計費。如果標準 Zhipu 端點出現 429 錯誤，Z.AI Coding Plan 端點可能仍有可用餘額。

#### 負載平衡

為相同模型名稱設定多個端點後，PicoClaw 會自動以循環方式在端點之間分配要求：

```json
{
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "provider": "openai",
      "model": "gpt-5.4",
      "api_base": "https://api1.example.com/v1",
      "api_keys": ["sk-key1"]
    },
    {
      "model_name": "gpt-5.4",
      "provider": "openai",
      "model": "gpt-5.4",
      "api_base": "https://api2.example.com/v1",
      "api_keys": ["sk-key2"]
    }
  ]
}
```

#### 自動模型容錯移轉（串聯）

在代理人模型設定中指定 `primary` 加 `fallbacks` 後，PicoClaw 即可自動進行容錯移轉。
如果遇到 HTTP `429`、配額／速率限制及逾時等可重試的錯誤，執行階段備援鏈會嘗試下一個候選模型。
系統也會分別追蹤每個候選模型的冷卻時間，避免立即重試剛失敗的目標。

```json
{
  "model_list": [
    {
      "model_name": "qwen-main",
      "provider": "openai",
      "model": "qwen3.5:cloud",
      "api_base": "https://api.example.com/v1",
      "api_keys": ["sk-main"]
    },
    {
      "model_name": "deepseek-backup",
      "provider": "deepseek",
      "model": "deepseek-chat",
      "api_keys": ["sk-backup-1"]
    },
    {
      "model_name": "gemini-backup",
      "provider": "gemini",
      "model": "gemini-2.5-flash",
      "api_keys": ["sk-backup-2"]
    }
  ],
  "agents": {
    "defaults": {
      "model_name": "qwen-main",
      "model_fallbacks": ["deepseek-backup", "gemini-backup"]
    }
  }
}
```

如果相同模型採用金鑰層級的容錯移轉，PicoClaw 可以依序嘗試其他由不同金鑰提供的候選模型，再改用其他備援模型。

#### 從舊版 `providers` 設定移轉

舊版 `providers` 設定**已淘汰**，並已從 V2 移除。系統會自動移轉現有的 V0／V1 設定。

**舊版設定（已淘汰）：**

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

**新版設定（建議）：**

```json
{
  "version": 3,
  "model_list": [
    {
      "model_name": "glm-4.7",
      "provider": "zhipu",
      "model": "glm-4.7",
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

詳細移轉方式請參閱 [migration/model-list-migration.md](../migration/model-list-migration.md)。

### 服務供應商架構

PicoClaw 會依通訊協定系列路由服務供應商：

- OpenAI 相容通訊協定：OpenRouter、OpenAI 相容閘道、Groq、Zhipu 及 vLLM 類型的端點。
- Gemini 原生通訊協定：使用原生 `models/*:generateContent` 及 `models/*:streamGenerateContent` 端點連線至 Google Gemini。
- Anthropic 通訊協定：Claude 原生 API 行為。
- Codex／OAuth 路徑：OpenAI OAuth／Token 驗證路由。

這種設計能讓執行階段保持輕量，也讓新增 OpenAI 相容後端時，通常只需設定 `api_base` 與 `api_keys`。

<details>
<summary><b>Zhipu</b></summary>

**1. 取得 API 金鑰與基底 URL**

* [取得 API 金鑰](https://bigmodel.cn/usercenter/proj-mgmt/apikeys)

**2. 設定**

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

**3. 執行**

```bash
picoclaw agent -m "Hello"
```

</details>

<details>
<summary><b>完整設定範例</b></summary>

```json
{
  "agents": {
    "defaults": {
      "model_name": "claude-opus-4-5"
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
      "exec_timeout_minutes": 5,
      "allow_command": true,
      "command_allowed_remotes": []
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

## 📝 API 金鑰比較

| 服務                  | 費用                    | 適用情境                                  |
| --------------------- | ----------------------- | ----------------------------------------- |
| **OpenRouter**        | 免費：每月 200K Token   | 多種模型（Claude、GPT-4 等）              |
| **Volcengine CodingPlan** | 首月 ¥9.9           | 適合中國使用者，提供多種頂尖模型（Doubao、DeepSeek 等） |
| **Zhipu**             | 免費：每月 200K Token   | 適合中國使用者                            |
| **Brave Search**      | 每 1000 次查詢 $5       | Web 搜尋功能                              |
| **SearXNG**           | 免費（自行架設）        | 重視隱私的整合搜尋（70 多個搜尋引擎）     |
| **Groq**              | 提供免費方案            | 快速推論（Llama、Mixtral）                |
| **Cerebras**          | 提供免費方案            | 快速推論（Llama、Qwen 等）                |
| **LongCat**           | 免費：每日最多 5M Token | 快速推論                                  |
| **ModelScope**        | 免費：每日 2000 次要求  | 推論（Qwen、GLM、DeepSeek 等）            |

---

<div align="center">
  <img src="../../assets/logo.jpg" alt="PicoClaw 迷因" width="512">
</div>
