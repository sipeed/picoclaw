# 🔌 プロバイダーとモデル設定

> [README](../project/README.ja.md) に戻る

### プロバイダー

> [!NOTE]
> Groq は Whisper による無料の音声文字起こしを提供しています。Groq を設定すると、任意のチャネルからの音声メッセージが Agent レベルで自動的にテキストに変換されます。

| プロバイダー         | 用途                         | API Key の取得                                                       |
| -------------------- | ---------------------------- | -------------------------------------------------------------------- |
| `gemini`             | LLM (Gemini 直接接続)       | [aistudio.google.com](https://aistudio.google.com)                   |
| `zhipu`              | LLM (Zhipu 直接接続)        | [bigmodel.cn](https://bigmodel.cn)                                   |
| `volcengine`         | LLM (Volcengine 直接接続)   | [volcengine.com](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) |
| `openrouter`         | LLM (推奨、全モデルアクセス可) | [openrouter.ai](https://openrouter.ai)                               |
| `anthropic`          | LLM (Claude 直接接続)       | [console.anthropic.com](https://console.anthropic.com)               |
| `openai`             | LLM (GPT 直接接続)          | [platform.openai.com](https://platform.openai.com)                   |
| `deepseek`           | LLM (DeepSeek 直接接続)     | [platform.deepseek.com](https://platform.deepseek.com)               |
| `qwen`               | LLM (Qwen 直接接続)         | [dashscope.console.aliyun.com](https://dashscope.console.aliyun.com) |
| `groq`               | LLM + **音声文字起こし** (Whisper) | [console.groq.com](https://console.groq.com)                         |
| `cerebras`           | LLM (Cerebras 直接接続)     | [cerebras.ai](https://cerebras.ai)                                   |
| `vivgrid`            | LLM (Vivgrid 直接接続)      | [vivgrid.com](https://vivgrid.com)                                   |
| `moonshot`           | LLM (Kimi/Moonshot 直接接続) | [platform.moonshot.cn](https://platform.moonshot.cn)                 |
| `minimax`            | LLM (Minimax 直接接続)      | [platform.minimaxi.com](https://platform.minimaxi.com)              |
| `avian`              | LLM (Avian 直接接続)        | [avian.io](https://avian.io)                                         |
| `mistral`            | LLM (Mistral 直接接続)      | [console.mistral.ai](https://console.mistral.ai)                    |
| `longcat`            | LLM (Longcat 直接接続)      | [longcat.ai](https://longcat.ai)                                     |
| `modelscope`         | LLM (ModelScope 直接接続)   | [modelscope.cn](https://modelscope.cn)                               |

<a id="モデル設定-model_list"></a>
### モデル設定 (model_list)

> **新機能！** PicoClaw は**モデル中心**の設定方式を採用しました。`ベンダー/モデル` 形式（例: `zhipu/glm-4.7`）を指定するだけで新しい provider を追加できます——**コード変更は一切不要です！**

この設計は**マルチ Agent シナリオ**もサポートし、柔軟な Provider 選択を提供します：

- **Agent ごとに異なる Provider**: 各 Agent が独自の LLM provider を使用可能
- **モデルフォールバック**: プライマリモデルとフォールバックモデルを設定し、信頼性を向上
- **ロードバランシング**: 複数の API エンドポイント間でリクエストを分散
- **一元管理**: すべての provider を一箇所で管理

#### 📋 サポートされている全ベンダー

| ベンダー            | `model` プレフィックス | デフォルト API Base                                 | プロトコル | API Key の取得                                                    |
| ------------------- | --------------------- | --------------------------------------------------- | ---------- | ----------------------------------------------------------------- |
| **OpenAI**          | `openai/`             | `https://api.openai.com/v1`                         | OpenAI     | [キーを取得](https://platform.openai.com)                         |
| **Anthropic**       | `anthropic/`          | `https://api.anthropic.com/v1`                      | Anthropic  | [キーを取得](https://console.anthropic.com)                       |
| **智谱 AI (GLM)**   | `zhipu/`              | `https://open.bigmodel.cn/api/paas/v4`              | OpenAI     | [キーを取得](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) |
| **DeepSeek**        | `deepseek/`           | `https://api.deepseek.com/v1`                       | OpenAI     | [キーを取得](https://platform.deepseek.com)                       |
| **Google Gemini**   | `gemini/`             | `https://generativelanguage.googleapis.com/v1beta`  | Gemini     | [キーを取得](https://aistudio.google.com/api-keys)                |
| **Groq**            | `groq/`               | `https://api.groq.com/openai/v1`                    | OpenAI     | [キーを取得](https://console.groq.com)                            |
| **Moonshot**        | `moonshot/`           | `https://api.moonshot.cn/v1`                        | OpenAI     | [キーを取得](https://platform.moonshot.cn)                        |
| **通義千問 (Qwen)** | `qwen/`               | `https://dashscope.aliyuncs.com/compatible-mode/v1` | OpenAI     | [キーを取得](https://dashscope.console.aliyun.com)                |
| **NVIDIA**          | `nvidia/`             | `https://integrate.api.nvidia.com/v1`               | OpenAI     | [キーを取得](https://build.nvidia.com)                            |
| **Ollama**          | `ollama/`             | `http://localhost:11434/v1`                         | OpenAI     | ローカル（キー不要）                                              |
| **OpenRouter**      | `openrouter/`         | `https://openrouter.ai/api/v1`                      | OpenAI     | [キーを取得](https://openrouter.ai/keys)                          |
| **LiteLLM Proxy**   | `litellm/`            | `http://localhost:4000/v1`                          | OpenAI     | LiteLLM プロキシキー                                              |
| **VLLM**            | `vllm/`               | `http://localhost:8000/v1`                          | OpenAI     | ローカル                                                          |
| **Cerebras**        | `cerebras/`           | `https://api.cerebras.ai/v1`                        | OpenAI     | [キーを取得](https://cerebras.ai)                                 |
| **VolcEngine (Doubao)** | `volcengine/`     | `https://ark.cn-beijing.volces.com/api/v3`          | OpenAI     | [キーを取得](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) |
| **神算云**          | `shengsuanyun/`       | `https://router.shengsuanyun.com/api/v1`            | OpenAI     | -                                                                 |
| **BytePlus**        | `byteplus/`           | `https://ark.ap-southeast.bytepluses.com/api/v3`    | OpenAI     | [キーを取得](https://www.byteplus.com)                            |
| **Vivgrid**         | `vivgrid/`            | `https://api.vivgrid.com/v1`                        | OpenAI     | [キーを取得](https://vivgrid.com)                                 |
| **LongCat**         | `longcat/`            | `https://api.longcat.chat/openai`                   | OpenAI     | [キーを取得](https://longcat.chat/platform)                       |
| **ModelScope (魔搭)**| `modelscope/`        | `https://api-inference.modelscope.cn/v1`            | OpenAI     | [トークンを取得](https://modelscope.cn/my/tokens)                |
| **Antigravity**     | `antigravity/`        | Google Cloud                                        | カスタム   | OAuth のみ                                                        |
| **GitHub Copilot**  | `github-copilot/`     | `localhost:4321`                                    | gRPC       | -                                                                 |

#### 基本設定

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

#### `model_list` エントリフィールド

| フィールド | 型 | 必須 | 説明 |
|-----------|------|------|------|
| `model_name` | string | はい | agent 設定でこのモデルを参照するための一意の名前 |
| `model` | string | はい | ベンダー/モデル識別子（例：`openai/gpt-5.4`、`azure/gpt-5.4`、`anthropic/claude-sonnet-4.6`） |
| `api_keys` | string[] | はい* | 認証キー。複数キーでリクエストごとのローテーションが可能。ローカル provider（Ollama、LM Studio、VLLM）には不要 |
| `api_base` | string | いいえ | デフォルトの API エンドポイント URL を上書き |
| `proxy` | string | いいえ | このモデルエントリの HTTP プロキシ URL |
| `user_agent` | string | いいえ | カスタム `User-Agent` リクエストヘッダー（OpenAI 互換、Gemini、Anthropic、Azure provider で対応） |
| `request_timeout` | int | いいえ | リクエストタイムアウト（秒）。デフォルト値は provider により異なる |
| `max_tokens_field` | string | いいえ | リクエストボディの max tokens フィールド名を上書き（例：o1 モデルでは `max_completion_tokens`） |
| `thinking_level` | string | いいえ | 拡張思考レベル：`off`、`low`、`medium`、`high`、`xhigh`、`adaptive` |
| `extra_body` | object | いいえ | 各リクエストボディに注入する追加フィールド |
| `rpm` | int | いいえ | 1 分あたりのリクエストレート制限 |
| `fallbacks` | string[] | いいえ | 自動フェイルオーバーのフォールバックモデル名 |
| `enabled` | bool | いいえ | このモデルエントリを有効にするかどうか（デフォルト：`true`） |

#### ベンダー別設定例

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

**LiteLLM Proxy**

```json
{
  "model_name": "lite-gpt4",
  "model": "litellm/lite-gpt4",
  "api_base": "http://localhost:4000/v1",
  "api_keys": ["sk-..."]
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

**Anthropic (API キー使用)**

```json
{
  "model_name": "claude-sonnet-4.6",
  "model": "anthropic/claude-sonnet-4.6",
  "api_keys": ["sk-ant-your-key"]
}
```

> `picoclaw auth login --provider anthropic` を実行して API トークンを設定してください。

**Anthropic Messages API（ネイティブ形式）**

Anthropic API への直接アクセスや、Anthropic のネイティブメッセージ形式のみをサポートするカスタムエンドポイント向け：

```json
{
  "model_name": "claude-opus-4-6",
  "model": "anthropic-messages/claude-opus-4-6",
  "api_keys": ["sk-ant-your-key"],
  "api_base": "https://api.anthropic.com"
}
```

> `anthropic-messages` プロトコルを使用するケース：
> - Anthropic のネイティブ `/v1/messages` エンドポイントのみをサポートするサードパーティプロキシを使用する場合（OpenAI 互換の `/v1/chat/completions` 非対応）
> - MiniMax、Synthetic など Anthropic のネイティブメッセージ形式を必要とするサービスに接続する場合
> - 既存の `anthropic` プロトコルが 404 エラーを返す場合（エンドポイントが OpenAI 互換形式をサポートしていないことを示す）
>
> **注意:** `anthropic` プロトコルは OpenAI 互換形式（`/v1/chat/completions`）を使用し、`anthropic-messages` は Anthropic のネイティブ形式（`/v1/messages`）を使用します。エンドポイントがサポートする形式に応じて選択してください。

**Ollama (ローカル)**

```json
{
  "model_name": "llama3",
  "model": "ollama/llama3"
}
```

**カスタムプロキシ/API**

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

PicoClaw はリクエスト送信前に外側の `litellm/` プレフィックスのみを除去するため、`litellm/lite-gpt4` は `lite-gpt4` を送信し、`litellm/openai/gpt-4o` は `openai/gpt-4o` を送信します。

#### ロードバランシング

同じモデル名に複数のエンドポイントを設定すると、PicoClaw が自動的にラウンドロビンで分散します：

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

#### レガシー `providers` 設定からの移行

旧 `providers` 設定形式は**非推奨**となり、V2 で削除されました。既存の V0/V1 設定は自動的に移行されます。

**旧設定（非推奨）：**

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

**新設定（推奨）：**

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

詳細な移行ガイドは [docs/migration/model-list-migration.md](../migration/model-list-migration.md) を参照してください。

### Provider アーキテクチャ

PicoClaw はプロトコルファミリーごとに Provider をルーティングします：

- OpenAI 互換プロトコル：OpenRouter、OpenAI 互換ゲートウェイ、Groq、Zhipu、vLLM スタイルのエンドポイント。
- Gemini ネイティブプロトコル：Google Gemini のネイティブ `models/*:generateContent` / `models/*:streamGenerateContent` エンドポイント。
- Anthropic プロトコル：Claude ネイティブ API 動作。
- Codex/OAuth パス：OpenAI OAuth/Token 認証ルート。

これによりランタイムを軽量に保ちつつ、新しい OpenAI 互換バックエンドの追加をほぼ設定操作（`api_base` + `api_keys`）のみで実現しています。

<details>
<summary><b>Zhipu 設定例</b></summary>

**1. API key と base URL を取得**

- [API key](https://bigmodel.cn/usercenter/proj-mgmt/apikeys) を取得

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

**3. 実行**

```bash
picoclaw agent -m "こんにちは"
```

</details>

<details>
<summary><b>完全な設定例</b></summary>

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

## 📝 API Key 比較表

| サービス         | Pricing                  | ユースケース                          |
| ---------------- | ------------------------ | ------------------------------------- |
| **OpenRouter**   | Free: 200K tokens/month  | マルチモデル (Claude, GPT-4 など)     |
| **Volcengine CodingPlan** | ¥9.9/first month | 中国ユーザー向け、複数の SOTA モデル (Doubao, DeepSeek など) |
| **Zhipu**        | Free: 200K tokens/month  | 中国ユーザー向け                      |
| **Brave Search** | $5/1000 queries          | Web 検索機能                          |
| **SearXNG**      | Free (self-hosted)       | プライバシー重視のメタ検索 (70+ engines) |
| **Groq**         | Free tier available      | 高速推論 (Llama, Mixtral)             |
| **Cerebras**     | Free tier available      | 高速推論 (Llama, Qwen など)           |
| **LongCat**      | Free: up to 5M tokens/day | 高速推論                             |
| **ModelScope**   | Free: 2000 requests/day  | 推論 (Qwen, GLM, DeepSeek など)       |

---

<div align="center">
  <img src="../../assets/logo.jpg" alt="PicoClaw Meme" width="512">
</div>
