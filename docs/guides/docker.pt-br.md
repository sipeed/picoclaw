# 🐳 Docker e Início Rápido

> Voltar ao [README](../project/README.pt-br.md)

## 🐳 Docker Compose

Você também pode executar o PicoClaw usando Docker Compose sem instalar nada localmente.

```bash
# 1. Clone este repositório
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. Primeira execução — gera automaticamente docker/data/config.json e encerra
#    (só é acionado quando config.json e workspace/ estão ambos ausentes)
docker compose -f docker/docker-compose.yml --profile gateway up
# O contêiner exibe "First-run setup complete." e para.

# 3. Configure suas chaves de API
vim docker/data/config.json   # Set provider API keys, bot tokens, etc.

# 4. Iniciar
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

> [!TIP]
> **Usuários Docker**: Por padrão, o Gateway escuta em `127.0.0.1`, que não é acessível a partir do host. Se você precisar acessar os endpoints de saúde ou expor portas, defina `PICOCLAW_GATEWAY_HOST=0.0.0.0` no seu ambiente ou atualize o `config.json`.

```bash
# 5. Verificar logs
docker compose -f docker/docker-compose.yml logs -f picoclaw-gateway

# 6. Parar
docker compose -f docker/docker-compose.yml --profile gateway down
```

### Modo Launcher (Console Web)

A imagem `launcher` inclui os três binários (`picoclaw`, `picoclaw-launcher`, `picoclaw-launcher-tui`) e inicia o console web por padrão, que fornece uma interface baseada em navegador para configuração e chat.

```bash
docker compose -f docker/docker-compose.yml --profile launcher up -d
```

Abra http://localhost:18800 no seu navegador. O launcher gerencia o processo do gateway automaticamente.

> [!WARNING]
> O console web ainda não suporta autenticação. Evite expô-lo na internet pública.

### Modo Agent (One-shot)

```bash
# Fazer uma pergunta
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent -m "What is 2+2?"

# Modo interativo
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent
```

### Atualização

```bash
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

### 🚀 Início Rápido

> [!TIP]
> Configure sua chave de API em `~/.picoclaw/config.json`. Obtenha chaves de API: [Volcengine (CodingPlan)](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) (LLM) · [OpenRouter](https://openrouter.ai/keys) (LLM) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) (LLM). A busca na web é opcional — obtenha gratuitamente uma [API Tavily](https://tavily.com) (1000 consultas gratuitas/mês) ou [API Brave Search](https://brave.com/search/api) (2000 consultas gratuitas/mês).

**1. Inicializar**

```bash
picoclaw onboard
```

**2. Configurar** (`~/.picoclaw/config.json`)

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

> **Novo**: O formato de configuração `model_list` permite adicionar provedores sem alteração de código. Veja [Configuração de Modelos](#configuração-de-modelos-model_list) para detalhes.
> `request_timeout` é opcional e usa segundos. Se omitido ou definido como `<= 0`, o PicoClaw usa o timeout padrão (120s).

**3. Obter chaves de API**

* **Provedor LLM**: [OpenRouter](https://openrouter.ai/keys) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) · [Anthropic](https://console.anthropic.com) · [OpenAI](https://platform.openai.com) · [Gemini](https://aistudio.google.com/api-keys)
* **Busca na Web** (opcional):
  * [Brave Search](https://brave.com/search/api) - Pago ($5/1000 consultas, ~$5-6/mês)
  * [Perplexity](https://www.perplexity.ai) - Busca com IA e interface de chat
  * [SearXNG](https://github.com/searxng/searxng) - Metabuscador auto-hospedado (gratuito, sem necessidade de chave de API)
  * [Tavily](https://tavily.com) - Otimizado para agentes de IA (1000 requisições/mês)
  * DuckDuckGo - Fallback integrado (sem necessidade de chave de API)

> **Nota**: Veja `config.example.json` para um modelo de configuração completo.

**4. Conversar**

```bash
picoclaw agent -m "What is 2+2?"
```

Pronto! Você tem um assistente de IA funcionando em 2 minutos.

---
