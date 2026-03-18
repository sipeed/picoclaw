# 💬 Configuração de Aplicativos de Chat

> Voltar ao [README](../../README.pt-br.md)

## 💬 Aplicativos de Chat

Converse com seu picoclaw através do Telegram, Discord, WhatsApp, Matrix, QQ, DingTalk, LINE, WeCom, Feishu, Slack, IRC, OneBot ou MaixCam

> **Nota**: Todos os canais baseados em webhook (LINE, WeCom, etc.) são servidos em um único servidor HTTP Gateway compartilhado (`gateway.host`:`gateway.port`, padrão `127.0.0.1:18790`). Não há portas por canal para configurar. Nota: Feishu usa o modo WebSocket/SDK e não utiliza o servidor HTTP webhook compartilhado.

| Channel      | Setup                              |
| ------------ | ---------------------------------- |
| **Telegram** | Easy (just a token)                |
| **Discord**  | Easy (bot token + intents)         |
| **WhatsApp** | Easy (native: QR scan; or bridge URL) |
| **Matrix**   | Medium (homeserver + bot access token) |
| **QQ**       | Easy (AppID + AppSecret)           |
| **DingTalk** | Medium (app credentials)           |
| **LINE**     | Medium (credentials + webhook URL) |
| **WeCom AI Bot** | Medium (Token + AES key)       |
| **Feishu**   | Medium (App ID + Secret, WebSocket mode) |
| **Slack**    | Medium (Bot token + App token) |
| **IRC**      | Medium (server + TLS config)   |
| **OneBot**   | Medium (QQ via OneBot protocol) |
| **MaixCam**  | Easy (Sipeed hardware integration) |
| **Pico**     | Native PicoClaw protocol           |

<details>
<summary><b>Telegram</b> (Recomendado)</summary>

**1. Criar um bot**

* Abra o Telegram, pesquise `@BotFather`
* Envie `/newbot`, siga as instruções
* Copie o token

**2. Configurar**

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
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
  "channels": {
    "discord": {
      "enabled": true,
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
  "channels": {
    "discord": {
      "group_trigger": { "mention_only": true }
    }
  }
}
```

Você também pode ativar por prefixos de palavras-chave (ex.: `!bot`):

```json
{
  "channels": {
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

<details>
<summary><b>WhatsApp</b> (nativo via whatsmeow)</summary>

O PicoClaw pode se conectar ao WhatsApp de duas formas:

- **Nativo (recomendado):** In-process usando [whatsmeow](https://github.com/tulir/whatsmeow). Sem bridge separado. Defina `"use_native": true` e deixe `bridge_url` vazio. Na primeira execução, escaneie o QR code com o WhatsApp (Dispositivos Vinculados). A sessão é armazenada no seu workspace (ex.: `workspace/whatsapp/`). O canal nativo é **opcional** para manter o binário padrão pequeno; compile com `-tags whatsapp_native` (ex.: `make build-whatsapp-native` ou `go build -tags whatsapp_native ./cmd/...`).
- **Bridge:** Conecte-se a um bridge WebSocket externo. Defina `bridge_url` (ex.: `ws://localhost:3001`) e mantenha `use_native` como false.

**Configurar (nativo)**

```json
{
  "channels": {
    "whatsapp": {
      "enabled": true,
      "use_native": true,
      "session_store_path": "",
      "allow_from": []
    }
  }
}
```

Se `session_store_path` estiver vazio, a sessão é armazenada em `<workspace>/whatsapp/`. Execute `picoclaw gateway`; na primeira execução, escaneie o QR code impresso no terminal com WhatsApp → Dispositivos Vinculados.

</details>

<details>
<summary><b>QQ</b></summary>

**1. Criar um bot**

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

> Defina `allow_from` como vazio para permitir todos os usuários, ou especifique números QQ para restringir o acesso.

**3. Executar**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>DingTalk</b></summary>

**1. Criar um bot**

* Acesse a [Open Platform](https://open.dingtalk.com/)
* Crie um aplicativo interno
* Copie o Client ID e o Client Secret

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

> Defina `allow_from` como vazio para permitir todos os usuários, ou especifique IDs de usuário DingTalk para restringir o acesso.

**3. Executar**

```bash
picoclaw gateway
```
</details>

<details>
<summary><b>Matrix</b></summary>

**1. Preparar conta do bot**

* Use seu homeserver preferido (ex.: `https://matrix.org` ou auto-hospedado)
* Crie um usuário bot e obtenha seu access token

**2. Configurar**

```json
{
  "channels": {
    "matrix": {
      "enabled": true,
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

Para opções completas (`device_id`, `join_on_invite`, `group_trigger`, `placeholder`, `reasoning_channel_id`), veja o [Guia de Configuração do Canal Matrix](docs/channels/matrix/README.md).

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

<details>
<summary><b>WeCom (企业微信)</b></summary>

O PicoClaw suporta três tipos de integração WeCom:

**Opção 1: WeCom Bot (Bot)** - Configuração mais fácil, suporta chats de grupo
**Opção 2: WeCom App (App Personalizado)** - Mais recursos, mensagens proativas, apenas chat privado
**Opção 3: WeCom AI Bot (AI Bot)** - AI Bot oficial, respostas em streaming, suporta chat de grupo e privado

Veja o [Guia de Configuração do WeCom AI Bot](docs/channels/wecom/wecom_aibot/README.zh.md) para instruções detalhadas de configuração.

**Configuração Rápida - WeCom Bot:**

**1. Criar um bot**

* Acesse o Console de Administração WeCom → Chat de Grupo → Adicionar Bot de Grupo
* Copie a URL do webhook (formato: `https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx`)

**2. Configurar**

```json
{
  "channels": {
    "wecom": {
      "enabled": true,
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
  "channels": {
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
* Nas configurações do AI Bot, configure a URL de callback: `http://your-server:18791/webhook/wecom-aibot`
* Copie o **Token** e clique em "Gerar Aleatoriamente" para o **EncodingAESKey**

**2. Configurar**

```json
{
  "channels": {
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
