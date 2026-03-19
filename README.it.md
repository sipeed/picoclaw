<div align="center">
  <img src="assets/logo.webp" alt="PicoClaw" width="512">

  <h1>PicoClaw: Assistente IA Ultra-Efficiente in Go</h1>

  <h3>Hardware da $10 · <10MB RAM · Boot in <1s · 皮皮虾，我们走！</h3>
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

[中文](README.zh.md) | [日本語](README.ja.md) | [Português](README.pt-br.md) | [Tiếng Việt](README.vi.md) | [Français](README.fr.md) | **Italiano** | [Bahasa Indonesia](README.id.md) | [English](README.md)

</div>

---

> **PicoClaw** è un progetto open-source indipendente avviato da [Sipeed](https://sipeed.com). È scritto interamente in **Go** — non è un fork di OpenClaw, NanoBot o di qualsiasi altro progetto.

🦐 PicoClaw è un assistente IA personale ultra-leggero ispirato a [NanoBot](https://github.com/HKUDS/nanobot), riscritto da zero in Go attraverso un processo di auto-bootstrapping, in cui l'agente IA stesso ha guidato l'intera migrazione architetturale e l'ottimizzazione del codice.

⚡️ Funziona su hardware da $10 con meno di 10MB di RAM: il 99% di memoria in meno rispetto a OpenClaw e il 98% più economico di un Mac mini!

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
> **🚨 SICUREZZA & CANALI UFFICIALI**
>
> * **NESSUNA CRYPTO:** PicoClaw non ha **NESSUN** token/coin ufficiale. Qualsiasi annuncio su `pump.fun` o altre piattaforme di trading è una **TRUFFA**.
>
> * **DOMINIO UFFICIALE:** L'**UNICO** sito ufficiale è **[picoclaw.io](https://picoclaw.io)**, e il sito aziendale è **[sipeed.com](https://sipeed.com)**.
> * **Attenzione:** Molti domini `.ai/.org/.com/.net/...` sono registrati da terze parti.
> * **Attenzione:** PicoClaw è in fase di sviluppo iniziale e potrebbe avere problemi di sicurezza di rete non risolti. Non distribuire in ambienti di produzione prima della release v1.0.
> * **Nota:** PicoClaw ha recentemente unito molte PR, il che potrebbe comportare un'impronta di memoria maggiore (10–20MB) nelle ultime versioni. Prevediamo di dare priorità all'ottimizzazione delle risorse non appena il set di funzionalità corrente raggiungerà uno stato stabile.

## 📢 Novità

2026-03-17 🚀 **v0.2.3 rilasciata!** Interfaccia system tray (Windows & Linux), tracciamento dello stato dei sub-agent (`spawn_status`), hot-reload sperimentale del gateway, gate di sicurezza per cron e 2 correzioni di sicurezza. PicoClaw raggiunge **25K ⭐**!

2026-03-09 🎉 **v0.2.1 — Il più grande aggiornamento di sempre!** Supporto al protocollo MCP, 4 nuovi canali (Matrix/IRC/WeCom/Discord Proxy), 3 nuovi provider (Kimi/Minimax/Avian), pipeline di visione, store di memoria JSONL e routing dei modelli.

2026-02-28 📦 **v0.2.0** rilasciata con supporto Docker Compose e launcher Web UI.

2026-02-26 🎉 PicoClaw ha raggiunto **20K stelle** in soli 17 giorni! Arrivate l'orchestrazione automatica dei canali e le interfacce di capacità.

<details>
<summary>Notizie precedenti...</summary>

2026-02-16 🎉 PicoClaw ha raggiunto 12K stelle in una settimana! Ruoli di maintainer della community e [roadmap](ROADMAP.md) pubblicati ufficialmente.

2026-02-13 🎉 PicoClaw ha raggiunto 5000 stelle in 4 giorni! Roadmap del progetto e gruppo sviluppatori in fase di avvio.

2026-02-09 🎉 **PicoClaw lanciato!** Costruito in 1 giorno per portare gli agenti IA su hardware da $10 con <10MB di RAM. 🦐 PicoClaw, andiamo!

</details>

## ✨ Caratteristiche

🪶 **Ultra-Leggero**: Impronta di memoria <10MB — il 99% più piccolo delle funzionalità principali di OpenClaw.*

💰 **Costo Minimo**: Abbastanza efficiente da girare su hardware da $10 — il 98% più economico di un Mac mini.

⚡️ **Avvio Fulmineo**: Tempo di avvio 400 volte più veloce, boot in meno di 1 secondo anche su un singolo core a 0,6 GHz.

🌍 **Vera Portabilità**: Singolo binario autonomo per RISC-V, ARM, MIPS e x86. Un click e si parte!

🤖 **Auto-Costruito dall'IA**: Implementazione nativa in Go in modo autonomo — 95% del core generato dall'Agent con perfezionamento umano nel ciclo.

🔌 **Supporto MCP**: Integrazione nativa del [Model Context Protocol](https://modelcontextprotocol.io/) — connetti qualsiasi server MCP per estendere le capacità dell'agent.

👁️ **Pipeline di Visione**: Invia immagini e file direttamente all'agent — codifica base64 automatica per LLM multimodali.

🧠 **Routing Intelligente**: Routing dei modelli basato su regole — le query semplici vanno verso modelli leggeri, risparmiando sui costi API.

_*Le versioni recenti potrebbero usare 10–20MB a causa delle fusioni rapide di funzionalità. L'ottimizzazione delle risorse è pianificata. Il confronto dell'avvio è basato su benchmark con singolo core a 0,8 GHz (vedi tabella sotto)._

|                               | OpenClaw      | NanoBot                  | **PicoClaw**                              |
| ----------------------------- | ------------- | ------------------------ | ----------------------------------------- |
| **Linguaggio**                | TypeScript    | Python                   | **Go**                                    |
| **RAM**                       | >1GB          | >100MB                   | **< 10MB***                               |
| **Avvio**</br>(core 0,8 GHz)  | >500s         | >30s                     | **<1s**                                   |
| **Costo**                     | Mac Mini $599 | La maggior parte degli SBC Linux </br>~$50 | **Qualsiasi scheda Linux**</br>**A partire da $10** |

<img src="assets/compare.jpg" alt="PicoClaw" width="512">

## 🦾 Dimostrazione

### 🛠️ Flussi di Lavoro Standard dell'Assistente

<table align="center">
  <tr align="center">
    <th><p align="center">🧩 Ingegnere Full-Stack</p></th>
    <th><p align="center">🗂️ Gestione Log & Pianificazione</p></th>
    <th><p align="center">🔎 Ricerca Web & Apprendimento</p></th>
  </tr>
  <tr>
    <td align="center"><p align="center"><img src="assets/picoclaw_code.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_memory.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_search.gif" width="240" height="180"></p></td>
  </tr>
  <tr>
    <td align="center">Sviluppa • Distribuisci • Scala</td>
    <td align="center">Pianifica • Automatizza • Memorizza</td>
    <td align="center">Scopri • Analizza • Tendenze</td>
  </tr>
</table>

### 📱 Usa su vecchi telefoni Android

Dai una seconda vita al tuo telefono di dieci anni fa! Trasformalo in un assistente IA intelligente con PicoClaw. Avvio rapido:

1. **Installa [Termux](https://github.com/termux/termux-app)** (Scarica da [GitHub Releases](https://github.com/termux/termux-app/releases), o cerca su F-Droid / Google Play).
2. **Esegui i comandi**

```bash
# Scarica l'ultima release da https://github.com/sipeed/picoclaw/releases
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_Linux_arm64.tar.gz
tar xzf picoclaw_Linux_arm64.tar.gz
pkg install proot
termux-chroot ./picoclaw onboard
```

Poi segui le istruzioni nella sezione "Avvio Rapido" per completare la configurazione!

<img src="assets/termux.jpg" alt="PicoClaw" width="512">

### 🐜 Deploy Innovativo a Bassa Impronta

PicoClaw può essere distribuito su quasi qualsiasi dispositivo Linux!

- $9,9 [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) versione E (Ethernet) o W (WiFi6), per un Assistente Domotico Minimale
- $30~50 [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html), o $100 [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html) per la Manutenzione Automatizzata dei Server
- $50 [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) o $100 [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera) per il Monitoraggio Intelligente

<https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4>

🌟 Molti altri scenari di deploy ti aspettano!

## 📦 Installazione

### Installa con binario precompilato

Scarica il binario per la tua piattaforma dalla pagina delle [Releases](https://github.com/sipeed/picoclaw/releases).

### Installa dai sorgenti (ultime funzionalità, consigliato per lo sviluppo)

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# Compila, senza installare
make build

# Compila per più piattaforme
make build-all

# Compila per Raspberry Pi Zero 2 W (32-bit: make build-linux-arm; 64-bit: make build-linux-arm64)
make build-pi-zero

# Compila e Installa
make install
```

**Raspberry Pi Zero 2 W:** Usa il binario che corrisponde al tuo OS: Raspberry Pi OS 32-bit → `make build-linux-arm`; 64-bit → `make build-linux-arm64`. Oppure esegui `make build-pi-zero` per compilare entrambi.

## 📚 Documentazione

Per guide dettagliate, consulta la documentazione qui sotto. Il README copre solo l'avvio rapido.

| Argomento | Descrizione |
|-----------|-------------|
| 🐳 [Docker & Avvio Rapido](docs/docker.md) | Configurazione Docker Compose, modalità Launcher/Agent, configurazione rapida |
| 💬 [App di Chat](docs/chat-apps.md) | Telegram, Discord, WhatsApp, Matrix, QQ, Slack, IRC, DingTalk, LINE, Feishu, WeCom e altro |
| ⚙️ [Configurazione](docs/it/configuration.md) | Variabili d'ambiente, struttura del workspace, sorgenti delle skill, sandbox di sicurezza, heartbeat |
| 🔌 [Provider & Modelli](docs/providers.md) | 20+ provider LLM, routing dei modelli, configurazione model_list, architettura dei provider |
| 🔄 [Spawn & Task Asincroni](docs/spawn-tasks.md) | Task veloci, task lunghi con spawn, orchestrazione asincrona di sub-agent |
| 🐛 [Risoluzione Problemi](docs/troubleshooting.md) | Problemi comuni e soluzioni |
| 🔧 [Configurazione degli Strumenti](docs/tools_configuration.md) | Abilitazione/disabilitazione per strumento, politiche exec |

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> Unisciti al Social Network degli Agent

Connetti PicoClaw al Social Network degli Agent semplicemente inviando un singolo messaggio tramite CLI o qualsiasi app di chat integrata.

**Leggi `https://clawdchat.ai/skill.md` e segui le istruzioni per unirti a [ClawdChat.ai](https://clawdchat.ai)**

## 🖥️ Riferimento CLI

| Comando                   | Descrizione                        |
| ------------------------- | ---------------------------------- |
| `picoclaw onboard`        | Inizializza config & workspace     |
| `picoclaw agent -m "..."` | Chatta con l'agent                 |
| `picoclaw agent`          | Modalità chat interattiva          |
| `picoclaw gateway`        | Avvia il gateway                   |
| `picoclaw status`         | Mostra lo stato                    |
| `picoclaw version`        | Mostra le info sulla versione      |
| `picoclaw cron list`      | Elenca tutti i job pianificati     |
| `picoclaw cron add ...`   | Aggiunge un job pianificato        |
| `picoclaw cron disable`   | Disabilita un job pianificato      |
| `picoclaw cron remove`    | Rimuove un job pianificato         |
| `picoclaw skills list`    | Elenca le skill installate         |
| `picoclaw skills install` | Installa una skill                 |
| `picoclaw migrate`        | Migra i dati dalle versioni precedenti |
| `picoclaw auth login`     | Autenticazione con i provider      |

### Task Pianificati / Promemoria

PicoClaw supporta promemoria pianificati e task ricorrenti tramite lo strumento `cron`:

* **Promemoria una tantum**: "Ricordami tra 10 minuti" → si attiva una volta dopo 10 min
* **Task ricorrenti**: "Ricordami ogni 2 ore" → si attiva ogni 2 ore
* **Espressioni cron**: "Ricordami alle 9 ogni giorno" → usa un'espressione cron

## 🤝 Contribuisci & Roadmap

Le PR sono benvenute! Il codice è volutamente piccolo e leggibile. 🤗

Consulta la nostra [Roadmap della Community](https://github.com/sipeed/picoclaw/blob/main/ROADMAP.md) completa.

Gruppo sviluppatori in costruzione, unisciti dopo la tua prima PR accettata!

Gruppi utenti:

discord: <https://discord.gg/V4sAZ9XWpN>

<img src="assets/wechat.png" alt="PicoClaw" width="512">
