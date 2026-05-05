<div align="center">
  <img src="../../assets/logo.webp" alt="PicoClaw" width="512">

  <h1>PicoClaw : Assistant IA Ultra-Efficace en Go</h1>

  <h3>Matériel à $10 · 10 Mo de RAM · Démarrage en ms · Let's Go, PicoClaw!</h3>
  <p>
    <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/Arch-x86__64%2C%20ARM64%2C%20MIPS%2C%20RISC--V%2C%20LoongArch-blue" alt="Hardware">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
    <br>
    <a href="https://picoclaw.io"><img src="https://img.shields.io/badge/Website-picoclaw.io-blue?style=flat&logo=google-chrome&logoColor=white" alt="Website"></a>
    <a href="https://docs.picoclaw.io/"><img src="https://img.shields.io/badge/Docs-Official-007acc?style=flat&logo=read-the-docs&logoColor=white" alt="Docs"></a>
    <a href="https://deepwiki.com/sipeed/picoclaw"><img src="https://img.shields.io/badge/Wiki-DeepWiki-FFA500?style=flat&logo=wikipedia&logoColor=white" alt="Wiki"></a>
    <br>
    <a href="https://x.com/SipeedIO"><img src="https://img.shields.io/badge/X_(Twitter)-SipeedIO-black?style=flat&logo=x&logoColor=white" alt="Twitter"></a>
    <a href="../../assets/wechat.png"><img src="https://img.shields.io/badge/WeChat-Group-41d56b?style=flat&logo=wechat&logoColor=white"></a>
    <a href="https://discord.gg/V4sAZ9XWpN"><img src="https://img.shields.io/badge/Discord-Community-4c60eb?style=flat&logo=discord&logoColor=white" alt="Discord"></a>
  </p>

[中文](README.zh.md) | [日本語](README.ja.md) | [한국어](README.ko.md) | [Português](README.pt-br.md) | [Tiếng Việt](README.vi.md) | **Français** | [Italiano](README.it.md) | [Bahasa Indonesia](README.id.md) | [Malay](README.ms.md) | [English](../../README.md)

</div>

---

> **PicoClaw** est un projet open-source indépendant initié par [Sipeed](https://sipeed.com), entièrement écrit en **Go** à partir de zéro — ce n'est pas un fork d'OpenClaw, de NanoBot ou de tout autre projet.

**PicoClaw** est un assistant personnel IA ultra-léger inspiré de [NanoBot](https://github.com/HKUDS/nanobot). Il a été entièrement reconstruit en **Go** via un processus d'auto-amorçage (self-bootstrapping) — l'Agent IA lui-même a piloté la migration architecturale et l'optimisation du code.

**Fonctionne sur du matériel à $10 avec <10 Mo de RAM** — c'est 99% de mémoire en moins qu'OpenClaw et 98% moins cher qu'un Mac mini !


<table align="center">
<tr align="center">
<td align="center" valign="top">
<p align="center">
<img src="../../assets/picoclaw_mem.gif" width="360" height="240">
</p>
</td>
<td align="center" valign="top">
<p align="center">
<img src="../../assets/licheervnano.png" width="400" height="240">
</p>
</td>
</tr>
</table>

> [!CAUTION]
> **Avis de sécurité**
>
> * **PAS DE CRYPTO :** PicoClaw n'a **pas** émis de tokens officiels ni de cryptomonnaie. Toute affirmation sur `pump.fun` ou d'autres plateformes de trading est une **arnaque**.
> * **DOMAINE OFFICIEL :** Le **SEUL** site officiel est **[picoclaw.io](https://picoclaw.io)**, et le site de l'entreprise est **[sipeed.com](https://sipeed.com)**
> * **ATTENTION :** De nombreux domaines `.ai/.org/.com/.net/...` ont été enregistrés par des tiers. Ne leur faites pas confiance.
> * **NOTE :** PicoClaw est en développement rapide précoce. Des problèmes de sécurité non résolus peuvent exister. Ne pas déployer en production avant la v1.0.
> * **NOTE :** PicoClaw a récemment fusionné de nombreuses PRs. Les builds récents peuvent utiliser 10-20 Mo de RAM. L'optimisation des ressources est prévue après la stabilisation des fonctionnalités.

## 📢 Actualités

2026-03-31 📱 **Support Android !** PicoClaw fonctionne maintenant sur Android ! Téléchargez l'APK sur [picoclaw.io](https://picoclaw.io/download)

2026-03-25 🚀 **v0.2.4 publiée !** Refonte de l'architecture Agent (SubTurn, Hooks, Steering, EventBus), intégration WeChat/WeCom, renforcement de la sécurité (.security.yml, filtrage des données sensibles), nouveaux providers (AWS Bedrock, Azure, Xiaomi MiMo), et 35 corrections de bugs. PicoClaw a atteint **26K Stars** !

2026-03-17 🚀 **v0.2.3 publiée !** Interface system tray (Windows & Linux), requête de statut des sous-agents (`spawn_status`), rechargement à chaud expérimental du Gateway, sécurisation Cron, et 2 correctifs de sécurité. PicoClaw a atteint **25K Stars** !

2026-03-09 🎉 **v0.2.1 — Plus grande mise à jour à ce jour !** Support du protocole MCP, 4 nouveaux channels (Matrix/IRC/WeCom/Discord Proxy), 3 nouveaux providers (Kimi/Minimax/Avian), pipeline vision, stockage mémoire JSONL, routage de modèles.

2026-02-28 📦 **v0.2.0** publiée avec support Docker Compose et Web UI Launcher.

<details>
<summary>Actualités précédentes...</summary>

2026-02-26 🎉 PicoClaw atteint **20K Stars** en seulement 17 jours ! L'orchestration automatique des channels et les interfaces de capacités sont disponibles.

2026-02-16 🎉 PicoClaw dépasse 12K Stars en une semaine ! Rôles de mainteneurs communautaires et [Roadmap](../../ROADMAP.md) officiellement lancés.

2026-02-13 🎉 PicoClaw dépasse 5000 Stars en 4 jours ! Roadmap du projet et groupes de développeurs en cours.

2026-02-09 🎉 **PicoClaw publié !** Construit en 1 jour pour apporter les Agents IA sur du matériel à $10 avec <10 Mo de RAM. Let's Go, PicoClaw !

</details>


## ✨ Fonctionnalités

🪶 **Ultra-léger** : Empreinte mémoire du cœur <10 Mo — 99% plus petit qu'OpenClaw.*

💰 **Coût minimal** : Suffisamment efficace pour fonctionner sur du matériel à $10 — 98% moins cher qu'un Mac mini.

⚡️ **Démarrage ultra-rapide** : 400x plus rapide au démarrage. Démarre en <1s même sur un processeur monocœur à 0,6 GHz.

🌍 **Vraiment portable** : Binaire unique pour les architectures RISC-V, ARM, MIPS et x86. Un seul binaire, fonctionne partout !

🤖 **Auto-amorcé par IA** : Implémentation native pure Go — 95% du code principal a été généré par un Agent et affiné via une révision humaine en boucle.

🔌 **Support MCP** : Intégration native du [Model Context Protocol](https://modelcontextprotocol.io/) — connectez n'importe quel serveur MCP pour étendre les capacités de l'Agent.

👁️ **Pipeline vision** : Envoyez des images et des fichiers directement à l'Agent — encodage base64 automatique pour les LLMs multimodaux.

🧠 **Routage intelligent** : Routage de modèles basé sur des règles — les requêtes simples vont vers des modèles légers, économisant les coûts API.

_*Les builds récents peuvent utiliser 10-20 Mo en raison des fusions rapides de PRs. L'optimisation des ressources est prévue. Comparaison de vitesse de démarrage basée sur des benchmarks monocœur à 0,8 GHz (voir tableau ci-dessous)._

<div align="center">

|                                | OpenClaw      | NanoBot                  | **PicoClaw**                           |
| ------------------------------ | ------------- | ------------------------ | -------------------------------------- |
| **Langage**                    | TypeScript    | Python                   | **Go**                                 |
| **RAM**                        | >1 Go         | >100 Mo                  | **< 10 Mo***                           |
| **Temps de démarrage**</br>(cœur 0,8 GHz) | >500s | >30s              | **<1s**                                |
| **Coût**                       | Mac Mini $599 | La plupart des cartes Linux ~$50 | **N'importe quelle carte Linux**</br>**à partir de $10** |

<img src="../../assets/compare.jpg" alt="PicoClaw" width="512">

</div>

> **[Liste de compatibilité matérielle](../guides/hardware-compatibility.fr.md)** — Voir toutes les cartes testées, du RISC-V à $5 au Raspberry Pi en passant par les téléphones Android. Votre carte n'est pas listée ? Soumettez une PR !

<p align="center">
<img src="../../assets/hardware-banner.jpg" alt="PicoClaw Hardware Compatibility" width="100%">
</p>

## 🦾 Démonstration

### 🛠️ Flux de travail standard de l'assistant

<table align="center">
<tr align="center">
<th><p align="center">Mode Ingénieur Full-Stack</p></th>
<th><p align="center">Journalisation & Planification</p></th>
<th><p align="center">Recherche Web & Apprentissage</p></th>
</tr>
<tr>
<td align="center"><p align="center"><img src="../../assets/picoclaw_code.gif" width="240" height="180"></p></td>
<td align="center"><p align="center"><img src="../../assets/picoclaw_memory.gif" width="240" height="180"></p></td>
<td align="center"><p align="center"><img src="../../assets/picoclaw_search.gif" width="240" height="180"></p></td>
</tr>
<tr>
<td align="center">Développer · Déployer · Mettre à l'échelle</td>
<td align="center">Planifier · Automatiser · Mémoriser</td>
<td align="center">Découvrir · Analyser · Tendances</td>
</tr>
</table>

### 🐜 Déploiement innovant à faible empreinte

PicoClaw peut être déployé sur pratiquement n'importe quel appareil Linux !

- $9,9 [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) édition E(Ethernet) ou W(WiFi6), pour un assistant domestique minimal
- $30~50 [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html), ou $100 [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html), pour des opérations serveur automatisées
- $50 [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) ou $100 [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera), pour la surveillance intelligente

<https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4>

🌟 D'autres cas de déploiement vous attendent !


## 📦 Installation

### Télécharger depuis picoclaw.io (Recommandé)

Visitez **[picoclaw.io](https://picoclaw.io)** — le site officiel détecte automatiquement votre plateforme et fournit un téléchargement en un clic. Pas besoin de choisir manuellement une architecture.

### Télécharger le binaire précompilé

Vous pouvez aussi télécharger le binaire pour votre plateforme depuis la page [GitHub Releases](https://github.com/sipeed/picoclaw/releases).

### Compiler depuis les sources (pour le développement)

Prérequis :

- Go 1.25+
- Node.js 22+ et pnpm 10.33.0+ pour les builds Web UI / launcher

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# Installer les dépendances frontend
(cd web/frontend && pnpm install --frozen-lockfile)

# Compiler le binaire principal
make build

# Compiler le Web UI Launcher (requis pour le mode WebUI)
make build-launcher

# Compiler les binaires core pour toutes les plateformes gérées par le Makefile
make build-all

# Compiler pour Raspberry Pi Zero 2 W (32 bits : make build-linux-arm ; 64 bits : make build-linux-arm64)
make build-pi-zero

# Compiler et installer
make install
```

**Raspberry Pi Zero 2 W :** Utilisez le binaire correspondant à votre OS : Raspberry Pi OS 32 bits -> `make build-linux-arm` ; 64 bits -> `make build-linux-arm64`. Ou exécutez `make build-pi-zero` pour compiler les deux.

## 🚀 Guide de démarrage rapide

### 🌐 WebUI Launcher (Recommandé pour le bureau)

Le WebUI Launcher fournit une interface basée sur navigateur pour la configuration et le chat. C'est la façon la plus simple de démarrer — aucune connaissance de la ligne de commande requise.

**Option 1 : Double-clic (Bureau)**

Après téléchargement depuis [picoclaw.io](https://picoclaw.io), double-cliquez sur `picoclaw-launcher` (ou `picoclaw-launcher.exe` sous Windows). Votre navigateur s'ouvrira automatiquement sur `http://localhost:18800`.

**Option 2 : Ligne de commande**

```bash
picoclaw-launcher
# Ouvrez http://localhost:18800 dans votre navigateur
```

> [!TIP]
> **Accès distant / Docker / VM :** Ajoutez le flag `-public` pour écouter sur toutes les interfaces :
> ```bash
> picoclaw-launcher -public
> ```

<p align="center">
<img src="../../assets/launcher-webui.jpg" alt="WebUI Launcher" width="600">
</p>

**Pour commencer :**

Ouvrez le WebUI, puis : **1)** Configurez un Provider (ajoutez votre clé API LLM) -> **2)** Configurez un Channel (ex. Telegram) -> **3)** Démarrez le Gateway -> **4)** Chattez !

Pour la documentation détaillée du WebUI, voir [docs.picoclaw.io](https://docs.picoclaw.io).

<details>
<summary><b>Docker (alternative)</b></summary>

```bash
# 1. Cloner ce dépôt
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. Premier lancement — génère automatiquement docker/data/config.json puis s'arrête
#    (se déclenche uniquement quand config.json et workspace/ sont tous deux absents)
docker compose -f docker/docker-compose.yml --profile launcher up
# Le conteneur affiche "First-run setup complete." et s'arrête.

# 3. Définir vos clés API
vim docker/data/config.json

# 4. Démarrer
docker compose -f docker/docker-compose.yml --profile launcher up -d
# Ouvrez http://localhost:18800
```

> **Utilisateurs Docker / VM :** Le Gateway écoute sur `127.0.0.1` par défaut. Définissez `PICOCLAW_GATEWAY_HOST=0.0.0.0` ou utilisez le flag `-public` pour le rendre accessible depuis l'hôte.

```bash
# Vérifier les logs
docker compose -f docker/docker-compose.yml logs -f

# Arrêter
docker compose -f docker/docker-compose.yml --profile launcher down

# Mettre à jour
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml --profile launcher up -d
```

</details>

<details>
<summary><b>macOS — Avertissement de sécurité au premier lancement</b></summary>

macOS peut bloquer `picoclaw-launcher` au premier lancement car il est téléchargé depuis Internet et n'est pas notarisé via le Mac App Store.

**Étape 1 :** Double-cliquez sur `picoclaw-launcher`. Un avertissement de sécurité s'affiche :

<p align="center">
<img src="../../assets/macos-gatekeeper-warning.jpg" alt="Avertissement macOS Gatekeeper" width="400">
</p>

> *"picoclaw-launcher" n'a pas pu être ouvert — Apple n'a pas pu vérifier que "picoclaw-launcher" ne contient pas de logiciel malveillant susceptible de nuire à votre Mac ou de compromettre votre confidentialité.*

**Étape 2 :** Ouvrez **Réglages Système** → **Confidentialité et sécurité** → faites défiler jusqu'à la section **Sécurité** → cliquez sur **Ouvrir quand même** → confirmez en cliquant sur **Ouvrir quand même** dans la boîte de dialogue.

<p align="center">
<img src="../../assets/macos-gatekeeper-allow.jpg" alt="macOS Confidentialité et sécurité — Ouvrir quand même" width="600">
</p>

Après cette étape unique, `picoclaw-launcher` s'ouvrira normalement lors des lancements suivants.

</details>

<a id="-run-on-old-android-phones"></a>
### 📱 Android

Donnez une seconde vie à votre téléphone vieux de dix ans ! Transformez-le en assistant IA intelligent avec PicoClaw.

**Option 1 : Installation APK**

Aperçu :

<table>
  <tr>
    <td><img src="../../assets/fui_main_page.jpg" width="200"></td>
    <td><img src="../../assets/fui_web_page.jpg" width="200"></td>
    <td><img src="../../assets/fui_log_page.jpg" width="200"></td>
    <td><img src="../../assets/fui_setting_page.jpg" width="200"></td>
  </tr>
</table>

Téléchargez l'APK depuis [picoclaw.io](https://picoclaw.io/download/) et installez-le directement. Pas besoin de Termux !

**Option 2 : Termux**

<details>
<summary><b>Terminal Launcher (pour les environnements à ressources limitées)</b></summary>

1. Installez [Termux](https://github.com/termux/termux-app) (téléchargez depuis [GitHub Releases](https://github.com/termux/termux-app/releases), ou cherchez dans F-Droid / Google Play)
2. Exécutez les commandes suivantes :

```bash
# Télécharger la dernière version
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_Linux_arm64.tar.gz
tar xzf picoclaw_Linux_arm64.tar.gz
pkg install proot
termux-chroot ./picoclaw onboard   # chroot fournit une arborescence Linux standard
```

Suivez ensuite la section Terminal Launcher ci-dessous pour terminer la configuration.

<img src="../../assets/termux.jpg" alt="PicoClaw on Termux" width="512">

Pour les environnements minimaux où seul le binaire principal `picoclaw` est disponible (sans Launcher UI), vous pouvez tout configurer via la ligne de commande et un fichier de configuration JSON.

**1. Initialiser**

```bash
picoclaw onboard
```

Cela crée `~/.picoclaw/config.json` et le répertoire workspace.

**2. Configurer** (`~/.picoclaw/config.json`)

```json
{
  "agents": {
    "defaults": {
      "model_name": "gpt-5.4"
    }
  },
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_key": "sk-your-api-key"
    }
  ]
}
```

> Voir `config/config.example.json` dans le dépôt pour un modèle de configuration complet avec toutes les options disponibles.

**3. Chatter**

```bash
# Question ponctuelle
picoclaw agent -m "What is 2+2?"

# Mode interactif
picoclaw agent

# Démarrer le gateway pour l'intégration d'applications de chat
picoclaw gateway
```

</details>


## 🔌 Providers (LLM)

PicoClaw supporte plus de 30 providers LLM via la configuration `model_list`. Utilisez le format `protocole/modèle` :

| Provider | Protocole | Clé API | Notes |
|----------|-----------|---------|-------|
| [OpenAI](https://platform.openai.com/api-keys) | `openai/` | Requise | GPT-5.4, GPT-4o, o3, etc. |
| [Anthropic](https://console.anthropic.com/settings/keys) | `anthropic/` | Requise | Claude Opus 4.6, Sonnet 4.6, etc. |
| [Google Gemini](https://aistudio.google.com/apikey) | `gemini/` | Requise | Gemini 3 Flash, 2.5 Pro, etc. |
| [OpenRouter](https://openrouter.ai/keys) | `openrouter/` | Requise | 200+ modèles, API unifiée |
| [Zhipu (GLM)](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) | `zhipu/` | Requise | GLM-4.7, GLM-5, etc. |
| [DeepSeek](https://platform.deepseek.com/api_keys) | `deepseek/` | Requise | DeepSeek-V3, DeepSeek-R1 |
| [Volcengine](https://console.volcengine.com) | `volcengine/` | Requise | Modèles Doubao, Ark |
| [Qwen](https://dashscope.console.aliyun.com/apiKey) | `qwen/` | Requise | Qwen3, Qwen-Max, etc. |
| [Groq](https://console.groq.com/keys) | `groq/` | Requise | Inférence rapide (Llama, Mixtral) |
| [Moonshot (Kimi)](https://platform.moonshot.cn/console/api-keys) | `moonshot/` | Requise | Modèles Kimi |
| [Minimax](https://platform.minimaxi.com/user-center/basic-information/interface-key) | `minimax/` | Requise | Modèles MiniMax |
| [Mistral](https://console.mistral.ai/api-keys) | `mistral/` | Requise | Mistral Large, Codestral |
| [NVIDIA NIM](https://build.nvidia.com/) | `nvidia/` | Requise | Modèles hébergés NVIDIA |
| [Cerebras](https://cloud.cerebras.ai/) | `cerebras/` | Requise | Inférence rapide |
| [Novita AI](https://novita.ai/) | `novita/` | Requise | Divers modèles open |
| [Xiaomi MiMo](https://platform.xiaomimimo.com/) | `mimo/` | Requise | Modèles MiMo |
| [Ollama](https://ollama.com/) | `ollama/` | Non requise | Modèles locaux, auto-hébergé |
| [vLLM](https://docs.vllm.ai/) | `vllm/` | Non requise | Déploiement local, compatible OpenAI |
| [OVMS](https://docs.openvino.ai/2026/model-server/ovms_docs_llm_quickstart.html) | `ovms/` | Non requise | Intel OpenVINO Model Server, compatible OpenAI |
| [LiteLLM](https://docs.litellm.ai/) | `litellm/` | Variable | Proxy pour 100+ providers |
| [Azure OpenAI](https://portal.azure.com/) | `azure/` | Requise | Déploiement Azure entreprise |
| [GitHub Copilot](https://github.com/features/copilot) | `github-copilot/` | OAuth | Connexion par code appareil |
| [Antigravity](https://console.cloud.google.com/) | `antigravity/` | OAuth | Google Cloud AI |

<details>
<summary><b>Déploiement local (Ollama, vLLM, OVMS, etc.)</b></summary>

**Ollama :**
```json
{
  "model_list": [
    {
      "model_name": "local-llama",
      "model": "ollama/llama3.1:8b",
      "api_base": "http://localhost:11434/v1"
    }
  ]
}
```

**vLLM :**
```json
{
  "model_list": [
    {
      "model_name": "local-vllm",
      "model": "vllm/your-model",
      "api_base": "http://localhost:8000/v1"
    }
  ]
}
```

**OVMS (OpenVINO Model Server) :**
```json
{
  "model_list": [
    {
      "model_name": "local-ovms",
      "model": "ovms/your-model",
      "api_base": "http://localhost:8000/v3"
    }
  ]
}
```

Pour les détails complets de configuration des providers, voir [Providers & Models](../guides/providers.fr.md).

</details>

## 💬 Channels (Applications de chat)

Parlez à votre PicoClaw via plus de 17 plateformes de messagerie :

| Channel | Configuration | Protocole | Docs |
|---------|---------------|-----------|------|
| **Telegram** | Facile (token bot) | Long polling | [Guide](../channels/telegram/README.fr.md) |
| **Discord** | Facile (token bot + intents) | WebSocket | [Guide](../channels/discord/README.fr.md) |
| **WhatsApp** | Facile (scan QR ou URL bridge) | Natif / Bridge | [Guide](../guides/chat-apps.fr.md#whatsapp) |
| **Weixin** | Facile (scan QR natif) | iLink API | [Guide](../guides/chat-apps.fr.md#weixin) |
| **QQ** | Facile (AppID + AppSecret) | WebSocket | [Guide](../channels/qq/README.fr.md) |
| **Slack** | Facile (token bot + app) | Socket Mode | [Guide](../channels/slack/README.fr.md) |
| **Matrix** | Moyen (homeserver + token) | Sync API | [Guide](../channels/matrix/README.fr.md) |
| **DingTalk** | Moyen (identifiants client) | Stream | [Guide](../channels/dingtalk/README.fr.md) |
| **Feishu / Lark** | Moyen (App ID + Secret) | WebSocket/SDK | [Guide](../channels/feishu/README.fr.md) |
| **LINE** | Moyen (identifiants + webhook) | Webhook | [Guide](../channels/line/README.fr.md) |
| **WeCom** | Facile (QR login ou manuel) | WebSocket | [Guide](../channels/wecom/README.fr.md) |
| **IRC** | Moyen (serveur + pseudo) | Protocole IRC | [Guide](../guides/chat-apps.fr.md#irc) |
| **OneBot** | Moyen (URL WebSocket) | OneBot v11 | [Guide](../channels/onebot/README.fr.md) |
| **MaixCam** | Facile (activer) | Socket TCP | [Guide](../channels/maixcam/README.fr.md) |
| **Pico** | Facile (activer) | Protocole natif | Intégré |
| **Pico Client** | Facile (URL WebSocket) | WebSocket | Intégré |

> Tous les channels basés sur webhook partagent un seul serveur HTTP Gateway (`gateway.host`:`gateway.port`, par défaut `127.0.0.1:18790`). Feishu utilise le mode WebSocket/SDK et n'utilise pas le serveur HTTP partagé.

> La verbosité des logs est contrôlée par `gateway.log_level` (par défaut : `warn`). Valeurs supportées : `debug`, `info`, `warn`, `error`, `fatal`. Peut aussi être défini via `PICOCLAW_LOG_LEVEL`. Voir [Configuration](../guides/configuration.fr.md#niveau-de-log-du-gateway) pour plus de détails.

Pour les instructions détaillées de configuration des channels, voir [Configuration des applications de chat](../guides/chat-apps.fr.md).

## 🔧 Outils

### 🔍 Recherche Web

PicoClaw peut effectuer des recherches sur le web pour fournir des informations à jour. Configurez dans `tools.web` :

| Moteur de recherche | Clé API | Niveau gratuit | Lien |
|--------------------|---------|----------------|------|
| DuckDuckGo | Non requise | Illimité | Fallback intégré |
| [Baidu Search](https://cloud.baidu.com/doc/qianfan-api/s/Wmbq4z7e5) | Requise | 1000 requêtes/jour | IA, optimisé pour le chinois |
| [Tavily](https://tavily.com) | Requise | 1000 requêtes/mois | Optimisé pour les Agents IA |
| [Brave Search](https://brave.com/search/api) | Requise | 2000 requêtes/mois | Rapide et privé |
| [Perplexity](https://www.perplexity.ai) | Requise | Payant | Recherche propulsée par IA |
| [SearXNG](https://github.com/searxng/searxng) | Non requise | Auto-hébergé | Métamoteur de recherche gratuit |
| [GLM Search](https://open.bigmodel.cn/) | Requise | Variable | Recherche web Zhipu |

### ⚙️ Autres outils

PicoClaw inclut des outils intégrés pour les opérations sur fichiers, l'exécution de code, la planification et plus encore. Voir [Configuration des outils](../reference/tools_configuration.fr.md) pour les détails.

## 🎯 Skills

Les Skills sont des capacités modulaires qui étendent votre Agent. Elles sont chargées depuis les fichiers `SKILL.md` dans votre workspace.

**Installer des Skills depuis ClawHub :**

```bash
picoclaw skills search "web scraping"
picoclaw skills install <skill-name>
```

**Configurer le token ClawHub** (optionnel, pour des limites de débit plus élevées) :

Ajoutez à votre `config.json` :
```json
{
  "tools": {
    "skills": {
      "registries": {
        "clawhub": {
          "auth_token": "your-clawhub-token"
        }
      }
    }
  }
}
```

Pour plus de détails, voir [Configuration des outils - Skills](../reference/tools_configuration.fr.md#skills-tool).

## 🔗 MCP (Model Context Protocol)

PicoClaw supporte nativement [MCP](https://modelcontextprotocol.io/) — connectez n'importe quel serveur MCP pour étendre les capacités de votre Agent avec des outils et sources de données externes.

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "servers": {
        "filesystem": {
          "enabled": true,
          "command": "npx",
          "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
        }
      }
    }
  }
}
```

Pour la configuration MCP complète (transports stdio, SSE, HTTP, Tool Discovery), voir [Configuration des outils - MCP](../reference/tools_configuration.fr.md#mcp-tool).

## <img src="../../assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> Rejoignez le réseau social des Agents

Connectez PicoClaw au réseau social des Agents simplement en envoyant un seul message via le CLI ou n'importe quelle application de chat intégrée.

**Lisez `https://clawdchat.ai/skill.md` et suivez les instructions pour rejoindre [ClawdChat.ai](https://clawdchat.ai)**

## 🖥️ Référence CLI

| Commande                  | Description                              |
| ------------------------- | ---------------------------------------- |
| `picoclaw onboard`        | Initialiser la config & le workspace     |
| `picoclaw auth weixin` | Connecter un compte WeChat via QR |
| `picoclaw agent -m "..."` | Chatter avec l'agent                     |
| `picoclaw agent`          | Mode chat interactif                     |
| `picoclaw gateway`        | Démarrer le gateway                      |
| `picoclaw status`         | Afficher le statut                       |
| `picoclaw version`        | Afficher les informations de version     |
| `picoclaw model`          | Voir ou changer le modèle par défaut     |
| `picoclaw cron list`      | Lister toutes les tâches planifiées      |
| `picoclaw cron add ...`   | Ajouter une tâche planifiée              |
| `picoclaw cron disable`   | Désactiver une tâche planifiée           |
| `picoclaw cron remove`    | Supprimer une tâche planifiée            |
| `picoclaw skills list`    | Lister les Skills installées             |
| `picoclaw skills install` | Installer une Skill                      |
| `picoclaw migrate`        | Migrer les données depuis d'anciennes versions |
| `picoclaw auth login`     | S'authentifier auprès des providers      |

### ⏰ Tâches planifiées / Rappels

PicoClaw supporte les rappels planifiés et les tâches récurrentes via l'outil `cron` :

* **Rappels ponctuels** : "Rappelle-moi dans 10 minutes" -> se déclenche une fois après 10 min
* **Tâches récurrentes** : "Rappelle-moi toutes les 2 heures" -> se déclenche toutes les 2 heures
* **Expressions cron** : "Rappelle-moi à 9h chaque jour" -> utilise une expression cron

## 📚 Documentation

Pour des guides détaillés au-delà de ce README :

| Sujet | Description |
|-------|-------------|
| [Docker & Démarrage rapide](../guides/docker.fr.md) | Configuration Docker Compose, modes Launcher/Agent |
| [Applications de chat](../guides/chat-apps.fr.md) | Guides de configuration pour les 17+ channels |
| [Configuration](../guides/configuration.fr.md) | Variables d'environnement, structure du workspace, sandbox de sécurité |
| [Providers & Modèles](../guides/providers.fr.md) | 30+ providers LLM, routage de modèles, configuration model_list |
| [Spawn & Tâches asynchrones](../guides/spawn-tasks.fr.md) | Tâches rapides, tâches longues avec spawn, orchestration de sous-agents asynchrones |
| [Hooks](../architecture/hooks/README.md) | Système de hooks événementiels : observateurs, intercepteurs, hooks d'approbation |
| [Steering](../architecture/steering.md) | Injecter des messages dans une boucle agent en cours d'exécution |
| [SubTurn](../architecture/subturn.md) | Coordination de subagents, contrôle de concurrence, cycle de vie |
| [Dépannage](../operations/troubleshooting.fr.md) | Problèmes courants et solutions |
| [Configuration des outils](../reference/tools_configuration.fr.md) | Activation/désactivation par outil, politiques d'exécution, MCP, Skills |
| [Compatibilité matérielle](../guides/hardware-compatibility.fr.md) | Cartes testées, exigences minimales |

## 🤝 Contribuer & Roadmap

Les PRs sont les bienvenues ! Le code source est intentionnellement petit et lisible.

Consultez notre [Roadmap communautaire](https://github.com/sipeed/picoclaw/issues/988) et [CONTRIBUTING.md](../../CONTRIBUTING.md) pour les directives.

Groupe de développeurs en construction, rejoignez-le après votre première PR fusionnée !

Groupes d'utilisateurs :

Discord : <https://discord.gg/V4sAZ9XWpN>

WeChat :
<img src="../../assets/wechat.png" alt="WeChat group QR code" width="512">
