# 🐳 Docker 与快速开始

> 返回 [README](../../README.zh.md)

## 🐳 Docker Compose

您也可以使用 Docker Compose 运行 PicoClaw，无需在本地安装任何环境。

```bash
# 1. 克隆仓库
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. 首次运行 — 自动生成 docker/data/config.json 后退出
#    （仅在 config.json 和 workspace/ 都不存在时触发）
docker compose -f docker/docker-compose.yml --profile gateway up
# 容器打印 "First-run setup complete." 后自动停止

# 3. 填写 API Key 等配置
vim docker/data/config.json   # 设置 provider API key、Bot Token 等

# 4. 正式启动
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

> [!TIP]
> **Docker 用户**: 默认情况下, Gateway 监听 `127.0.0.1`，该端口不会暴露到容器外。如果需要通过端口映射访问健康检查接口，请在环境变量中设置 `PICOCLAW_GATEWAY_HOST=0.0.0.0` 或修改 `config.json`。

```bash
# 5. 查看日志
docker compose -f docker/docker-compose.yml logs -f picoclaw-gateway

# 6. 停止
docker compose -f docker/docker-compose.yml --profile gateway down
```

### Launcher 模式 (Web 控制台)

`launcher` 镜像包含所有三个二进制文件（`picoclaw`、`picoclaw-launcher`、`picoclaw-launcher-tui`），默认启动 Web 控制台，提供基于浏览器的配置和聊天界面。

```bash
docker compose -f docker/docker-compose.yml --profile launcher up -d
```

在浏览器中打开 <http://localhost:18800>。Launcher 会自动管理 Gateway 进程。

> [!WARNING]
> Web 控制台通过 dashboard 令牌鉴权（默认每次启动在内存中生成；可用 `PICOCLAW_LAUNCHER_TOKEN` 固定）。**不要**将启动器暴露到不可信网络或公网。完整说明见 [配置指南](configuration.md) 中的「Web 启动器控制台」一节。

### Agent 模式 (一次性运行)

```bash
# 提问
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent -m "2+2 等于几？"

# 交互模式
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent
```

### 更新镜像

```bash
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

---

## 🚀 快速开始

> [!TIP]
> 在 `~/.picoclaw/config.json` 中设置您的 API Key。获取 API Key: [火山引擎 (CodingPlan)](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) (LLM) · [OpenRouter](https://openrouter.ai/keys) (LLM) · [Zhipu (智谱)](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) (LLM)。网络搜索是 **可选的** — 获取免费的 [Tavily API](https://tavily.com) (每月 1000 次免费查询) 或 [Brave Search API](https://brave.com/search/api) (每月 2000 次免费查询)。

**1. 初始化 (Initialize)**

```bash
picoclaw onboard
```

**2. 配置 (Configure)** (`~/.picoclaw/config.json`)

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

> **新功能**: `model_list` 配置格式支持零代码添加 provider。详见[模型配置](providers.md#模型配置-model_list)章节。
> `request_timeout` 为可选项，单位为秒。若省略或设置为 `<= 0`，PicoClaw 使用默认超时（120 秒）。

**3. 获取 API Key**

* **LLM 提供商**: [OpenRouter](https://openrouter.ai/keys) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) · [Anthropic](https://console.anthropic.com) · [OpenAI](https://platform.openai.com) · [Gemini](https://aistudio.google.com/api-keys)
* **网络搜索** (可选):
  * [Brave Search](https://brave.com/search/api) - 付费 ($5/1000 次查询，约 $5-6/月)
  * [Perplexity](https://www.perplexity.ai) - AI 驱动的搜索与聊天界面
  * [SearXNG](https://github.com/searxng/searxng) - 自建元搜索引擎（免费，无需 API Key）
  * [Tavily](https://tavily.com) - 专为 AI Agent 优化 (1000 请求/月)
  * DuckDuckGo - 内置回退（无需 API Key）

> **注意**: 完整的配置模板请参考 `config.example.json`。

**4. 对话 (Chat)**

```bash
picoclaw agent -m "2+2 等于几？"
```

就是这样！您在 2 分钟内就拥有了一个可工作的 AI 助手。

---
