# 🔧 Configuração de Ferramentas

> Voltar ao [README](../../README.pt-br.md)

A configuração de ferramentas do PicoClaw está localizada no campo `tools` do `config.json`.

## Estrutura de diretórios

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

## Ferramentas Web

As ferramentas web são usadas para pesquisa e busca de páginas web.

### Web Fetcher
Configurações gerais para busca e processamento de conteúdo de páginas web.

| Config              | Tipo   | Padrão        | Descrição                                                                                     |
|---------------------|--------|---------------|-----------------------------------------------------------------------------------------------|
| `enabled`           | bool   | true          | Habilitar a capacidade de busca de páginas web.                                               |
| `fetch_limit_bytes` | int    | 10485760      | Tamanho máximo do payload da página web a ser buscado, em bytes (padrão é 10MB).              |
| `format`            | string | "plaintext"   | Formato de saída do conteúdo buscado. Opções: `plaintext` ou `markdown` (recomendado).        |

### DuckDuckGo

| Config        | Tipo | Padrão | Descrição                      |
|---------------|------|--------|--------------------------------|
| `enabled`     | bool | true   | Habilitar pesquisa DuckDuckGo  |
| `max_results` | int  | 5      | Número máximo de resultados    |

### Baidu Search

| Config        | Tipo   | Padrão                                                          | Descrição                          |
|---------------|--------|-----------------------------------------------------------------|------------------------------------|
| `enabled`     | bool   | false                                                           | Habilitar pesquisa Baidu           |
| `api_key`     | string | -                                                               | Chave API Qianfan                  |
| `base_url`    | string | `https://qianfan.baidubce.com/v2/ai_search/web_search`         | URL da API Baidu Search            |
| `max_results` | int    | 10                                                              | Número máximo de resultados        |

```json
{
  "tools": {
    "web": {
      "baidu_search": {
        "enabled": true,
        "api_key": "YOUR_BAIDU_QIANFAN_API_KEY",
        "max_results": 10
      }
    }
  }
}
```

### Perplexity

| Config        | Tipo   | Padrão | Descrição                      |
|---------------|--------|--------|--------------------------------|
| `enabled`     | bool     | false  | Habilitar pesquisa Perplexity                                    |
| `api_key`     | string   | -      | Chave API do Perplexity                                          |
| `api_keys`    | string[] | -      | Várias chaves API do Perplexity para rotação (prioridade sobre `api_key`) |
| `max_results` | int      | 5      | Número máximo de resultados                                      |

### Brave

| Config        | Tipo   | Padrão | Descrição                  |
|---------------|--------|--------|----------------------------|
| `enabled`     | bool     | false  | Habilitar pesquisa Brave                                         |
| `api_key`     | string   | -      | Chave API única do Brave Search                                  |
| `api_keys`    | string[] | -      | Várias chaves API do Brave para rotação (prioridade sobre `api_key`) |
| `max_results` | int      | 5      | Número máximo de resultados                                      |

### Tavily

| Config        | Tipo   | Padrão | Descrição                          |
|---------------|--------|--------|------------------------------------|
| `enabled`     | bool   | false  | Habilitar pesquisa Tavily          |
| `api_key`     | string | -      | Chave API do Tavily                |
| `base_url`    | string | -      | URL base personalizada do Tavily   |
| `max_results` | int    | 0      | Número máximo de resultados (0 = padrão) |

### SearXNG

| Config        | Tipo   | Padrão                   | Descrição                      |
|---------------|--------|--------------------------|--------------------------------|
| `enabled`     | bool   | false                    | Habilitar pesquisa SearXNG     |
| `base_url`    | string | `http://localhost:8888`  | URL da instância SearXNG       |
| `max_results` | int    | 5                        | Número máximo de resultados    |

### GLM Search

| Config          | Tipo   | Padrão                                               | Descrição                  |
|-----------------|--------|------------------------------------------------------|----------------------------|
| `enabled`       | bool   | false                                                | Habilitar GLM Search       |
| `api_key`       | string | -                                                    | Chave API GLM              |
| `base_url`      | string | `https://open.bigmodel.cn/api/paas/v4/web_search`   | URL da API GLM Search      |
| `search_engine` | string | `search_std`                                         | Tipo de motor de busca     |
| `max_results`   | int    | 5                                                    | Número máximo de resultados |

## Ferramenta Exec

A ferramenta exec é usada para executar comandos shell.

| Config                 | Tipo  | Padrão | Descrição                                      |
|------------------------|-------|--------|-------------------------------------------------|
| `enabled`              | bool  | true   | Habilitar a ferramenta exec                     |
| `enable_deny_patterns` | bool  | true   | Habilitar bloqueio padrão de comandos perigosos |
| `custom_deny_patterns` | array | []     | Padrões de negação personalizados (expressões regulares) |

### Desabilitando a Ferramenta Exec

Para desabilitar completamente a ferramenta `exec`, defina `enabled` como `false`:

**Via arquivo de configuração:**
```json
{
  "tools": {
    "exec": {
      "enabled": false
    }
  }
}
```

**Via variável de ambiente:**
```bash
PICOCLAW_TOOLS_EXEC_ENABLED=false
```

> **Nota:** Quando desabilitada, o agent não poderá executar comandos shell. Isso também afeta a capacidade da ferramenta Cron de executar comandos shell agendados.

### Funcionalidade

- **`enable_deny_patterns`**: Defina como `false` para desabilitar completamente os padrões de bloqueio de comandos perigosos padrão
- **`custom_deny_patterns`**: Adicione padrões regex de negação personalizados; comandos correspondentes serão bloqueados

### Padrões de comandos bloqueados por padrão

Por padrão, o PicoClaw bloqueia os seguintes comandos perigosos:

- Comandos de exclusão: `rm -rf`, `del /f/q`, `rmdir /s`
- Operações de disco: `format`, `mkfs`, `diskpart`, `dd if=`, escrita em `/dev/sd*`
- Operações do sistema: `shutdown`, `reboot`, `poweroff`
- Substituição de comandos: `$()`, `${}`, crases
- Pipe para shell: `| sh`, `| bash`
- Escalação de privilégios: `sudo`, `chmod`, `chown`
- Controle de processos: `pkill`, `killall`, `kill -9`
- Operações remotas: `curl | sh`, `wget | sh`, `ssh`
- Gerenciamento de pacotes: `apt`, `yum`, `dnf`, `npm install -g`, `pip install --user`
- Contêineres: `docker run`, `docker exec`
- Git: `git push`, `git force`
- Outros: `eval`, `source *.sh`

### Limitação arquitetural conhecida

O guarda exec apenas valida o comando de nível superior enviado ao PicoClaw. Ele **não** inspeciona recursivamente processos filhos gerados por ferramentas de build ou scripts após o início desse comando.

Exemplos de fluxos de trabalho que podem contornar o guarda de comando direto uma vez que o comando inicial é permitido:

- `make run`
- `go run ./cmd/...`
- `cargo run`
- `npm run build`

Isso significa que o guarda é útil para bloquear comandos diretos obviamente perigosos, mas **não** é um sandbox completo para pipelines de build não revisados. Se seu modelo de ameaça inclui código não confiável no workspace, use isolamento mais forte, como contêineres, VMs ou um fluxo de aprovação em torno de comandos de build e execução.

### Exemplo de configuração

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

## Ferramenta Cron

A ferramenta cron é usada para agendar tarefas periódicas.

| Config                 | Tipo | Padrão | Descrição                                          |
|------------------------|------|--------|-----------------------------------------------------|
| `exec_timeout_minutes` | int  | 5      | Tempo limite de execução em minutos, 0 significa sem limite |

## Ferramenta MCP

A ferramenta MCP permite a integração com servidores Model Context Protocol externos.

### Descoberta de ferramentas (carregamento preguiçoso)

Ao conectar a vários servidores MCP, expor centenas de ferramentas simultaneamente pode esgotar a janela de contexto do LLM e aumentar os custos de API. O recurso **Discovery** resolve isso mantendo as ferramentas MCP *ocultas* por padrão.

Em vez de carregar todas as ferramentas, o LLM recebe uma ferramenta de pesquisa leve (usando correspondência de palavras-chave BM25 ou Regex). Quando o LLM precisa de uma capacidade específica, ele pesquisa a biblioteca oculta. As ferramentas correspondentes são então temporariamente "desbloqueadas" e injetadas no contexto por um número configurado de turnos (`ttl`).

### Configuração global

| Config      | Tipo   | Padrão | Descrição                                    |
|-------------|--------|--------|----------------------------------------------|
| `enabled`   | bool   | false  | Habilitar integração MCP globalmente         |
| `discovery` | object | `{}`   | Configuração de descoberta de ferramentas (veja abaixo) |
| `servers`   | object | `{}`   | Mapa de nome do servidor para configuração do servidor |

### Configuração Discovery (`discovery`)

| Config               | Tipo | Padrão | Descrição                                                                                                                         |
|----------------------|------|--------|-----------------------------------------------------------------------------------------------------------------------------------|
| `enabled`            | bool | false  | Se true, as ferramentas MCP ficam ocultas e são carregadas sob demanda via pesquisa. Se false, todas as ferramentas são carregadas |
| `ttl`                | int  | 5      | Número de turnos de conversa que uma ferramenta descoberta permanece desbloqueada                                                 |
| `max_search_results` | int  | 5      | Número máximo de ferramentas retornadas por consulta de pesquisa                                                                  |
| `use_bm25`           | bool | true   | Habilitar a ferramenta de pesquisa por linguagem natural/palavras-chave (`tool_search_tool_bm25`). **Aviso**: consome mais recursos que a pesquisa regex |
| `use_regex`          | bool | false  | Habilitar a ferramenta de pesquisa por padrão regex (`tool_search_tool_regex`)                                                    |

> **Nota:** Se `discovery.enabled` for `true`, você **deve** habilitar pelo menos um mecanismo de pesquisa (`use_bm25` ou `use_regex`),
> caso contrário a aplicação falhará ao iniciar.

### Configuração por servidor

| Config     | Tipo   | Obrigatório | Descrição                                  |
|------------|--------|-------------|--------------------------------------------|
| `enabled`  | bool   | sim         | Habilitar este servidor MCP                |
| `type`     | string | não         | Tipo de transporte: `stdio`, `sse`, `http` |
| `command`  | string | stdio       | Comando executável para transporte stdio   |
| `args`     | array  | não         | Argumentos do comando para transporte stdio |
| `env`      | object | não         | Variáveis de ambiente para processo stdio  |
| `env_file` | string | não         | Caminho para arquivo de ambiente para processo stdio |
| `url`      | string | sse/http    | URL do endpoint para transporte `sse`/`http` |
| `headers`  | object | não         | Cabeçalhos HTTP para transporte `sse`/`http` |

### Comportamento do transporte

- Se `type` for omitido, o transporte é detectado automaticamente:
    - `url` está definido → `sse`
    - `command` está definido → `stdio`
- `http` e `sse` ambos usam `url` + `headers` opcionais.
- `env` e `env_file` são aplicados apenas a servidores `stdio`.

### Exemplos de configuração

#### 1) Servidor MCP Stdio

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

#### 2) Servidor MCP remoto SSE/HTTP

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

#### 3) Configuração MCP massiva com descoberta de ferramentas habilitada

*Neste exemplo, o LLM verá apenas o `tool_search_tool_bm25`. Ele pesquisará e desbloqueará ferramentas do Github ou Postgres dinamicamente apenas quando solicitado pelo usuário.*

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

## Ferramenta Skills

A ferramenta skills configura a descoberta e instalação de habilidades via registros como o ClawHub.

### Registros

| Config                             | Tipo   | Padrão               | Descrição                                    |
|------------------------------------|--------|-----------------------|----------------------------------------------|
| `registries.clawhub.enabled`       | bool   | true                  | Habilitar registro ClawHub                   |
| `registries.clawhub.base_url`      | string | `https://clawhub.ai`  | URL base do ClawHub                          |
| `registries.clawhub.auth_token`    | string | `""`                  | Token Bearer opcional para limites de taxa mais altos |
| `registries.clawhub.search_path`   | string | `/api/v1/search`      | Caminho da API de pesquisa                   |
| `registries.clawhub.skills_path`   | string | `/api/v1/skills`      | Caminho da API de Skills                     |
| `registries.clawhub.download_path` | string | `/api/v1/download`    | Caminho da API de download                   |

### Exemplo de configuração

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

## Variáveis de ambiente

Todas as opções de configuração podem ser substituídas via variáveis de ambiente com o formato `PICOCLAW_TOOLS_<SECTION>_<KEY>`:

Por exemplo:

- `PICOCLAW_TOOLS_WEB_BRAVE_ENABLED=true`
- `PICOCLAW_TOOLS_EXEC_ENABLED=false`
- `PICOCLAW_TOOLS_EXEC_ENABLE_DENY_PATTERNS=false`
- `PICOCLAW_TOOLS_CRON_EXEC_TIMEOUT_MINUTES=10`
- `PICOCLAW_TOOLS_MCP_ENABLED=true`

Nota: Configuração de tipo mapa aninhado (por exemplo `tools.mcp.servers.<name>.*`) é configurada no `config.json` em vez de variáveis de ambiente.
