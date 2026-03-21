# 💬 Configuration des Applications de Chat

> Retour au [README](../../README.fr.md)

## 💬 Applications de Chat

Communiquez avec votre PicoClaw via Telegram, Discord, WhatsApp, Matrix, QQ, DingTalk, LINE, WeCom, Feishu, Slack, IRC, OneBot ou MaixCam.

> **Note** : Tous les canaux basés sur les webhooks (LINE, WeCom, etc.) sont servis sur un seul serveur HTTP Gateway partagé (`gateway.host`:`gateway.port`, par défaut `127.0.0.1:18790`). Il n'y a pas de ports par canal à configurer. Note : Feishu utilise le mode WebSocket/SDK et n'utilise pas le serveur HTTP webhook partagé.

| Canal                | Difficulté         | Description                                           | Documentation                                                                                                    |
| -------------------- | ------------------ | ----------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| **Telegram**         | ⭐ Facile          | Recommandé, transcription vocale, long polling (pas d'IP publique requise) | [Documentation](../channels/telegram/README.fr.md)                                                  |
| **Discord**          | ⭐ Facile          | Socket Mode, groupes/DM, écosystème bot riche         | [Documentation](../channels/discord/README.fr.md)                                                               |
| **WhatsApp**         | ⭐ Facile          | Natif (scan QR) ou Bridge URL                         | [Documentation](#whatsapp)                                                                                       |
| **Slack**            | ⭐ Facile          | **Socket Mode** (pas d'IP publique requise), entreprise | [Documentation](../channels/slack/README.fr.md)                                                                |
| **Matrix**           | ⭐⭐ Moyen         | Protocole fédéré, auto-hébergement possible           | [Documentation](../channels/matrix/README.fr.md)                                                                |
| **QQ**               | ⭐⭐ Moyen         | API bot officielle, communauté chinoise               | [Documentation](../channels/qq/README.fr.md)                                                                    |
| **DingTalk**         | ⭐⭐ Moyen         | Mode Stream (pas d'IP publique requise), entreprise   | [Documentation](../channels/dingtalk/README.fr.md)                                                              |
| **LINE**             | ⭐⭐⭐ Avancé      | HTTPS Webhook requis                                  | [Documentation](../channels/line/README.fr.md)                                                                  |
| **WeCom (企业微信)** | ⭐⭐⭐ Avancé      | Bot groupe (Webhook), app personnalisée (API), AI Bot | [Bot](../channels/wecom/wecom_bot/README.fr.md) / [App](../channels/wecom/wecom_app/README.fr.md) / [AI Bot](../channels/wecom/wecom_aibot/README.fr.md) |
| **Feishu (飞书)**    | ⭐⭐⭐ Avancé      | Collaboration entreprise, fonctionnalités riches      | [Documentation](../channels/feishu/README.fr.md)                                                                |
| **IRC**              | ⭐⭐ Moyen         | Serveur + configuration TLS                           | -                                                                                                                |
| **OneBot**           | ⭐⭐ Moyen         | Compatible NapCat/Go-CQHTTP, écosystème communautaire | [Documentation](../channels/onebot/README.fr.md)                                                                |
| **MaixCam**          | ⭐ Facile          | Canal d'intégration matérielle pour caméras AI Sipeed | [Documentation](../channels/maixcam/README.fr.md)                                                               |
| **Pico**             | ⭐ Facile          | Canal protocole natif PicoClaw                        |                                                                                                                  |

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

**Configuration rapide (recommandée)**

QQ Open Platform propose une page de configuration en un clic pour les bots compatibles OpenClaw :

1. Ouvrez [QQ Bot Quick Start](https://q.qq.com/qqbot/openclaw/index.html) et scannez le QR code pour vous connecter
2. Un bot est créé automatiquement — copiez l'**App ID** et l'**App Secret**
3. Configurez PicoClaw :

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

4. Lancez `picoclaw gateway` et ouvrez QQ pour discuter avec votre bot

> L'App Secret n'est affiché qu'une seule fois. Enregistrez-le immédiatement — le consulter à nouveau forcera une réinitialisation.
>
> Les bots créés via la page de configuration rapide sont initialement réservés au créateur et ne prennent pas en charge les discussions de groupe. Pour activer l'accès en groupe, configurez le mode sandbox sur la [QQ Open Platform](https://q.qq.com/).

**Configuration manuelle**

Si vous préférez créer le bot manuellement :

* Connectez-vous sur [QQ Open Platform](https://q.qq.com/) pour vous inscrire en tant que développeur
* Créez un bot QQ — personnalisez son avatar et son nom
* Copiez l'**App ID** et l'**App Secret** depuis les paramètres du bot
* Configurez comme indiqué ci-dessus et lancez `picoclaw gateway`

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

Pour toutes les options (`device_id`, `join_on_invite`, `group_trigger`, `placeholder`, `reasoning_channel_id`), voir le [Guide de Configuration du Canal Matrix](../channels/matrix/README.md).

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

Voir le [Guide de Configuration WeCom AI Bot](../channels/wecom/wecom_aibot/README.fr.md) pour les instructions détaillées.

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
* Dans les paramètres du AI Bot, configurez l'URL de callback : `http://your-server:18790/webhook/wecom-aibot`
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
      "welcome_message": "Hello! How can I help you?",
      "processing_message": "⏳ Processing, please wait. The results will be sent shortly."
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

PicoClaw se connecte à Feishu via le mode WebSocket/SDK — aucune URL webhook publique ni serveur de callback nécessaire.

**1. Créer une application**

* Allez sur [Feishu Open Platform](https://open.feishu.cn/) et créez une application
* Dans les paramètres de l'application, activez la capacité **Bot**
* Créez une version et publiez l'application (l'application doit être publiée pour prendre effet)
* Copiez l'**App ID** (commence par `cli_`) et l'**App Secret**

**2. Configurer**

```json
{
  "channels": {
    "feishu": {
      "enabled": true,
      "app_id": "cli_xxx",
      "app_secret": "YOUR_APP_SECRET",
      "allow_from": []
    }
  }
}
```

Optionnel : `encrypt_key` et `verification_token` pour le chiffrement des événements (recommandé en production).

**3. Lancer et discuter**

```bash
picoclaw gateway
```

Ouvrez Feishu, recherchez le nom de votre bot et commencez à discuter. Vous pouvez aussi ajouter le bot à un groupe — utilisez `group_trigger.mention_only: true` pour ne répondre que lorsqu'il est @mentionné.

Pour toutes les options, voir le [Guide de Configuration du Canal Feishu](../channels/feishu/README.fr.md).

</details>

<details>
<summary><b>Slack</b></summary>

**1. Créer une application Slack**

* Allez sur [Slack API](https://api.slack.com/apps) et créez une nouvelle application
* Sous **OAuth & Permissions**, ajoutez les scopes bot : `chat:write`, `app_mentions:read`, `im:history`, `im:read`, `im:write`
* Installez l'application dans votre workspace
* Copiez le **Bot Token** (`xoxb-...`) et l'**App-Level Token** (`xapp-...`, activez Socket Mode pour l'obtenir)

**2. Configurer**

```json
{
  "channels": {
    "slack": {
      "enabled": true,
      "bot_token": "xoxb-YOUR-BOT-TOKEN",
      "app_token": "xapp-YOUR-APP-TOKEN",
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

**1. Configurer**

```json
{
  "channels": {
    "irc": {
      "enabled": true,
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

Optionnel : `nickserv_password` pour l'authentification NickServ, `sasl_user`/`sasl_password` pour l'authentification SASL.

**2. Lancer**

```bash
picoclaw gateway
```

Le bot se connectera au serveur IRC et rejoindra les canaux spécifiés.

</details>

<details>
<summary><b>OneBot (QQ via protocole OneBot)</b></summary>

OneBot est un protocole ouvert pour les bots QQ. PicoClaw se connecte à toute implémentation compatible OneBot v11 (par ex. [Lagrange](https://github.com/LagrangeDev/Lagrange.Core), [NapCat](https://github.com/NapNeko/NapCatQQ)) via WebSocket.

**1. Configurer une implémentation OneBot**

Installez et exécutez un framework de bot QQ compatible OneBot v11. Activez son serveur WebSocket.

**2. Configurer**

```json
{
  "channels": {
    "onebot": {
      "enabled": true,
      "ws_url": "ws://127.0.0.1:8080",
      "access_token": "",
      "allow_from": []
    }
  }
}
```

| Champ | Description |
|-------|-------------|
| `ws_url` | URL WebSocket de l'implémentation OneBot |
| `access_token` | Token d'accès pour l'authentification (si configuré dans OneBot) |
| `reconnect_interval` | Intervalle de reconnexion en secondes (par défaut : 5) |

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
