# ⚙️ Guia de Configuração

> Voltar ao [README](../../README.pt-br.md)

## ⚙️ Configuração

Arquivo de configuração: `~/.picoclaw/config.json`

### Variáveis de Ambiente

Você pode substituir os caminhos padrão usando variáveis de ambiente. Isso é útil para instalações portáteis, implantações em contêineres ou execução do picoclaw como serviço do sistema. Essas variáveis são independentes e controlam caminhos diferentes.

| Variável          | Descrição                                                                                                                             | Caminho Padrão              |
|-------------------|-----------------------------------------------------------------------------------------------------------------------------------------|---------------------------|
| `PICOCLAW_CONFIG` | Substitui o caminho para o arquivo de configuração. Isso indica diretamente ao picoclaw qual `config.json` carregar, ignorando todos os outros locais. | `~/.picoclaw/config.json` |
| `PICOCLAW_HOME`   | Substitui o diretório raiz para dados do picoclaw. Isso altera o local padrão do `workspace` e outros diretórios de dados.          | `~/.picoclaw`             |

**Exemplos:**

```bash
# Executar picoclaw usando um arquivo de configuração específico
# O caminho do workspace será lido de dentro desse arquivo de configuração
PICOCLAW_CONFIG=/etc/picoclaw/production.json picoclaw gateway

# Executar picoclaw com todos os dados armazenados em /opt/picoclaw
# A configuração será carregada do padrão ~/.picoclaw/config.json
# O workspace será criado em /opt/picoclaw/workspace
PICOCLAW_HOME=/opt/picoclaw picoclaw agent

# Usar ambos para uma configuração totalmente personalizada
PICOCLAW_HOME=/srv/picoclaw PICOCLAW_CONFIG=/srv/picoclaw/main.json picoclaw gateway
```

### Layout do Workspace

O PicoClaw armazena dados no seu workspace configurado (padrão: `~/.picoclaw/workspace`):

```
~/.picoclaw/workspace/
├── sessions/          # Sessões de conversa e histórico
├── memory/           # Memória de longo prazo (MEMORY.md)
├── state/            # Estado persistente (último canal, etc.)
├── cron/             # Banco de dados de tarefas agendadas
├── skills/           # Skills personalizadas
├── AGENT.md          # Guia de comportamento do agente
├── HEARTBEAT.md      # Prompts de tarefas periódicas (verificados a cada 30 min)
├── IDENTITY.md       # Identidade do agente
├── SOUL.md           # Alma do agente
└── USER.md           # Preferências do usuário
```

> **Nota:** Alterações em `AGENT.md`, `SOUL.md`, `USER.md` e `memory/MEMORY.md` são detectadas automaticamente em tempo de execução via rastreamento de data de modificação (mtime). **Não é necessário reiniciar o gateway** após editar esses arquivos — o agente carrega o novo conteúdo na próxima requisição.

### Fontes de Skills

Por padrão, as skills são carregadas de:

1. `~/.picoclaw/workspace/skills` (workspace)
2. `~/.picoclaw/skills` (global)
3. `<caminho-embutido-na-compilação>/skills` (embutido)

Para configurações avançadas/de teste, você pode substituir o diretório raiz de skills builtin com:

```bash
export PICOCLAW_BUILTIN_SKILLS=/path/to/skills
```

### Política Unificada de Execução de Comandos

- Comandos slash genéricos são executados através de um único caminho em `pkg/agent/loop.go` via `commands.Executor`.
- Os adaptadores de canal não consomem mais comandos genéricos localmente; eles encaminham o texto de entrada para o caminho bus/agent. O Telegram ainda registra automaticamente os comandos suportados na inicialização.
- Comando slash desconhecido (por exemplo `/foo`) passa para o processamento normal do LLM.
- Comando registrado mas não suportado no canal atual (por exemplo `/show` no WhatsApp) retorna um erro explícito ao usuário e interrompe o processamento.

### 🔒 Sandbox de Segurança

O PicoClaw é executado em um ambiente sandbox por padrão. O agente só pode acessar arquivos e executar comandos dentro do workspace configurado.

#### Configuração Padrão

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true
    }
  }
}
```

| Opção                   | Padrão                  | Descrição                                 |
| ----------------------- | ----------------------- | ----------------------------------------- |
| `workspace`             | `~/.picoclaw/workspace` | Diretório de trabalho do agente           |
| `restrict_to_workspace` | `true`                  | Restringir acesso a arquivos/comandos ao workspace |

#### Ferramentas Protegidas

Quando `restrict_to_workspace: true`, as seguintes ferramentas são isoladas:

| Ferramenta    | Função           | Restrição                              |
| ------------- | ---------------- | -------------------------------------- |
| `read_file`   | Ler arquivos     | Apenas arquivos dentro do workspace    |
| `write_file`  | Escrever arquivos| Apenas arquivos dentro do workspace    |
| `list_dir`    | Listar diretórios| Apenas diretórios dentro do workspace  |
| `edit_file`   | Editar arquivos  | Apenas arquivos dentro do workspace    |
| `append_file` | Anexar a arquivos| Apenas arquivos dentro do workspace    |
| `exec`        | Executar comandos| Caminhos de comando devem estar dentro do workspace |

#### Proteção Adicional do Exec

Mesmo com `restrict_to_workspace: false`, a ferramenta `exec` bloqueia estes comandos perigosos:

* `rm -rf`, `del /f`, `rmdir /s` — Exclusão em massa
* `format`, `mkfs`, `diskpart` — Formatação de disco
* `dd if=` — Imagem de disco
* Escrita em `/dev/sd[a-z]` — Escritas diretas em disco
* `shutdown`, `reboot`, `poweroff` — Desligamento do sistema
* Fork bomb `:(){ :|:& };:`

### Controle de Acesso a Arquivos

| Config Key | Type | Default | Description |
|------------|------|---------|-------------|
| `tools.allow_read_paths` | string[] | `[]` | Additional paths allowed for reading outside workspace |
| `tools.allow_write_paths` | string[] | `[]` | Additional paths allowed for writing outside workspace |

### Segurança do Exec

| Config Key | Type | Default | Description |
|------------|------|---------|-------------|
| `tools.exec.allow_remote` | bool | `false` | Allow exec tool from remote channels (Telegram/Discord etc.) |
| `tools.exec.enable_deny_patterns` | bool | `true` | Enable dangerous command interception |
| `tools.exec.custom_deny_patterns` | string[] | `[]` | Custom regex patterns to block |
| `tools.exec.custom_allow_patterns` | string[] | `[]` | Custom regex patterns to allow |

> **Nota de Segurança:** A proteção contra symlinks é habilitada por padrão — todos os caminhos de arquivo são resolvidos através de `filepath.EvalSymlinks` antes da correspondência com a whitelist, prevenindo ataques de escape via symlink.

#### Limitação Conhecida: Processos Filhos de Ferramentas de Build

O guard de segurança do exec inspeciona apenas a linha de comando que o PicoClaw executa diretamente. Ele não inspeciona recursivamente processos filhos gerados por ferramentas de desenvolvimento permitidas como `make`, `go run`, `cargo`, `npm run` ou scripts de build personalizados.

Isso significa que um comando de nível superior ainda pode compilar ou executar outros binários após passar pela verificação inicial do guard. Na prática, trate scripts de build, Makefiles, scripts de pacotes e binários gerados como código executável que precisa do mesmo nível de revisão que um comando shell direto.

Para ambientes de maior risco:

* Revise scripts de build antes da execução.
* Prefira aprovação/revisão manual para fluxos de trabalho de compilação e execução.
* Execute o PicoClaw dentro de um contêiner ou VM se precisar de isolamento mais forte do que o guard integrado oferece.

#### Exemplos de Erro

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (path outside working dir)}
```

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (dangerous pattern detected)}
```

#### Desabilitando Restrições (Risco de Segurança)

Se você precisar que o agente acesse caminhos fora do workspace:

**Método 1: Arquivo de configuração**

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": false
    }
  }
}
```

**Método 2: Variável de ambiente**

```bash
export PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE=false
```

> ⚠️ **Aviso**: Desabilitar esta restrição permite que o agente acesse qualquer caminho no seu sistema. Use com cautela apenas em ambientes controlados.

#### Consistência do Limite de Segurança

A configuração `restrict_to_workspace` se aplica consistentemente em todos os caminhos de execução:

| Caminho de Execução | Limite de Segurança          |
| -------------------- | ---------------------------- |
| Main Agent           | `restrict_to_workspace` ✅   |
| Subagent / Spawn     | Herda a mesma restrição ✅   |
| Heartbeat tasks      | Herda a mesma restrição ✅   |

Todos os caminhos compartilham a mesma restrição de workspace — não há como contornar o limite de segurança através de subagentes ou tarefas agendadas.

### Heartbeat (Tarefas Periódicas)

O PicoClaw pode executar tarefas periódicas automaticamente. Crie um arquivo `HEARTBEAT.md` no seu workspace:

```markdown
# Tarefas Periódicas

- Verificar meu e-mail para mensagens importantes
- Revisar meu calendário para eventos próximos
- Verificar a previsão do tempo
```

O agente lerá este arquivo a cada 30 minutos (configurável) e executará quaisquer tarefas usando as ferramentas disponíveis.

#### Tarefas Assíncronas com Spawn

Para tarefas de longa duração (busca na web, chamadas de API), use a ferramenta `spawn` para criar um **subagente**:

```markdown
# Tarefas Periódicas

## Tarefas Rápidas (responder diretamente)

- Informar a hora atual

## Tarefas Longas (usar spawn para assíncrono)

- Pesquisar notícias de IA na web e resumir
- Verificar e-mails e reportar mensagens importantes
```

**Comportamentos principais:**

| Funcionalidade   | Descrição                                                          |
| ---------------- | ------------------------------------------------------------------ |
| **spawn**        | Cria subagente assíncrono, não bloqueia o heartbeat                |
| **Contexto independente** | Subagente tem seu próprio contexto, sem histórico de sessão |
| **message tool** | Subagente comunica diretamente com o usuário via message tool      |
| **Não-bloqueante** | Após o spawn, o heartbeat continua para a próxima tarefa         |

#### Fluxo de Comunicação do Subagente

```
Heartbeat disparado
    ↓
Agent lê HEARTBEAT.md
    ↓
Tarefa longa: spawn subagente
    ↓                           ↓
Continua próxima tarefa    Subagente trabalha independentemente
    ↓                           ↓
Todas tarefas concluídas   Subagente usa ferramenta "message"
    ↓                           ↓
Responde HEARTBEAT_OK      Usuário recebe resultado diretamente
```

**Configuração:**

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

| Opção      | Padrão | Descrição                              |
| ---------- | ------ | -------------------------------------- |
| `enabled`  | `true` | Ativar/desativar heartbeat             |
| `interval` | `30`   | Intervalo em minutos (mínimo: 5)       |

**Variáveis de ambiente:**

* `PICOCLAW_HEARTBEAT_ENABLED=false` para desativar
* `PICOCLAW_HEARTBEAT_INTERVAL=60` para alterar o intervalo

### Providers

> [!NOTE]
> O Groq fornece transcrição de voz gratuita via Whisper. Se configurado, mensagens de áudio de qualquer canal serão automaticamente transcritas no nível do agente.

| Provider     | Finalidade                              | Obter API Key                                                |
| ------------ | --------------------------------------- | ------------------------------------------------------------ |
| `gemini`     | LLM (Gemini direto)                     | [aistudio.google.com](https://aistudio.google.com)           |
| `zhipu`      | LLM (Zhipu direto)                      | [bigmodel.cn](https://bigmodel.cn)                           |
| `volcengine` | LLM (Volcengine direto)                 | [volcengine.com](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) |
| `openrouter` | LLM (recomendado, acesso a todos modelos) | [openrouter.ai](https://openrouter.ai)                     |
| `anthropic`  | LLM (Claude direto)                     | [console.anthropic.com](https://console.anthropic.com)       |
| `openai`     | LLM (GPT direto)                        | [platform.openai.com](https://platform.openai.com)           |
| `deepseek`   | LLM (DeepSeek direto)                   | [platform.deepseek.com](https://platform.deepseek.com)       |
| `qwen`       | LLM (Qwen direto)                       | [dashscope.console.aliyun.com](https://dashscope.console.aliyun.com) |
| `groq`       | LLM + **Transcrição de voz** (Whisper)  | [console.groq.com](https://console.groq.com)                 |
| `cerebras`   | LLM (Cerebras direto)                   | [cerebras.ai](https://cerebras.ai)                           |
| `vivgrid`    | LLM (Vivgrid direto)                    | [vivgrid.com](https://vivgrid.com)                           |

### Configuração de Modelos (model_list)

> **Novidade:** PicoClaw agora usa uma abordagem **centrada no modelo**. Basta especificar o formato `vendor/model` (ex.: `zhipu/glm-4.7`) para adicionar novos providers — **sem alterações de código!**

#### Todos os Vendors Suportados

| Vendor                  | Prefixo `model` | API Base padrão                                     | Protocolo | API Key                                                          |
| ----------------------- | --------------- | --------------------------------------------------- | --------- | ---------------------------------------------------------------- |
| **OpenAI**              | `openai/`       | `https://api.openai.com/v1`                         | OpenAI    | [Obter](https://platform.openai.com)                             |
| **Anthropic**           | `anthropic/`    | `https://api.anthropic.com/v1`                      | Anthropic | [Obter](https://console.anthropic.com)                           |
| **智谱 AI (GLM)**       | `zhipu/`        | `https://open.bigmodel.cn/api/paas/v4`              | OpenAI    | [Obter](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys)   |
| **DeepSeek**            | `deepseek/`     | `https://api.deepseek.com/v1`                       | OpenAI    | [Obter](https://platform.deepseek.com)                           |
| **Google Gemini**       | `gemini/`       | `https://generativelanguage.googleapis.com/v1beta`  | OpenAI    | [Obter](https://aistudio.google.com/api-keys)                    |
| **Groq**                | `groq/`         | `https://api.groq.com/openai/v1`                    | OpenAI    | [Obter](https://console.groq.com)                                |
| **通义千问 (Qwen)**     | `qwen/`         | `https://dashscope.aliyuncs.com/compatible-mode/v1` | OpenAI    | [Obter](https://dashscope.console.aliyun.com)                    |
| **Ollama**              | `ollama/`       | `http://localhost:11434/v1`                         | OpenAI    | Local (sem chave)                                                |
| **OpenRouter**          | `openrouter/`   | `https://openrouter.ai/api/v1`                      | OpenAI    | [Obter](https://openrouter.ai/keys)                              |
| **VolcEngine (Doubao)** | `volcengine/`   | `https://ark.cn-beijing.volces.com/api/v3`          | OpenAI    | [Obter](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) |
| **Antigravity**         | `antigravity/`  | Google Cloud                                        | Custom    | Somente OAuth                                                    |

#### Balanceamento de Carga

Configure múltiplos endpoints para o mesmo nome de modelo — PicoClaw fará round-robin automaticamente:

```json
{
  "model_list": [
    { "model_name": "gpt-5.4", "model": "openai/gpt-5.4", "api_base": "https://api1.example.com/v1", "api_key": "sk-key1" },
    { "model_name": "gpt-5.4", "model": "openai/gpt-5.4", "api_base": "https://api2.example.com/v1", "api_key": "sk-key2" }
  ]
}
```

#### Migração da Configuração Legada `providers`

A configuração antiga `providers` está **depreciada** mas ainda é suportada. Veja [docs/migration/model-list-migration.md](../migration/model-list-migration.md).

### Arquitetura de Providers

PicoClaw roteia providers por família de protocolo:

- **Compatível com OpenAI**: OpenRouter, Groq, Zhipu, endpoints vLLM e a maioria dos outros.
- **Anthropic**: Comportamento nativo da API Claude.
- **Codex/OAuth**: Rota de autenticação OAuth/token OpenAI.

### Tarefas Agendadas / Lembretes

PicoClaw suporta tarefas agendadas via ferramenta `cron`.

```json
{
  "tools": {
    "cron": {
      "enabled": true,
      "exec_timeout_minutes": 5
    }
  }
}
```

As tarefas agendadas persistem após reinicializações em `~/.picoclaw/workspace/cron/`.

### Tópicos Avançados

| Tópico | Descrição |
| ------ | --------- |
| [Sistema de Hooks](../hooks/README.md) | Hooks orientados a eventos: observadores, interceptores, hooks de aprovação |
| [Steering](../steering.md) | Injetar mensagens em um loop de agente em execução |
| [SubTurn](../subturn.md) | Coordenação de subagentes, controle de concorrência, ciclo de vida |
| [Gerenciamento de Contexto](../agent-refactor/context.md) | Detecção de limites de contexto, compressão |
