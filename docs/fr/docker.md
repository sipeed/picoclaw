# 🐳 Docker et Démarrage Rapide

> Retour au [README](../../README.fr.md)

## 🐳 Docker Compose

Vous pouvez également exécuter PicoClaw avec Docker Compose sans rien installer localement.

```bash
# 1. Cloner ce dépôt
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. Premier lancement — génère automatiquement docker/data/config.json puis s'arrête
#    (se déclenche uniquement quand config.json et workspace/ sont tous deux absents)
docker compose -f docker/docker-compose.yml --profile gateway up
# Le conteneur affiche "First-run setup complete." et s'arrête.

# 3. Configurer vos clés API
vim docker/data/config.json   # Set provider API keys, bot tokens, etc.

# 4. Démarrer
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

> [!TIP]
> **Utilisateurs Docker** : Par défaut, le Gateway écoute sur `127.0.0.1`, ce qui n'est pas accessible depuis l'hôte. Si vous devez accéder aux endpoints de santé ou exposer des ports, définissez `PICOCLAW_GATEWAY_HOST=0.0.0.0` dans votre environnement ou mettez à jour `config.json`.

```bash
# 5. Vérifier les logs
docker compose -f docker/docker-compose.yml logs -f picoclaw-gateway

# 6. Arrêter
docker compose -f docker/docker-compose.yml --profile gateway down
```

### Mode Launcher (Console Web)

L'image `launcher` inclut les trois binaires (`picoclaw`, `picoclaw-launcher`, `picoclaw-launcher-tui`) et démarre la console web par défaut, qui fournit une interface navigateur pour la configuration et le chat.

```bash
docker compose -f docker/docker-compose.yml --profile launcher up -d
```

Ouvrez http://localhost:18800 dans votre navigateur. Le launcher gère automatiquement le processus gateway.

> [!WARNING]
> La console web ne prend pas encore en charge l'authentification. Évitez de l'exposer sur Internet public.

### Mode Agent (One-shot)

```bash
# Poser une question
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent -m "What is 2+2?"

# Mode interactif
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent
```

### Mise à jour

```bash
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

### 🚀 Démarrage Rapide

> [!TIP]
> Configurez votre clé API dans `~/.picoclaw/config.json`. Obtenir des clés API : [Volcengine (CodingPlan)](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) (LLM) · [OpenRouter](https://openrouter.ai/keys) (LLM) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) (LLM). La recherche web est optionnelle — obtenez gratuitement une [API Tavily](https://tavily.com) (1000 requêtes gratuites/mois) ou une [API Brave Search](https://brave.com/search/api) (2000 requêtes gratuites/mois).

**1. Initialiser**

```bash
picoclaw onboard
```

**2. Configurer** (`~/.picoclaw/config.json`)

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

> **Nouveau** : Le format de configuration `model_list` permet l'ajout de fournisseurs sans modification de code. Voir [Configuration des Modèles](#configuration-des-modèles-model_list) pour plus de détails.
> `request_timeout` est optionnel et utilise les secondes. S'il est omis ou défini à `<= 0`, PicoClaw utilise le timeout par défaut (120s).

**3. Obtenir des clés API**

* **Fournisseur LLM** : [OpenRouter](https://openrouter.ai/keys) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) · [Anthropic](https://console.anthropic.com) · [OpenAI](https://platform.openai.com) · [Gemini](https://aistudio.google.com/api-keys)
* **Recherche Web** (optionnel) :
  * [Brave Search](https://brave.com/search/api) - Payant ($5/1000 requêtes, ~$5-6/mois)
  * [Perplexity](https://www.perplexity.ai) - Recherche alimentée par l'IA avec interface de chat
  * [SearXNG](https://github.com/searxng/searxng) - Métamoteur auto-hébergé (gratuit, pas de clé API nécessaire)
  * [Tavily](https://tavily.com) - Optimisé pour les agents IA (1000 requêtes/mois)
  * DuckDuckGo - Solution de repli intégrée (pas de clé API requise)

> **Note** : Voir `config.example.json` pour un modèle de configuration complet.

**4. Discuter**

```bash
picoclaw agent -m "What is 2+2?"
```

C'est tout ! Vous avez un assistant IA fonctionnel en 2 minutes.

---
