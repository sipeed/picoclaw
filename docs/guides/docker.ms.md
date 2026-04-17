# 🐳 Panduan Docker & Quick Start

> Kembali ke [README](../project/README.ms.md)

## 🐳 Docker Compose

Anda juga boleh menjalankan PicoClaw menggunakan Docker Compose tanpa memasang apa-apa secara setempat.

```bash
# 1. Clone repo ini
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. Larian pertama — jana docker/data/config.json secara automatik kemudian keluar
docker compose -f docker/docker-compose.yml --profile gateway up
# Container akan memaparkan "First-run setup complete." dan berhenti.

# 3. Tetapkan kunci API anda
vim docker/data/config.json   # Tetapkan API key penyedia, token bot, dan sebagainya.

# 4. Mula
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

> [!TIP]
> **Pengguna Docker**: Secara lalai, Gateway mendengar pada `127.0.0.1` yang tidak boleh diakses dari host. Jika anda perlu mengakses health endpoint atau mendedahkan port, tetapkan `PICOCLAW_GATEWAY_HOST=0.0.0.0` dalam persekitaran anda atau kemas kini `config.json`.

```bash
# 5. Semak log
docker compose -f docker/docker-compose.yml logs -f picoclaw-gateway

# 6. Hentikan
docker compose -f docker/docker-compose.yml --profile gateway down
```

### Mod Launcher (Konsol Web)

Imej `launcher` merangkumi ketiga-tiga binari (`picoclaw`, `picoclaw-launcher`, `picoclaw-launcher-tui`) dan memulakan konsol web secara lalai, yang menyediakan UI berasaskan pelayar untuk konfigurasi dan sembang.

```bash
docker compose -f docker/docker-compose.yml --profile launcher up -d
```

Buka http://localhost:18800 dalam pelayar anda. Launcher mengurus proses gateway secara automatik.

> [!WARNING]
> Konsol web belum menyokong autentikasi. Elakkan mendedahkannya ke internet awam.

### Mod Agent (One-shot)

```bash
# Tanyakan soalan
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent -m "What is 2+2?"

# Mod interaktif
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent
```

### Kemas kini

```bash
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

### 🚀 Quick Start

> [!TIP]
> Tetapkan API Key anda dalam `~/.picoclaw/config.json`. Dapatkan API Key: [Volcengine (CodingPlan)](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) (LLM) · [OpenRouter](https://openrouter.ai/keys) (LLM) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) (LLM). Carian web adalah pilihan — dapatkan [Tavily API](https://tavily.com) percuma (1000 pertanyaan percuma/bulan) atau [Brave Search API](https://brave.com/search/api) (2000 pertanyaan percuma/bulan).

**1. Inisialisasi**

```bash
picoclaw onboard
```

**2. Konfigurasi** (`~/.picoclaw/config.json`)

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
      "api_keys": ["sk-your-api-key"],
      "api_base":"https://ark.cn-beijing.volces.com/api/coding/v3"
    },
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_keys": ["your-api-key"],
      "request_timeout": 300
    },
    {
      "model_name": "claude-sonnet-4.6",
      "model": "anthropic/claude-sonnet-4.6",
      "api_keys": ["your-anthropic-key"]
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

> **Baharu**: Format konfigurasi `model_list` membolehkan penambahan penyedia tanpa perubahan kod. Lihat [Konfigurasi Model](#konfigurasi-model-model_list) untuk butiran.
> `request_timeout` adalah pilihan dan menggunakan saat. Jika diabaikan atau ditetapkan kepada `<= 0`, PicoClaw menggunakan timeout lalai (120s).

**3. Dapatkan API Key**

* **Penyedia LLM**: [OpenRouter](https://openrouter.ai/keys) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) · [Anthropic](https://console.anthropic.com) · [OpenAI](https://platform.openai.com) · [Gemini](https://aistudio.google.com/api-keys)
* **Carian Web** (pilihan):
  * [Brave Search](https://brave.com/search/api) - Berbayar ($5/1000 pertanyaan, ~$5-6/bulan)
  * [Perplexity](https://www.perplexity.ai) - Carian berkuasa AI dengan antara muka sembang
  * [SearXNG](https://github.com/searxng/searxng) - Enjin meta-carian hos kendiri (percuma, tidak perlu API key)
  * [Tavily](https://tavily.com) - Dioptimumkan untuk AI Agents (1000 permintaan/bulan)
  * DuckDuckGo - Fallback terbina dalam (tidak memerlukan API key)

> **Nota**: Lihat `config.example.json` untuk templat konfigurasi penuh.

**4. Sembang**

```bash
picoclaw agent -m "What is 2+2?"
```

Itu sahaja! Anda kini mempunyai pembantu AI yang berfungsi dalam masa 2 minit.

---
