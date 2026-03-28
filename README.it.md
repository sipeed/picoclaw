<div align="center">
<img src="assets/logo.webp" alt="PicoClaw" width="512">

<h1>PicoClaw: Assistente IA Ultra-Efficiente in Go</h1>

<h3>Hardware da $10 · 10MB di RAM · Avvio in ms · Let's Go, PicoClaw!</h3>
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

[中文](README.zh.md) | [日本語](README.ja.md) | [Português](README.pt-br.md) | [Tiếng Việt](README.vi.md) | [Français](README.fr.md) | **Italiano** | [Bahasa Indonesia](README.id.md) | [Malay](README.my.md) | [English](README.md)

</div>

---

> **PicoClaw** è un progetto open-source indipendente avviato da [Sipeed](https://sipeed.com), scritto interamente in **Go** da zero — non è un fork di OpenClaw, NanoBot o di qualsiasi altro progetto.

**PicoClaw** è un assistente IA personale ultra-leggero ispirato a [NanoBot](https://github.com/HKUDS/nanobot). È stato riscritto da zero in **Go** attraverso un processo di "auto-bootstrapping" — l'Agent IA stesso ha guidato la migrazione architetturale e l'ottimizzazione del codice.

**Funziona su hardware da $10 con <10MB di RAM** — il 99% di memoria in meno rispetto a OpenClaw e il 98% più economico di un Mac mini!

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
> **Avviso di Sicurezza**
>
> * **NESSUNA CRYPTO:** PicoClaw **non** ha emesso token o criptovalute ufficiali. Qualsiasi annuncio su `pump.fun` o altre piattaforme di trading è una **truffa**.
> * **DOMINIO UFFICIALE:** L'**UNICO** sito ufficiale è **[picoclaw.io](https://picoclaw.io)**, e il sito aziendale è **[sipeed.com](https://sipeed.com)**
> * **ATTENZIONE:** Molti domini `.ai/.org/.com/.net/...` sono stati registrati da terze parti. Non fidarti di essi.
> * **NOTA:** PicoClaw è in fase di sviluppo iniziale rapido. Potrebbero esserci problemi di sicurezza non risolti. Non distribuire in produzione prima della v1.0.
> * **NOTA:** PicoClaw ha recentemente unito molte PR. Le build recenti potrebbero usare 10-20MB di RAM. L'ottimizzazione delle risorse è pianificata dopo la stabilizzazione delle funzionalità.

## 📢 Novità

2026-03-25 🚀 **v0.2.4 rilasciata!** Revisione dell'architettura Agent (SubTurn, Hooks, Steering, EventBus), integrazione WeChat/WeCom, rafforzamento della sicurezza (.security.yml, filtraggio dati sensibili), nuovi provider (AWS Bedrock, Azure, Xiaomi MiMo) e 35 correzioni di bug. PicoClaw raggiunge **26K Stars**!

2026-03-17 🚀 **v0.2.3 rilasciata!** Interfaccia system tray (Windows & Linux), query sullo stato dei sub-agent (`spawn_status`), hot-reload sperimentale del Gateway, gate di sicurezza per Cron e 2 correzioni di sicurezza. PicoClaw raggiunge **25K Stars**!

2026-03-09 🎉 **v0.2.1 — Il più grande aggiornamento di sempre!** Supporto al protocollo MCP, 4 nuovi canali (Matrix/IRC/WeCom/Discord Proxy), 3 nuovi provider (Kimi/Minimax/Avian), pipeline visiva, archivio memoria JSONL, routing dei modelli.

2026-02-28 📦 **v0.2.0** rilasciata con supporto Docker Compose e Web UI Launcher.

<details>
<summary>Notizie precedenti...</summary>

2026-02-26 🎉 PicoClaw raggiunge **20K stelle** in soli 17 giorni! Orchestrazione automatica dei canali e interfacce di capacità sono attive.

2026-02-16 🎉 PicoClaw supera 12K stelle in una settimana! Ruoli di maintainer della community e [Roadmap](ROADMAP.md) pubblicati ufficialmente.

2026-02-13 🎉 PicoClaw supera 5000 stelle in 4 giorni! Roadmap del progetto e gruppi sviluppatori in fase di avvio.

2026-02-09 🎉 **PicoClaw lanciato!** Costruito in 1 giorno per portare gli AI Agent su hardware da $10 con <10MB di RAM. Let's Go, PicoClaw!

</details>

## ✨ Caratteristiche

🪶 **Ultra-Leggero**: Impronta di memoria <10MB — il 99% più piccolo rispetto a OpenClaw.*

💰 **Costo Minimo**: Abbastanza efficiente da girare su hardware da $10 — il 98% più economico di un Mac mini.

⚡️ **Avvio Fulmineo**: Avvio 400 volte più veloce. Boot in meno di 1 secondo anche su un singolo core a 0,6 GHz.

🌍 **Vera Portabilità**: Singolo binario per RISC-V, ARM, MIPS e x86. Un binario, funziona ovunque!

🤖 **Auto-Costruito dall'IA**: Implementazione nativa in Go — il 95% del codice core è stato generato da un Agent e perfezionato tramite revisione umana nel ciclo.

🔌 **Supporto MCP**: Integrazione nativa del [Model Context Protocol](https://modelcontextprotocol.io/) — connetti qualsiasi server MCP per estendere le capacità dell'Agent.

👁️ **Pipeline di Visione**: Invia immagini e file direttamente all'Agent — codifica base64 automatica per LLM multimodali.

🧠 **Routing Intelligente**: Routing dei modelli basato su regole — le query semplici vanno verso modelli leggeri, risparmiando sui costi API.

_*Le build recenti potrebbero usare 10-20MB a causa delle fusioni rapide di PR. L'ottimizzazione delle risorse è pianificata. Il confronto dell'avvio è basato su benchmark con singolo core a 0,8 GHz (vedi tabella sotto)._

<div align="center">

|                                | OpenClaw      | NanoBot                  | **PicoClaw**                           |
| ------------------------------ | ------------- | ------------------------ | -------------------------------------- |
| **Linguaggio**                 | TypeScript    | Python                   | **Go**                                 |
| **RAM**                        | >1GB          | >100MB                   | **< 10MB***                            |
| **Avvio**</br>(core 0,8 GHz)  | >500s         | >30s                     | **<1s**                                |
| **Costo**                      | Mac Mini $599 | La maggior parte degli SBC Linux ~$50 | **Qualsiasi scheda Linux**</br>**a partire da $10** |

<img src="assets/compare.jpg" alt="PicoClaw" width="512">

</div>

> **[Lista di Compatibilità Hardware](docs/hardware-compatibility.md)** — Vedi tutte le schede testate, dai $5 RISC-V al Raspberry Pi ai telefoni Android. La tua scheda non è elencata? Invia una PR!

<p align="center">
<img src="assets/hardware-banner.jpg" alt="PicoClaw Hardware Compatibility" width="100%">
</p>

## 🦾 Dimostrazione

### 🛠️ Flussi di Lavoro Standard dell'Assistente

<table align="center">
<tr align="center">
<th><p align="center">Modalità Ingegnere Full-Stack</p></th>
<th><p align="center">Log & Pianificazione</p></th>
<th><p align="center">Ricerca Web & Apprendimento</p></th>
</tr>
<tr>
<td align="center"><p align="center"><img src="assets/picoclaw_code.gif" width="240" height="180"></p></td>
<td align="center"><p align="center"><img src="assets/picoclaw_memory.gif" width="240" height="180"></p></td>
<td align="center"><p align="center"><img src="assets/picoclaw_search.gif" width="240" height="180"></p></td>
</tr>
<tr>
<td align="center">Sviluppa · Distribuisci · Scala</td>
<td align="center">Pianifica · Automatizza · Memorizza</td>
<td align="center">Scopri · Analizza · Tendenze</td>
</tr>
</table>

### 🐜 Deploy Innovativo a Bassa Impronta

PicoClaw può essere distribuito su quasi qualsiasi dispositivo Linux!

- $9,9 [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) versione E (Ethernet) o W (WiFi6), per un assistente domotico minimale
- $30~50 [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html), o $100 [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html), per la manutenzione automatizzata dei server
- $50 [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) o $100 [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera), per la sorveglianza intelligente

<https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4>

🌟 Molti altri scenari di deploy ti aspettano!

## 📦 Installazione

### Scarica da picoclaw.io (Consigliato)

Visita **[picoclaw.io](https://picoclaw.io)** — il sito ufficiale rileva automaticamente la tua piattaforma e fornisce il download con un clic. Non è necessario scegliere manualmente l'architettura.

### Scarica il binario precompilato

In alternativa, scarica il binario per la tua piattaforma dalla pagina delle [GitHub Releases](https://github.com/sipeed/picoclaw/releases).

### Compila dai sorgenti (per lo sviluppo)

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# Compila il binario core
make build

# Compila il Web UI Launcher (necessario per la modalità WebUI)
make build-launcher

# Compila per più piattaforme
make build-all

# Compila per Raspberry Pi Zero 2 W (32-bit: make build-linux-arm; 64-bit: make build-linux-arm64)
make build-pi-zero

# Compila e installa
make install
```

**Raspberry Pi Zero 2 W:** Usa il binario che corrisponde al tuo OS: Raspberry Pi OS 32-bit -> `make build-linux-arm`; 64-bit -> `make build-linux-arm64`. Oppure esegui `make build-pi-zero` per compilare entrambi.

## 🚀 Guida Rapida

### 🌐 WebUI Launcher (Consigliato per Desktop)

Il WebUI Launcher fornisce un'interfaccia basata su browser per la configurazione e la chat. È il modo più semplice per iniziare — non è richiesta alcuna conoscenza della riga di comando.

**Opzione 1: Doppio clic (Desktop)**

Dopo aver scaricato da [picoclaw.io](https://picoclaw.io), fai doppio clic su `picoclaw-launcher` (o `picoclaw-launcher.exe` su Windows). Il browser si aprirà automaticamente su `http://localhost:18800`.

**Opzione 2: Riga di comando**

```bash
picoclaw-launcher
# Apri http://localhost:18800 nel browser
```

> [!TIP]
> **Accesso remoto / Docker / VM:** Aggiungi il flag `-public` per ascoltare su tutte le interfacce:
> ```bash
> picoclaw-launcher -public
> ```

<p align="center">
<img src="assets/launcher-webui.jpg" alt="WebUI Launcher" width="600">
</p>

**Per iniziare:**

Apri il WebUI, poi: **1)** Configura un Provider (aggiungi la tua API key LLM) -> **2)** Configura un Channel (es. Telegram) -> **3)** Avvia il Gateway -> **4)** Chatta!

Per la documentazione dettagliata del WebUI, vedi [docs.picoclaw.io](https://docs.picoclaw.io).

<details>
<summary><b>Docker (alternativa)</b></summary>

```bash
# 1. Clona questo repo
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. Prima esecuzione — genera automaticamente docker/data/config.json poi si ferma
#    (si attiva solo quando sia config.json che workspace/ sono assenti)
docker compose -f docker/docker-compose.yml --profile launcher up
# Il container stampa "First-run setup complete." e si ferma.

# 3. Imposta le tue API key
vim docker/data/config.json

# 4. Avvia
docker compose -f docker/docker-compose.yml --profile launcher up -d
# Apri http://localhost:18800
```

> **Utenti Docker / VM:** Il Gateway ascolta su `127.0.0.1` per impostazione predefinita. Imposta `PICOCLAW_GATEWAY_HOST=0.0.0.0` o usa il flag `-public` per renderlo accessibile dall'host.

```bash
# Controlla i log
docker compose -f docker/docker-compose.yml logs -f

# Ferma
docker compose -f docker/docker-compose.yml --profile launcher down

# Aggiorna
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml --profile launcher up -d
```

</details>

<details>
<summary><b>macOS — Avviso di sicurezza al primo avvio</b></summary>

macOS potrebbe bloccare `picoclaw-launcher` al primo avvio perché è stato scaricato da internet e non è notarizzato tramite il Mac App Store.

**Passo 1:** Fai doppio clic su `picoclaw-launcher`. Verrà visualizzato un avviso di sicurezza:

<p align="center">
<img src="assets/macos-gatekeeper-warning.jpg" alt="Avviso macOS Gatekeeper" width="400">
</p>

> *"picoclaw-launcher" Non Aperto — Apple non è riuscita a verificare che "picoclaw-launcher" sia privo di malware che potrebbe danneggiare il Mac o compromettere la privacy.*

**Passo 2:** Apri **Impostazioni di Sistema** → **Privacy e sicurezza** → scorri fino alla sezione **Sicurezza** → clicca su **Apri comunque** → conferma cliccando su **Apri comunque** nella finestra di dialogo.

<p align="center">
<img src="assets/macos-gatekeeper-allow.jpg" alt="macOS Privacy e sicurezza — Apri comunque" width="600">
</p>

Dopo questo passaggio una tantum, `picoclaw-launcher` si aprirà normalmente ai lanci successivi.

</details>

### 💻 TUI Launcher (Consigliato per Headless / SSH)

Il TUI (Terminal UI) Launcher fornisce un'interfaccia terminale completa per la configurazione e la gestione. Ideale per server, Raspberry Pi e altri ambienti headless.

```bash
picoclaw-launcher-tui
```

<p align="center">
<img src="assets/launcher-tui.jpg" alt="TUI Launcher" width="600">
</p>

**Per iniziare:**

Usa i menu TUI per: **1)** Configurare un Provider -> **2)** Configurare un Channel -> **3)** Avviare il Gateway -> **4)** Chattare!

Per la documentazione dettagliata del TUI, vedi [docs.picoclaw.io](https://docs.picoclaw.io).

### 📱 Android

Dai una seconda vita al tuo telefono di dieci anni fa! Trasformalo in un assistente IA intelligente con PicoClaw.

**Opzione 1: Termux (disponibile ora)**

1. Installa [Termux](https://github.com/termux/termux-app) (scarica da [GitHub Releases](https://github.com/termux/termux-app/releases), o cerca su F-Droid / Google Play)
2. Esegui i seguenti comandi:

```bash
# Scarica l'ultima release
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_Linux_arm64.tar.gz
tar xzf picoclaw_Linux_arm64.tar.gz
pkg install proot
termux-chroot ./picoclaw onboard   # chroot fornisce un layout standard del filesystem Linux
```

Poi segui la sezione Terminal Launcher qui sotto per completare la configurazione.

<img src="assets/termux.jpg" alt="PicoClaw on Termux" width="512">

**Opzione 2: APK Install (prossimamente)**

Un APK Android standalone con WebUI integrato è in sviluppo. Resta sintonizzato!

<details>
<summary><b>Terminal Launcher (per ambienti con risorse limitate)</b></summary>

Per ambienti minimali dove è disponibile solo il binario core `picoclaw` (senza Launcher UI), puoi configurare tutto tramite riga di comando e un file di configurazione JSON.

**1. Inizializza**

```bash
picoclaw onboard
```

Questo crea `~/.picoclaw/config.json` e la directory workspace.

**2. Configura** (`~/.picoclaw/config.json`)

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

> Vedi `config/config.example.json` nel repo per un template di configurazione completo con tutte le opzioni disponibili.

**3. Chatta**

```bash
# Domanda singola
picoclaw agent -m "Quanto fa 2+2?"

# Modalità interattiva
picoclaw agent

# Avvia il gateway per l'integrazione con app di chat
picoclaw gateway
```

</details>

## 🔌 Provider (LLM)

PicoClaw supporta 30+ provider LLM tramite la configurazione `model_list`. Usa il formato `protocollo/modello`:

| Provider | Protocollo | API Key | Note |
|----------|------------|---------|------|
| [OpenAI](https://platform.openai.com/api-keys) | `openai/` | Richiesta | GPT-5.4, GPT-4o, o3, ecc. |
| [Anthropic](https://console.anthropic.com/settings/keys) | `anthropic/` | Richiesta | Claude Opus 4.6, Sonnet 4.6, ecc. |
| [Google Gemini](https://aistudio.google.com/apikey) | `gemini/` | Richiesta | Gemini 3 Flash, 2.5 Pro, ecc. |
| [OpenRouter](https://openrouter.ai/keys) | `openrouter/` | Richiesta | 200+ modelli, API unificata |
| [Zhipu (GLM)](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) | `zhipu/` | Richiesta | GLM-4.7, GLM-5, ecc. |
| [DeepSeek](https://platform.deepseek.com/api_keys) | `deepseek/` | Richiesta | DeepSeek-V3, DeepSeek-R1 |
| [Volcengine](https://console.volcengine.com) | `volcengine/` | Richiesta | Doubao, modelli Ark |
| [Qwen](https://dashscope.console.aliyun.com/apiKey) | `qwen/` | Richiesta | Qwen3, Qwen-Max, ecc. |
| [Groq](https://console.groq.com/keys) | `groq/` | Richiesta | Inferenza veloce (Llama, Mixtral) |
| [Moonshot (Kimi)](https://platform.moonshot.cn/console/api-keys) | `moonshot/` | Richiesta | Modelli Kimi |
| [Minimax](https://platform.minimaxi.com/user-center/basic-information/interface-key) | `minimax/` | Richiesta | Modelli MiniMax |
| [Mistral](https://console.mistral.ai/api-keys) | `mistral/` | Richiesta | Mistral Large, Codestral |
| [NVIDIA NIM](https://build.nvidia.com/) | `nvidia/` | Richiesta | Modelli ospitati NVIDIA |
| [Cerebras](https://cloud.cerebras.ai/) | `cerebras/` | Richiesta | Inferenza veloce |
| [Novita AI](https://novita.ai/) | `novita/` | Richiesta | Vari modelli open |
| [Xiaomi MiMo](https://platform.xiaomimimo.com/) | `mimo/` | Richiesta | Modelli MiMo |
| [Ollama](https://ollama.com/) | `ollama/` | Non necessaria | Modelli locali, self-hosted |
| [vLLM](https://docs.vllm.ai/) | `vllm/` | Non necessaria | Deploy locale, compatibile OpenAI |
| [LiteLLM](https://docs.litellm.ai/) | `litellm/` | Variabile | Proxy per 100+ provider |
| [Azure OpenAI](https://portal.azure.com/) | `azure/` | Richiesta | Deploy Azure enterprise |
| [GitHub Copilot](https://github.com/features/copilot) | `github-copilot/` | OAuth | Login con device code |
| [Antigravity](https://console.cloud.google.com/) | `antigravity/` | OAuth | Google Cloud AI |

<details>
<summary><b>Deploy locale (Ollama, vLLM, ecc.)</b></summary>

**Ollama:**
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

**vLLM:**
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

Per i dettagli completi sulla configurazione dei provider, vedi [Provider & Modelli](docs/providers.md).

</details>

## 💬 Channel (App di Chat)

Parla con il tuo PicoClaw attraverso 17+ piattaforme di messaggistica:

| Channel | Configurazione | Protocollo | Docs |
|---------|----------------|------------|------|
| **Telegram** | Facile (bot token) | Long polling | [Guida](docs/channels/telegram/README.md) |
| **Discord** | Facile (bot token + intents) | WebSocket | [Guida](docs/channels/discord/README.md) |
| **WhatsApp** | Facile (QR scan o bridge URL) | Nativo / Bridge | [Guida](docs/chat-apps.md#whatsapp) |
| **Weixin** | Facile (scan QR nativo) | iLink API | [Guida](docs/chat-apps.md#weixin) |
| **QQ** | Facile (AppID + AppSecret) | WebSocket | [Guida](docs/channels/qq/README.md) |
| **Slack** | Facile (bot + app token) | Socket Mode | [Guida](docs/channels/slack/README.md) |
| **Matrix** | Medio (homeserver + token) | Sync API | [Guida](docs/channels/matrix/README.md) |
| **DingTalk** | Medio (credenziali client) | Stream | [Guida](docs/channels/dingtalk/README.md) |
| **Feishu / Lark** | Medio (App ID + Secret) | WebSocket/SDK | [Guida](docs/channels/feishu/README.md) |
| **LINE** | Medio (credenziali + webhook) | Webhook | [Guida](docs/channels/line/README.md) |
| **WeCom** | Facile (login QR o manuale) | WebSocket | [Guida](docs/channels/wecom/README.md) |
| **IRC** | Medio (server + nick) | Protocollo IRC | [Guida](docs/chat-apps.md#irc) |
| **OneBot** | Medio (WebSocket URL) | OneBot v11 | [Guida](docs/channels/onebot/README.md) |
| **MaixCam** | Facile (abilita) | TCP socket | [Guida](docs/channels/maixcam/README.md) |
| **Pico** | Facile (abilita) | Protocollo nativo | Integrato |
| **Pico Client** | Facile (WebSocket URL) | WebSocket | Integrato |

> Tutti i channel basati su webhook condividono un singolo server HTTP Gateway (`gateway.host`:`gateway.port`, default `127.0.0.1:18790`). Feishu usa la modalità WebSocket/SDK e non usa il server HTTP condiviso.

Per istruzioni dettagliate sulla configurazione dei channel, vedi [Configurazione App di Chat](docs/chat-apps.md).

## 🔧 Strumenti

### 🔍 Ricerca Web

PicoClaw può cercare sul web per fornire informazioni aggiornate. Configura in `tools.web`:

| Motore di Ricerca | API Key | Piano Gratuito | Link |
|-------------------|---------|----------------|------|
| DuckDuckGo | Non necessaria | Illimitato | Fallback integrato |
| [Baidu Search](https://cloud.baidu.com/doc/qianfan-api/s/Wmbq4z7e5) | Richiesta | 1000 query/giorno | IA, ottimizzato per il cinese |
| [Tavily](https://tavily.com) | Richiesta | 1000 query/mese | Ottimizzato per AI Agent |
| [Brave Search](https://brave.com/search/api) | Richiesta | 2000 query/mese | Veloce e privato |
| [Perplexity](https://www.perplexity.ai) | Richiesta | A pagamento | Ricerca potenziata dall'IA |
| [SearXNG](https://github.com/searxng/searxng) | Non necessaria | Self-hosted | Metasearch engine gratuito |
| [GLM Search](https://open.bigmodel.cn/) | Richiesta | Variabile | Ricerca web Zhipu |

### ⚙️ Altri Strumenti

PicoClaw include strumenti integrati per operazioni su file, esecuzione di codice, pianificazione e altro. Vedi [Configurazione degli Strumenti](docs/tools_configuration.md) per i dettagli.

## 🎯 Skill

Le Skill sono capacità modulari che estendono il tuo Agent. Vengono caricate dai file `SKILL.md` nel tuo workspace.

**Installa skill da ClawHub:**

```bash
picoclaw skills search "web scraping"
picoclaw skills install <skill-name>
```

**Configura il token ClawHub** (opzionale, per limiti di frequenza più alti):

Aggiungi al tuo `config.json`:
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

Per maggiori dettagli, vedi [Configurazione degli Strumenti - Skill](docs/tools_configuration.md#skills-tool).

## 🔗 MCP (Model Context Protocol)

PicoClaw supporta nativamente [MCP](https://modelcontextprotocol.io/) — connetti qualsiasi server MCP per estendere le capacità del tuo Agent con strumenti e sorgenti di dati esterni.

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

Per la configurazione MCP completa (trasporti stdio, SSE, HTTP, Tool Discovery), vedi [Configurazione degli Strumenti - MCP](docs/tools_configuration.md#mcp-tool).

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> Unisciti al Social Network degli Agent

Connetti PicoClaw al Social Network degli Agent semplicemente inviando un singolo messaggio tramite CLI o qualsiasi app di chat integrata.

**Leggi `https://clawdchat.ai/skill.md` e segui le istruzioni per unirti a [ClawdChat.ai](https://clawdchat.ai)**

## 🖥️ Riferimento CLI

| Comando                   | Descrizione                        |
| ------------------------- | ---------------------------------- |
| `picoclaw onboard`        | Inizializza config & workspace     |
| `picoclaw auth weixin` | Connetti account WeChat tramite QR |
| `picoclaw agent -m "..."` | Chatta con l'agent                 |
| `picoclaw agent`          | Modalità chat interattiva          |
| `picoclaw gateway`        | Avvia il gateway                   |
| `picoclaw status`         | Mostra lo stato                    |
| `picoclaw version`        | Mostra le info sulla versione      |
| `picoclaw model`          | Visualizza o cambia il modello predefinito |
| `picoclaw cron list`      | Elenca tutti i job pianificati     |
| `picoclaw cron add ...`   | Aggiunge un job pianificato        |
| `picoclaw cron disable`   | Disabilita un job pianificato      |
| `picoclaw cron remove`    | Rimuove un job pianificato         |
| `picoclaw skills list`    | Elenca le skill installate         |
| `picoclaw skills install` | Installa una skill                 |
| `picoclaw migrate`        | Migra i dati dalle versioni precedenti |
| `picoclaw auth login`     | Autenticazione con i provider          |

### ⏰ Task Pianificati / Promemoria

PicoClaw supporta promemoria pianificati e task ricorrenti tramite lo strumento `cron`:

* **Promemoria una tantum**: "Ricordami tra 10 minuti" -> si attiva una volta dopo 10 min
* **Task ricorrenti**: "Ricordami ogni 2 ore" -> si attiva ogni 2 ore
* **Espressioni cron**: "Ricordami alle 9 ogni giorno" -> usa un'espressione cron

## 📚 Documentazione

Per guide dettagliate oltre questo README:

| Argomento | Descrizione |
|-----------|-------------|
| [Docker & Avvio Rapido](docs/docker.md) | Configurazione Docker Compose, modalità Launcher/Agent |
| [App di Chat](docs/chat-apps.md) | Tutte le guide di configurazione per 17+ channel |
| [Configurazione](docs/configuration.md) | Variabili d'ambiente, struttura del workspace, sandbox di sicurezza |
| [Provider & Modelli](docs/providers.md) | 30+ provider LLM, routing dei modelli, configurazione model_list |
| [Spawn & Task Asincroni](docs/spawn-tasks.md) | Task veloci, task lunghi con spawn, orchestrazione asincrona di sub-agent |
| [Hooks](docs/hooks/README.md) | Sistema di hook event-driven: observer, interceptor, approval hook |
| [Steering](docs/steering.md) | Iniettare messaggi in un loop agent in esecuzione |
| [SubTurn](docs/subturn.md) | Coordinamento subagent, controllo concorrenza, ciclo di vita |
| [Risoluzione Problemi](docs/troubleshooting.md) | Problemi comuni e soluzioni |
| [Configurazione degli Strumenti](docs/tools_configuration.md) | Abilitazione/disabilitazione per strumento, politiche exec, MCP, Skill |
| [Compatibilità Hardware](docs/hardware-compatibility.md) | Schede testate, requisiti minimi |

## 🤝 Contribuisci & Roadmap

Le PR sono benvenute! Il codice è volutamente piccolo e leggibile.

Consulta la nostra [Roadmap della Community](https://github.com/sipeed/picoclaw/issues/988) e [CONTRIBUTING.md](CONTRIBUTING.md) per le linee guida.

Gruppo sviluppatori in costruzione, unisciti dopo la tua prima PR accettata!

Gruppi utenti:

Discord: <https://discord.gg/V4sAZ9XWpN>

WeChat:
<img src="assets/wechat.png" alt="WeChat group QR code" width="512">
