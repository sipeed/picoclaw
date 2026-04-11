# 💬 Configuração de Aplicativos de Chat

> Voltar ao [README](../../README.pt-br.md)

## 💬 Aplicativos de Chat

Converse com seu picoclaw através do Telegram, Discord, WhatsApp, Matrix, QQ, DingTalk, LINE, WeCom, Feishu, Slack, IRC, OneBot ou MaixCam

> **Nota**: Todos os canais baseados em webhook (LINE, WeCom, etc.) são servidos em um único servidor HTTP Gateway compartilhado (`gateway.host`:`gateway.port`, padrão `127.0.0.1:18790`). Não há portas por canal para configurar. Nota: Feishu usa o modo WebSocket/SDK e não utiliza o servidor HTTP webhook compartilhado.

| Canal                | Dificuldade        | Descrição                                             | Documentação                                                                                                     |
| -------------------- | ------------------ | ----------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| **Telegram**         | ⭐ Fácil           | Recomendado, voz para texto, long polling (sem IP público) | [Documentação](../channels/telegram/README.pt-br.md)                                                       |
| **Discord**          | ⭐ Fácil           | Socket Mode, suporte a grupos/DM, ecossistema bot rico | [Documentação](../channels/discord/README.pt-br.md)                                                            |
| **WhatsApp**         | ⭐ Fácil           | Nativo (scan QR) ou Bridge URL                        | [Documentação](#whatsapp)                                                                                        |
| **Weixin**           | ⭐ Fácil           | Scan QR nativo (API Tencent iLink)                    | [Documentação](#weixin)                                                                                          |
| **Slack**            | ⭐ Fácil           | **Socket Mode** (sem IP público), empresarial         | [Documentação](../channels/slack/README.pt-br.md)                                                               |
| **Matrix**           | ⭐⭐ Médio         | Protocolo federado, suporte a auto-hospedagem         | [Documentação](../channels/matrix/README.pt-br.md)                                                              |
| **QQ**               | ⭐⭐ Médio         | API bot oficial, comunidade chinesa                   | [Documentação](../channels/qq/README.pt-br.md)                                                                  |
| **DingTalk**         | ⭐⭐ Médio         | Modo Stream (sem IP público), empresarial             | [Documentação](../channels/dingtalk/README.pt-br.md)                                                            |
| **LINE**             | ⭐⭐⭐ Avançado    | HTTPS Webhook obrigatório                             | [Documentação](../channels/line/README.pt-br.md)                                                                |
| **WeCom (企业微信)** | ⭐⭐⭐ Avançado    | Bot de grupo (Webhook), app personalizado (API), AI Bot | [Bot](../channels/wecom/wecom_bot/README.pt-br.md) / [App](../channels/wecom/wecom_app/README.pt-br.md) / [AI Bot](../channels/wecom/wecom_aibot/README.pt-br.md) |
| **Feishu (飞书)**    | ⭐⭐⭐ Avançado    | Colaboração empresarial, rico em recursos             | [Documentação](../channels/feishu/README.pt-br.md)                                                              |
| **IRC**              | ⭐⭐ Médio         | Servidor + configuração TLS                           | [Documentação](#irc) |
| **OneBot**           | ⭐⭐ Médio         | Compatível com NapCat/Go-CQHTTP, ecossistema comunitário | [Documentação](../channels/onebot/README.pt-br.md)                                                           |
| **MaixCam**          | ⭐ Fácil           | Canal de integração de hardware para câmeras AI Sipeed | [Documentação](../channels/maixcam/README.pt-br.md)                                                            |
| **Pico**             | ⭐ Fácil           | Canal de protocolo nativo PicoClaw                    |                                                                                                                  |

<a id="telegram"></a>
<details>
<summary><b>Telegram</b> (Recomendado)</summary>

**1. Criar um bot**

* Abra o Telegram, pesquise `@BotFather`
* Envie `/newbot`, siga as instruções
* Copie o token

**2. Configurar**

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

> Obtenha seu ID de usuário com `@userinfobot` no Telegram.

**3. Executar**

```bash
picoclaw gateway
```

**4. Menu de comandos do Telegram (registrado automaticamente na inicialização)**

O PicoClaw agora mantém definições de comandos em um registro compartilhado. Na inicialização, o Telegram registrará automaticamente os comandos de bot suportados (por exemplo `/start`, `/help`, `/show`, `/list`) para que o menu de comandos e o comportamento em tempo de execução permaneçam sincronizados.
O registro do menu de comandos do Telegram permanece como descoberta UX local do canal; a execução genérica de comandos é tratada centralmente no loop do agente via commands executor.

Se o registro de comandos falhar (erros transitórios de rede/API), o canal ainda inicia e o PicoClaw tenta novamente o registro em segundo plano.

</details>

<a id="discord"></a>
<details>
<summary><b>Discord</b></summary>

**1. Criar um bot**

* Acesse <https://discord.com/developers/applications>
* Crie um aplicativo → Bot → Add Bot
* Copie o token do bot

**2. Habilitar intents**

* Nas configurações do Bot, habilite **MESSAGE CONTENT INTENT**
* (Opcional) Habilite **SERVER MEMBERS INTENT** se planeja usar listas de permissão baseadas em dados de membros

**3. Obter seu User ID**
* Configurações do Discord → Avançado → habilite **Developer Mode**
* Clique com o botão direito no seu avatar → **Copy User ID**

**4. Configurar**

```json
{
  "channel_list": {
    "discord": {
      "enabled": true,
      "type": "discord",
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

**5. Convidar o bot**

* OAuth2 → URL Generator
* Scopes: `bot`
* Bot Permissions: `Send Messages`, `Read Message History`
* Abra a URL de convite gerada e adicione o bot ao seu servidor

**Opcional: Modo de ativação em grupo**

Por padrão, o bot responde a todas as mensagens em um canal do servidor. Para restringir respostas apenas a @menções, adicione:

```json
{
  "channel_list": {
    "discord": {
      "group_trigger": { "mention_only": true }
    }
  }
}
```

Você também pode ativar por prefixos de palavras-chave (ex.: `!bot`):

```json
{
  "channel_list": {
    "discord": {
      "group_trigger": { "prefixes": ["!bot"] }
    }
  }
}
```

**6. Executar**

```bash
picoclaw gateway
```

</details>

<a id="whatsapp"></a>
<details>
<summary><b>WhatsApp</b> (nativo via whatsmeow)</summary>

O PicoClaw pode se conectar ao WhatsApp de duas formas:

- **Nativo (recomendado):** In-process usando [whatsmeow](https://github.com/tulir/whatsmeow). Sem bridge separado. Defina `"use_native": true` e deixe `bridge_url` vazio. Na primeira execução, escaneie o QR code com o WhatsApp (Dispositivos Vinculados). A sessão é armazenada no seu workspace (ex.: `workspace/whatsapp/`). O canal nativo é **opcional** para manter o binário padrão pequeno; compile com `-tags whatsapp_native` (ex.: `make build-whatsapp-native` ou `go build -tags whatsapp_native ./cmd/...`).
- **Bridge:** Conecte-se a um bridge WebSocket externo. Defina `bridge_url` (ex.: `ws://localhost:3001`) e mantenha `use_native` como false.

**Configurar (nativo)**

```json
{
  "channel_list": {
    "whatsapp": {
      "enabled": true,
      "type": "whatsapp",
      "use_native": true,
      "session_store_path": "",
      "allow_from": []
    }
  }
}
```

Se `session_store_path` estiver vazio, a sessão é armazenada em `<workspace>/whatsapp/`. Execute `picoclaw gateway`; na primeira execução, escaneie o QR code impresso no terminal com WhatsApp → Dispositivos Vinculados.

</details>

<a id="weixin"></a>
<details>
<summary><b>Weixin</b> (WeChat Pessoal)</summary>

O PicoClaw suporta conexão com sua conta pessoal do WeChat usando a API oficial Tencent iLink.

**1. Login**

Execute o fluxo de login interativo por QR code:
```bash
picoclaw auth weixin
```
Escaneie o QR code exibido com seu aplicativo WeChat mobile. Após o login bem-sucedido, o token é salvo na sua configuração.

**2. Configurar**

(Opcional) Adicione seu ID de usuário WeChat em `allow_from` para restringir quem pode enviar mensagens ao bot:
```json
{
  "channel_list": {
    "weixin": {
      "enabled": true,
      "type": "weixin",
      "token": "YOUR_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

**3. Executar**
```bash
picoclaw gateway
```

</details>

<a id="qq"></a>
<details>
<summary><b>QQ</b></summary>

**Configuração rápida (recomendada)**

A QQ Open Platform oferece uma página de configuração com um clique para bots compatíveis com OpenClaw:

1. Abra o [QQ Bot Quick Start](https://q.qq.com/qqbot/openclaw/index.html) e escaneie o QR code para fazer login
2. Um bot é criado automaticamente — copie o **App ID** e o **App Secret**
3. Configure o PicoClaw:

```json
{
  "channel_list": {
    "qq": {
      "enabled": true,
      "type": "qq",
      "app_id": "YOUR_APP_ID",
      "app_secret": "YOUR_APP_SECRET",
      "allow_from": []
    }
  }
}
```

4. Execute `picoclaw gateway` e abra o QQ para conversar com seu bot

> O App Secret é exibido apenas uma vez. Salve-o imediatamente — visualizá-lo novamente forçará uma redefinição.
>
> Bots criados pela página de configuração rápida são inicialmente apenas para o criador e não suportam chats de grupo. Para habilitar o acesso em grupo, configure o modo sandbox na [QQ Open Platform](https://q.qq.com/).

**Configuração manual**

Se preferir criar o bot manualmente:

* Faça login na [QQ Open Platform](https://q.qq.com/) para se registrar como desenvolvedor
* Crie um bot QQ — personalize seu avatar e nome
* Copie o **App ID** e o **App Secret** nas configurações do bot
* Configure conforme mostrado acima e execute `picoclaw gateway`

</details>

<a id="dingtalk"></a>
<details>
<summary><b>DingTalk</b></summary>

**1. Criar um bot**

* Acesse a [Open Platform](https://open.dingtalk.com/)
* Crie um aplicativo interno
* Copie o Client ID e o Client Secret

**2. Configurar**

```json
{
  "channel_list": {
    "dingtalk": {
      "enabled": true,
      "type": "dingtalk",
      "client_id": "YOUR_CLIENT_ID",
      "client_secret": "YOUR_CLIENT_SECRET",
      "allow_from": []
    }
  }
}
```

> Defina `allow_from` como vazio para permitir todos os usuários, ou especifique IDs de usuário DingTalk para restringir o acesso.

**3. Executar**

```bash
picoclaw gateway
```

</details>

<a id="maixcam"></a>
<details>
<summary><b>MaixCam</b></summary>

Canal de integração projetado especificamente para hardware de câmera AI Sipeed.

```json
{
  "channel_list": {
    "maixcam": {
      "enabled": true,
      "type": "maixcam"
    }
  }
}
```

```bash
picoclaw gateway
```

</details>


<a id="matrix"></a>
<details>
<summary><b>Matrix</b></summary>

**1. Preparar conta do bot**

* Use seu homeserver preferido (ex.: `https://matrix.org` ou auto-hospedado)
* Crie um usuário bot e obtenha seu access token

**2. Configurar**

```json
{
  "channel_list": {
    "matrix": {
      "enabled": true,
      "type": "matrix",
      "homeserver": "https://matrix.org",
      "user_id": "@your-bot:matrix.org",
      "access_token": "YOUR_MATRIX_ACCESS_TOKEN",
      "allow_from": []
    }
  }
}
```

**3. Executar**

```bash
picoclaw gateway
```

Para opções completas (`device_id`, `join_on_invite`, `group_trigger`, `placeholder`, `reasoning_channel_id`), veja o [Guia de Configuração do Canal Matrix](../channels/matrix/README.md).

</details>

<a id="line"></a>
<details>
<summary><b>LINE</b></summary>

**1. Criar uma Conta Oficial LINE**

- Acesse o [LINE Developers Console](https://developers.line.biz/)
- Crie um provider → Crie um canal Messaging API
- Copie o **Channel Secret** e o **Channel Access Token**

**2. Configurar**

```json
{
  "channel_list": {
    "line": {
      "enabled": true,
      "type": "line",
      "channel_secret": "YOUR_CHANNEL_SECRET",
      "channel_access_token": "YOUR_CHANNEL_ACCESS_TOKEN",
      "webhook_path": "/webhook/line",
      "allow_from": []
    }
  }
}
```

> O webhook do LINE é servido no servidor Gateway compartilhado (`gateway.host`:`gateway.port`, padrão `127.0.0.1:18790`).

**3. Configurar URL do Webhook**

O LINE requer HTTPS para webhooks. Use um proxy reverso ou túnel:

```bash
# Exemplo com ngrok (porta padrão do gateway é 18790)
ngrok http 18790
```

Em seguida, defina a URL do Webhook no LINE Developers Console como `https://your-domain/webhook/line` e habilite **Use webhook**.

**4. Executar**

```bash
picoclaw gateway
```

> Em chats de grupo, o bot responde apenas quando @mencionado. As respostas citam a mensagem original.

</details>

<a id="wecom"></a>
<details>
<summary><b>WeCom (企业微信)</b></summary>

O PicoClaw suporta três tipos de integração WeCom:

**Opção 1: WeCom Bot (Bot)** - Configuração mais fácil, suporta chats de grupo
**Opção 2: WeCom App (App Personalizado)** - Mais recursos, mensagens proativas, apenas chat privado
**Opção 3: WeCom AI Bot (AI Bot)** - AI Bot oficial, respostas em streaming, suporta chat de grupo e privado

Veja o [Guia de Configuração do WeCom AI Bot](../channels/wecom/wecom_aibot/README.pt-br.md) para instruções detalhadas de configuração.

**Configuração Rápida - WeCom Bot:**

**1. Criar um bot**

* Acesse o Console de Administração WeCom → Chat de Grupo → Adicionar Bot de Grupo
* Copie a URL do webhook (formato: `https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx`)

**2. Configurar**

```json
{
  "channel_list": {
    "wecom": {
      "enabled": true,
      "type": "wecom",
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "webhook_url": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY",
      "webhook_path": "/webhook/wecom",
      "allow_from": []
    }
  }
}
```

> O webhook do WeCom é servido no servidor Gateway compartilhado (`gateway.host`:`gateway.port`, padrão `127.0.0.1:18790`).

**Configuração Rápida - WeCom App:**

**1. Criar um aplicativo**

* Acesse o Console de Administração WeCom → Gerenciamento de Apps → Criar App
* Copie o **AgentId** e o **Secret**
* Acesse a página "Minha Empresa", copie o **CorpID**

**2. Configurar recebimento de mensagens**

* Nos detalhes do App, clique em "Receber Mensagem" → "Configurar API"
* Defina a URL como `http://your-server:18790/webhook/wecom-app`
* Gere o **Token** e o **EncodingAESKey**

**3. Configurar**

```json
{
  "channel_list": {
    "wecom_app": {
      "enabled": true,
      "corp_id": "wwxxxxxxxxxxxxxxxx",
      "corp_secret": "YOUR_CORP_SECRET",
      "agent_id": 1000002,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "webhook_path": "/webhook/wecom-app",
      "allow_from": []
    }
  }
}
```

**4. Executar**

```bash
picoclaw gateway
```

> **Nota**: Os callbacks de webhook do WeCom são servidos na porta do Gateway (padrão 18790). Use um proxy reverso para HTTPS.

**Configuração Rápida - WeCom AI Bot:**

**1. Criar um AI Bot**

* Acesse o Console de Administração WeCom → Gerenciamento de Apps → AI Bot
* Nas configurações do AI Bot, configure a URL de callback: `http://your-server:18790/webhook/wecom-aibot`
* Copie o **Token** e clique em "Gerar Aleatoriamente" para o **EncodingAESKey**

**2. Configurar**

```json
{
  "channel_list": {
    "wecom_aibot": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_43_CHAR_ENCODING_AES_KEY",
      "webhook_path": "/webhook/wecom-aibot",
      "allow_from": [],
      "welcome_message": "Hello! How can I help you?"
    }
  }
}
```

**3. Executar**

```bash
picoclaw gateway
```

> **Nota**: O WeCom AI Bot usa protocolo de streaming pull — sem preocupações com timeout de resposta. Tarefas longas (>30 segundos) mudam automaticamente para entrega via `response_url` push.

</details>

<a id="feishu"></a>
<details>
<summary><b>Feishu (Lark)</b></summary>

O PicoClaw se conecta ao Feishu via modo WebSocket/SDK — não é necessário URL de webhook público nem servidor de callback.

**1. Criar um aplicativo**

* Acesse a [Feishu Open Platform](https://open.feishu.cn/) e crie um aplicativo
* Nas configurações do aplicativo, habilite a capacidade **Bot**
* Crie uma versão e publique o aplicativo (o aplicativo deve ser publicado para funcionar)
* Copie o **App ID** (começa com `cli_`) e o **App Secret**

**2. Configurar**

```json
{
  "channel_list": {
    "feishu": {
      "enabled": true,
      "type": "feishu",
      "app_id": "cli_xxx",
      "app_secret": "YOUR_APP_SECRET",
      "allow_from": []
    }
  }
}
```

Opcional: `encrypt_key` e `verification_token` para criptografia de eventos (recomendado para produção).

**3. Executar e conversar**

```bash
picoclaw gateway
```

Abra o Feishu, pesquise o nome do seu bot e comece a conversar. Você também pode adicionar o bot a um grupo — use `group_trigger.mention_only: true` para responder apenas quando @mencionado.

Para opções completas, veja o [Guia de Configuração do Canal Feishu](../channels/feishu/README.pt-br.md).

</details>

<a id="slack"></a>
<details>
<summary><b>Slack</b></summary>

**1. Criar um aplicativo Slack**

* Acesse a [Slack API](https://api.slack.com/apps) e crie um novo aplicativo
* Em **OAuth & Permissions**, adicione os escopos do bot: `chat:write`, `app_mentions:read`, `im:history`, `im:read`, `im:write`
* Instale o aplicativo no seu workspace
* Copie o **Bot Token** (`xoxb-...`) e o **App-Level Token** (`xapp-...`, habilite Socket Mode para obtê-lo)

**2. Configurar**

```json
{
  "channel_list": {
    "slack": {
      "enabled": true,
      "type": "slack",
      "bot_token": "xoxb-YOUR-BOT-TOKEN",
      "app_token": "xapp-YOUR-APP-TOKEN",
      "allow_from": []
    }
  }
}
```

**3. Executar**

```bash
picoclaw gateway
```

</details>

<a id="irc"></a>
<details>
<summary><b>IRC</b></summary>

**1. Configurar**

```json
{
  "channel_list": {
    "irc": {
      "enabled": true,
      "type": "irc",
      "server": "irc.libera.chat:6697",
      "tls": true,
      "nick": "picoclaw-bot",
      "channels": ["#your-channel"],
      "password": "",
      "allow_from": []
    }
  }
}
```

Opcional: `nickserv_password` para autenticação NickServ, `sasl_user`/`sasl_password` para autenticação SASL.

**2. Executar**

```bash
picoclaw gateway
```

O bot se conectará ao servidor IRC e entrará nos canais especificados.

</details>

<a id="onebot"></a>
<details>
<summary><b>OneBot (QQ via protocolo OneBot)</b></summary>

OneBot é um protocolo aberto para bots QQ. O PicoClaw se conecta a qualquer implementação compatível com OneBot v11 (ex.: [Lagrange](https://github.com/LagrangeDev/Lagrange.Core), [NapCat](https://github.com/NapNeko/NapCatQQ)) via WebSocket.

**1. Configurar uma implementação OneBot**

Instale e execute um framework de bot QQ compatível com OneBot v11. Habilite seu servidor WebSocket.

**2. Configurar**

```json
{
  "channel_list": {
    "onebot": {
      "enabled": true,
      "type": "onebot",
      "ws_url": "ws://127.0.0.1:8080",
      "access_token": "",
      "allow_from": []
    }
  }
}
```

| Campo | Descrição |
|-------|-----------|
| `ws_url` | URL WebSocket da implementação OneBot |
| `access_token` | Token de acesso para autenticação (se configurado no OneBot) |
| `reconnect_interval` | Intervalo de reconexão em segundos (padrão: 5) |

**3. Executar**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>MaixCam</b></summary>

Canal de integração projetado especificamente para hardware de câmera AI Sipeed.

```json
{
  "channel_list": {
    "maixcam": {
      "enabled": true,
      "type": "maixcam"
    }
  }
}
```

```bash
picoclaw gateway
```

</details>
