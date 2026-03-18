# 💬 Configuration des Applications de Chat

> Retour au [README](../../README.fr.md)

## 💬 Applications de Chat

Communiquez avec votre PicoClaw via Telegram, Discord, WhatsApp, Matrix, QQ, DingTalk, LINE, WeCom, Feishu, Slack, IRC, OneBot ou MaixCam.

> **Note** : Tous les canaux basés sur les webhooks (LINE, WeCom, etc.) sont servis sur un seul serveur HTTP Gateway partagé (`gateway.host`:`gateway.port`, par défaut `127.0.0.1:18790`). Il n'y a pas de ports par canal à configurer. Note : Feishu utilise le mode WebSocket/SDK et n'utilise pas le serveur HTTP webhook partagé.

| Canal        | Configuration                          |
| ------------ | -------------------------------------- |
| **Telegram** | Facile (juste un token)                |
| **Discord**  | Facile (bot token + intents)           |
| **WhatsApp** | Facile (natif : scan QR ; ou bridge URL) |
| **Matrix**   | Moyen (homeserver + bot access token)  |
| **QQ**       | Facile (AppID + AppSecret)             |
| **DingTalk** | Moyen (identifiants de l'application)  |
| **LINE**     | Moyen (identifiants + webhook URL)     |
| **WeCom AI Bot** | Moyen (Token + clé AES)           |
| **Feishu**   | Moyen (App ID + Secret, mode WebSocket) |
| **Slack**    | Moyen (Bot token + App token)          |
| **IRC**      | Moyen (serveur + configuration TLS)    |
| **OneBot**   | Moyen (QQ via protocole OneBot)        |
| **MaixCam**  | Facile (intégration matérielle Sipeed) |
| **Pico**     | Native PicoClaw protocol           |

<details>
<summary><b>Telegram</b> (Recommandé)</summary>

**1. Créer un bot**

* Ouvrez Telegram, recherchez `@BotFather`
* Envoyez `/newbot`, suivez les instructions
* Copiez le token

**2. Configurer**

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

> Obtenez votre identifiant utilisateur via `@userinfobot` sur Telegram.

**3. Lancer**

```bash
picoclaw gateway
```

**4. Menu de commandes Telegram (enregistré automatiquement au démarrage)**

PicoClaw conserve les définitions de commandes dans un registre partagé unique. Au démarrage, Telegram enregistre automatiquement les commandes bot prises en charge (par exemple `/start`, `/help`, `/show`, `/list`) afin que le menu de commandes et le comportement à l'exécution restent synchronisés.
L'enregistrement du menu de commandes Telegram reste une découverte UX locale au canal ; l'exécution générique des commandes est gérée de manière centralisée dans la boucle agent via l'exécuteur de commandes.

Si l'enregistrement des commandes échoue (erreurs transitoires réseau/API), le canal démarre quand même et PicoClaw réessaie l'enregistrement en arrière-plan.

</details>

<details>
<summary><b>Discord</b></summary>

**1. Créer un bot**

* Allez sur <https://discord.com/developers/applications>
* Créez une application → Bot → Add Bot
* Copiez le token du bot

**2. Activer les intents**

* Dans les paramètres du Bot, activez **MESSAGE CONTENT INTENT**
* (Optionnel) Activez **SERVER MEMBERS INTENT** si vous prévoyez d'utiliser des listes d'autorisation basées sur les données des membres

**3. Obtenir votre identifiant utilisateur**
* Paramètres Discord → Avancé → activez **Developer Mode**
* Clic droit sur votre avatar → **Copy User ID**

**4. Configurer**

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

**5. Inviter le bot**

* OAuth2 → URL Generator
* Scopes : `bot`
* Bot Permissions : `Send Messages`, `Read Message History`
* Ouvrez l'URL d'invitation générée et ajoutez le bot à votre serveur

**Mode déclenchement en groupe (optionnel)**

Par défaut, le bot répond à tous les messages dans un canal de serveur. Pour limiter les réponses aux @mentions uniquement, ajoutez :

```json
{
  "channels": {
    "discord": {
      "group_trigger": { "mention_only": true }
    }
  }
}
```

Vous pouvez également déclencher par préfixes de mots-clés (par ex. `!bot`) :

```json
{
  "channels": {
    "discord": {
      "group_trigger": { "prefixes": ["!bot"] }
    }
  }
}
```

**6. Lancer**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>WhatsApp</b> (natif via whatsmeow)</summary>

PicoClaw peut se connecter à WhatsApp de deux manières :

- **Natif (recommandé) :** En processus via [whatsmeow](https://github.com/tulir/whatsmeow). Pas de bridge séparé. Définissez `"use_native": true` et laissez `bridge_url` vide. Au premier lancement, scannez le code QR avec WhatsApp (Appareils liés). La session est stockée dans votre workspace (par ex. `workspace/whatsapp/`). Le canal natif est **optionnel** pour garder le binaire par défaut léger ; compilez avec `-tags whatsapp_native` (par ex. `make build-whatsapp-native` ou `go build -tags whatsapp_native ./cmd/...`).
- **Bridge :** Connectez-vous à un bridge WebSocket externe. Définissez `bridge_url` (par ex. `ws://localhost:3001`) et gardez `use_native` à false.

**Configurer (natif)**

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

Si `session_store_path` est vide, la session est stockée dans `<workspace>/whatsapp/`. Lancez `picoclaw gateway` ; au premier lancement, scannez le code QR affiché dans le terminal avec WhatsApp → Appareils liés.

</details>

<details>
<summary><b>QQ</b></summary>

**1. Créer un bot**

- Allez sur [QQ Open Platform](https://q.qq.com/#)
- Créez une application → Obtenez **AppID** et **AppSecret**

**2. Configurer**

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

> Définissez `allow_from` vide pour autoriser tous les utilisateurs, ou spécifiez des numéros QQ pour restreindre l'accès.

**3. Lancer**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>DingTalk</b></summary>

**1. Créer un bot**

* Allez sur [Open Platform](https://open.dingtalk.com/)
* Créez une application interne
* Copiez le Client ID et le Client Secret

**2. Configurer**

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

> Définissez `allow_from` vide pour autoriser tous les utilisateurs, ou spécifiez des identifiants DingTalk pour restreindre l'accès.

**3. Lancer**

```bash
picoclaw gateway
```
</details>

<details>
<summary><b>Matrix</b></summary>

**1. Préparer le compte bot**

* Utilisez votre homeserver préféré (par ex. `https://matrix.org` ou auto-hébergé)
* Créez un utilisateur bot et obtenez son access token

**2. Configurer**

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

**3. Lancer**

```bash
picoclaw gateway
```

Pour toutes les options (`device_id`, `join_on_invite`, `group_trigger`, `placeholder`, `reasoning_channel_id`), voir le [Guide de Configuration du Canal Matrix](docs/channels/matrix/README.md).

</details>

<details>
<summary><b>LINE</b></summary>

**1. Créer un compte officiel LINE**

- Allez sur [LINE Developers Console](https://developers.line.biz/)
- Créez un provider → Créez un canal Messaging API
- Copiez le **Channel Secret** et le **Channel Access Token**

**2. Configurer**

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

> Le webhook LINE est servi sur le serveur Gateway partagé (`gateway.host`:`gateway.port`, par défaut `127.0.0.1:18790`).

**3. Configurer l'URL du Webhook**

LINE nécessite HTTPS pour les webhooks. Utilisez un reverse proxy ou un tunnel :

```bash
# Exemple avec ngrok (le port par défaut du gateway est 18790)
ngrok http 18790
```

Puis définissez l'URL du Webhook dans la console LINE Developers à `https://your-domain/webhook/line` et activez **Use webhook**.

**4. Lancer**

```bash
picoclaw gateway
```

> Dans les discussions de groupe, le bot ne répond que lorsqu'il est @mentionné. Les réponses citent le message original.

</details>

<details>
<summary><b>WeCom (企业微信)</b></summary>

PicoClaw prend en charge trois types d'intégration WeCom :

**Option 1 : WeCom Bot (Bot)** - Configuration plus facile, prend en charge les discussions de groupe
**Option 2 : WeCom App (Application personnalisée)** - Plus de fonctionnalités, messagerie proactive, chat privé uniquement
**Option 3 : WeCom AI Bot (Bot IA)** - Bot IA officiel, réponses en streaming, prend en charge les discussions de groupe et privées

Voir le [Guide de Configuration WeCom AI Bot](docs/channels/wecom/wecom_aibot/README.zh.md) pour les instructions détaillées.

**Configuration rapide - WeCom Bot :**

**1. Créer un bot**

* Allez dans la console d'administration WeCom → Discussion de groupe → Ajouter un bot de groupe
* Copiez l'URL du webhook (format : `https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx`)

**2. Configurer**

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

> Le webhook WeCom est servi sur le serveur Gateway partagé (`gateway.host`:`gateway.port`, par défaut `127.0.0.1:18790`).

**Configuration rapide - WeCom App :**

**1. Créer une application**

* Allez dans la console d'administration WeCom → Gestion des applications → Créer une application
* Copiez **AgentId** et **Secret**
* Allez sur la page "Mon entreprise", copiez **CorpID**

**2. Configurer la réception des messages**

* Dans les détails de l'application, cliquez sur "Recevoir les messages" → "Configurer l'API"
* Définissez l'URL à `http://your-server:18790/webhook/wecom-app`
* Générez **Token** et **EncodingAESKey**

**3. Configurer**

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

**4. Lancer**

```bash
picoclaw gateway
```

> **Note** : Les callbacks webhook WeCom sont servis sur le port Gateway (par défaut 18790). Utilisez un reverse proxy pour HTTPS.

**Configuration rapide - WeCom AI Bot :**

**1. Créer un AI Bot**

* Allez dans la console d'administration WeCom → Gestion des applications → AI Bot
* Dans les paramètres du AI Bot, configurez l'URL de callback : `http://your-server:18791/webhook/wecom-aibot`
* Copiez **Token** et cliquez sur "Générer aléatoirement" pour **EncodingAESKey**

**2. Configurer**

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

**3. Lancer**

```bash
picoclaw gateway
```

> **Note** : WeCom AI Bot utilise le protocole streaming pull — pas de problème de timeout de réponse. Les tâches longues (>30 secondes) basculent automatiquement vers la livraison push via `response_url`.

</details>

<details>
<summary><b>Feishu (飞书)</b></summary>

**1. Créer une application**

* Allez sur [Feishu Open Platform](https://open.feishu.cn/)
* Créez une application → Obtenez **App ID** et **App Secret**

**2. Configurer**

```json
{
  "channels": {
    "feishu": {
      "enabled": true,
      "app_id": "cli_xxx",
      "app_secret": "xxx",
      "encrypt_key": "",
      "verification_token": "",
      "allow_from": []
    }
  }
}
```

> Feishu utilise le mode WebSocket/SDK et ne nécessite pas de serveur webhook.

**3. Lancer**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>Slack</b></summary>

**1. Créer une application Slack**

* Allez sur [Slack API](https://api.slack.com/apps)
* Créez une nouvelle application
* Obtenez le **Bot Token** et l'**App Token**

**2. Configurer**

```json
{
  "channels": {
    "slack": {
      "enabled": true,
      "bot_token": "xoxb-your-bot-token",
      "app_token": "xapp-your-app-token",
      "allow_from": []
    }
  }
}
```

**3. Lancer**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>IRC</b></summary>

**1. Configurer le serveur IRC**

* Préparez les informations de votre serveur IRC (adresse, port, canal)

**2. Configurer**

```json
{
  "channels": {
    "irc": {
      "enabled": true,
      "server": "irc.example.com:6697",
      "nick": "picoclaw-bot",
      "channel": "#your-channel",
      "use_tls": true,
      "allow_from": []
    }
  }
}
```

**3. Lancer**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>OneBot</b></summary>

**1. Configurer OneBot**

* Installez une implémentation OneBot compatible (par ex. go-cqhttp, Lagrange)
* Configurez la connexion WebSocket

**2. Configurer**

```json
{
  "channels": {
    "onebot": {
      "enabled": true,
      "ws_url": "ws://localhost:8080",
      "allow_from": []
    }
  }
}
```

> OneBot permet d'utiliser QQ via le protocole OneBot standard.

**3. Lancer**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>MaixCam</b></summary>

**1. Préparer le matériel**

* Obtenez un appareil [Sipeed MaixCam](https://wiki.sipeed.com/maixcam)

**2. Configurer**

```json
{
  "channels": {
    "maixcam": {
      "enabled": true,
      "allow_from": []
    }
  }
}
```

> MaixCam est une intégration matérielle Sipeed pour l'interaction IA embarquée.

**3. Lancer**

```bash
picoclaw gateway
```

</details>
