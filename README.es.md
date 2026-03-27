<div align="center">
<img src="assets/logo.webp" alt="PicoClaw" width="512">

<h1>PicoClaw: Asistente de IA Ultra-Eficiente en Go</h1>

<h3>Hardware de $10 · 10MB de RAM · Boot en ms · ¡Vamos, PicoClaw!</h3>
  <p>
    <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/Arch-x86__64%2C%20ARM64%2C%20MIPS%2C%20RISC--V%2C%20LoongArch-blue" alt="Hardware">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
    <br>
    <a href="https://picoclaw.io"><img src="https://img.shields.io/badge/Sitio-picoclaw.io-blue?style=flat&logo=google-chrome&logoColor=white" alt="Website"></a>
    <a href="https://docs.picoclaw.io/"><img src="https://img.shields.io/badge/Docs-Oficial-007acc?style=flat&logo=read-the-docs&logoColor=white" alt="Docs"></a>
    <a href="https://deepwiki.com/sipeed/picoclaw"><img src="https://img.shields.io/badge/Wiki-DeepWiki-FFA500?style=flat&logo=wikipedia&logoColor=white" alt="Wiki"></a>
    <br>
    <a href="https://x.com/SipeedIO"><img src="https://img.shields.io/badge/X_(Twitter)-SipeedIO-black?style=flat&logo=x&logoColor=white" alt="Twitter"></a>
    <a href="./assets/wechat.png"><img src="https://img.shields.io/badge/WeChat-Grupo-41d56b?style=flat&logo=wechat&logoColor=white"></a>
    <a href="https://discord.gg/V4sAZ9XWpN"><img src="https://img.shields.io/badge/Discord-Comunidad-4c60eb?style=flat&logo=discord&logoColor=white" alt="Discord"></a>
  </p>

[中文](README.zh.md) | [日本語](README.ja.md) | [Português](README.pt-br.md) | [Tiếng Việt](README.vi.md) | [Français](README.fr.md) | [Italiano](README.it.md) | [Bahasa Indonesia](README.id.md) | [English](README.md) | **Español**

</div>

---

> **PicoClaw** es un proyecto de código abierto independiente iniciado por [Sipeed](https://sipeed.com), escrito completamente en **Go** desde cero — no es un fork de OpenClaw, NanoBot ni ningún otro proyecto.

**PicoClaw** es un asistente personal de IA ultra-ligero inspirado en [NanoBot](https://github.com/HKUDS/nanobot). Fue reconstruido desde cero en **Go** mediante un proceso de "auto-bootstrapping" — el propio Agente de IA impulsó la migración de arquitectura y la optimización del código.

**Funciona en hardware de $10 con menos de 10MB de RAM** — eso es un 99% menos de memoria que OpenClaw y un 98% más barato que una Mac mini.

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
> **Aviso de Seguridad**
>
> * **SIN CRIPTO:** PicoClaw **no** ha emitido ningún token oficial ni criptomoneda. Todas las afirmaciones en `pump.fun` u otras plataformas de trading son **estafas**.
> * **DOMINIO OFICIAL:** El **ÚNICO** sitio web oficial es **[picoclaw.io](https://picoclaw.io)**, y el sitio de la empresa es **[sipeed.com](https://sipeed.com)**
> * **CUIDADO:** Muchos dominios `.ai/.org/.com/.net/...` han sido registrados por terceros. No confíes en ellos.
> * **NOTA:** PicoClaw está en desarrollo temprano y rápido. Puede haber problemas de seguridad sin resolver. No despliegues en producción antes de la v1.0.
> * **NOTA:** PicoClaw ha fusionado recientemente muchos PRs. Las versiones recientes pueden usar entre 10 y 20MB de RAM. La optimización de recursos está planificada tras la estabilización de funcionalidades.

## 📢 Novedades

2026-03-17 🚀 **¡v0.2.3 lanzada!** UI en bandeja del sistema (Windows y Linux), consulta de estado de sub-agentes (`spawn_status`), recarga en caliente experimental del Gateway, control de seguridad en Cron, y 2 correcciones de seguridad. ¡PicoClaw alcanzó las **25K Stars**!

2026-03-09 🎉 **v0.2.1 — ¡La mayor actualización hasta ahora!** Soporte del protocolo MCP, 4 nuevos canales (Matrix/IRC/WeCom/Discord Proxy), 3 nuevos proveedores (Kimi/Minimax/Avian), pipeline de visión, almacenamiento de memoria JSONL, enrutamiento de modelos.

2026-02-28 📦 **v0.2.0** lanzada con soporte para Docker Compose y Web UI Launcher.

2026-02-26 🎉 ¡PicoClaw alcanza las **20K Stars** en solo 17 días! La auto-orquestación de canales e interfaces de capacidades ya están disponibles.

<details>
<summary>Noticias anteriores...</summary>

2026-02-16 🎉 ¡PicoClaw supera las 12K Stars en una semana! Roles de mantenedores de la comunidad y [Roadmap](ROADMAP.md) lanzados oficialmente.

2026-02-13 🎉 ¡PicoClaw supera las 5000 Stars en 4 días! Hoja de ruta del proyecto y grupos de desarrolladores en proceso.

2026-02-09 🎉 **¡PicoClaw lanzado!** Construido en 1 día para llevar Agentes de IA a hardware de $10 con menos de 10MB de RAM. ¡Vamos, PicoClaw!

</details>

## ✨ Características

🪶 **Ultra-ligero**: Huella de memoria del núcleo menor a 10MB — un 99% más pequeño que OpenClaw.*

💰 **Costo mínimo**: Lo suficientemente eficiente para funcionar en hardware de $10 — un 98% más barato que una Mac mini.

⚡️ **Arranque ultrarrápido**: 400 veces más rápido en el inicio. Arranca en menos de 1 segundo incluso en un procesador de un solo núcleo a 0.6GHz.

🌍 **Verdaderamente portable**: Un único binario para arquitecturas RISC-V, ARM, MIPS y x86. ¡Un binario, funciona en todas partes!

🤖 **Auto-bootstrapped con IA**: Implementación nativa en Go puro — el 95% del código base fue generado por un Agente y perfeccionado mediante revisión humana.

🔌 **Soporte MCP**: Integración nativa con el [Model Context Protocol](https://modelcontextprotocol.io/) — conecta cualquier servidor MCP para extender las capacidades del Agente.

👁️ **Pipeline de visión**: Envía imágenes y archivos directamente al Agente — codificación automática en base64 para LLMs multimodales.

🧠 **Enrutamiento inteligente**: Enrutamiento de modelos basado en reglas — las consultas simples van a modelos ligeros, ahorrando costos de API.

_*Las versiones recientes pueden usar entre 10 y 20MB debido a la fusión rápida de PRs. La optimización de recursos está planificada. La comparación de velocidad de arranque se basa en benchmarks de un solo núcleo a 0.8GHz (ver tabla a continuación)._

<div align="center">

|                                | OpenClaw      | NanoBot                  | **PicoClaw**                           |
| ------------------------------ | ------------- | ------------------------ | -------------------------------------- |
| **Lenguaje**                   | TypeScript    | Python                   | **Go**                                 |
| **RAM**                        | >1GB          | >100MB                   | **< 10MB***                            |
| **Tiempo de arranque**</br>(núcleo 0.8GHz) | >500s | >30s               | **<1s**                                |
| **Costo**                      | Mac Mini $599 | La mayoría de placas Linux ~$50 | **Cualquier placa Linux**</br>**desde $10** |

<img src="assets/compare.jpg" alt="PicoClaw" width="512">

</div>

> **[Lista de Compatibilidad de Hardware](docs/hardware-compatibility.md)** — Consulta todas las placas probadas, desde RISC-V de $5 hasta Raspberry Pi y teléfonos Android. ¿Tu placa no aparece? ¡Envía un PR!

<p align="center">
<img src="assets/hardware-banner.jpg" alt="Compatibilidad de Hardware PicoClaw" width="100%">
</p>

## 🦾 Demostración

### 🛠️ Flujos de Trabajo Estándar del Asistente

<table align="center">
<tr align="center">
<th><p align="center">Modo Ingeniero Full-Stack</p></th>
<th><p align="center">Registro y Planificación</p></th>
<th><p align="center">Búsqueda Web y Aprendizaje</p></th>
</tr>
<tr>
<td align="center"><p align="center"><img src="assets/picoclaw_code.gif" width="240" height="180"></p></td>
<td align="center"><p align="center"><img src="assets/picoclaw_memory.gif" width="240" height="180"></p></td>
<td align="center"><p align="center"><img src="assets/picoclaw_search.gif" width="240" height="180"></p></td>
</tr>
<tr>
<td align="center">Desarrollar · Desplegar · Escalar</td>
<td align="center">Programar · Automatizar · Recordar</td>
<td align="center">Descubrir · Analizar · Tendencias</td>
</tr>
</table>

### 🐜 Despliegue Innovador de Bajo Consumo

¡PicoClaw puede desplegarse en prácticamente cualquier dispositivo Linux!

- $9.9 [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) edición E (Ethernet) o W (WiFi6), para un asistente doméstico minimalista
- $30~50 [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html), o $100 [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html), para operaciones automatizadas de servidor
- $50 [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) o $100 [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera), para vigilancia inteligente

<https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4>

🌟 ¡Más casos de despliegue esperan ser descubiertos!

## 📦 Instalación

### Descargar desde picoclaw.io (Recomendado)

Visita **[picoclaw.io](https://picoclaw.io)** — el sitio oficial detecta automáticamente tu plataforma y proporciona descarga con un solo clic. No es necesario seleccionar manualmente la arquitectura.

### Descargar binario precompilado

Alternativamente, descarga el binario para tu plataforma desde la página de [GitHub Releases](https://github.com/sipeed/picoclaw/releases).

### Compilar desde el código fuente (para desarrollo)

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# Compilar binario principal
make build

# Compilar Web UI Launcher (requerido para el modo WebUI)
make build-launcher

# Compilar para múltiples plataformas
make build-all

# Compilar para Raspberry Pi Zero 2 W (32 bits: make build-linux-arm; 64 bits: make build-linux-arm64)
make build-pi-zero

# Compilar e instalar
make install
```

**Raspberry Pi Zero 2 W:** Usa el binario que coincida con tu sistema operativo: Raspberry Pi OS de 32 bits -> `make build-linux-arm`; de 64 bits -> `make build-linux-arm64`. O ejecuta `make build-pi-zero` para compilar ambos.

## 🚀 Guía de Inicio Rápido

### 🌐 WebUI Launcher (Recomendado para Escritorio)

El WebUI Launcher proporciona una interfaz basada en el navegador para configuración y chat. Esta es la forma más sencilla de comenzar — no se requieren conocimientos de línea de comandos.

**Opción 1: Doble clic (Escritorio)**

Después de descargar desde [picoclaw.io](https://picoclaw.io), haz doble clic en `picoclaw-launcher` (o `picoclaw-launcher.exe` en Windows). Tu navegador se abrirá automáticamente en `http://localhost:18800`.

**Opción 2: Línea de comandos**

```bash
picoclaw-launcher
# Abre http://localhost:18800 en tu navegador
```

> [!TIP]
> **Acceso remoto / Docker / VM:** Agrega el flag `-public` para escuchar en todas las interfaces:
> ```bash
> picoclaw-launcher -public
> ```

<p align="center">
<img src="assets/launcher-webui.jpg" alt="WebUI Launcher" width="600">
</p>

**Primeros pasos:**

Abre el WebUI, luego: **1)** Configura un Proveedor (agrega tu clave de API de LLM) -> **2)** Configura un Canal (por ejemplo, Telegram) -> **3)** Inicia el Gateway -> **4)** ¡Empieza a chatear!

Para documentación detallada del WebUI, visita [docs.picoclaw.io](https://docs.picoclaw.io).

<details>
<summary><b>Docker (alternativa)</b></summary>

```bash
# 1. Clona este repositorio
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. Primera ejecución — genera automáticamente docker/data/config.json y se detiene
#    (solo se activa cuando config.json y workspace/ no existen)
docker compose -f docker/docker-compose.yml --profile launcher up
# El contenedor imprime "First-run setup complete." y se detiene.

# 3. Configura tus claves de API
vim docker/data/config.json

# 4. Iniciar
docker compose -f docker/docker-compose.yml --profile launcher up -d
# Abre http://localhost:18800
```

> **Usuarios de Docker / VM:** El Gateway escucha en `127.0.0.1` por defecto. Establece `PICOCLAW_GATEWAY_HOST=0.0.0.0` o usa el flag `-public` para hacerlo accesible desde el host.

```bash
# Ver logs
docker compose -f docker/docker-compose.yml logs -f

# Detener
docker compose -f docker/docker-compose.yml --profile launcher down

# Actualizar
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml --profile launcher up -d
```

</details>

### 💻 TUI Launcher (Recomendado para Entornos sin Cabeza / SSH)

El TUI (Terminal UI) Launcher proporciona una interfaz de terminal completa para configuración y gestión. Ideal para servidores, Raspberry Pi y otros entornos sin pantalla.

```bash
picoclaw-launcher-tui
```

<p align="center">
<img src="assets/launcher-tui.jpg" alt="TUI Launcher" width="600">
</p>

**Primeros pasos:**

Usa los menús del TUI para: **1)** Configurar un Proveedor -> **2)** Configurar un Canal -> **3)** Iniciar el Gateway -> **4)** ¡Chatear!

Para documentación detallada del TUI, visita [docs.picoclaw.io](https://docs.picoclaw.io).

### 📱 Android

¡Dale una segunda vida a tu teléfono de hace una década! Conviértelo en un Asistente de IA inteligente con PicoClaw.

**Opción 1: Termux (disponible ahora)**

1. Instala [Termux](https://github.com/termux/termux-app) (descarga desde [GitHub Releases](https://github.com/termux/termux-app/releases), o búscalo en F-Droid / Google Play)
2. Ejecuta los siguientes comandos:

```bash
# Descarga la última versión
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_Linux_arm64.tar.gz
tar xzf picoclaw_Linux_arm64.tar.gz
pkg install proot
termux-chroot ./picoclaw onboard   # chroot proporciona un diseño estándar del sistema de archivos Linux
```

Luego sigue la sección Terminal Launcher a continuación para completar la configuración.

<img src="assets/termux.jpg" alt="PicoClaw en Termux" width="512">

**Opción 2: Instalación por APK (próximamente)**

¡Un APK independiente para Android con WebUI integrado está en desarrollo. ¡Mantente atento!

<details>
<summary><b>Terminal Launcher (para entornos con recursos limitados)</b></summary>

Para entornos mínimos donde solo está disponible el binario principal `picoclaw` (sin Launcher UI), puedes configurar todo mediante la línea de comandos y un archivo de configuración JSON.

**1. Inicializar**

```bash
picoclaw onboard
```

Esto crea `~/.picoclaw/config.json` y el directorio workspace.

**2. Configurar** (`~/.picoclaw/config.json`)

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
      "model": "openai/gpt-5.4"
      // api_key ahora se carga desde .security.yml
    }
  ]
}
```

> Consulta `config/config.example.json` en el repositorio para una plantilla de configuración completa con todas las opciones disponibles.
>
> Nota: el formato config.example.json es la versión 0, contiene códigos sensibles y será migrado automáticamente a la versión 1+. Luego, config.json solo almacenará datos no sensibles; los códigos sensibles se guardarán en .security.yml. Si necesitas modificar los códigos manualmente, consulta `docs/security_configuration.md` para más detalles.

**3. Chatear**

```bash
# Pregunta única
picoclaw agent -m "¿Cuánto es 2+2?"

# Modo interactivo
picoclaw agent

# Iniciar gateway para integración con apps de chat
picoclaw gateway
```

</details>

## 🔌 Proveedores (LLM)

PicoClaw soporta más de 30 proveedores de LLM a través de la configuración `model_list`. Usa el formato `protocolo/modelo`:

| Proveedor | Protocolo | Clave API | Notas |
|-----------|-----------|-----------|-------|
| [OpenAI](https://platform.openai.com/api-keys) | `openai/` | Requerida | GPT-5.4, GPT-4o, o3, etc. |
| [Anthropic](https://console.anthropic.com/settings/keys) | `anthropic/` | Requerida | Claude Opus 4.6, Sonnet 4.6, etc. |
| [Google Gemini](https://aistudio.google.com/apikey) | `gemini/` | Requerida | Gemini 3 Flash, 2.5 Pro, etc. |
| [OpenRouter](https://openrouter.ai/keys) | `openrouter/` | Requerida | 200+ modelos, API unificada |
| [Zhipu (GLM)](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) | `zhipu/` | Requerida | GLM-4.7, GLM-5, etc. |
| [DeepSeek](https://platform.deepseek.com/api_keys) | `deepseek/` | Requerida | DeepSeek-V3, DeepSeek-R1 |
| [Volcengine](https://console.volcengine.com) | `volcengine/` | Requerida | Modelos Doubao, Ark |
| [Qwen](https://dashscope.console.aliyun.com/apiKey) | `qwen/` | Requerida | Qwen3, Qwen-Max, etc. |
| [Groq](https://console.groq.com/keys) | `groq/` | Requerida | Inferencia rápida (Llama, Mixtral) |
| [Moonshot (Kimi)](https://platform.moonshot.cn/console/api-keys) | `moonshot/` | Requerida | Modelos Kimi |
| [Minimax](https://platform.minimaxi.com/user-center/basic-information/interface-key) | `minimax/` | Requerida | Modelos MiniMax |
| [Mistral](https://console.mistral.ai/api-keys) | `mistral/` | Requerida | Mistral Large, Codestral |
| [NVIDIA NIM](https://build.nvidia.com/) | `nvidia/` | Requerida | Modelos alojados en NVIDIA |
| [Cerebras](https://cloud.cerebras.ai/) | `cerebras/` | Requerida | Inferencia rápida |
| [Novita AI](https://novita.ai/) | `novita/` | Requerida | Varios modelos abiertos |
| [Ollama](https://ollama.com/) | `ollama/` | No requerida | Modelos locales, auto-alojados |
| [vLLM](https://docs.vllm.ai/) | `vllm/` | No requerida | Despliegue local, compatible con OpenAI |
| [LiteLLM](https://docs.litellm.ai/) | `litellm/` | Variable | Proxy para 100+ proveedores |
| [Azure OpenAI](https://portal.azure.com/) | `azure/` | Requerida | Despliegue empresarial en Azure |
| [GitHub Copilot](https://github.com/features/copilot) | `github-copilot/` | OAuth | Login con código de dispositivo |
| [Antigravity](https://console.cloud.google.com/) | `antigravity/` | OAuth | Google Cloud AI |
| [AWS Bedrock](https://console.aws.amazon.com/bedrock)* | `bedrock/` | Credenciales AWS | Claude, Llama, Mistral en AWS |

> \* AWS Bedrock requiere el tag de compilación: `go build -tags bedrock`. Establece `api_base` con un nombre de región (p. ej., `us-east-1`) para la resolución automática de endpoints en todas las particiones de AWS (aws, aws-cn, aws-us-gov). Si usas una URL de endpoint completa en su lugar, también debes configurar `AWS_REGION` mediante variable de entorno o configuración/perfil de AWS.

<details>
<summary><b>Despliegue local (Ollama, vLLM, etc.)</b></summary>

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

Para detalles completos de configuración de proveedores, consulta [Proveedores y Modelos](docs/providers.md).

</details>

## 💬 Canales (Apps de Chat)

Habla con tu PicoClaw a través de más de 17 plataformas de mensajería:

| Canal | Configuración | Protocolo | Docs |
|-------|---------------|-----------|------|
| **Telegram** | Fácil (token de bot) | Long polling | [Guía](docs/channels/telegram/README.md) |
| **Discord** | Fácil (token de bot + intents) | WebSocket | [Guía](docs/channels/discord/README.md) |
| **WhatsApp** | Fácil (escaneo QR o URL de bridge) | Nativo / Bridge | [Guía](docs/chat-apps.md#whatsapp) |
| **Weixin** | Fácil (escaneo QR nativo) | API iLink | [Guía](docs/chat-apps.md#weixin) |
| **QQ** | Fácil (AppID + AppSecret) | WebSocket | [Guía](docs/channels/qq/README.md) |
| **Slack** | Fácil (bot + app token) | Socket Mode | [Guía](docs/channels/slack/README.md) |
| **Matrix** | Medio (homeserver + token) | Sync API | [Guía](docs/channels/matrix/README.md) |
| **DingTalk** | Medio (credenciales de cliente) | Stream | [Guía](docs/channels/dingtalk/README.md) |
| **Feishu / Lark** | Medio (App ID + Secret) | WebSocket/SDK | [Guía](docs/channels/feishu/README.md) |
| **LINE** | Medio (credenciales + webhook) | Webhook | [Guía](docs/channels/line/README.md) |
| **WeCom Bot** | Medio (URL de webhook) | Webhook | [Guía](docs/channels/wecom/wecom_bot/README.md) |
| **WeCom App** | Medio (credenciales corporativas) | Webhook | [Guía](docs/channels/wecom/wecom_app/README.md) |
| **WeCom AI Bot** | Medio (token + clave AES) | WebSocket / Webhook | [Guía](docs/channels/wecom/wecom_aibot/README.md) |
| **IRC** | Medio (servidor + nick) | Protocolo IRC | [Guía](docs/chat-apps.md#irc) |
| **OneBot** | Medio (URL WebSocket) | OneBot v11 | [Guía](docs/channels/onebot/README.md) |
| **MaixCam** | Fácil (habilitar) | TCP socket | [Guía](docs/channels/maixcam/README.md) |
| **Pico** | Fácil (habilitar) | Protocolo nativo | Integrado |
| **Pico Client** | Fácil (URL WebSocket) | WebSocket | Integrado |

> Todos los canales basados en webhook comparten un único servidor HTTP del Gateway (`gateway.host`:`gateway.port`, por defecto `127.0.0.1:18790`). Feishu usa el modo WebSocket/SDK y no utiliza el servidor HTTP compartido.

Para instrucciones detalladas de configuración de canales, consulta [Configuración de Apps de Chat](docs/chat-apps.md).

## 🔧 Herramientas

### 🔍 Búsqueda Web

PicoClaw puede buscar en la web para proporcionar información actualizada. Configura en `tools.web`:

| Motor de Búsqueda | Clave API | Plan Gratuito | Enlace |
|-------------------|-----------|---------------|--------|
| DuckDuckGo | No requerida | Ilimitado | Fallback integrado |
| [Baidu Search](https://cloud.baidu.com/doc/qianfan-api/s/Wmbq4z7e5) | Requerida | 1000 consultas/día | Potenciado por IA, optimizado para China |
| [Tavily](https://tavily.com) | Requerida | 1000 consultas/mes | Optimizado para Agentes de IA |
| [Brave Search](https://brave.com/search/api) | Requerida | 2000 consultas/mes | Rápido y privado |
| [Perplexity](https://www.perplexity.ai) | Requerida | De pago | Búsqueda potenciada por IA |
| [SearXNG](https://github.com/searxng/searxng) | No requerida | Auto-alojado | Motor de metabúsqueda gratuito |
| [GLM Search](https://open.bigmodel.cn/) | Requerida | Variable | Búsqueda web de Zhipu |

### ⚙️ Otras Herramientas

PicoClaw incluye herramientas integradas para operaciones de archivos, ejecución de código, programación y más. Consulta [Configuración de Herramientas](docs/tools_configuration.md) para más detalles.

## 🎯 Skills (Habilidades)

Las Skills son capacidades modulares que extienden tu Agente. Se cargan desde archivos `SKILL.md` en tu workspace.

**Instalar skills desde ClawHub:**

```bash
picoclaw skills search "web scraping"
picoclaw skills install <nombre-skill>
```

**Configurar token de ClawHub** (opcional, para límites de tasa más altos):

Agrega a tu `config.json`:
```json
{
  "tools": {
    "skills": {
      "registries": {
        "clawhub": {
          "auth_token": "tu-token-de-clawhub"
        }
      }
    }
  }
}
```

Para más detalles, consulta [Configuración de Herramientas - Skills](docs/tools_configuration.md#skills-tool).

## 🔗 MCP (Model Context Protocol)

PicoClaw soporta nativamente el [MCP](https://modelcontextprotocol.io/) — conecta cualquier servidor MCP para extender las capacidades de tu Agente con herramientas y fuentes de datos externas.

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

Para la configuración completa de MCP (transportes stdio, SSE, HTTP, Descubrimiento de Herramientas), consulta [Configuración de Herramientas - MCP](docs/tools_configuration.md#mcp-tool).

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> Únete a la Red Social de Agentes

Conecta PicoClaw a la Red Social de Agentes simplemente enviando un único mensaje desde la CLI o cualquier App de Chat integrada.

**Lee `https://clawdchat.ai/skill.md` y sigue las instrucciones para unirte a [ClawdChat.ai](https://clawdchat.ai)**

## 🖥️ Referencia de CLI

| Comando                   | Descripción                      |
| ------------------------- | -------------------------------- |
| `picoclaw onboard`        | Inicializar configuración y workspace |
| `picoclaw auth weixin`    | Conectar cuenta de WeChat vía QR |
| `picoclaw agent -m "..."` | Chatear con el agente            |
| `picoclaw agent`          | Modo de chat interactivo         |
| `picoclaw gateway`        | Iniciar el gateway               |
| `picoclaw status`         | Mostrar estado                   |
| `picoclaw version`        | Mostrar información de versión   |
| `picoclaw model`          | Ver o cambiar el modelo por defecto |
| `picoclaw cron list`      | Listar todas las tareas programadas |
| `picoclaw cron add ...`   | Agregar una tarea programada     |
| `picoclaw cron disable`   | Deshabilitar una tarea programada |
| `picoclaw cron remove`    | Eliminar una tarea programada    |
| `picoclaw skills list`    | Listar skills instaladas         |
| `picoclaw skills install` | Instalar una skill               |
| `picoclaw migrate`        | Migrar datos de versiones anteriores |
| `picoclaw auth login`     | Autenticarse con proveedores     |

### ⏰ Tareas Programadas / Recordatorios

PicoClaw soporta recordatorios programados y tareas recurrentes a través de la herramienta `cron`:

* **Recordatorios únicos**: "Recuérdame en 10 minutos" -> se activa una vez después de 10 minutos
* **Tareas recurrentes**: "Recuérdame cada 2 horas" -> se activa cada 2 horas
* **Expresiones cron**: "Recuérdame todos los días a las 9am" -> usa expresión cron

## 📚 Documentación

Para guías detalladas más allá de este README:

| Tema | Descripción |
|------|-------------|
| [Docker y Inicio Rápido](docs/docker.md) | Configuración de Docker Compose, modos Launcher/Agent |
| [Apps de Chat](docs/chat-apps.md) | Guías de configuración de todos los 17+ canales |
| [Configuración](docs/configuration.md) | Variables de entorno, diseño del workspace, sandbox de seguridad |
| [Proveedores y Modelos](docs/providers.md) | 30+ proveedores de LLM, enrutamiento de modelos, configuración de model_list |
| [Tareas Spawn y Asíncronas](docs/spawn-tasks.md) | Tareas rápidas, tareas largas con spawn, orquestación asíncrona de sub-agentes |
| [Hooks](docs/hooks/README.md) | Sistema de hooks dirigido por eventos: observadores, interceptores, hooks de aprobación |
| [Steering](docs/steering.md) | Inyectar mensajes en un bucle de agente en ejecución entre llamadas a herramientas |
| [SubTurn](docs/subturn.md) | Coordinación de sub-agentes, control de concurrencia, ciclo de vida |
| [Solución de Problemas](docs/troubleshooting.md) | Problemas comunes y soluciones |
| [Configuración de Herramientas](docs/tools_configuration.md) | Habilitación/deshabilitación por herramienta, políticas de ejecución, MCP, Skills |
| [Compatibilidad de Hardware](docs/hardware-compatibility.md) | Placas probadas, requisitos mínimos |

## 🤝 Contribuir y Hoja de Ruta

¡Los PRs son bienvenidos! El código base es intencionalmente pequeño y legible.

Consulta nuestra [Hoja de Ruta de la Comunidad](https://github.com/sipeed/picoclaw/issues/988) y [CONTRIBUTING.md](CONTRIBUTING.md) para las pautas.

¡Grupo de desarrolladores en construcción, únete después de tu primer PR fusionado!

Grupos de Usuarios:

Discord: <https://discord.gg/V4sAZ9XWpN>

WeChat:
<img src="assets/wechat.png" alt="Código QR del grupo de WeChat" width="512">
