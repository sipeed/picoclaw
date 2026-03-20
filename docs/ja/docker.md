# 🐳 Docker とクイックスタート

> [README](../../README.ja.md) に戻る

## 🐳 Docker Compose

Docker Compose を使用して PicoClaw を実行できます。ローカルに何もインストールする必要はありません。

```bash
# 1. リポジトリをクローン
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. 初回実行 — docker/data/config.json を自動生成して終了
docker compose -f docker/docker-compose.yml --profile gateway up
# コンテナが "First-run setup complete." と表示して停止します

# 3. API Key を設定
vim docker/data/config.json   # provider API key、Bot Token などを設定

# 4. 起動
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

> [!TIP]
> **Docker ユーザー**: デフォルトでは Gateway は `127.0.0.1` でリッスンしており、コンテナ外からはアクセスできません。ヘルスチェックエンドポイントへのアクセスやポート公開が必要な場合は、環境変数で `PICOCLAW_GATEWAY_HOST=0.0.0.0` を設定するか、`config.json` を更新してください。

```bash
# 5. ログを確認
docker compose -f docker/docker-compose.yml logs -f picoclaw-gateway

# 6. 停止
docker compose -f docker/docker-compose.yml --profile gateway down
```

### Launcher モード (Web コンソール)

`launcher` イメージには 3 つのバイナリ（`picoclaw`、`picoclaw-launcher`、`picoclaw-launcher-tui`）がすべて含まれており、デフォルトで Web コンソールを起動します。ブラウザベースの設定・チャット画面を提供します。

```bash
docker compose -f docker/docker-compose.yml --profile launcher up -d
```

ブラウザで http://localhost:18800 を開いてください。Launcher が Gateway プロセスを自動管理します。

> [!WARNING]
> Web コンソールはまだ認証をサポートしていません。公開インターネットに公開しないでください。

### Agent モード (ワンショット)

```bash
# 質問する
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent -m "2+2は？"

# インタラクティブモード
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent
```

### イメージの更新

```bash
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

---

## 🚀 クイックスタート

> [!TIP]
> `~/.picoclaw/config.json` に API Key を設定してください。API Key の取得先: [Volcengine (CodingPlan)](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) (LLM) · [OpenRouter](https://openrouter.ai/keys) (LLM) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) (LLM)。Web 検索は**オプション**です — 無料の [Tavily API](https://tavily.com) (月 1000 回無料) または [Brave Search API](https://brave.com/search/api) (月 2000 回無料) を取得できます。

**1. 初期化**

```bash
picoclaw onboard
```

**2. 設定** (`~/.picoclaw/config.json`)

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model_name": "gpt-5.4",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "model_list": [
    {
      "model_name": "ark-code-latest",
      "model": "volcengine/ark-code-latest",
      "api_key": "sk-your-api-key",
      "api_base":"https://ark.cn-beijing.volces.com/api/coding/v3"
    },
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_key": "your-api-key",
      "request_timeout": 300
    },
    {
      "model_name": "claude-sonnet-4.6",
      "model": "anthropic/claude-sonnet-4.6",
      "api_key": "your-anthropic-key"
    }
  ],
  "tools": {
    "web": {
      "enabled": true,
      "fetch_limit_bytes": 10485760,
      "format": "plaintext",
      "brave": {
        "enabled": false,
        "api_key": "YOUR_BRAVE_API_KEY",
        "max_results": 5
      },
      "tavily": {
        "enabled": false,
        "api_key": "YOUR_TAVILY_API_KEY",
        "max_results": 5
      },
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      },
      "perplexity": {
        "enabled": false,
        "api_key": "YOUR_PERPLEXITY_API_KEY",
        "max_results": 5
      },
      "searxng": {
        "enabled": false,
        "base_url": "http://your-searxng-instance:8888",
        "max_results": 5
      }
    }
  }
}
```

> **新機能**: `model_list` 設定形式により、コード変更なしで provider を追加できます。詳細は[モデル設定](providers.md#モデル設定-model_list)を参照してください。
> `request_timeout` はオプションで、単位は秒です。省略または `<= 0` に設定した場合、PicoClaw はデフォルトのタイムアウト（120 秒）を使用します。

**3. API Key の取得**

* **LLM プロバイダー**: [OpenRouter](https://openrouter.ai/keys) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) · [Anthropic](https://console.anthropic.com) · [OpenAI](https://platform.openai.com) · [Gemini](https://aistudio.google.com/api-keys)
* **Web 検索** (オプション):
  * [Brave Search](https://brave.com/search/api) - 有料 ($5/1000 queries, ~$5-6/month)
  * [Perplexity](https://www.perplexity.ai) - AI 搭載の検索・チャットインターフェース
  * [SearXNG](https://github.com/searxng/searxng) - セルフホスト型メタ検索エンジン（無料、API Key 不要）
  * [Tavily](https://tavily.com) - AI Agent 向けに最適化 (1000 requests/month)
  * DuckDuckGo - 組み込みフォールバック（API Key 不要）

> **注意**: 完全な設定テンプレートは `config.example.json` を参照してください。

**4. チャット**

```bash
picoclaw agent -m "2+2は？"
```

以上です！2 分で動作する AI アシスタントが手に入ります。

---
