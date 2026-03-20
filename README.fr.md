<div align="center">
  <img src="assets/logo.webp" alt="PicoClaw" width="512">

  <h1>PicoClaw : Assistant IA Ultra-Efficace en Go</h1>

  <h3>Matériel à $10 · <10 Mo de RAM · Démarrage en <1s · 皮皮虾，我们走！</h3>
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
    <a href="./assets/wechat.png"><img src="https://img.shields.io/badge/WeChat-Group-41d56b?style=flat&logo=wechat&logoColor=white"></a>
    <a href="https://discord.gg/V4sAZ9XWpN"><img src="https://img.shields.io/badge/Discord-Community-4c60eb?style=flat&logo=discord&logoColor=white" alt="Discord"></a>
  </p>

[中文](README.zh.md) | [日本語](README.ja.md) | [Português](README.pt-br.md) | [Tiếng Việt](README.vi.md) | **Français** | [Italiano](README.it.md) | [Bahasa Indonesia](README.id.md) | [English](README.md)

</div>

---

> **PicoClaw** est un projet open-source indépendant initié par [Sipeed](https://sipeed.com). Il est entièrement écrit en **Go** — ce n'est pas un fork d'OpenClaw, de NanoBot ou de tout autre projet.

🦐 **PicoClaw** est un assistant personnel IA ultra-léger inspiré de [NanoBot](https://github.com/HKUDS/nanobot), entièrement réécrit en **Go** via un processus d'auto-amorçage (self-bootstrapping) — où l'agent IA lui-même a piloté l'intégralité de la migration architecturale et de l'optimisation du code.

⚡️ **Extrêmement léger :** Fonctionne sur du matériel à seulement **$10** avec **<10 Mo** de RAM. C'est 99% de mémoire en moins qu'OpenClaw et 98% moins cher qu'un Mac mini !

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
>
> * **DOMAINE OFFICIEL :** Le **SEUL** site officiel est **[picoclaw.io](https://picoclaw.io)**, et le site de l'entreprise est **[sipeed.com](https://sipeed.com)**.
> * **Attention :** De nombreux domaines `.ai/.org/.com/.net/...` sont enregistrés par des tiers.
> * **Attention :** PicoClaw est en phase de développement précoce et peut présenter des problèmes de sécurité réseau non résolus. Ne déployez pas en environnement de production avant la version v1.0.
> * **Note :** PicoClaw a récemment fusionné de nombreuses PR, ce qui peut entraîner une empreinte mémoire plus importante (10–20 Mo) dans les dernières versions. Nous prévoyons de prioriser l'optimisation des ressources dès que l'ensemble des fonctionnalités sera stabilisé.

## 📢 Actualités

2026-03-17 🚀 **v0.2.3 publié !** Interface système tray (Windows & Linux), suivi de statut des sous-agents (`spawn_status`), rechargement à chaud expérimental du gateway, portes de sécurité cron, et 2 correctifs de sécurité. PicoClaw atteint **25K ⭐** !

2026-03-09 🎉 **v0.2.1 — Plus grande mise à jour !** Support du protocole MCP, 4 nouveaux canaux (Matrix/IRC/WeCom/Discord Proxy), 3 nouveaux fournisseurs (Kimi/Minimax/Avian), pipeline de vision, stockage mémoire JSONL, et routage de modèles.

2026-02-28 📦 **v0.2.0** publié avec support Docker Compose et lanceur Web UI.

2026-02-26 🎉 PicoClaw a atteint **20K étoiles** en seulement 17 jours ! L'orchestration automatique des canaux et les interfaces de capacités sont arrivées.

<details>
<summary>Actualités précédentes...</summary>

2026-02-16 🎉 PicoClaw a atteint 12K étoiles en une semaine ! Les rôles de mainteneurs communautaires et la [feuille de route](ROADMAP.md) sont officiellement publiés.

2026-02-13 🎉 PicoClaw a atteint 5000 étoiles en 4 jours ! La Feuille de Route du Projet et le Groupe de Développeurs sont en cours de mise en place.

2026-02-09 🎉 **PicoClaw est lancé !** Construit en 1 jour pour apporter les Agents IA au matériel à $10 avec <10 Mo de RAM. 🦐 PicoClaw, c'est parti !

</details>

## ✨ Fonctionnalités

🪶 **Ultra-Léger** : Empreinte mémoire <10 Mo — 99% plus petit que les fonctionnalités essentielles d'OpenClaw.*

💰 **Coût Minimal** : Suffisamment efficace pour fonctionner sur du matériel à $10 — 98% moins cher qu'un Mac mini.

⚡️ **Démarrage Éclair** : Temps de démarrage 400X plus rapide, boot en <1 seconde même sur un cœur unique à 0,6 GHz.

🌍 **Véritable Portabilité** : Un seul binaire autonome pour RISC-V, ARM, MIPS et x86. Un clic et c'est parti !

🤖 **Auto-Construit par l'IA** : Implémentation native en Go de manière autonome — 95% du cœur généré par l'Agent avec affinement humain dans la boucle.

🔌 **Support MCP** : Intégration native du [Model Context Protocol](https://modelcontextprotocol.io/) — connectez n'importe quel serveur MCP pour étendre les capacités de l'agent.

👁️ **Pipeline de Vision** : Envoyez des images et fichiers directement à l'agent — encodage base64 automatique pour les LLM multimodaux.

🧠 **Routage Intelligent** : Routage de modèles basé sur des règles — les requêtes simples vont vers des modèles légers, économisant les coûts API.

_*Les versions récentes peuvent utiliser 10–20 Mo en raison des fusions rapides de fonctionnalités. L'optimisation des ressources est prévue. La comparaison de démarrage est basée sur des benchmarks à cœur unique 0,8 GHz (voir tableau ci-dessous)._

|                               | OpenClaw      | NanoBot                  | **PicoClaw**                              |
| ----------------------------- | ------------- | ------------------------ | ----------------------------------------- |
| **Langage**                   | TypeScript    | Python                   | **Go**                                    |
| **RAM**                       | >1 Go         | >100 Mo                  | **< 10 Mo***                              |
| **Démarrage**</br>(cœur 0,8 GHz) | >500s     | >30s                     | **<1s**                                   |
| **Coût**                      | Mac Mini $599 | La plupart des SBC Linux </br>~$50 | **N'importe quelle carte Linux**</br>**À partir de $10** |

<img src="assets/compare.jpg" alt="PicoClaw" width="512">

> 📋 **[Liste de Compatibilité Matérielle](docs/hardware-compatibility.md)** — Voir toutes les cartes testées, du RISC-V à $5 au Raspberry Pi en passant par les téléphones Android. Votre carte n'est pas listée ? Soumettez une PR !

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

1. **Installez [Termux](https://github.com/termux/termux-app)** (Téléchargez depuis [GitHub Releases](https://github.com/termux/termux-app/releases), ou recherchez sur F-Droid / Google Play).
2. **Exécutez les commandes**

```bash
# Téléchargez la dernière version depuis https://github.com/sipeed/picoclaw/releases
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_Linux_arm64.tar.gz
tar xzf picoclaw_Linux_arm64.tar.gz
pkg install proot
termux-chroot ./picoclaw onboard   # chroot fournit une disposition standard du système de fichiers Linux
```

Puis suivez les instructions de la section « Démarrage Rapide » pour terminer la configuration !

<img src="assets/termux.jpg" alt="PicoClaw" width="512">

### 🐜 Déploiement Innovant à Faible Empreinte

PicoClaw peut être déployé sur pratiquement n'importe quel appareil Linux !

- 9,9$ [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) version E (Ethernet) ou W (WiFi6), pour un Assistant Domotique Minimaliste
- 30~$50 [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html), ou 100$ [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html) pour la Maintenance Automatisée de Serveurs
- 50$ [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) ou 100$ [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera) pour la Surveillance Intelligente

<https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4>

🌟 Encore plus de scénarios de déploiement vous attendent !

## 📦 Installation

### Télécharger depuis picoclaw.io (Recommandé)

Visitez **[picoclaw.io](https://picoclaw.io)** — le site officiel détecte automatiquement votre plateforme et propose un téléchargement en un clic. Pas besoin de choisir manuellement une architecture.

### Télécharger le binaire précompilé

Vous pouvez aussi télécharger le binaire pour votre plateforme depuis la page [GitHub Releases](https://github.com/sipeed/picoclaw/releases).

### Compiler depuis les sources (pour le développement)

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# Compiler, pas besoin d'installer
make build

# Compiler pour plusieurs plateformes
make build-all

# Compiler pour Raspberry Pi Zero 2 W (32-bit : make build-linux-arm ; 64-bit : make build-linux-arm64)
make build-pi-zero

# Compiler et Installer
make install
```

**Raspberry Pi Zero 2 W :** Utilisez le binaire correspondant à votre OS : Raspberry Pi OS 32-bit → `make build-linux-arm` ; 64-bit → `make build-linux-arm64`. Ou exécutez `make build-pi-zero` pour compiler les deux.

## 📚 Documentation

Pour des guides détaillés, consultez la documentation ci-dessous. Ce README ne couvre que le démarrage rapide.

| Sujet | Description |
|-------|-------------|
| 🐳 [Docker & Démarrage Rapide](docs/fr/docker.md) | Configuration Docker Compose, modes Launcher/Agent, configuration rapide |
| 💬 [Applications de Chat](docs/fr/chat-apps.md) | Telegram, Discord, WhatsApp, Matrix, QQ, Slack, IRC, DingTalk, LINE, Feishu, WeCom, et plus |
| ⚙️ [Configuration](docs/fr/configuration.md) | Variables d'environnement, structure du workspace, sources de compétences, bac à sable de sécurité, heartbeat |
| 🔌 [Fournisseurs & Modèles](docs/fr/providers.md) | 20+ fournisseurs LLM, routage de modèles, configuration model_list, architecture des fournisseurs |
| 🔄 [Spawn & Tâches Asynchrones](docs/fr/spawn-tasks.md) | Tâches rapides, tâches longues avec spawn, orchestration asynchrone de sous-agents |
| 🐛 [Dépannage](docs/fr/troubleshooting.md) | Problèmes courants et solutions |
| 🔧 [Configuration des Outils](docs/fr/tools_configuration.md) | Activation/désactivation par outil, politiques exec |
| 📋 [Compatibilité Matérielle](docs/hardware-compatibility.md) | Cartes testées, exigences minimales, comment ajouter votre carte |

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> Rejoignez le Réseau Social d'Agents

Connectez PicoClaw au Réseau Social d'Agents simplement en envoyant un seul message via le CLI ou n'importe quelle application de chat intégrée.

**Lisez `https://clawdchat.ai/skill.md` et suivez les instructions pour rejoindre [ClawdChat.ai](https://clawdchat.ai)**

## 🖥️ Référence CLI

| Commande                  | Description                        |
| ------------------------- | ---------------------------------- |
| `picoclaw onboard`        | Initialiser la config & le workspace |
| `picoclaw agent -m "..."` | Discuter avec l'agent              |
| `picoclaw agent`          | Mode chat interactif               |
| `picoclaw gateway`        | Démarrer le gateway                |
| `picoclaw status`         | Afficher le statut                 |
| `picoclaw version`        | Afficher les infos de version      |
| `picoclaw cron list`      | Lister les tâches planifiées       |
| `picoclaw cron add ...`   | Ajouter une tâche planifiée        |
| `picoclaw cron disable`   | Désactiver une tâche planifiée     |
| `picoclaw cron remove`    | Supprimer une tâche planifiée      |
| `picoclaw skills list`    | Lister les compétences installées  |
| `picoclaw skills install` | Installer une compétence           |
| `picoclaw migrate`        | Migrer les données des anciennes versions |
| `picoclaw auth login`     | S'authentifier auprès des fournisseurs |
| `picoclaw model`          | Voir ou changer le modèle par défaut |

### Tâches Planifiées / Rappels

PicoClaw prend en charge les rappels planifiés et les tâches récurrentes via l'outil `cron` :

* **Rappels ponctuels** : « Rappelle-moi dans 10 minutes » → se déclenche une fois après 10 min
* **Tâches récurrentes** : « Rappelle-moi toutes les 2 heures » → se déclenche toutes les 2 heures
* **Expressions cron** : « Rappelle-moi à 9h chaque jour » → utilise une expression cron

## 🤝 Contribuer & Feuille de Route

Les PR sont les bienvenues ! Le code est intentionnellement petit et lisible. 🤗

Consultez notre [Feuille de Route Communautaire](https://github.com/sipeed/picoclaw/blob/main/ROADMAP.md) complète.

Groupe de développeurs en construction, rejoignez-nous après votre première PR fusionnée !

Groupes d'utilisateurs :

discord : <https://discord.gg/V4sAZ9XWpN>

<img src="assets/wechat.png" alt="PicoClaw" width="512">
