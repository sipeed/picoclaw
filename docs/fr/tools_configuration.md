# 🔧 Configuration des Outils

> Retour au [README](../../README.fr.md)

La configuration des outils de PicoClaw se trouve dans le champ `tools` de `config.json`.

## Structure du répertoire

```json
{
  "tools": {
    "web": {
      ...
    },
    "mcp": {
      ...
    },
    "exec": {
      ...
    },
    "cron": {
      ...
    },
    "skills": {
      ...
    }
  }
}
```

## Outils Web

Les outils web sont utilisés pour la recherche et la récupération de pages web.

### Web Fetcher
Paramètres généraux pour la récupération et le traitement du contenu des pages web.

| Config              | Type   | Par défaut    | Description                                                                                   |
|---------------------|--------|---------------|-----------------------------------------------------------------------------------------------|
| `enabled`           | bool   | true          | Activer la capacité de récupération de pages web.                                             |
| `fetch_limit_bytes` | int    | 10485760      | Taille maximale du contenu de la page web à récupérer, en octets (par défaut 10 Mo).          |
| `format`            | string | "plaintext"   | Format de sortie du contenu récupéré. Options : `plaintext` ou `markdown` (recommandé).       |

### Brave

| Config        | Type   | Par défaut | Description               |
|---------------|--------|------------|---------------------------|
| `enabled`     | bool   | false      | Activer la recherche Brave |
| `api_key`     | string | -          | Clé API Brave Search      |
| `max_results` | int    | 5          | Nombre maximum de résultats |

### DuckDuckGo

| Config        | Type | Par défaut | Description                    |
|---------------|------|------------|--------------------------------|
| `enabled`     | bool | true       | Activer la recherche DuckDuckGo |
| `max_results` | int  | 5          | Nombre maximum de résultats    |

### Perplexity

| Config        | Type   | Par défaut | Description                    |
|---------------|--------|------------|--------------------------------|
| `enabled`     | bool   | false      | Activer la recherche Perplexity |
| `api_key`     | string | -          | Clé API Perplexity             |
| `max_results` | int    | 5          | Nombre maximum de résultats    |

## Outil Exec

L'outil exec est utilisé pour exécuter des commandes shell.

| Config                 | Type  | Par défaut | Description                                    |
|------------------------|-------|------------|------------------------------------------------|
| `enabled`              | bool  | true       | Activer l'outil exec                           |
| `enable_deny_patterns` | bool  | true       | Activer le blocage par défaut des commandes dangereuses |
| `custom_deny_patterns` | array | []         | Modèles de refus personnalisés (expressions régulières) |

### Désactivation de l'Outil Exec

Pour désactiver complètement l'outil `exec`, définissez `enabled` à `false` :

**Via le fichier de configuration :**
```json
{
  "tools": {
    "exec": {
      "enabled": false
    }
  }
}
```

**Via la variable d'environnement :**
```bash
PICOCLAW_TOOLS_EXEC_ENABLED=false
```

> **Note :** Lorsqu'il est désactivé, l'agent ne pourra pas exécuter de commandes shell. Cela affecte également la capacité de l'outil Cron à exécuter des commandes shell planifiées.

### Fonctionnalité

- **`enable_deny_patterns`** : Définir à `false` pour désactiver complètement les modèles de blocage par défaut des commandes dangereuses
- **`custom_deny_patterns`** : Ajouter des modèles regex de refus personnalisés ; les commandes correspondantes seront bloquées

### Modèles de commandes bloquées par défaut

Par défaut, PicoClaw bloque les commandes dangereuses suivantes :

- Commandes de suppression : `rm -rf`, `del /f/q`, `rmdir /s`
- Opérations disque : `format`, `mkfs`, `diskpart`, `dd if=`, écriture vers `/dev/sd*`
- Opérations système : `shutdown`, `reboot`, `poweroff`
- Substitution de commandes : `$()`, `${}`, backticks
- Pipe vers shell : `| sh`, `| bash`
- Élévation de privilèges : `sudo`, `chmod`, `chown`
- Contrôle de processus : `pkill`, `killall`, `kill -9`
- Opérations distantes : `curl | sh`, `wget | sh`, `ssh`
- Gestion de paquets : `apt`, `yum`, `dnf`, `npm install -g`, `pip install --user`
- Conteneurs : `docker run`, `docker exec`
- Git : `git push`, `git force`
- Autres : `eval`, `source *.sh`

### Limitation architecturale connue

Le garde exec ne valide que la commande de niveau supérieur envoyée à PicoClaw. Il n'inspecte **pas** récursivement les processus enfants générés par les outils de build ou les scripts après le démarrage de cette commande.

Exemples de workflows pouvant contourner le garde de commande directe une fois la commande initiale autorisée :

- `make run`
- `go run ./cmd/...`
- `cargo run`
- `npm run build`

Cela signifie que le garde est utile pour bloquer les commandes directes manifestement dangereuses, mais ce n'est **pas** un bac à sable complet pour les pipelines de build non vérifiés. Si votre modèle de menace inclut du code non fiable dans l'espace de travail, utilisez une isolation plus forte comme des conteneurs, des VM ou un flux d'approbation autour des commandes de build et d'exécution.

### Exemple de configuration

```json
{
  "tools": {
    "exec": {
      "enable_deny_patterns": true,
      "custom_deny_patterns": [
        "\\brm\\s+-r\\b",
        "\\bkillall\\s+python"
      ]
    }
  }
}
```

## Outil Cron

L'outil cron est utilisé pour planifier des tâches périodiques.

| Config                 | Type | Par défaut | Description                                        |
|------------------------|------|------------|----------------------------------------------------|
| `exec_timeout_minutes` | int  | 5          | Délai d'expiration en minutes, 0 signifie sans limite |

## Outil MCP

L'outil MCP permet l'intégration avec des serveurs Model Context Protocol externes.

### Découverte d'outils (chargement paresseux)

Lors de la connexion à plusieurs serveurs MCP, exposer simultanément des centaines d'outils peut épuiser la fenêtre de contexte du LLM et augmenter les coûts API. La fonctionnalité **Discovery** résout ce problème en gardant les outils MCP *masqués* par défaut.

Au lieu de charger tous les outils, le LLM reçoit un outil de recherche léger (utilisant la correspondance par mots-clés BM25 ou les expressions régulières). Lorsque le LLM a besoin d'une capacité spécifique, il recherche dans la bibliothèque masquée. Les outils correspondants sont alors temporairement « déverrouillés » et injectés dans le contexte pour un nombre configuré de tours (`ttl`).

### Configuration globale

| Config      | Type   | Par défaut | Description                                  |
|-------------|--------|------------|----------------------------------------------|
| `enabled`   | bool   | false      | Activer l'intégration MCP globalement        |
| `discovery` | object | `{}`       | Configuration de la découverte d'outils (voir ci-dessous) |
| `servers`   | object | `{}`       | Mappage du nom de serveur à la configuration du serveur |

### Configuration Discovery (`discovery`)

| Config               | Type | Par défaut | Description                                                                                                                       |
|----------------------|------|------------|-----------------------------------------------------------------------------------------------------------------------------------|
| `enabled`            | bool | false      | Si true, les outils MCP sont masqués et chargés à la demande via la recherche. Si false, tous les outils sont chargés             |
| `ttl`                | int  | 5          | Nombre de tours de conversation pendant lesquels un outil découvert reste déverrouillé                                            |
| `max_search_results` | int  | 5          | Nombre maximum d'outils retournés par requête de recherche                                                                        |
| `use_bm25`           | bool | true       | Activer l'outil de recherche par langage naturel/mots-clés (`tool_search_tool_bm25`). **Attention** : consomme plus de ressources que la recherche regex |
| `use_regex`          | bool | false      | Activer l'outil de recherche par motif regex (`tool_search_tool_regex`)                                                           |

> **Note :** Si `discovery.enabled` est `true`, vous **devez** activer au moins un moteur de recherche (`use_bm25` ou `use_regex`),
> sinon l'application ne démarrera pas.

### Configuration par serveur

| Config     | Type   | Requis   | Description                                |
|------------|--------|----------|--------------------------------------------|
| `enabled`  | bool   | oui      | Activer ce serveur MCP                     |
| `type`     | string | non      | Type de transport : `stdio`, `sse`, `http` |
| `command`  | string | stdio    | Commande exécutable pour le transport stdio |
| `args`     | array  | non      | Arguments de commande pour le transport stdio |
| `env`      | object | non      | Variables d'environnement pour le processus stdio |
| `env_file` | string | non      | Chemin vers le fichier d'environnement pour le processus stdio |
| `url`      | string | sse/http | URL du point de terminaison pour le transport `sse`/`http` |
| `headers`  | object | non      | En-têtes HTTP pour le transport `sse`/`http` |

### Comportement du transport

- Si `type` est omis, le transport est détecté automatiquement :
    - `url` est défini → `sse`
    - `command` est défini → `stdio`
- `http` et `sse` utilisent tous deux `url` + `headers` optionnels.
- `env` et `env_file` ne sont appliqués qu'aux serveurs `stdio`.

### Exemples de configuration

#### 1) Serveur MCP Stdio

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "servers": {
        "filesystem": {
          "enabled": true,
          "command": "npx",
          "args": [
            "-y",
            "@modelcontextprotocol/server-filesystem",
            "/tmp"
          ]
        }
      }
    }
  }
}
```

#### 2) Serveur MCP distant SSE/HTTP

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "servers": {
        "remote-mcp": {
          "enabled": true,
          "type": "sse",
          "url": "https://example.com/mcp",
          "headers": {
            "Authorization": "Bearer YOUR_TOKEN"
          }
        }
      }
    }
  }
}
```

#### 3) Configuration MCP massive avec découverte d'outils activée

*Dans cet exemple, le LLM ne verra que `tool_search_tool_bm25`. Il recherchera et déverrouillera dynamiquement les outils Github ou Postgres uniquement lorsque l'utilisateur le demande.*

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "discovery": {
        "enabled": true,
        "ttl": 5,
        "max_search_results": 5,
        "use_bm25": true,
        "use_regex": false
      },
      "servers": {
        "github": {
          "enabled": true,
          "command": "npx",
          "args": [
            "-y",
            "@modelcontextprotocol/server-github"
          ],
          "env": {
            "GITHUB_PERSONAL_ACCESS_TOKEN": "YOUR_GITHUB_TOKEN"
          }
        },
        "postgres": {
          "enabled": true,
          "command": "npx",
          "args": [
            "-y",
            "@modelcontextprotocol/server-postgres",
            "postgresql://user:password@localhost/dbname"
          ]
        },
        "slack": {
          "enabled": true,
          "command": "npx",
          "args": [
            "-y",
            "@modelcontextprotocol/server-slack"
          ],
          "env": {
            "SLACK_BOT_TOKEN": "YOUR_SLACK_BOT_TOKEN",
            "SLACK_TEAM_ID": "YOUR_SLACK_TEAM_ID"
          }
        }
      }
    }
  }
}
```

## Outil Skills

L'outil skills configure la découverte et l'installation de compétences via des registres comme ClawHub.

### Registres

| Config                             | Type   | Par défaut           | Description                                  |
|------------------------------------|--------|----------------------|----------------------------------------------|
| `registries.clawhub.enabled`       | bool   | true                 | Activer le registre ClawHub                  |
| `registries.clawhub.base_url`      | string | `https://clawhub.ai` | URL de base ClawHub                          |
| `registries.clawhub.auth_token`    | string | `""`                 | Jeton Bearer optionnel pour des limites de débit plus élevées |
| `registries.clawhub.search_path`   | string | `/api/v1/search`     | Chemin de l'API de recherche                 |
| `registries.clawhub.skills_path`   | string | `/api/v1/skills`     | Chemin de l'API Skills                       |
| `registries.clawhub.download_path` | string | `/api/v1/download`   | Chemin de l'API de téléchargement            |

### Exemple de configuration

```json
{
  "tools": {
    "skills": {
      "registries": {
        "clawhub": {
          "enabled": true,
          "base_url": "https://clawhub.ai",
          "auth_token": "",
          "search_path": "/api/v1/search",
          "skills_path": "/api/v1/skills",
          "download_path": "/api/v1/download"
        }
      }
    }
  }
}
```

## Variables d'environnement

Toutes les options de configuration peuvent être remplacées via des variables d'environnement au format `PICOCLAW_TOOLS_<SECTION>_<KEY>` :

Par exemple :

- `PICOCLAW_TOOLS_WEB_BRAVE_ENABLED=true`
- `PICOCLAW_TOOLS_EXEC_ENABLED=false`
- `PICOCLAW_TOOLS_EXEC_ENABLE_DENY_PATTERNS=false`
- `PICOCLAW_TOOLS_CRON_EXEC_TIMEOUT_MINUTES=10`
- `PICOCLAW_TOOLS_MCP_ENABLED=true`

Note : La configuration de type map imbriquée (par exemple `tools.mcp.servers.<name>.*`) est configurée dans `config.json` plutôt que via des variables d'environnement.
