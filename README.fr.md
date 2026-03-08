<div align="center">
  <img src="assets/logo.jpg" alt="PicoClaw" width="512">

  <h1>PicoClaw : Assistant IA Ultra-Efficace en Go</h1>

  <h3>Matériel à 10$ · 10 Mo de RAM · Démarrage en 1s · 皮皮虾，我们走！</h3>

  <p>
    <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/Arch-x86__64%2C%20ARM64%2C%20MIPS%2C%20RISC--V-blue" alt="Hardware">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
    <br>
    <a href="https://picoclaw.io"><img src="https://img.shields.io/badge/Website-picoclaw.io-blue?style=flat&logo=google-chrome&logoColor=white" alt="Website"></a>
    <a href="https://x.com/SipeedIO"><img src="https://img.shields.io/badge/X_(Twitter)-SipeedIO-black?style=flat&logo=x&logoColor=white" alt="Twitter"></a>
  </p>

 [中文](README.zh.md) | [日本語](README.ja.md) | [Português](README.pt-br.md) | [Tiếng Việt](README.vi.md) | [English](README.md) | **Français**
</div>

---

🦐 **PicoClaw** est un assistant personnel IA ultra-léger inspiré de [nanobot](https://github.com/HKUDS/nanobot), entièrement réécrit en **Go** via un processus d'auto-amorçage (self-bootstrapping) — où l'agent IA lui-même a piloté l'intégralité de la migration architecturale et de l'optimisation du code.

⚡️ **Extrêmement léger :** Fonctionne sur du matériel à seulement **10$** avec **<10 Mo** de RAM. C'est 99% de mémoire en moins qu'OpenClaw et 98% moins cher qu'un Mac mini !

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
> **🚨 SÉCURITÉ & CANAUX OFFICIELS**
>
> * **PAS DE CRYPTO :** PicoClaw n'a **AUCUN** token/jeton officiel. Toute annonce sur `pump.fun` ou d'autres plateformes de trading est une **ARNAQUE**.
> * **DOMAINE OFFICIEL :** Le **SEUL** site officiel est **[picoclaw.io](https://picoclaw.io)**, et le site de l'entreprise est **[sipeed.com](https://sipeed.com)**.
> * **Attention :** De nombreux domaines `.ai/.org/.com/.net/...` sont enregistrés par des tiers et ne nous appartiennent pas.
> * **Attention :** PicoClaw est en phase de développement précoce et peut présenter des problèmes de sécurité réseau non résolus. Ne déployez pas en environnement de production avant la version v1.0.
> * **Note :** PicoClaw a récemment fusionné de nombreuses PR, ce qui peut entraîner une empreinte mémoire plus importante (10–20 Mo) dans les dernières versions. Nous prévoyons de prioriser l'optimisation des ressources dès que l'ensemble des fonctionnalités sera stabilisé.


## 📢 Actualités

2026-02-16 🎉 PicoClaw a atteint 12K étoiles en une semaine ! Merci à tous pour votre soutien ! PicoClaw grandit plus vite que nous ne l'avions jamais imaginé. Vu le volume élevé de PR, nous avons un besoin urgent de mainteneurs communautaires. Nos rôles de bénévoles et notre feuille de route sont officiellement publiés [ici](docs/ROADMAP.md) — nous avons hâte de vous accueillir !

2026-02-13 🎉 PicoClaw a atteint 5000 étoiles en 4 jours ! Merci à la communauté ! Nous finalisons la **Feuille de Route du Projet** et mettons en place le **Groupe de Développeurs** pour accélérer le développement de PicoClaw.
🚀 **Appel à l'action :** Soumettez vos demandes de fonctionnalités dans les GitHub Discussions. Nous les examinerons et les prioriserons lors de notre prochaine réunion hebdomadaire.

2026-02-09 🎉 PicoClaw est lancé ! Construit en 1 jour pour apporter les Agents IA au matériel à 10$ avec <10 Mo de RAM. 🦐 PicoClaw, c'est parti !

## ✨ Fonctionnalités

🪶 **Ultra-Léger** : Empreinte mémoire <10 Mo — 99% plus petit que Clawdbot pour les fonctionnalités essentielles.

💰 **Coût Minimal** : Suffisamment efficace pour fonctionner sur du matériel à 10$ — 98% moins cher qu'un Mac mini.

⚡️ **Démarrage Éclair** : Temps de démarrage 400X plus rapide, boot en 1 seconde même sur un cœur unique à 0,6 GHz.

🌍 **Véritable Portabilité** : Un seul binaire autonome pour RISC-V, ARM, MIPS et x86. Un clic et c'est parti !

🤖 **Auto-Construit par l'IA** : Implémentation native en Go de manière autonome — 95% du cœur généré par l'Agent avec affinement humain dans la boucle.

|                               | OpenClaw      | NanoBot                  | **PicoClaw**                              |
| ----------------------------- | ------------- | ------------------------ | ----------------------------------------- |
| **Langage**                   | TypeScript    | Python                   | **Go**                                    |
| **RAM**                       | >1 Go         | >100 Mo                  | **< 10 Mo**                               |
| **Démarrage**</br>(cœur 0,8 GHz) | >500s     | >30s                     | **<1s**                                   |
| **Coût**                      | Mac Mini 599$ | La plupart des SBC Linux </br>~50$ | **N'importe quelle carte Linux**</br>**À partir de 10$** |

<img src="assets/compare.jpg" alt="PicoClaw" width="512">

## 🦾 Démonstration

### 🛠️ Flux de Travail Standard de l'Assistant

<table align="center">
  <tr align="center">
    <th><p align="center">🧩 Ingénieur Full-Stack</p></th>
    <th><p align="center">🗂️ Gestion des Logs & Planification</p></th>
    <th><p align="center">🔎 Recherche Web & Apprentissage</p></th>
  </tr>
  <tr>
    <td align="center"><p align="center"><img src="assets/picoclaw_code.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_memory.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_search.gif" width="240" height="180"></p></td>
  </tr>
  <tr>
    <td align="center">Développer • Déployer • Mettre à l'échelle</td>
    <td align="center">Planifier • Automatiser • Mémoriser</td>
    <td align="center">Découvrir • Analyser • Tendances</td>
  </tr>
</table>

### 📱 Utiliser sur d'anciens téléphones Android

Donnez une seconde vie à votre téléphone d'il y a dix ans ! Transformez-le en assistant IA intelligent avec PicoClaw. Démarrage rapide :

1. **Installez Termux** (disponible sur F-Droid ou Google Play).
2. **Exécutez les commandes**

```bash
# Note : Remplacez v0.1.1 par la dernière version depuis la page des Releases
wget https://github.com/sipeed/picoclaw/releases/download/v0.1.1/picoclaw-linux-arm64
chmod +x picoclaw-linux-arm64
pkg install proot
termux-chroot ./picoclaw-linux-arm64 onboard
```

Puis suivez les instructions de la section « Démarrage Rapide » pour terminer la configuration !

<img src="assets/termux.jpg" alt="PicoClaw" width="512">

### 🐜 Déploiement Innovant à Faible Empreinte

PicoClaw peut être déployé sur pratiquement n'importe quel appareil Linux !

- 9,9$ [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) version E (Ethernet) ou W (WiFi6), pour un Assistant Domotique Minimaliste
- 30~50$ [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html), ou 100$ [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html) pour la Maintenance Automatisée de Serveurs
- 50$ [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) ou 100$ [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera) pour la Surveillance Intelligente

<https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4>

🌟 Encore plus de scénarios de déploiement vous attendent !

## 📦 Installation

### Installer avec un binaire précompilé

Téléchargez le binaire pour votre plateforme depuis la page des [releases](https://github.com/sipeed/picoclaw/releases).

### Installer depuis les sources (dernières fonctionnalités, recommandé pour le développement)

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# Compiler, pas besoin d'installer
make build

# Compiler pour plusieurs plateformes
make build-all

# Compiler et Installer
make install
```

## 🐳 Docker Compose

Vous pouvez également exécuter PicoClaw avec Docker Compose sans rien installer localement.

```bash
# 1. Clonez ce dépôt
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. Premier lancement — génère docker/data/config.json puis s'arrête
docker compose -f docker/docker-compose.yml --profile gateway up
# Le conteneur affiche "First-run setup complete." puis s'arrête.

# 3. Configurez vos clés API
vim docker/data/config.json   # Clés API du fournisseur, tokens de bot, etc.

# 4. Démarrer
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

> [!TIP]
> **Utilisateurs Docker** : Par défaut, le Gateway écoute sur `127.0.0.1`, ce qui n'est pas accessible depuis l'hôte. Si vous avez besoin d'accéder aux endpoints de santé ou d'exposer des ports, définissez `PICOCLAW_GATEWAY_HOST=0.0.0.0` dans votre environnement ou mettez à jour `config.json`.

```bash
# 5. Voir les logs
docker compose -f docker/docker-compose.yml logs -f picoclaw-gateway

# 6. Arrêter
docker compose -f docker/docker-compose.yml --profile gateway down
```

### Mode Agent (exécution unique)

```bash
# Poser une question
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent -m "Combien font 2+2 ?"

# Mode interactif
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent
```

### Mettre à jour

```bash
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

### 🚀 Démarrage Rapide

> [!TIP]
> Configurez votre clé API dans `~/.picoclaw/config.json`.
> Obtenir des clés API : [OpenRouter](https://openrouter.ai/keys) (LLM) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) (LLM)
> La recherche web est **optionnelle** — obtenez gratuitement l'[API Brave Search](https://brave.com/search/api) (2000 requêtes gratuites/mois) ou utilisez le repli automatique intégré.

**1. Initialiser**

```bash
picoclaw onboard
```

**2. Configurer** (`~/.picoclaw/config.json`)

```json
{
  "model_list": [
    {
      "model_name": "gpt4",
      "model": "openai/gpt-5.2",
      "api_key": "sk-your-openai-key",
      "request_timeout": 300,
      "api_base": "https://api.openai.com/v1"
    }
  ],
  "agents": {
    "defaults": {
      "model_name": "gpt4"
    }
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "VOTRE_TOKEN_BOT",
      "allow_from": ["VOTRE_USER_ID"]
    }
  },
  "tools": {
    "web": {
      "brave": {
        "enabled": false,
        "api_key": "VOTRE_CLE_API_BRAVE",
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

> **Nouveau** : Le format de configuration `model_list` permet d'ajouter des fournisseurs sans modifier le code. Voir [Configuration de Modèle](#configuration-de-modèle-model_list) pour plus de détails.
> `request_timeout` est optionnel et s'exprime en secondes. S'il est omis ou défini à `<= 0`, PicoClaw utilise le délai d'expiration par défaut (120s).

**3. Obtenir des Clés API**

* **Fournisseur LLM** : [OpenRouter](https://openrouter.ai/keys) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) · [Anthropic](https://console.anthropic.com) · [OpenAI](https://platform.openai.com) · [Gemini](https://aistudio.google.com/api-keys)
* **Recherche Web** (optionnel) : [Brave Search](https://brave.com/search/api) - Offre gratuite disponible (2000 requêtes/mois)

> **Note** : Consultez `config.example.json` pour un modèle de configuration complet.

**4. Discuter**

```bash
picoclaw agent -m "Combien font 2+2 ?"
```

Et voilà ! Vous avez un assistant IA fonctionnel en 2 minutes.

---

## 💬 Applications de Chat

Discutez avec votre PicoClaw via Telegram, Discord, DingTalk, LINE ou WeCom

| Canal        | Configuration                          |
| ------------ | -------------------------------------- |
| **Telegram** | Facile (juste un token)                |
| **Discord**  | Facile (token bot + intents)           |
| **QQ**       | Facile (AppID + AppSecret)             |
| **DingTalk** | Moyen (identifiants de l'application)  |
| **LINE**     | Moyen (identifiants + URL de webhook)  |
| **WeCom AI Bot** | Moyen (Token + clé AES)            |

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
      "token": "VOTRE_TOKEN_BOT",
      "allow_from": ["VOTRE_USER_ID"]
    }
  }
}
```

> Obtenez votre User ID via `@userinfobot` sur Telegram.

**3. Lancer**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>Discord</b></summary>

**1. Créer un bot**

* Rendez-vous sur <https://discord.com/developers/applications>
* Créez une application → Bot → Add Bot
* Copiez le token du bot

**2. Activer les intents**

* Dans les paramètres du Bot, activez **MESSAGE CONTENT INTENT**
* (Optionnel) Activez **SERVER MEMBERS INTENT** si vous souhaitez utiliser des listes d'autorisation basées sur les données des membres

**3. Obtenir votre User ID**

* Paramètres Discord → Avancé → activez le **Mode Développeur**
* Clic droit sur votre avatar → **Copier l'identifiant**

**4. Configurer**

```json
{
  "channels": {
    "discord": {
      "enabled": true,
      "token": "VOTRE_TOKEN_BOT",
      "allow_from": ["VOTRE_USER_ID"]
    }
  }
}
```

**5. Inviter le bot**

* OAuth2 → URL Generator
* Scopes : `bot`
* Permissions du Bot : `Send Messages`, `Read Message History`
* Ouvrez l'URL d'invitation générée et ajoutez le bot à votre serveur

**6. Lancer**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>QQ</b></summary>

**1. Créer un bot**

- Rendez-vous sur la [QQ Open Platform](https://q.qq.com/#)
- Créez une application → Obtenez l'**AppID** et l'**AppSecret**

**2. Configurer**

```json
{
  "channels": {
    "qq": {
      "enabled": true,
      "app_id": "VOTRE_APP_ID",
      "app_secret": "VOTRE_APP_SECRET",
      "allow_from": []
    }
  }
}
```

> Laissez `allow_from` vide pour autoriser tous les utilisateurs, ou spécifiez des numéros QQ pour restreindre l'accès.

**3. Lancer**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>DingTalk</b></summary>

**1. Créer un bot**

* Rendez-vous sur la [Open Platform](https://open.dingtalk.com/)
* Créez une application interne
* Copiez le Client ID et le Client Secret

**2. Configurer**

```json
{
  "channels": {
    "dingtalk": {
      "enabled": true,
      "client_id": "VOTRE_CLIENT_ID",
      "client_secret": "VOTRE_CLIENT_SECRET",
      "allow_from": []
    }
  }
}
```

> Laissez `allow_from` vide pour autoriser tous les utilisateurs, ou spécifiez des identifiants pour restreindre l'accès.

**3. Lancer**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>LINE</b></summary>

**1. Créer un Compte Officiel LINE**

- Rendez-vous sur la [LINE Developers Console](https://developers.line.biz/)
- Créez un provider → Créez un canal Messaging API
- Copiez le **Channel Secret** et le **Channel Access Token**

**2. Configurer**

```json
{
  "channels": {
    "line": {
      "enabled": true,
      "channel_secret": "VOTRE_CHANNEL_SECRET",
      "channel_access_token": "VOTRE_CHANNEL_ACCESS_TOKEN",
      "webhook_path": "/webhook/line",
      "allow_from": []
    }
  }
}
```

**3. Configurer l'URL du Webhook**

LINE exige HTTPS pour les webhooks. Utilisez un reverse proxy ou un tunnel :

```bash
# Exemple avec ngrok (tunnel vers le serveur Gateway partagé)
ngrok http 18790
```

Puis configurez l'URL du Webhook dans la LINE Developers Console sur `https://votre-domaine/webhook/line` et activez **Use webhook**.

> **Note** : Le webhook LINE est servi par le serveur Gateway partagé (par défaut `127.0.0.1:18790`). Si vous utilisez ngrok ou un proxy inverse, faites pointer le tunnel vers le port `18790`.

**4. Lancer**

```bash
picoclaw gateway
```

> Dans les discussions de groupe, le bot répond uniquement lorsqu'il est mentionné avec @. Les réponses citent le message original.

> **Docker Compose** : Si vous avez besoin d'exposer le webhook LINE via Docker, mappez le port du Gateway partagé (par défaut `18790`) vers l'hôte, par exemple `ports: ["18790:18790"]`. Notez que le serveur Gateway sert les webhooks de tous les canaux à partir de ce port.

</details>

<details>
<summary><b>WeCom (WeChat Work)</b></summary>

PicoClaw prend en charge trois types d'intégration WeCom :

**Option 1 : WeCom Bot (Robot)** - Configuration plus facile, prend en charge les discussions de groupe
**Option 2 : WeCom App (Application Personnalisée)** - Plus de fonctionnalités, messagerie proactive, chat privé uniquement
**Option 3 : WeCom AI Bot (Bot Intelligent)** - Bot IA officiel, réponses en streaming, prend en charge groupe et privé

Voir le [Guide de Configuration WeCom AI Bot](docs/channels/wecom/wecom_aibot/README.zh.md) pour des instructions détaillées.

**Configuration Rapide - WeCom Bot :**

**1. Créer un bot**

* Accédez à la Console d'Administration WeCom → Discussion de Groupe → Ajouter un Bot de Groupe
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

**Configuration Rapide - WeCom App :**

**1. Créer une application**

* Accédez à la Console d'Administration WeCom → Gestion des Applications → Créer une Application
* Copiez l'**AgentId** et le **Secret**
* Accédez à la page "Mon Entreprise", copiez le **CorpID**

**2. Configurer la réception des messages**

* Dans les détails de l'application, cliquez sur "Recevoir les Messages" → "Configurer l'API"
* Définissez l'URL sur `http://your-server:18790/webhook/wecom-app`
* Générez le **Token** et l'**EncodingAESKey**

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

> **Note** : Les callbacks webhook WeCom App sont servis par le serveur Gateway partagé (par défaut `127.0.0.1:18790`). Assurez-vous que le port `18790` est accessible ou utilisez un proxy inverse HTTPS en production.

**Configuration Rapide - WeCom AI Bot :**

**1. Créer un AI Bot**

* Accédez à la Console d'Administration WeCom → Gestion des Applications → AI Bot
* Configurez l'URL de callback : `http://your-server:18791/webhook/wecom-aibot`
* Copiez le **Token** et générez l'**EncodingAESKey**

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
      "welcome_message": "Bonjour ! Comment puis-je vous aider ?"
    }
  }
}
```

**3. Lancer**

```bash
picoclaw gateway
```

> **Note** : WeCom AI Bot utilise le protocole pull en streaming — pas de problème de timeout. Les tâches longues (>5,5 min) basculent automatiquement vers la livraison via `response_url`.

</details>

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> Rejoignez le Réseau Social d'Agents

Connectez PicoClaw au Réseau Social d'Agents simplement en envoyant un seul message via le CLI ou n'importe quelle application de chat intégrée.

**Lisez `https://clawdchat.ai/skill.md` et suivez les instructions pour rejoindre [ClawdChat.ai](https://clawdchat.ai)**

## ⚙️ Configuration

Fichier de configuration : `~/.picoclaw/config.json`

### Variables d'Environnement

Vous pouvez remplacer les chemins par défaut à l'aide de variables d'environnement. Ceci est utile pour les installations portables, les déploiements conteneurisés ou l'exécution de picoclaw en tant que service système. Ces variables sont indépendantes et contrôlent différents chemins.

| Variable          | Description                                                                                                                             | Chemin par Défaut         |
|-------------------|-----------------------------------------------------------------------------------------------------------------------------------------|---------------------------|
| `PICOCLAW_CONFIG` | Remplace le chemin du fichier de configuration. Cela indique directement à picoclaw quel `config.json` charger, en ignorant tous les autres emplacements. | `~/.picoclaw/config.json` |
| `PICOCLAW_HOME`   | Remplace le répertoire racine des données picoclaw. Cela modifie l'emplacement par défaut du `workspace` et des autres répertoires de données.          | `~/.picoclaw`             |

**Exemples :**

```bash
# Exécuter picoclaw en utilisant un fichier de configuration spécifique
# Le chemin du workspace sera lu à partir de ce fichier de configuration
PICOCLAW_CONFIG=/etc/picoclaw/production.json picoclaw gateway

# Exécuter picoclaw avec toutes ses données stockées dans /opt/picoclaw
# La configuration sera chargée à partir du fichier par défaut ~/.picoclaw/config.json
# Le workspace sera créé dans /opt/picoclaw/workspace
PICOCLAW_HOME=/opt/picoclaw picoclaw agent

# Utiliser les deux pour une configuration entièrement personnalisée
PICOCLAW_HOME=/srv/picoclaw PICOCLAW_CONFIG=/srv/picoclaw/main.json picoclaw gateway
```

### Structure du Workspace

PicoClaw stocke les données dans votre workspace configuré (par défaut : `~/.picoclaw/workspace`) :

```
~/.picoclaw/workspace/
├── sessions/          # Sessions de conversation et historique
├── memory/           # Mémoire à long terme (MEMORY.md)
├── state/            # État persistant (dernier canal, etc.)
├── cron/             # Base de données des tâches planifiées
├── skills/           # Compétences personnalisées
├── AGENTS.md         # Guide de comportement de l'Agent
├── HEARTBEAT.md      # Invites de tâches périodiques (vérifiées toutes les 30 min)
├── IDENTITY.md       # Identité de l'Agent
├── SOUL.md           # Âme de l'Agent
├── TOOLS.md          # Description des outils
└── USER.md           # Préférences utilisateur
```

### 🔒 Bac à Sable de Sécurité

PicoClaw s'exécute dans un environnement sandboxé par défaut. L'agent ne peut accéder aux fichiers et exécuter des commandes qu'au sein du workspace configuré.

#### Configuration par Défaut

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

| Option | Par défaut | Description |
|--------|------------|-------------|
| `workspace` | `~/.picoclaw/workspace` | Répertoire de travail de l'agent |
| `restrict_to_workspace` | `true` | Restreindre l'accès fichiers/commandes au workspace |

#### Outils Protégés

Lorsque `restrict_to_workspace: true`, les outils suivants sont restreints au bac à sable :

| Outil | Fonction | Restriction |
|-------|----------|-------------|
| `read_file` | Lire des fichiers | Uniquement les fichiers dans le workspace |
| `write_file` | Écrire des fichiers | Uniquement les fichiers dans le workspace |
| `list_dir` | Lister des répertoires | Uniquement les répertoires dans le workspace |
| `edit_file` | Éditer des fichiers | Uniquement les fichiers dans le workspace |
| `append_file` | Ajouter à des fichiers | Uniquement les fichiers dans le workspace |
| `exec` | Exécuter des commandes | Les chemins doivent être dans le workspace |

#### Protection Supplémentaire d'Exec

Même avec `restrict_to_workspace: false`, l'outil `exec` bloque ces commandes dangereuses :

* `rm -rf`, `del /f`, `rmdir /s` — Suppression en masse
* `format`, `mkfs`, `diskpart` — Formatage de disque
* `dd if=` — Écriture d'image disque
* Écriture vers `/dev/sd[a-z]` — Écriture directe sur le disque
* `shutdown`, `reboot`, `poweroff` — Arrêt du système
* Fork bomb `:(){ :|:& };:`

#### Exemples d'Erreurs

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (path outside working dir)}
```

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (dangerous pattern detected)}
```

#### Désactiver les Restrictions (Risque de Sécurité)

Si vous avez besoin que l'agent accède à des chemins en dehors du workspace :

**Méthode 1 : Fichier de configuration**

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": false
    }
  }
}
```

**Méthode 2 : Variable d'environnement**

```bash
export PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE=false
```

> ⚠️ **Attention** : Désactiver cette restriction permet à l'agent d'accéder à n'importe quel chemin sur votre système. À utiliser avec précaution uniquement dans des environnements contrôlés.

#### Cohérence du Périmètre de Sécurité

Le paramètre `restrict_to_workspace` s'applique de manière cohérente sur tous les chemins d'exécution :

| Chemin d'Exécution | Périmètre de Sécurité |
|--------------------|----------------------|
| Agent Principal | `restrict_to_workspace` ✅ |
| Sous-agent / Spawn | Hérite de la même restriction ✅ |
| Tâches Heartbeat | Hérite de la même restriction ✅ |

Tous les chemins partagent la même restriction de workspace — il est impossible de contourner le périmètre de sécurité via des sous-agents ou des tâches planifiées.

### Heartbeat (Tâches Périodiques)

PicoClaw peut exécuter des tâches périodiques automatiquement. Créez un fichier `HEARTBEAT.md` dans votre workspace :

```markdown
# Tâches Périodiques

- Vérifier mes e-mails pour les messages importants
- Consulter mon agenda pour les événements à venir
- Vérifier les prévisions météo
```

L'agent lira ce fichier toutes les 30 minutes (configurable) et exécutera les tâches à l'aide des outils disponibles.

#### Tâches Asynchrones avec Spawn

Pour les tâches de longue durée (recherche web, appels API), utilisez l'outil `spawn` pour créer un **sous-agent** :

```markdown
# Tâches Périodiques

## Tâches Rapides (réponse directe)
- Indiquer l'heure actuelle

## Tâches Longues (utiliser spawn pour l'asynchrone)
- Rechercher les actualités IA sur le web et les résumer
- Vérifier les e-mails et signaler les messages importants
```

**Comportements clés :**

| Fonctionnalité | Description |
|----------------|-------------|
| **spawn** | Crée un sous-agent asynchrone, ne bloque pas le heartbeat |
| **Contexte indépendant** | Le sous-agent a son propre contexte, sans historique de session |
| **Outil message** | Le sous-agent communique directement avec l'utilisateur via l'outil message |
| **Non-bloquant** | Après le spawn, le heartbeat continue vers la tâche suivante |

#### Fonctionnement de la Communication du Sous-agent

```
Le Heartbeat se déclenche
    ↓
L'Agent lit HEARTBEAT.md
    ↓
Pour une tâche longue : spawn d'un sous-agent
    ↓                           ↓
Continue la tâche suivante   Le sous-agent travaille indépendamment
    ↓                           ↓
Toutes les tâches terminées  Le sous-agent utilise l'outil "message"
    ↓                           ↓
Répond HEARTBEAT_OK          L'utilisateur reçoit le résultat directement
```

Le sous-agent a accès aux outils (message, web_search, etc.) et peut communiquer avec l'utilisateur indépendamment sans passer par l'agent principal.

**Configuration :**

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

| Option | Par défaut | Description |
|--------|------------|-------------|
| `enabled` | `true` | Activer/désactiver le heartbeat |
| `interval` | `30` | Intervalle de vérification en minutes (min : 5) |

**Variables d'environnement :**

* `PICOCLAW_HEARTBEAT_ENABLED=false` pour désactiver
* `PICOCLAW_HEARTBEAT_INTERVAL=60` pour modifier l'intervalle

### Fournisseurs

> [!NOTE]
> Groq fournit la transcription vocale gratuite via Whisper. Si configuré, les messages audio de n'importe quel canal seront automatiquement transcrits au niveau de l'agent.

| Fournisseur              | Utilisation                              | Obtenir une Clé API                                    |
| ------------------------ | ---------------------------------------- | ------------------------------------------------------ |
| `gemini`                 | LLM (Gemini direct)                      | [aistudio.google.com](https://aistudio.google.com)     |
| `zhipu`                  | LLM (Zhipu direct)                       | [bigmodel.cn](bigmodel.cn)                             |
| `openrouter` (À tester)  | LLM (recommandé, accès à tous les modèles) | [openrouter.ai](https://openrouter.ai)               |
| `anthropic` (À tester)   | LLM (Claude direct)                      | [console.anthropic.com](https://console.anthropic.com) |
| `openai` (À tester)      | LLM (GPT direct)                         | [platform.openai.com](https://platform.openai.com)     |
| `deepseek` (À tester)    | LLM (DeepSeek direct)                    | [platform.deepseek.com](https://platform.deepseek.com) |
| `qwen`                   | LLM (Alibaba Qwen)                      | [dashscope.aliyuncs.com](https://dashscope.aliyuncs.com/compatible-mode/v1) |
| `cerebras`               | LLM (Cerebras)                           | [cerebras.ai](https://api.cerebras.ai/v1)              |
| `groq`                   | LLM + **Transcription vocale** (Whisper) | [console.groq.com](https://console.groq.com)           |

<details>
<summary><b>Configuration Zhipu</b></summary>

**1. Obtenir la clé API**

* Obtenez la [clé API](https://bigmodel.cn/usercenter/proj-mgmt/apikeys)

**2. Configurer**

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
      "api_key": "Votre Clé API",
      "api_base": "https://open.bigmodel.cn/api/paas/v4"
    }
  }
}
```

**3. Lancer**

```bash
picoclaw agent -m "Bonjour, comment ça va ?"
```

</details>

<details>
<summary><b>Exemple de configuration complète</b></summary>

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

### Configuration de Modèle (model_list)

> **Nouveau !** PicoClaw utilise désormais une approche de configuration **centrée sur le modèle**. Spécifiez simplement le format `fournisseur/modèle` (par exemple, `zhipu/glm-4.7`) pour ajouter de nouveaux fournisseurs—**aucune modification de code requise !**

Cette conception permet également le **support multi-agent** avec une sélection flexible de fournisseurs :

- **Différents agents, différents fournisseurs** : Chaque agent peut utiliser son propre fournisseur LLM
- **Modèles de secours (Fallbacks)** : Configurez des modèles primaires et de secours pour la résilience
- **Équilibrage de charge** : Répartissez les requêtes sur plusieurs points de terminaison
- **Configuration centralisée** : Gérez tous les fournisseurs en un seul endroit

#### 📋 Tous les Fournisseurs Supportés

| Fournisseur | Préfixe `model` | API Base par Défaut | Protocole | Clé API |
|-------------|-----------------|---------------------|----------|---------|
| **OpenAI** | `openai/` | `https://api.openai.com/v1` | OpenAI | [Obtenir Clé](https://platform.openai.com) |
| **Anthropic** | `anthropic/` | `https://api.anthropic.com/v1` | Anthropic | [Obtenir Clé](https://console.anthropic.com) |
| **Zhipu AI (GLM)** | `zhipu/` | `https://open.bigmodel.cn/api/paas/v4` | OpenAI | [Obtenir Clé](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) |
| **DeepSeek** | `deepseek/` | `https://api.deepseek.com/v1` | OpenAI | [Obtenir Clé](https://platform.deepseek.com) |
| **Google Gemini** | `gemini/` | `https://generativelanguage.googleapis.com/v1beta` | OpenAI | [Obtenir Clé](https://aistudio.google.com/api-keys) |
| **Groq** | `groq/` | `https://api.groq.com/openai/v1` | OpenAI | [Obtenir Clé](https://console.groq.com) |
| **Moonshot** | `moonshot/` | `https://api.moonshot.cn/v1` | OpenAI | [Obtenir Clé](https://platform.moonshot.cn) |
| **Qwen (Alibaba)** | `qwen/` | `https://dashscope.aliyuncs.com/compatible-mode/v1` | OpenAI | [Obtenir Clé](https://dashscope.console.aliyun.com) |
| **NVIDIA** | `nvidia/` | `https://integrate.api.nvidia.com/v1` | OpenAI | [Obtenir Clé](https://build.nvidia.com) |
| **Ollama** | `ollama/` | `http://localhost:11434/v1` | OpenAI | Local (pas de clé nécessaire) |
| **OpenRouter** | `openrouter/` | `https://openrouter.ai/api/v1` | OpenAI | [Obtenir Clé](https://openrouter.ai/keys) |
| **VLLM** | `vllm/` | `http://localhost:8000/v1` | OpenAI | Local |
| **Cerebras** | `cerebras/` | `https://api.cerebras.ai/v1` | OpenAI | [Obtenir Clé](https://cerebras.ai) |
| **Volcengine** | `volcengine/` | `https://ark.cn-beijing.volces.com/api/v3` | OpenAI | [Obtenir Clé](https://console.volcengine.com) |
| **ShengsuanYun** | `shengsuanyun/` | `https://router.shengsuanyun.com/api/v1` | OpenAI | - |
| **Antigravity** | `antigravity/` | Google Cloud | Custom | OAuth uniquement |
| **GitHub Copilot** | `github-copilot/` | `localhost:4321` | gRPC | - |

#### Configuration de Base

```json
{
  "model_list": [
    {
      "model_name": "gpt-5.2",
      "model": "openai/gpt-5.2",
      "api_key": "sk-your-openai-key"
    },
    {
      "model_name": "claude-sonnet-4.6",
      "model": "anthropic/claude-sonnet-4.6",
      "api_key": "sk-ant-your-key"
    },
    {
      "model_name": "glm-4.7",
      "model": "zhipu/glm-4.7",
      "api_key": "your-zhipu-key"
    }
  ],
  "agents": {
    "defaults": {
      "model": "gpt-5.2"
    }
  }
}
```

#### Exemples par Fournisseur

**OpenAI**
```json
{
  "model_name": "gpt-5.2",
  "model": "openai/gpt-5.2",
  "api_key": "sk-..."
}
```

**Zhipu AI (GLM)**
```json
{
  "model_name": "glm-4.7",
  "model": "zhipu/glm-4.7",
  "api_key": "your-key"
}
```

**Anthropic (avec OAuth)**
```json
{
  "model_name": "claude-sonnet-4.6",
  "model": "anthropic/claude-sonnet-4.6",
  "auth_method": "oauth"
}
```
> Exécutez `picoclaw auth login --provider anthropic` pour configurer les identifiants OAuth.

**Proxy/API personnalisée**
```json
{
  "model_name": "my-custom-model",
  "model": "openai/custom-model",
  "api_base": "https://my-proxy.com/v1",
  "api_key": "sk-...",
  "request_timeout": 300
}
```

#### Équilibrage de Charge

Configurez plusieurs points de terminaison pour le même nom de modèle—PicoClaw utilisera automatiquement le round-robin entre eux :

```json
{
  "model_list": [
    {
      "model_name": "gpt-5.2",
      "model": "openai/gpt-5.2",
      "api_base": "https://api1.example.com/v1",
      "api_key": "sk-key1"
    },
    {
      "model_name": "gpt-5.2",
      "model": "openai/gpt-5.2",
      "api_base": "https://api2.example.com/v1",
      "api_key": "sk-key2"
    }
  ]
}
```

#### Migration depuis l'Ancienne Configuration `providers`

L'ancienne configuration `providers` est **dépréciée** mais toujours supportée pour la rétrocompatibilité.

**Ancienne Configuration (dépréciée) :**
```json
{
  "providers": {
    "zhipu": {
      "api_key": "your-key",
      "api_base": "https://open.bigmodel.cn/api/paas/v4"
    }
  },
  "agents": {
    "defaults": {
      "provider": "zhipu",
      "model": "glm-4.7"
    }
  }
}
```

**Nouvelle Configuration (recommandée) :**
```json
{
  "model_list": [
    {
      "model_name": "glm-4.7",
      "model": "zhipu/glm-4.7",
      "api_key": "your-key"
    }
  ],
  "agents": {
    "defaults": {
      "model": "glm-4.7"
    }
  }
}
```

Pour le guide de migration détaillé, voir [docs/migration/model-list-migration.md](docs/migration/model-list-migration.md).

## Référence CLI

| Commande                  | Description                           |
| ------------------------- | ------------------------------------- |
| `picoclaw onboard`        | Initialiser la configuration & le workspace |
| `picoclaw agent -m "..."` | Discuter avec l'agent                 |
| `picoclaw agent`          | Mode de discussion interactif         |
| `picoclaw gateway`        | Démarrer la passerelle                |
| `picoclaw status`         | Afficher le statut                    |
| `picoclaw cron list`      | Lister toutes les tâches planifiées   |
| `picoclaw cron add ...`   | Ajouter une tâche planifiée           |

### Tâches Planifiées / Rappels

PicoClaw prend en charge les rappels planifiés et les tâches récurrentes via l'outil `cron` :

* **Rappels ponctuels** : « Rappelle-moi dans 10 minutes » → se déclenche une fois après 10 min
* **Tâches récurrentes** : « Rappelle-moi toutes les 2 heures » → se déclenche toutes les 2 heures
* **Expressions Cron** : « Rappelle-moi à 9h tous les jours » → utilise une expression cron

Les tâches sont stockées dans `~/.picoclaw/workspace/cron/` et traitées automatiquement.

## 🤝 Contribuer & Feuille de Route

Les PR sont les bienvenues ! Le code source est volontairement petit et lisible. 🤗

Feuille de route à venir...

Groupe de développeurs en construction. Condition d'entrée : au moins 1 PR fusionnée.

Groupes d'utilisateurs :

Discord : <https://discord.gg/V4sAZ9XWpN>

<img src="assets/wechat.png" alt="PicoClaw" width="512">

## 🐛 Dépannage

### La recherche web affiche « API 配置问题 »

C'est normal si vous n'avez pas encore configuré de clé API de recherche. PicoClaw fournira des liens utiles pour la recherche manuelle.

Pour activer la recherche web :

1. **Option 1 (Recommandé)** : Obtenez une clé API gratuite sur [https://brave.com/search/api](https://brave.com/search/api) (2000 requêtes gratuites/mois) pour les meilleurs résultats.
2. **Option 2 (Sans carte bancaire)** : Si vous n'avez pas de clé, le système bascule automatiquement sur **DuckDuckGo** (aucune clé requise).

Ajoutez la clé dans `~/.picoclaw/config.json` si vous utilisez Brave :

```json
{
  "tools": {
    "web": {
      "brave": {
        "enabled": false,
        "api_key": "VOTRE_CLE_API_BRAVE",
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

### Erreurs de filtrage de contenu

Certains fournisseurs (comme Zhipu) disposent d'un filtrage de contenu. Essayez de reformuler votre requête ou utilisez un modèle différent.

### Le bot Telegram affiche « Conflict: terminated by other getUpdates »

Cela se produit lorsqu'une autre instance du bot est en cours d'exécution. Assurez-vous qu'un seul `picoclaw gateway` fonctionne à la fois.

---

## 📝 Comparaison des Clés API

| Service          | Offre Gratuite       | Cas d'Utilisation                     |
| ---------------- | -------------------- | ------------------------------------- |
| **OpenRouter**   | 200K tokens/mois     | Multiples modèles (Claude, GPT-4, etc.) |
| **Zhipu**        | 200K tokens/mois     | Idéal pour les utilisateurs chinois   |
| **Brave Search** | 2000 requêtes/mois   | Fonctionnalité de recherche web       |
| **Groq**         | Offre gratuite dispo | Inférence ultra-rapide (Llama, Mixtral) |
