<div align="center">
<img src="assets/logo.jpg" alt="PicoClaw" width="512">

<h1>PicoClaw: Assistente de IA Ultra-Eficiente em Go</h1>

<h3>Hardware de $10 · 10MB de RAM · Boot em 1s · 皮皮虾，我们走！</h3>

  <p>
    <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/Arch-x86__64%2C%20ARM64%2C%20RISC--V-blue" alt="Hardware">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
    <br>
    <a href="https://picoclaw.io"><img src="https://img.shields.io/badge/Website-picoclaw.io-blue?style=flat&logo=google-chrome&logoColor=white" alt="Website"></a>
    <a href="https://x.com/SipeedIO"><img src="https://img.shields.io/badge/X_(Twitter)-SipeedIO-black?style=flat&logo=x&logoColor=white" alt="Twitter"></a>
  </p>

 [中文](README.zh.md) | [日本語](README.ja.md) | [English](README.md) | **Português**
</div>

---

🦐 **PicoClaw** é um assistente pessoal de IA ultra-leve inspirado no [nanobot](https://github.com/HKUDS/nanobot), reescrito do zero em **Go** por meio de um processo de "auto-inicialização" (self-bootstrapping) — onde o próprio agente de IA conduziu toda a migração de arquitetura e otimização de código.

⚡️ **Extremamente leve:** Roda em hardware de apenas **$10** com **<10MB** de RAM. Isso é 99% menos memória que o OpenClaw e 98% mais barato que um Mac mini!

<table align="center">
<tr align="center">
<td align="center" valign="top">
<p align="center">
<img src="assets/picoclaw_mem.gif" width="360" height="240">
</p>
</td>
<td align="center" valign="top">
<p align="center">
<img src="assets/licheervnano.png" width="400" height="240">
</p>
</td>
</tr>
</table>

> [!CAUTION]
> **🚨 DECLARACAO DE SEGURANCA & CANAIS OFICIAIS**
>
> * **SEM CRIPTOMOEDAS:** O PicoClaw **NAO** possui nenhum token/moeda oficial. Todas as alegacoes no `pump.fun` ou outras plataformas de negociacao sao **GOLPES**.
> * **DOMINIO OFICIAL:** O **UNICO** site oficial e **[picoclaw.io](https://picoclaw.io)**, e o site da empresa e **[sipeed.com](https://sipeed.com)**.
> * **Aviso:** Muitos dominios `.ai/.org/.com/.net/...` foram registrados por terceiros, nao sao nossos.
> * **Aviso:** O PicoClaw esta em fase inicial de desenvolvimento e pode ter problemas de seguranca de rede nao resolvidos. Nao implante em ambientes de producao antes da versao v1.0.
> * **Nota:** O PicoClaw recentemente fez merge de muitos PRs, o que pode resultar em maior consumo de memoria (10-20MB) nas versoes mais recentes. Planejamos priorizar a otimizacao de recursos assim que o conjunto de funcionalidades estiver estavel.


## 📢 Novidades

2026-02-16 🎉 PicoClaw atingiu 12K stars em uma semana! Obrigado a todos pelo apoio! O PicoClaw esta crescendo mais rapido do que jamais imaginamos. Dado o alto volume de PRs, precisamos urgentemente de maintainers da comunidade. Nossos papeis de voluntarios e roadmap foram publicados oficialmente [aqui](docs/picoclaw_community_roadmap_260216.md) — estamos ansiosos para ter voce a bordo!

2026-02-13 🎉 PicoClaw atingiu 5000 stars em 4 dias! Obrigado a comunidade! Estamos finalizando o **Roadmap do Projeto** e configurando o **Grupo de Desenvolvedores** para acelerar o desenvolvimento do PicoClaw.
🚀 **Chamada para Acao:** Envie suas solicitacoes de funcionalidades nas GitHub Discussions. Revisaremos e priorizaremos na proxima reuniao semanal.

2026-02-09 🎉 PicoClaw lancado oficialmente! Construido em 1 dia para trazer Agentes de IA para hardware de $10 com <10MB de RAM. 🦐 PicoClaw, Partiu!

## ✨ Funcionalidades

🪶 **Ultra-Leve**: Consumo de memoria <10MB — 99% menor que o Clawdbot para funcionalidades essenciais.

💰 **Custo Minimo**: Eficiente o suficiente para rodar em hardware de $10 — 98% mais barato que um Mac mini.

⚡️ **Inicializacao Relampago**: Tempo de inicializacao 400X mais rapido, boot em 1 segundo mesmo em CPU single-core de 0.6GHz.

🌍 **Portabilidade Real**: Um unico binario auto-contido para RISC-V, ARM e x86. Um clique e ja era!

🤖 **Auto-Construido por IA**: Implementacao nativa em Go de forma autonoma — 95% do nucleo gerado pelo Agente com refinamento humano no loop.

|                               | OpenClaw      | NanoBot                  | **PicoClaw**                              |
| ----------------------------- | ------------- | ------------------------ | ----------------------------------------- |
| **Linguagem**                 | TypeScript    | Python                   | **Go**                                    |
| **RAM**                       | >1GB          | >100MB                   | **< 10MB**                                |
| **Inicializacao**</br>(CPU 0.8GHz) | >500s         | >30s                     | **<1s**                                   |
| **Custo**                     | Mac Mini $599 | Maioria dos SBC Linux </br>~$50 | **Qualquer placa Linux**</br>**A partir de $10** |

<img src="assets/compare.jpg" alt="PicoClaw" width="512">

## 🦾 Demonstracao

### 🛠️ Fluxos de Trabalho Padrao do Assistente

<table align="center">
<tr align="center">
<th><p align="center">🧩 Engenharia Full-Stack</p></th>
<th><p align="center">🗂️ Gerenciamento de Logs & Planejamento</p></th>
<th><p align="center">🔎 Busca Web & Aprendizado</p></th>
</tr>
<tr>
<td align="center"><p align="center"><img src="assets/picoclaw_code.gif" width="240" height="180"></p></td>
<td align="center"><p align="center"><img src="assets/picoclaw_memory.gif" width="240" height="180"></p></td>
<td align="center"><p align="center"><img src="assets/picoclaw_search.gif" width="240" height="180"></p></td>
</tr>
<tr>
<td align="center">Desenvolver • Implantar • Escalar</td>
<td align="center">Agendar • Automatizar • Memorizar</td>
<td align="center">Descobrir • Analisar • Tendencias</td>
</tr>
</table>

### 📱 Rode em celulares Android antigos

De uma segunda vida ao seu celular de dez anos atras! Transforme-o em um assistente de IA inteligente com o PicoClaw. Inicio rapido:

1. **Instale o Termux** (Disponivel no F-Droid ou Google Play).
2. **Execute os comandos**

```bash
# Nota: Substitua v0.1.1 pela versao mais recente da pagina de Releases
wget https://github.com/sipeed/picoclaw/releases/download/v0.1.1/picoclaw-linux-arm64
chmod +x picoclaw-linux-arm64
pkg install proot
termux-chroot ./picoclaw-linux-arm64 onboard
```

Depois siga as instrucoes na secao "Inicio Rapido" para completar a configuracao!

<img src="assets/termux.jpg" alt="PicoClaw" width="512">

### 🐜 Implantacao Inovadora com Baixo Consumo

O PicoClaw pode ser implantado em praticamente qualquer dispositivo Linux!

- $9.9 [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) versao E (Ethernet) ou W (WiFi6), para Assistente Domestico Minimalista
- $30~50 [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html), ou $100 [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html) para Manutencao Automatizada de Servidores
- $50 [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) ou $100 [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera) para Monitoramento Inteligente

https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4

🌟 Mais cenarios de implantacao aguardam voce!

## 📦 Instalacao

### Instalar com binario pre-compilado

Baixe o binario para sua plataforma na pagina de [releases](https://github.com/sipeed/picoclaw/releases).

### Instalar a partir do codigo-fonte (funcionalidades mais recentes, recomendado para desenvolvimento)

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# Build, sem necessidade de instalar
make build

# Build para multiplas plataformas
make build-all

# Build e Instalar
make install
```

## 🐳 Docker Compose

Voce tambem pode rodar o PicoClaw usando Docker Compose sem instalar nada localmente.

```bash
# 1. Clone este repositorio
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. Configure suas API keys
cp config/config.example.json config/config.json
vim config/config.json      # Configure DISCORD_BOT_TOKEN, API keys, etc.

# 3. Build & Iniciar
docker compose --profile gateway up -d

# 4. Ver logs
docker compose logs -f picoclaw-gateway

# 5. Parar
docker compose --profile gateway down
```

### Modo Agente (Execucao unica)

```bash
# Fazer uma pergunta
docker compose run --rm picoclaw-agent -m "Quanto e 2+2?"

# Modo interativo
docker compose run --rm picoclaw-agent
```

### Rebuild

```bash
docker compose --profile gateway build --no-cache
docker compose --profile gateway up -d
```

### 🚀 Inicio Rapido

> [!TIP]
> Configure sua API key em `~/.picoclaw/config.json`.
> Obtenha API keys: [OpenRouter](https://openrouter.ai/keys) (LLM) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) (LLM)
> Busca web e **opcional** — obtenha a [Brave Search API](https://brave.com/search/api) gratuita (2000 consultas gratis/mes) ou use o fallback automatico integrado.

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
      "model": "glm-4.7",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "xxx",
      "api_base": "https://openrouter.ai/api/v1"
    }
  },
  "tools": {
    "web": {
      "brave": {
        "enabled": false,
        "api_key": "YOUR_BRAVE_API_KEY",
        "max_results": 5
      },
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      }
    }
  }
}
```

**3. Obter API Keys**

* **Provedor de LLM**: [OpenRouter](https://openrouter.ai/keys) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) · [Anthropic](https://console.anthropic.com) · [OpenAI](https://platform.openai.com) · [Gemini](https://aistudio.google.com/api-keys)
* **Busca Web** (opcional): [Brave Search](https://brave.com/search/api) - Plano gratuito disponivel (2000 consultas/mes)

> **Nota**: Veja `config.example.json` para um modelo de configuracao completo.

**4. Conversar**

```bash
picoclaw agent -m "Quanto e 2+2?"
```

Pronto! Voce tem um assistente de IA funcionando em 2 minutos.

---

## 💬 Integracao com Apps de Chat

Converse com seu PicoClaw via Telegram, Discord, DingTalk ou LINE.

| Canal | Nivel de Configuracao |
| --- | --- |
| **Telegram** | Facil (apenas um token) |
| **Discord** | Facil (bot token + intents) |
| **QQ** | Facil (AppID + AppSecret) |
| **DingTalk** | Medio (credenciais do app) |
| **LINE** | Medio (credenciais + webhook URL) |

<details>
<summary><b>Telegram</b> (Recomendado)</summary>

**1. Criar o bot**

* Abra o Telegram, busque `@BotFather`
* Envie `/newbot`, siga as instrucoes
* Copie o token

**2. Configurar**

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allowFrom": ["YOUR_USER_ID"]
    }
  }
}
```

> Obtenha seu User ID pelo `@userinfobot` no Telegram.

**3. Executar**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>Discord</b></summary>

**1. Criar o bot**

* Acesse <https://discord.com/developers/applications>
* Crie um aplicativo → Bot → Add Bot
* Copie o token do bot

**2. Habilitar Intents**

* Nas configuracoes do Bot, habilite **MESSAGE CONTENT INTENT**
* (Opcional) Habilite **SERVER MEMBERS INTENT** se quiser usar lista de permissoes baseada em dados dos membros

**3. Obter seu User ID**

* Configuracoes do Discord → Avancado → habilite **Modo Desenvolvedor**
* Clique com botao direito no seu avatar → **Copiar ID do Usuario**

**4. Configurar**

```json
{
  "channels": {
    "discord": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allowFrom": ["YOUR_USER_ID"]
    }
  }
}
```

**5. Convidar o bot**

* OAuth2 → URL Generator
* Scopes: `bot`
* Bot Permissions: `Send Messages`, `Read Message History`
* Abra a URL de convite gerada e adicione o bot ao seu servidor

**6. Executar**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>QQ</b></summary>

**1. Criar o bot**

- Acesse a [QQ Open Platform](https://q.qq.com/#)
- Crie um aplicativo → Obtenha **AppID** e **AppSecret**

**2. Configurar**

```json
{
  "channels": {
    "qq": {
      "enabled": true,
      "app_id": "YOUR_APP_ID",
      "app_secret": "YOUR_APP_SECRET",
      "allow_from": []
    }
  }
}
```

> Deixe `allow_from` vazio para permitir todos os usuarios, ou especifique numeros QQ para restringir o acesso.

**3. Executar**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>DingTalk</b></summary>

**1. Criar o bot**

* Acesse a [Open Platform](https://open.dingtalk.com/)
* Crie um app interno
* Copie o Client ID e Client Secret

**2. Configurar**

```json
{
  "channels": {
    "dingtalk": {
      "enabled": true,
      "client_id": "YOUR_CLIENT_ID",
      "client_secret": "YOUR_CLIENT_SECRET",
      "allow_from": []
    }
  }
}
```

> Deixe `allow_from` vazio para permitir todos os usuarios, ou especifique IDs para restringir o acesso.

**3. Executar**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>LINE</b></summary>

**1. Criar uma Conta Oficial LINE**

- Acesse o [LINE Developers Console](https://developers.line.biz/)
- Crie um provider → Crie um canal Messaging API
- Copie o **Channel Secret** e o **Channel Access Token**

**2. Configurar**

```json
{
  "channels": {
    "line": {
      "enabled": true,
      "channel_secret": "YOUR_CHANNEL_SECRET",
      "channel_access_token": "YOUR_CHANNEL_ACCESS_TOKEN",
      "webhook_host": "0.0.0.0",
      "webhook_port": 18791,
      "webhook_path": "/webhook/line",
      "allow_from": []
    }
  }
}
```

**3. Configurar URL do Webhook**

O LINE requer HTTPS para webhooks. Use um reverse proxy ou tunnel:

```bash
# Exemplo com ngrok
ngrok http 18791
```

Em seguida, configure a Webhook URL no LINE Developers Console para `https://seu-dominio/webhook/line` e habilite **Use webhook**.

**4. Executar**

```bash
picoclaw gateway
```

> Em chats de grupo, o bot responde apenas quando mencionado com @. As respostas citam a mensagem original.

> **Docker Compose**: Adicione `ports: ["18791:18791"]` ao servico `picoclaw-gateway` para expor a porta do webhook.

</details>

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> Junte-se a Rede Social de Agentes

Conecte o PicoClaw a Rede Social de Agentes simplesmente enviando uma unica mensagem via CLI ou qualquer App de Chat integrado.

**Leia `https://clawdchat.ai/skill.md` e siga as instrucoes para entrar no [ClawdChat.ai](https://clawdchat.ai)**

## ⚙️ Configuracao Detalhada

Arquivo de configuracao: `~/.picoclaw/config.json`

### Estrutura do Workspace

O PicoClaw armazena dados no workspace configurado (padrao: `~/.picoclaw/workspace`):

```
~/.picoclaw/workspace/
├── sessions/          # Sessoes de conversa e historico
├── memory/           # Memoria de longo prazo (MEMORY.md)
├── state/            # Estado persistente (ultimo canal, etc.)
├── cron/             # Banco de dados de tarefas agendadas
├── skills/           # Skills personalizadas
├── AGENTS.md         # Guia de comportamento do Agente
├── HEARTBEAT.md      # Prompts de tarefas periodicas (verificado a cada 30 min)
├── IDENTITY.md       # Identidade do Agente
├── SOUL.md           # Alma do Agente
├── TOOLS.md          # Descricao das ferramentas
└── USER.md           # Preferencias do usuario
```

### 🔒 Sandbox de Seguranca

O PicoClaw roda em um ambiente sandbox por padrao. O agente so pode acessar arquivos e executar comandos dentro do workspace configurado.

#### Configuracao Padrao

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

| Opcao | Padrao | Descricao |
|-------|--------|-----------|
| `workspace` | `~/.picoclaw/workspace` | Diretorio de trabalho do agente |
| `restrict_to_workspace` | `true` | Restringir acesso de arquivos/comandos ao workspace |

#### Ferramentas Protegidas

Quando `restrict_to_workspace: true`, as seguintes ferramentas sao restritas ao sandbox:

| Ferramenta | Funcao | Restricao |
|------------|--------|-----------|
| `read_file` | Ler arquivos | Apenas arquivos dentro do workspace |
| `write_file` | Escrever arquivos | Apenas arquivos dentro do workspace |
| `list_dir` | Listar diretorios | Apenas diretorios dentro do workspace |
| `edit_file` | Editar arquivos | Apenas arquivos dentro do workspace |
| `append_file` | Adicionar a arquivos | Apenas arquivos dentro do workspace |
| `exec` | Executar comandos | Caminhos dos comandos devem estar dentro do workspace |

#### Protecao Adicional do Exec

Mesmo com `restrict_to_workspace: false`, a ferramenta `exec` bloqueia estes comandos perigosos:

* `rm -rf`, `del /f`, `rmdir /s` — Exclusao em massa
* `format`, `mkfs`, `diskpart` — Formatacao de disco
* `dd if=` — Criacao de imagem de disco
* Escrita em `/dev/sd[a-z]` — Escrita direta no disco
* `shutdown`, `reboot`, `poweroff` — Desligamento do sistema
* Fork bomb `:(){ :|:& };:`

#### Exemplos de Erro

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (path outside working dir)}
```

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (dangerous pattern detected)}
```

#### Desabilitar Restricoes (Risco de Seguranca)

Se voce precisa que o agente acesse caminhos fora do workspace:

**Metodo 1: Arquivo de configuracao**

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": false
    }
  }
}
```

**Metodo 2: Variavel de ambiente**

```bash
export PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE=false
```

> ⚠️ **Aviso**: Desabilitar esta restricao permite que o agente acesse qualquer caminho no seu sistema. Use com cuidado apenas em ambientes controlados.

#### Consistencia do Limite de Seguranca

A configuracao `restrict_to_workspace` se aplica consistentemente em todos os caminhos de execucao:

| Caminho de Execucao | Limite de Seguranca |
|----------------------|---------------------|
| Agente Principal | `restrict_to_workspace` ✅ |
| Subagente / Spawn | Herda a mesma restricao ✅ |
| Tarefas Heartbeat | Herda a mesma restricao ✅ |

Todos os caminhos compartilham a mesma restricao de workspace — nao ha como contornar o limite de seguranca por meio de subagentes ou tarefas agendadas.

### Heartbeat (Tarefas Periodicas)

O PicoClaw pode executar tarefas periodicas automaticamente. Crie um arquivo `HEARTBEAT.md` no seu workspace:

```markdown
# Tarefas Periodicas

- Verificar meu email para mensagens importantes
- Revisar minha agenda para proximos eventos
- Verificar a previsao do tempo
```

O agente lera este arquivo a cada 30 minutos (configuravel) e executara as tarefas usando as ferramentas disponiveis.

#### Tarefas Assincronas com Spawn

Para tarefas de longa duracao (busca web, chamadas de API), use a ferramenta `spawn` para criar um **subagente**:

```markdown
# Tarefas Periodicas

## Tarefas Rapidas (resposta direta)
- Informar hora atual

## Tarefas Longas (usar spawn para async)
- Buscar noticias de IA na web e resumir
- Verificar email e reportar mensagens importantes
```

**Comportamentos principais:**

| Funcionalidade | Descricao |
|----------------|-----------|
| **spawn** | Cria subagente assincrono, nao bloqueia o heartbeat |
| **Contexto independente** | Subagente tem seu proprio contexto, sem historico de sessao |
| **Ferramenta message** | Subagente se comunica diretamente com o usuario via ferramenta message |
| **Nao-bloqueante** | Apos o spawn, o heartbeat continua para a proxima tarefa |

#### Como Funciona a Comunicacao do Subagente

```
Heartbeat dispara
    ↓
Agente le HEARTBEAT.md
    ↓
Para tarefa longa: spawn subagente
    ↓                           ↓
Continua proxima tarefa    Subagente trabalha independentemente
    ↓                           ↓
Todas tarefas concluidas   Subagente usa ferramenta "message"
    ↓                           ↓
Responde HEARTBEAT_OK      Usuario recebe resultado diretamente
```

O subagente tem acesso as ferramentas (message, web_search, etc.) e pode se comunicar com o usuario independentemente sem passar pelo agente principal.

**Configuracao:**

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

| Opcao | Padrao | Descricao |
|-------|--------|-----------|
| `enabled` | `true` | Habilitar/desabilitar heartbeat |
| `interval` | `30` | Intervalo de verificacao em minutos (min: 5) |

**Variaveis de ambiente:**

* `PICOCLAW_HEARTBEAT_ENABLED=false` para desabilitar
* `PICOCLAW_HEARTBEAT_INTERVAL=60` para alterar o intervalo

### Provedores

> [!NOTE]
> O Groq fornece transcricao de voz gratuita via Whisper. Se configurado, mensagens de voz do Telegram serao automaticamente transcritas.

| Provedor | Finalidade | Obter API Key |
| --- | --- | --- |
| `gemini` | LLM (Gemini direto) | [aistudio.google.com](https://aistudio.google.com) |
| `zhipu` | LLM (Zhipu direto) | [bigmodel.cn](bigmodel.cn) |
| `openrouter` (Em teste) | LLM (recomendado, acesso a todos os modelos) | [openrouter.ai](https://openrouter.ai) |
| `anthropic` (Em teste) | LLM (Claude direto) | [console.anthropic.com](https://console.anthropic.com) |
| `openai` (Em teste) | LLM (GPT direto) | [platform.openai.com](https://platform.openai.com) |
| `deepseek` (Em teste) | LLM (DeepSeek direto) | [platform.deepseek.com](https://platform.deepseek.com) |
| `groq` | LLM + **Transcricao de voz** (Whisper) | [console.groq.com](https://console.groq.com) |

<details>
<summary><b>Configuracao Zhipu</b></summary>

**1. Obter API key**

* Obtenha a [API key](https://bigmodel.cn/usercenter/proj-mgmt/apikeys)

**2. Configurar**

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "glm-4.7",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "providers": {
    "zhipu": {
      "api_key": "Sua API Key",
      "api_base": "https://open.bigmodel.cn/api/paas/v4"
    }
  }
}
```

**3. Executar**

```bash
picoclaw agent -m "Ola, como vai?"
```

</details>

<details>
<summary><b>Exemplo de configuracao completa</b></summary>

```json
{
  "agents": {
    "defaults": {
      "model": "anthropic/claude-opus-4-5"
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-xxx"
    },
    "groq": {
      "api_key": "gsk_xxx"
    }
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "123456:ABC...",
      "allow_from": ["123456789"]
    },
    "discord": {
      "enabled": true,
      "token": "",
      "allow_from": [""]
    },
    "whatsapp": {
      "enabled": false
    },
    "feishu": {
      "enabled": false,
      "app_id": "cli_xxx",
      "app_secret": "xxx",
      "encrypt_key": "",
      "verification_token": "",
      "allow_from": []
    },
    "qq": {
      "enabled": false,
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

## Referencia CLI

| Comando | Descricao |
| --- | --- |
| `picoclaw onboard` | Inicializar configuracao & workspace |
| `picoclaw agent -m "..."` | Conversar com o agente |
| `picoclaw agent` | Modo de chat interativo |
| `picoclaw gateway` | Iniciar o gateway (para bots de chat) |
| `picoclaw status` | Mostrar status |
| `picoclaw cron list` | Listar todas as tarefas agendadas |
| `picoclaw cron add ...` | Adicionar uma tarefa agendada |

### Tarefas Agendadas / Lembretes

O PicoClaw suporta lembretes agendados e tarefas recorrentes por meio da ferramenta `cron`:

* **Lembretes unicos**: "Remind me in 10 minutes" (Me lembre em 10 minutos) → dispara uma vez apos 10min
* **Tarefas recorrentes**: "Remind me every 2 hours" (Me lembre a cada 2 horas) → dispara a cada 2 horas
* **Expressoes Cron**: "Remind me at 9am daily" (Me lembre as 9h todos os dias) → usa expressao cron

As tarefas sao armazenadas em `~/.picoclaw/workspace/cron/` e processadas automaticamente.

## 🤝 Contribuir & Roadmap

PRs sao bem-vindos! O codigo-fonte e intencionalmente pequeno e legivel. 🤗

Roadmap em breve...

Grupo de desenvolvedores em formacao. Requisito de entrada: Pelo menos 1 PR com merge.

Grupos de usuarios:

Discord: <https://discord.gg/V4sAZ9XWpN>

<img src="assets/wechat.png" alt="PicoClaw" width="512">

## 🐛 Solucao de Problemas

### Busca web mostra "API 配置问题"

Isso e normal se voce ainda nao configurou uma API key de busca. O PicoClaw fornecera links uteis para busca manual.

Para habilitar a busca web:

1. **Opcao 1 (Recomendado)**: Obtenha uma API key gratuita em [https://brave.com/search/api](https://brave.com/search/api) (2000 consultas gratis/mes) para os melhores resultados.
2. **Opcao 2 (Sem Cartao de Credito)**: Se voce nao tem uma key, o sistema automaticamente usa o **DuckDuckGo** como fallback (sem necessidade de key).

Adicione a key em `~/.picoclaw/config.json` se usar o Brave:

```json
{
  "tools": {
    "web": {
      "brave": {
        "enabled": true,
        "api_key": "YOUR_BRAVE_API_KEY",
        "max_results": 5
      },
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      }
    }
  }
}
```

### Erros de filtragem de conteudo

Alguns provedores (como Zhipu) possuem filtragem de conteudo. Tente reformular sua pergunta ou use um modelo diferente.

### Bot do Telegram diz "Conflict: terminated by other getUpdates"

Isso acontece quando outra instancia do bot esta rodando. Certifique-se de que apenas um `picoclaw gateway` esteja rodando por vez.

---

## 📝 Comparacao de API Keys

| Servico | Plano Gratuito | Caso de Uso |
| --- | --- | --- |
| **OpenRouter** | 200K tokens/mes | Multiplos modelos (Claude, GPT-4, etc.) |
| **Zhipu** | 200K tokens/mes | Melhor para usuarios chineses |
| **Brave Search** | 2000 consultas/mes | Funcionalidade de busca web |
| **Groq** | Plano gratuito disponivel | Inferencia ultra-rapida (Llama, Mixtral) |
