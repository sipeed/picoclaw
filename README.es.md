<div align="center">
  <img src="assets/logo.webp" alt="PicoClaw" width="512">

  <h1>PicoClaw: Asistente de IA Ultra-Eficiente en Go</h1>

  <h3>Hardware de $10 · 10 MB de RAM · Arranque en 1 s · 皮皮虾，我们走！</h3>

  <p>
    <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/Arch-x86__64%2C%20ARM64%2C%20MIPS%2C%20RISC--V-blue" alt="Hardware">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
    <br>
    <a href="https://picoclaw.io"><img src="https://img.shields.io/badge/Website-picoclaw.io-blue?style=flat&logo=google-chrome&logoColor=white" alt="Website"></a>
    <a href="https://x.com/SipeedIO"><img src="https://img.shields.io/badge/X_(Twitter)-SipeedIO-black?style=flat&logo=x&logoColor=white" alt="Twitter"></a>
    <br>
    <a href="./assets/wechat.png"><img src="https://img.shields.io/badge/WeChat-Group-41d56b?style=flat&logo=wechat&logoColor=white"></a>
    <a href="https://discord.gg/V4sAZ9XWpN"><img src="https://img.shields.io/badge/Discord-Community-4c60eb?style=flat&logo=discord&logoColor=white" alt="Discord"></a>
  </p>

**Español** | [中文](README.zh.md) | [日本語](README.ja.md) | [Português](README.pt-br.md) | [Tiếng Việt](README.vi.md) | [Français](README.fr.md) | [English](README.md)

</div>

---

🦐 PicoClaw es un asistente de IA personal ultraligero inspirado en [nanobot](https://github.com/HKUDS/nanobot), reescrito desde cero en Go mediante un proceso de auto-generación en el que el propio agente de IA dirigió toda la migración arquitectónica y la optimización del código.

⚡️ Funciona en hardware de $10 con menos de 10 MB de RAM: ¡un 99 % menos de memoria que OpenClaw y un 98 % más barato que una Mac mini!

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
> **🚨 SEGURIDAD Y CANALES OFICIALES / 安全声明**
>
> * **SIN CRIPTOMONEDAS:** PicoClaw **NO** tiene token ni moneda oficial. Cualquier afirmación en `pump.fun` u otras plataformas de trading son **ESTAFAS**.
>
> * **DOMINIO OFICIAL:** El **ÚNICO** sitio web oficial es **[picoclaw.io](https://picoclaw.io)**, y el sitio de la empresa es **[sipeed.com](https://sipeed.com)**
> * **Advertencia:** Muchos dominios `.ai/.org/.com/.net/...` han sido registrados por terceros.
> * **Advertencia:** PicoClaw se encuentra en desarrollo temprano y puede tener problemas de seguridad de red sin resolver. No lo despliegues en entornos de producción antes del lanzamiento de la versión v1.0.
> * **Nota:** PicoClaw ha incorporado recientemente muchos PRs, lo que puede resultar en un mayor consumo de memoria (10–20 MB) en las versiones más recientes. Tenemos previsto priorizar la optimización de recursos en cuanto el conjunto de características actual alcance un estado estable.

## 📢 Noticias

2026-02-16 🎉 ¡PicoClaw alcanzó las 12 000 estrellas en una semana! ¡Gracias a todos por vuestro apoyo! PicoClaw está creciendo más rápido de lo que jamás imaginamos. Dado el gran volumen de PRs, necesitamos urgentemente mantenedores de la comunidad. Los roles de voluntario y la hoja de ruta están publicados oficialmente [aquí](ROADMAP.md) — ¡estamos deseando tenerte a bordo!

2026-02-13 🎉 ¡PicoClaw alcanzó las 5000 estrellas en 4 días! ¡Gracias a la comunidad! Han llegado muchísimos PRs e issues (durante las vacaciones del Año Nuevo Chino); estamos finalizando la Hoja de Ruta del Proyecto y configurando el Grupo de Desarrolladores para acelerar el desarrollo de PicoClaw.  
🚀 Llamada a la acción: Por favor, envía tus solicitudes de funcionalidades en GitHub Discussions. Las revisaremos y priorizaremos durante nuestra próxima reunión semanal.

2026-02-09 🎉 ¡PicoClaw ha sido lanzado! Construido en 1 día para llevar agentes de IA a hardware de $10 con menos de 10 MB de RAM. 🦐 ¡PicoClaw, vamos!

## ✨ Características

🪶 **Ultra-Ligero**: Huella de memoria inferior a 10 MB — un 99 % más pequeño que Clawdbot en su funcionalidad principal.

💰 **Coste mínimo**: Lo suficientemente eficiente para ejecutarse en hardware de $10 — un 98 % más barato que una Mac mini.

⚡️ **Increíblemente rápido**: Tiempo de arranque 400 veces más rápido, inicia en 1 segundo incluso en un núcleo a 0,6 GHz.

🌍 **Verdadera portabilidad**: Binario único autocontenido para RISC-V, ARM, MIPS y x86. ¡Un clic para ejecutarlo!

🤖 **Impulsado por IA**: Implementación nativa en Go de forma autónoma — núcleo generado en un 95 % por agentes con refinamiento humano.

|                               | OpenClaw      | NanoBot                  | **PicoClaw**                              |
| ----------------------------- | ------------- | ------------------------ | ----------------------------------------- |
| **Lenguaje**                  | TypeScript    | Python                   | **Go**                                    |
| **RAM**                       | >1 GB         | >100 MB                  | **< 10 MB**                               |
| **Arranque**</br>(núcleo a 0,8 GHz) | >500 s   | >30 s                    | **<1 s**                                  |
| **Coste**                     | Mac Mini 599$ | La mayoría de SBC Linux </br>~50$ | **Cualquier placa Linux**</br>**Desde $10** |

<img src="assets/compare.jpg" alt="PicoClaw" width="512">

## 🦾 Demostración

### 🛠️ Flujos de Trabajo Estándar del Asistente

<table align="center">
  <tr align="center">
    <th><p align="center">🧩 Ingeniería Full-Stack</p></th>
    <th><p align="center">🗂️ Gestión de Registros y Planificación</p></th>
    <th><p align="center">🔎 Búsqueda Web y Aprendizaje</p></th>
  </tr>
  <tr>
    <td align="center"><p align="center"><img src="assets/picoclaw_code.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_memory.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_search.gif" width="240" height="180"></p></td>
  </tr>
  <tr>
    <td align="center">Desarrollar · Desplegar · Escalar</td>
    <td align="center">Planificar · Automatizar · Recordar</td>
    <td align="center">Descubrir · Analizar · Tendencias</td>
  </tr>
</table>

### 📱 Ejecución en Teléfonos Android Antiguos

¡Dale una segunda vida a tu teléfono de hace una década! Conviértelo en un asistente de IA inteligente con PicoClaw. Inicio rápido:

1. **Instala Termux** (disponible en F-Droid o Google Play).
2. **Ejecuta los comandos**

```bash
# Nota: Reemplaza v0.1.1 con la versión más reciente de la página de Releases
wget https://github.com/sipeed/picoclaw/releases/download/v0.1.1/picoclaw-linux-arm64
chmod +x picoclaw-linux-arm64
pkg install proot
termux-chroot ./picoclaw-linux-arm64 onboard
```

¡Luego sigue las instrucciones de la sección "Inicio Rápido" para completar la configuración!
<img src="assets/termux.jpg" alt="PicoClaw" width="512">

### 🐜 Despliegue Innovador de Bajo Consumo

¡PicoClaw puede desplegarse en prácticamente cualquier dispositivo Linux!

- $9,9 [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) versión E (Ethernet) o W (WiFi6), para un asistente doméstico mínimo
- $30–50 [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html), o $100 [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html) para mantenimiento automatizado de servidores
- $50 [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) o $100 [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera) para monitorización inteligente

<https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4>

🌟 ¡Más casos de despliegue están por llegar!

## 📦 Instalación

### Instalación con binario precompilado

Descarga el firmware para tu plataforma desde la página de [releases](https://github.com/sipeed/picoclaw/releases).

### Instalación desde el código fuente (últimas funcionalidades, recomendado para desarrollo)

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# Compilar, sin necesidad de instalar
make build

# Compilar para múltiples plataformas
make build-all

# Compilar para Raspberry Pi Zero 2 W (32 bits: make build-linux-arm; 64 bits: make build-linux-arm64)
make build-pi-zero

# Compilar e instalar
make install
```

**Raspberry Pi Zero 2 W:** Usa el binario que corresponda a tu sistema operativo: Raspberry Pi OS de 32 bits → `make build-linux-arm` (salida: `build/picoclaw-linux-arm`); 64 bits → `make build-linux-arm64` (salida: `build/picoclaw-linux-arm64`). O ejecuta `make build-pi-zero` para compilar ambos.

## 🐳 Docker Compose

También puedes ejecutar PicoClaw con Docker Compose sin instalar nada localmente.

```bash
# 1. Clona este repositorio
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. Primera ejecución: genera automáticamente docker/data/config.json y luego se detiene
docker compose -f docker/docker-compose.yml --profile gateway up
# El contenedor imprime "First-run setup complete." y se detiene.

# 3. Configura tus claves API
vim docker/data/config.json   # Establece las claves API del proveedor, tokens de bot, etc.

# 4. Inicia
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

> [!TIP]
> **Usuarios de Docker**: Por defecto, el Gateway escucha en `127.0.0.1`, lo que no es accesible desde el host. Si necesitas acceder a los endpoints de salud o exponer puertos, establece `PICOCLAW_GATEWAY_HOST=0.0.0.0` en tu entorno o actualiza `config.json`.

```bash
# 5. Revisar los registros
docker compose -f docker/docker-compose.yml logs -f picoclaw-gateway

# 6. Detener
docker compose -f docker/docker-compose.yml --profile gateway down
```

### Modo Launcher (Consola Web)

La imagen `launcher` incluye los tres binarios (`picoclaw`, `picoclaw-launcher`, `picoclaw-launcher-tui`) e inicia la consola web por defecto, que proporciona una interfaz basada en navegador para configuración y chat.

```bash
docker compose -f docker/docker-compose.yml --profile launcher up -d
```

Abre http://localhost:18800 en tu navegador. El launcher gestiona el proceso del gateway automáticamente.

> [!WARNING]
> La consola web aún no admite autenticación. Evita exponerla a Internet público.

### Modo Agente (Ejecución puntual)

```bash
# Hacer una pregunta
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent -m "¿Cuánto es 2+2?"

# Modo interactivo
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent
```

### Actualizar

```bash
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

### 🚀 Inicio Rápido

> [!TIP]
> Establece tu clave API en `~/.picoclaw/config.json`. Obtén claves API en: [Volcengine (CodingPlan)](https://console.volcengine.com) (LLM) · [OpenRouter](https://openrouter.ai/keys) (LLM) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) (LLM). La búsqueda web es opcional — obtén una [API de Tavily](https://tavily.com) gratuita (1000 consultas/mes) o una [API de Brave Search](https://brave.com/search/api) (2000 consultas gratuitas/mes).

**1. Inicializar**

```bash
picoclaw onboard
```

**2. Configurar** (`~/.picoclaw/config.json`)

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model_name": "gpt-5.4",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "model_list": [
    {
      "model_name": "ark-code-latest",
      "model": "volcengine/ark-code-latest",
      "api_key": "sk-your-api-key"
    },
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_key": "your-api-key",
      "request_timeout": 300
    },
    {
      "model_name": "claude-sonnet-4.6",
      "model": "anthropic/claude-sonnet-4.6",
      "api_key": "your-anthropic-key"
    }
  ],
  "tools": {
    "web": {
      "brave": {
        "enabled": false,
        "api_key": "YOUR_BRAVE_API_KEY",
        "max_results": 5
      },
      "tavily": {
        "enabled": false,
        "api_key": "YOUR_TAVILY_API_KEY",
        "max_results": 5
      },
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      },
      "perplexity": {
        "enabled": false,
        "api_key": "YOUR_PERPLEXITY_API_KEY",
        "max_results": 5
      },
      "searxng": {
        "enabled": false,
        "base_url": "http://your-searxng-instance:8888",
        "max_results": 5
      }
    }
  }
}
```

> **Nuevo**: El formato de configuración `model_list` permite añadir proveedores sin modificar el código. Consulta [Configuración de Modelos](#configuración-de-modelos-model_list) para más detalles.
> `request_timeout` es opcional y se expresa en segundos. Si se omite o se establece en `<= 0`, PicoClaw usa el tiempo de espera predeterminado (120 s).

**3. Obtener claves API**

* **Proveedor LLM**: [OpenRouter](https://openrouter.ai/keys) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) · [Anthropic](https://console.anthropic.com) · [OpenAI](https://platform.openai.com) · [Gemini](https://aistudio.google.com/api-keys)
* **Búsqueda web** (opcional):
  * [Brave Search](https://brave.com/search/api) - De pago ($5/1000 consultas, ~$5–6/mes)
  * [Perplexity](https://www.perplexity.ai) - Búsqueda con IA e interfaz de chat
  * [SearXNG](https://github.com/searxng/searxng) - Motor de metabúsqueda autoalojado (gratuito, sin clave API)
  * [Tavily](https://tavily.com) - Optimizado para agentes de IA (1000 peticiones/mes)
  * DuckDuckGo - Alternativa incorporada (sin clave API requerida)

> **Nota**: Consulta `config.example.json` para una plantilla de configuración completa.

**4. Chatear**

```bash
picoclaw agent -m "¿Cuánto es 2+2?"
```

¡Eso es todo! Tendrás un asistente de IA funcionando en 2 minutos.

---

## 💬 Aplicaciones de Chat

Habla con PicoClaw a través de Telegram, Discord, WhatsApp, Matrix, QQ, DingTalk, LINE o WeCom.

> **Nota**: Todos los canales basados en webhooks (LINE, WeCom, etc.) se sirven en un único servidor HTTP Gateway compartido (`gateway.host`:`gateway.port`, por defecto `127.0.0.1:18790`). No hay puertos individuales por canal. Nota: Feishu usa el modo WebSocket/SDK y no utiliza el servidor HTTP de webhooks compartido.

| Canal        | Configuración                              |
| ------------ | ------------------------------------------ |
| **Telegram** | Sencilla (solo un token)                   |
| **Discord**  | Sencilla (token de bot + intents)          |
| **WhatsApp** | Sencilla (nativo: escaneo QR; o URL de puente) |
| **Matrix**   | Media (servidor homeserver + token de acceso del bot) |
| **QQ**       | Sencilla (AppID + AppSecret)               |
| **DingTalk** | Media (credenciales de la app)             |
| **LINE**     | Media (credenciales + URL de webhook)      |
| **WeCom AI Bot** | Media (Token + clave AES)             |

<details>
<summary><b>Telegram</b> (Recomendado)</summary>

**1. Crear un bot**

* Abre Telegram y busca `@BotFather`
* Envía `/newbot` y sigue las instrucciones
* Copia el token

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

> Obtén tu ID de usuario desde `@userinfobot` en Telegram.

**3. Ejecutar**

```bash
picoclaw gateway
```

**4. Menú de comandos de Telegram (registro automático al inicio)**

PicoClaw mantiene las definiciones de comandos en un registro compartido. Al iniciarse, Telegram registrará automáticamente los comandos del bot compatibles (por ejemplo, `/start`, `/help`, `/show`, `/list`), de modo que el menú de comandos y el comportamiento en tiempo de ejecución estén sincronizados.
El registro del menú de comandos de Telegram es un proceso de descubrimiento local al canal; la ejecución genérica de comandos se gestiona de forma centralizada en el bucle del agente a través del ejecutor de comandos.

Si el registro de comandos falla (errores transitorios de red o de la API), el canal sigue iniciándose y PicoClaw reintenta el registro en segundo plano.

</details>

<details>
<summary><b>Discord</b></summary>

**1. Crear un bot**

* Ve a <https://discord.com/developers/applications>
* Crea una aplicación → Bot → Añadir Bot
* Copia el token del bot

**2. Habilitar intents**

* En la configuración del Bot, habilita **MESSAGE CONTENT INTENT**
* (Opcional) Habilita **SERVER MEMBERS INTENT** si planeas usar listas de permisos basadas en datos de miembros

**3. Obtener tu ID de usuario**
* Ajustes de Discord → Avanzado → habilita el **Modo desarrollador**
* Haz clic derecho en tu avatar → **Copiar ID de usuario**

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

**5. Invitar al bot**

* OAuth2 → Generador de URL
* Ámbitos: `bot`
* Permisos del bot: `Send Messages`, `Read Message History`
* Abre la URL de invitación generada y añade el bot a tu servidor

**Opcional: Modo de activación en grupos**

Por defecto, el bot responde a todos los mensajes en un canal de servidor. Para restringir las respuestas solo a menciones con @, añade:

```json
{
  "channels": {
    "discord": {
      "group_trigger": { "mention_only": true }
    }
  }
}
```

También puedes activarlo mediante prefijos de palabras clave (por ejemplo, `!bot`):

```json
{
  "channels": {
    "discord": {
      "group_trigger": { "prefixes": ["!bot"] }
    }
  }
}
```

**6. Ejecutar**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>WhatsApp</b> (nativo a través de whatsmeow)</summary>

PicoClaw puede conectarse a WhatsApp de dos formas:

- **Nativo (recomendado):** En proceso usando [whatsmeow](https://github.com/tulir/whatsmeow). Sin puente externo. Establece `"use_native": true` y deja `bridge_url` vacío. En la primera ejecución, escanea el código QR con WhatsApp (Dispositivos vinculados). La sesión se almacena en tu espacio de trabajo (por ejemplo, `workspace/whatsapp/`). El canal nativo es **opcional** para mantener el binario predeterminado pequeño; compila con `-tags whatsapp_native` (por ejemplo, `make build-whatsapp-native` o `go build -tags whatsapp_native ./cmd/...`).
- **Puente:** Conéctate a un puente WebSocket externo. Establece `bridge_url` (por ejemplo, `ws://localhost:3001`) y mantén `use_native` en false.

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

Si `session_store_path` está vacío, la sesión se almacena en `<workspace>/whatsapp/`. Ejecuta `picoclaw gateway`; en la primera ejecución, escanea el código QR que aparece en el terminal con WhatsApp → Dispositivos vinculados.

</details>

<details>
<summary><b>QQ</b></summary>

**1. Crear un bot**

- Ve a la [Plataforma Abierta de QQ](https://q.qq.com/#)
- Crea una aplicación → Obtén el **AppID** y el **AppSecret**

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

> Deja `allow_from` vacío para permitir todos los usuarios, o especifica números de QQ para restringir el acceso.

**3. Ejecutar**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>DingTalk</b></summary>

**1. Crear un bot**

* Ve a la [Plataforma Abierta](https://open.dingtalk.com/)
* Crea una aplicación interna
* Copia el Client ID y el Client Secret

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

> Deja `allow_from` vacío para permitir todos los usuarios, o especifica IDs de usuario de DingTalk para restringir el acceso.

**3. Ejecutar**

```bash
picoclaw gateway
```
</details>

<details>
<summary><b>Matrix</b></summary>

**1. Preparar la cuenta del bot**

* Usa tu servidor homeserver preferido (por ejemplo, `https://matrix.org` o uno autoalojado)
* Crea un usuario bot y obtén su token de acceso

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

**3. Ejecutar**

```bash
picoclaw gateway
```

Para opciones completas (`device_id`, `join_on_invite`, `group_trigger`, `placeholder`, `reasoning_channel_id`), consulta la [Guía de Configuración del Canal Matrix](docs/channels/matrix/README.md).

</details>

<details>
<summary><b>LINE</b></summary>

**1. Crear una cuenta oficial de LINE**

- Ve a la [Consola de Desarrolladores de LINE](https://developers.line.biz/)
- Crea un proveedor → Crea un canal de API de Mensajería
- Copia el **Channel Secret** y el **Channel Access Token**

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

> El webhook de LINE se sirve en el servidor Gateway compartido (`gateway.host`:`gateway.port`, por defecto `127.0.0.1:18790`).

**3. Configurar la URL del webhook**

LINE requiere HTTPS para los webhooks. Usa un proxy inverso o un túnel:

```bash
# Ejemplo con ngrok (el puerto predeterminado del gateway es 18790)
ngrok http 18790
```

Luego establece la URL del webhook en la Consola de Desarrolladores de LINE como `https://your-domain/webhook/line` y habilita **Usar webhook**.

**4. Ejecutar**

```bash
picoclaw gateway
```

> En chats de grupo, el bot responde solo cuando se le menciona con @. Las respuestas citan el mensaje original.

</details>

<details>
<summary><b>WeCom (企业微信)</b></summary>

PicoClaw admite tres tipos de integración con WeCom:

**Opción 1: Bot de WeCom (Bot)** - Configuración más sencilla, admite chats de grupo
**Opción 2: App de WeCom (App personalizada)** - Más funciones, mensajería proactiva, solo chat privado
**Opción 3: Bot de IA de WeCom (AI Bot)** - Bot de IA oficial, respuestas en streaming, admite grupos y chat privado

Consulta la [Guía de Configuración del Bot de IA de WeCom](docs/channels/wecom/wecom_aibot/README.zh.md) para instrucciones detalladas de configuración.

**Configuración rápida - Bot de WeCom:**

**1. Crear un bot**

* Ve a la Consola de Administración de WeCom → Chat de grupo → Añadir bot de grupo
* Copia la URL del webhook (formato: `https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx`)

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

> El webhook de WeCom se sirve en el servidor Gateway compartido (`gateway.host`:`gateway.port`, por defecto `127.0.0.1:18790`).

**Configuración rápida - App de WeCom:**

**1. Crear una app**

* Ve a la Consola de Administración de WeCom → Gestión de apps → Crear app
* Copia el **AgentId** y el **Secret**
* Ve a la página "Mi empresa" y copia el **CorpID**

**2. Configurar la recepción de mensajes**

* En los detalles de la app, haz clic en "Recibir mensajes" → "Establecer API"
* Establece la URL como `http://your-server:18790/webhook/wecom-app`
* Genera el **Token** y el **EncodingAESKey**

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

**4. Ejecutar**

```bash
picoclaw gateway
```

> **Nota**: Las callbacks de webhook de WeCom se sirven en el puerto del Gateway (por defecto 18790). Usa un proxy inverso para HTTPS.

**Configuración rápida - Bot de IA de WeCom:**

**1. Crear un Bot de IA**

* Ve a la Consola de Administración de WeCom → Gestión de apps → Bot de IA
* En la configuración del Bot de IA, configura la URL de callback: `http://your-server:18791/webhook/wecom-aibot`
* Copia el **Token** y haz clic en "Generar aleatoriamente" para el **EncodingAESKey**

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
      "welcome_message": "¡Hola! ¿En qué puedo ayudarte?"
    }
  }
}
```

**3. Ejecutar**

```bash
picoclaw gateway
```

> **Nota**: El Bot de IA de WeCom usa el protocolo de extracción en streaming — sin preocupaciones por tiempos de espera en las respuestas. Las tareas largas (más de 30 segundos) cambian automáticamente a entrega mediante `response_url`.

</details>

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> Únete a la Red Social de Agentes

Conecta PicoClaw a la Red Social de Agentes simplemente enviando un mensaje a través de la CLI o cualquier aplicación de chat integrada.

**Lee `https://clawdchat.ai/skill.md` y sigue las instrucciones para unirte a [ClawdChat.ai](https://clawdchat.ai)**

## ⚙️ Configuración

Archivo de configuración: `~/.picoclaw/config.json`

### Variables de Entorno

Puedes sobrescribir las rutas predeterminadas usando variables de entorno. Esto resulta útil para instalaciones portátiles, despliegues en contenedores o para ejecutar PicoClaw como un servicio del sistema. Estas variables son independientes y controlan rutas diferentes.

| Variable          | Descripción                                                                                                                                    | Ruta predeterminada       |
|-------------------|------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------|
| `PICOCLAW_CONFIG` | Sobrescribe la ruta al archivo de configuración. Indica directamente a PicoClaw qué `config.json` cargar, ignorando todas las demás ubicaciones. | `~/.picoclaw/config.json` |
| `PICOCLAW_HOME`   | Sobrescribe el directorio raíz de datos de PicoClaw. Cambia la ubicación predeterminada del `workspace` y otros directorios de datos.           | `~/.picoclaw`             |

**Ejemplos:**

```bash
# Ejecutar PicoClaw usando un archivo de configuración específico
# La ruta del workspace se leerá desde ese archivo de configuración
PICOCLAW_CONFIG=/etc/picoclaw/production.json picoclaw gateway

# Ejecutar PicoClaw con todos sus datos almacenados en /opt/picoclaw
# La configuración se cargará desde el ~/.picoclaw/config.json predeterminado
# El workspace se creará en /opt/picoclaw/workspace
PICOCLAW_HOME=/opt/picoclaw picoclaw agent

# Usar ambas para una configuración totalmente personalizada
PICOCLAW_HOME=/srv/picoclaw PICOCLAW_CONFIG=/srv/picoclaw/main.json picoclaw gateway
```

### Estructura del Workspace

PicoClaw almacena los datos en tu workspace configurado (por defecto: `~/.picoclaw/workspace`):

```
~/.picoclaw/workspace/
├── sessions/          # Sesiones de conversación e historial
├── memory/           # Memoria a largo plazo (MEMORY.md)
├── state/            # Estado persistente (último canal, etc.)
├── cron/             # Base de datos de tareas programadas
├── skills/           # Habilidades personalizadas
├── AGENTS.md         # Guía de comportamiento del agente
├── HEARTBEAT.md      # Prompts de tareas periódicas (verificado cada 30 min)
├── IDENTITY.md       # Identidad del agente
├── SOUL.md           # Alma del agente
└── USER.md           # Preferencias del usuario
```

### Fuentes de Habilidades

Por defecto, las habilidades se cargan desde:

1. `~/.picoclaw/workspace/skills` (workspace)
2. `~/.picoclaw/skills` (global)
3. `<directorio-de-trabajo-actual>/skills` (incorporado)

Para configuraciones avanzadas o de prueba, puedes sobrescribir la raíz de habilidades incorporadas con:

```bash
export PICOCLAW_BUILTIN_SKILLS=/path/to/skills
```

### Política Unificada de Ejecución de Comandos

- Los comandos con barra diagonal genéricos se ejecutan a través de una única ruta en `pkg/agent/loop.go` mediante `commands.Executor`.
- Los adaptadores de canal ya no procesan comandos genéricos localmente; reenvían el texto entrante al bus/agente. Telegram sigue registrando automáticamente los comandos compatibles al inicio.
- Los comandos con barra diagonal desconocidos (por ejemplo, `/foo`) se pasan al procesamiento normal del LLM.
- Un comando registrado pero no admitido en el canal actual (por ejemplo, `/show` en WhatsApp) devuelve un error explícito al usuario y detiene el procesamiento.

### 🔒 Sandbox de Seguridad

PicoClaw se ejecuta en un entorno aislado (sandbox) por defecto. El agente solo puede acceder a archivos y ejecutar comandos dentro del workspace configurado.

#### Configuración Predeterminada

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

| Opción                  | Predeterminado          | Descripción                                           |
| ----------------------- | ----------------------- | ----------------------------------------------------- |
| `workspace`             | `~/.picoclaw/workspace` | Directorio de trabajo del agente                      |
| `restrict_to_workspace` | `true`                  | Restringe el acceso a archivos/comandos al workspace  |

#### Herramientas Protegidas

Cuando `restrict_to_workspace: true`, las siguientes herramientas están aisladas en el sandbox:

| Herramienta   | Función              | Restricción                                         |
| ------------- | -------------------- | --------------------------------------------------- |
| `read_file`   | Leer archivos        | Solo archivos dentro del workspace                  |
| `write_file`  | Escribir archivos    | Solo archivos dentro del workspace                  |
| `list_dir`    | Listar directorios   | Solo directorios dentro del workspace               |
| `edit_file`   | Editar archivos      | Solo archivos dentro del workspace                  |
| `append_file` | Añadir a archivos    | Solo archivos dentro del workspace                  |
| `exec`        | Ejecutar comandos    | Las rutas de los comandos deben estar en el workspace |

#### Protección Adicional para exec

Incluso con `restrict_to_workspace: false`, la herramienta `exec` bloquea estos comandos peligrosos:

* `rm -rf`, `del /f`, `rmdir /s` — Eliminación masiva
* `format`, `mkfs`, `diskpart` — Formateo de disco
* `dd if=` — Imagen de disco
* Escritura en `/dev/sd[a-z]` — Escritura directa en disco
* `shutdown`, `reboot`, `poweroff` — Apagado del sistema
* Fork bomb `:(){ :|:& };:`

#### Ejemplos de Errores

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (path outside working dir)}
```

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (dangerous pattern detected)}
```

#### Deshabilitar las Restricciones (Riesgo de Seguridad)

Si necesitas que el agente acceda a rutas fuera del workspace:

**Método 1: Archivo de configuración**

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": false
    }
  }
}
```

**Método 2: Variable de entorno**

```bash
export PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE=false
```

> ⚠️ **Advertencia**: Deshabilitar esta restricción permite al agente acceder a cualquier ruta del sistema. Úsalo con precaución únicamente en entornos controlados.

#### Consistencia del Perímetro de Seguridad

La configuración `restrict_to_workspace` se aplica de forma coherente en todas las rutas de ejecución:

| Ruta de Ejecución      | Perímetro de Seguridad              |
| ---------------------- | ----------------------------------- |
| Agente principal       | `restrict_to_workspace` ✅          |
| Subagente / Spawn      | Hereda la misma restricción ✅      |
| Tareas de Heartbeat    | Hereda la misma restricción ✅      |

Todas las rutas comparten la misma restricción de workspace — no hay forma de saltarse el perímetro de seguridad a través de subagentes o tareas programadas.

### Heartbeat (Tareas Periódicas)

PicoClaw puede realizar tareas periódicas de forma automática. Crea un archivo `HEARTBEAT.md` en tu workspace:

```markdown
# Tareas Periódicas

- Revisar mi correo electrónico en busca de mensajes importantes
- Consultar mi calendario para ver los próximos eventos
- Comprobar el pronóstico del tiempo
```

El agente leerá este archivo cada 30 minutos (configurable) y ejecutará las tareas usando las herramientas disponibles.

#### Tareas Asíncronas con Spawn

Para tareas de larga duración (búsqueda web, llamadas a API), usa la herramienta `spawn` para crear un **subagente**:

```markdown
# Tareas Periódicas

## Tareas Rápidas (responder directamente)

- Reportar la hora actual

## Tareas Largas (usar spawn para ejecución asíncrona)

- Buscar noticias de IA en la web y hacer un resumen
- Revisar el correo electrónico e informar de los mensajes importantes
```

**Comportamientos clave:**

| Característica          | Descripción                                                         |
| ----------------------- | ------------------------------------------------------------------- |
| **spawn**               | Crea un subagente asíncrono, no bloquea el heartbeat                |
| **Contexto independiente** | El subagente tiene su propio contexto, sin historial de sesión   |
| **herramienta message** | El subagente se comunica directamente con el usuario mediante la herramienta message |
| **No bloqueante**       | Tras el spawn, el heartbeat continúa con la siguiente tarea         |

#### Cómo Funciona la Comunicación del Subagente

```
El heartbeat se activa
    ↓
El agente lee HEARTBEAT.md
    ↓
Para tarea larga: lanza un subagente (spawn)
    ↓                                   ↓
Continúa con la siguiente tarea    El subagente trabaja de forma independiente
    ↓                                   ↓
Todas las tareas completadas       El subagente usa la herramienta "message"
    ↓                                   ↓
Responde HEARTBEAT_OK              El usuario recibe el resultado directamente
```

El subagente tiene acceso a herramientas (message, web_search, etc.) y puede comunicarse con el usuario de forma independiente sin pasar por el agente principal.

**Configuración:**

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

| Opción     | Predeterminado | Descripción                              |
| ---------- | -------------- | ---------------------------------------- |
| `enabled`  | `true`         | Habilitar/deshabilitar el heartbeat      |
| `interval` | `30`           | Intervalo de verificación en minutos (mínimo: 5) |

**Variables de entorno:**

* `PICOCLAW_HEARTBEAT_ENABLED=false` para deshabilitar
* `PICOCLAW_HEARTBEAT_INTERVAL=60` para cambiar el intervalo

### Proveedores

> [!NOTE]
> Groq proporciona transcripción de voz gratuita a través de Whisper. Si está configurado, los mensajes de audio de cualquier canal se transcribirán automáticamente a nivel del agente.

| Proveedor                  | Propósito                                | Obtener clave API                                                    |
| -------------------------- | ---------------------------------------- | -------------------------------------------------------------------- |
| `gemini`                   | LLM (Gemini directo)                     | [aistudio.google.com](https://aistudio.google.com)                   |
| `zhipu`                    | LLM (Zhipu directo)                      | [bigmodel.cn](https://bigmodel.cn)                                   |
| `openrouter (por probar)`  | LLM (recomendado, acceso a todos los modelos) | [openrouter.ai](https://openrouter.ai)                          |
| `anthropic (por probar)`   | LLM (Claude directo)                     | [console.anthropic.com](https://console.anthropic.com)               |
| `openai (por probar)`      | LLM (GPT directo)                        | [platform.openai.com](https://platform.openai.com)                   |
| `deepseek (por probar)`    | LLM (DeepSeek directo)                   | [platform.deepseek.com](https://platform.deepseek.com)               |
| `qwen`                     | LLM (Qwen directo)                       | [dashscope.console.aliyun.com](https://dashscope.console.aliyun.com) |
| `groq`                     | LLM + **Transcripción de voz** (Whisper) | [console.groq.com](https://console.groq.com)                         |
| `cerebras`                 | LLM (Cerebras directo)                   | [cerebras.ai](https://cerebras.ai)                                   |
| `vivgrid`                  | LLM (Vivgrid directo)                    | [vivgrid.com](https://vivgrid.com)                                   |

### Configuración de Modelos (model_list)

> **¿Qué hay de nuevo?** PicoClaw ahora usa un enfoque de configuración **centrado en el modelo**. Simplemente especifica el formato `proveedor/modelo` (por ejemplo, `zhipu/glm-4.7`) para añadir nuevos proveedores — **¡sin cambios de código!**

Este diseño también permite el **soporte multi-agente** con selección flexible de proveedores:

- **Diferentes agentes, diferentes proveedores**: Cada agente puede usar su propio proveedor LLM
- **Fallbacks de modelo**: Configura modelos primarios y de respaldo para mayor resiliencia
- **Balanceo de carga**: Distribuye las peticiones entre múltiples endpoints
- **Configuración centralizada**: Gestiona todos los proveedores en un solo lugar

#### 📋 Todos los Proveedores Compatibles

| Proveedor              | Prefijo `model`   | Base de API predeterminada                          | Protocolo | Clave API                                                        |
| ---------------------- | ----------------- | --------------------------------------------------- | --------- | ---------------------------------------------------------------- |
| **OpenAI**             | `openai/`         | `https://api.openai.com/v1`                         | OpenAI    | [Obtener clave](https://platform.openai.com)                     |
| **Anthropic**          | `anthropic/`      | `https://api.anthropic.com/v1`                      | Anthropic | [Obtener clave](https://console.anthropic.com)                   |
| **智谱 AI (GLM)**      | `zhipu/`          | `https://open.bigmodel.cn/api/paas/v4`              | OpenAI    | [Obtener clave](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) |
| **DeepSeek**           | `deepseek/`       | `https://api.deepseek.com/v1`                       | OpenAI    | [Obtener clave](https://platform.deepseek.com)                   |
| **Google Gemini**      | `gemini/`         | `https://generativelanguage.googleapis.com/v1beta`  | OpenAI    | [Obtener clave](https://aistudio.google.com/api-keys)            |
| **Groq**               | `groq/`           | `https://api.groq.com/openai/v1`                    | OpenAI    | [Obtener clave](https://console.groq.com)                        |
| **Moonshot**           | `moonshot/`       | `https://api.moonshot.cn/v1`                        | OpenAI    | [Obtener clave](https://platform.moonshot.cn)                    |
| **通义千问 (Qwen)**    | `qwen/`           | `https://dashscope.aliyuncs.com/compatible-mode/v1` | OpenAI    | [Obtener clave](https://dashscope.console.aliyun.com)            |
| **NVIDIA**             | `nvidia/`         | `https://integrate.api.nvidia.com/v1`               | OpenAI    | [Obtener clave](https://build.nvidia.com)                        |
| **Ollama**             | `ollama/`         | `http://localhost:11434/v1`                         | OpenAI    | Local (sin clave requerida)                                      |
| **OpenRouter**         | `openrouter/`     | `https://openrouter.ai/api/v1`                      | OpenAI    | [Obtener clave](https://openrouter.ai/keys)                      |
| **LiteLLM Proxy**      | `litellm/`        | `http://localhost:4000/v1`                          | OpenAI    | Tu clave del proxy LiteLLM                                       |
| **VLLM**               | `vllm/`           | `http://localhost:8000/v1`                          | OpenAI    | Local                                                            |
| **Cerebras**           | `cerebras/`       | `https://api.cerebras.ai/v1`                        | OpenAI    | [Obtener clave](https://cerebras.ai)                             |
| **VolcEngine (Doubao)**| `volcengine/`     | `https://ark.cn-beijing.volces.com/api/v3`          | OpenAI    | [Obtener clave](https://console.volcengine.com)                  |
| **神算云**             | `shengsuanyun/`   | `https://router.shengsuanyun.com/api/v1`            | OpenAI    | -                                                                |
| **BytePlus**           | `byteplus/`       | `https://ark.ap-southeast.bytepluses.com/api/v3`   | OpenAI    | [Obtener clave](https://console.volcengine.com)                  |
| **Vivgrid**            | `vivgrid/`        | `https://api.vivgrid.com/v1`                        | OpenAI    | [Obtener clave](https://vivgrid.com)                             |
| **LongCat**            | `longcat/`        | `https://api.longcat.chat/openai`                   | OpenAI    | [Obtener clave](https://longcat.chat/platform)                   |
| **Antigravity**        | `antigravity/`    | Google Cloud                                        | Custom    | Solo OAuth                                                       |
| **GitHub Copilot**     | `github-copilot/` | `localhost:4321`                                    | gRPC      | -                                                                |

#### Configuración Básica

```json
{
  "model_list": [
    {
      "model_name": "ark-code-latest",
      "model": "volcengine/ark-code-latest",
      "api_key": "sk-your-api-key"
    },
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
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
      "model": "gpt-5.4"
    }
  }
}
```

#### Ejemplos por Proveedor

**OpenAI**

```json
{
  "model_name": "gpt-5.4",
  "model": "openai/gpt-5.4",
  "api_key": "sk-..."
}
```

**VolcEngine (Doubao)**

```json
{
  "model_name": "ark-code-latest",
  "model": "volcengine/ark-code-latest",
  "api_key": "sk-..."
}
```

**智谱 AI (GLM)**

```json
{
  "model_name": "glm-4.7",
  "model": "zhipu/glm-4.7",
  "api_key": "your-key"
}
```

**DeepSeek**

```json
{
  "model_name": "deepseek-chat",
  "model": "deepseek/deepseek-chat",
  "api_key": "sk-..."
}
```

**Anthropic (con clave API)**

```json
{
  "model_name": "claude-sonnet-4.6",
  "model": "anthropic/claude-sonnet-4.6",
  "api_key": "sk-ant-your-key"
}
```

> Ejecuta `picoclaw auth login --provider anthropic` para pegar tu token de API.

**Ollama (local)**

```json
{
  "model_name": "llama3",
  "model": "ollama/llama3"
}
```

**Proxy/API personalizado**

```json
{
  "model_name": "my-custom-model",
  "model": "openai/custom-model",
  "api_base": "https://my-proxy.com/v1",
  "api_key": "sk-...",
  "request_timeout": 300
}
```

**LiteLLM Proxy**

```json
{
  "model_name": "lite-gpt4",
  "model": "litellm/lite-gpt4",
  "api_base": "http://localhost:4000/v1",
  "api_key": "sk-..."
}
```

PicoClaw elimina únicamente el prefijo externo `litellm/` antes de enviar la petición, por lo que los alias del proxy como `litellm/lite-gpt4` envían `lite-gpt4`, mientras que `litellm/openai/gpt-4o` envía `openai/gpt-4o`.

#### Balanceo de Carga

Configura múltiples endpoints para el mismo nombre de modelo — PicoClaw alternará automáticamente entre ellos en round-robin:

```json
{
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_base": "https://api1.example.com/v1",
      "api_key": "sk-key1"
    },
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_base": "https://api2.example.com/v1",
      "api_key": "sk-key2"
    }
  ]
}
```

#### Migración desde la Configuración `providers` Antigua

La configuración antigua de `providers` está **obsoleta** pero sigue siendo compatible por razones de retrocompatibilidad.

**Configuración antigua (obsoleta):**

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

**Nueva configuración (recomendada):**

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

Para una guía de migración detallada, consulta [docs/migration/model-list-migration.md](docs/migration/model-list-migration.md).

### Arquitectura de Proveedores

PicoClaw enruta los proveedores por familia de protocolo:

- Protocolo compatible con OpenAI: OpenRouter, gateways compatibles con OpenAI, Groq, Zhipu y endpoints tipo vLLM.
- Protocolo Anthropic: comportamiento nativo de la API de Claude.
- Ruta Codex/OAuth: ruta de autenticación OAuth/token de OpenAI.

Esto mantiene el tiempo de ejecución ligero y hace que la incorporación de nuevos backends compatibles con OpenAI sea principalmente una operación de configuración (`api_base` + `api_key`).

<details>
<summary><b>Zhipu</b></summary>

**1. Obtener clave API y URL base**

* Obtén tu [clave API](https://bigmodel.cn/usercenter/proj-mgmt/apikeys)

**2. Configurar**

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
      "api_key": "Tu clave API",
      "api_base": "https://open.bigmodel.cn/api/paas/v4"
    }
  }
}
```

**3. Ejecutar**

```bash
picoclaw agent -m "Hola"
```

</details>

<details>
<summary><b>Ejemplo de configuración completa</b></summary>

```json
{
  "agents": {
    "defaults": {
      "model": "anthropic/claude-opus-4-5"
    }
  },
  "session": {
    "dm_scope": "per-channel-peer",
    "backlog_limit": 20
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
      "enabled": false,
      "bridge_url": "ws://localhost:3001",
      "use_native": false,
      "session_store_path": "",
      "allow_from": []
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
      },
      "perplexity": {
        "enabled": false,
        "api_key": "",
        "max_results": 5
      },
      "searxng": {
        "enabled": false,
        "base_url": "http://localhost:8888",
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

## Referencia de la CLI

| Comando                   | Descripción                              |
| ------------------------- | ---------------------------------------- |
| `picoclaw onboard`        | Inicializar la configuración y el workspace |
| `picoclaw agent -m "..."` | Chatear con el agente                    |
| `picoclaw agent`          | Modo de chat interactivo                 |
| `picoclaw gateway`        | Iniciar el gateway                       |
| `picoclaw status`         | Mostrar el estado                        |
| `picoclaw cron list`      | Listar todas las tareas programadas      |
| `picoclaw cron add ...`   | Añadir una tarea programada              |

### Tareas Programadas / Recordatorios

PicoClaw admite recordatorios programados y tareas recurrentes a través de la herramienta `cron`:

* **Recordatorios únicos**: "Recuérdame en 10 minutos" → se activa una vez tras 10 min
* **Tareas recurrentes**: "Recuérdame cada 2 horas" → se activa cada 2 horas
* **Expresiones cron**: "Recuérdame a las 9 de la mañana cada día" → usa una expresión cron

Los trabajos se almacenan en `~/.picoclaw/workspace/cron/` y se procesan automáticamente.

## 🤝 Contribuir y Hoja de Ruta

¡Los PRs son bienvenidos! El código es intencionalmente pequeño y legible. 🤗

Consulta nuestra [Hoja de Ruta de la Comunidad](https://github.com/sipeed/picoclaw/blob/main/ROADMAP.md) completa.

¡El grupo de desarrolladores está en formación, únete tras tu primer PR aceptado!

Grupos de usuarios:

Discord: <https://discord.gg/V4sAZ9XWpN>

<img src="assets/wechat.png" alt="PicoClaw" width="512">

## 🐛 Solución de Problemas

### La búsqueda web indica "problema de configuración de clave API"

Esto es normal si aún no has configurado una clave API de búsqueda. PicoClaw proporcionará enlaces útiles para realizar búsquedas manuales.

#### Prioridad de Proveedores de Búsqueda

PicoClaw selecciona automáticamente el mejor proveedor de búsqueda disponible en este orden:
1. **Perplexity** (si está habilitado y tiene clave API configurada) - Búsqueda con IA y citas
2. **Brave Search** (si está habilitado y tiene clave API configurada) - API de pago orientada a la privacidad ($5/1000 consultas)
3. **SearXNG** (si está habilitado y tiene `base_url` configurada) - Metabuscador autoalojado que agrega más de 70 motores (gratuito)
4. **DuckDuckGo** (si está habilitado, alternativa predeterminada) - Sin clave API requerida (gratuito)

#### Opciones de Configuración de Búsqueda Web

**Opción 1 (Mejores resultados)**: Búsqueda con IA de Perplexity
```json
{
  "tools": {
    "web": {
      "perplexity": {
        "enabled": true,
        "api_key": "YOUR_PERPLEXITY_API_KEY",
        "max_results": 5
      }
    }
  }
}
```

**Opción 2 (API de pago)**: Obtén una clave API en [https://brave.com/search/api](https://brave.com/search/api) ($5/1000 consultas, ~$5–6/mes)
```json
{
  "tools": {
    "web": {
      "brave": {
        "enabled": true,
        "api_key": "YOUR_BRAVE_API_KEY",
        "max_results": 5
      }
    }
  }
}
```

**Opción 3 (Autoalojado)**: Despliega tu propia instancia de [SearXNG](https://github.com/searxng/searxng)
```json
{
  "tools": {
    "web": {
      "searxng": {
        "enabled": true,
        "base_url": "http://your-server:8888",
        "max_results": 5
      }
    }
  }
}
```

Ventajas de SearXNG:
- **Sin coste**: Sin tarifas de API ni límites de uso
- **Orientado a la privacidad**: Autoalojado, sin rastreo
- **Resultados agregados**: Consulta más de 70 motores de búsqueda simultáneamente
- **Ideal para VMs en la nube**: Resuelve problemas de bloqueo de IPs de centros de datos (Oracle Cloud, GCP, AWS, Azure)
- **Sin clave API**: Solo despliega y configura la URL base

**Opción 4 (Sin configuración previa)**: DuckDuckGo está habilitado por defecto como alternativa (sin clave API requerida)

Añade la clave a `~/.picoclaw/config.json` si usas Brave:

```json
{
  "tools": {
    "web": {
      "brave": {
        "enabled": false,
        "api_key": "YOUR_BRAVE_API_KEY",
        "max_results": 5
      },
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      },
      "perplexity": {
        "enabled": false,
        "api_key": "YOUR_PERPLEXITY_API_KEY",
        "max_results": 5
      },
      "searxng": {
        "enabled": false,
        "base_url": "http://your-searxng-instance:8888",
        "max_results": 5
      }
    }
  }
}
```

### Aparecen errores de filtrado de contenido

Algunos proveedores (como Zhipu) aplican filtros de contenido. Intenta reformular tu consulta o utiliza un modelo diferente.

### El bot de Telegram indica "Conflict: terminated by other getUpdates"

Esto ocurre cuando hay otra instancia del bot en ejecución. Asegúrate de que solo haya un proceso `picoclaw gateway` activo a la vez.

---

## 📝 Comparativa de Claves API

| Servicio          | Nivel Gratuito                  | Caso de Uso                                      |
| ----------------- | ------------------------------- | ------------------------------------------------ |
| **OpenRouter**    | 200 000 tokens/mes              | Múltiples modelos (Claude, GPT-4, etc.)          |
| **Volcengine CodingPlan** | ¥9,9/primer mes        | Ideal para usuarios chinos, múltiples modelos SOTA (Doubao, DeepSeek, etc.) |
| **Zhipu**         | 200 000 tokens/mes              | Adecuado para usuarios chinos                    |
| **Brave Search**  | De pago ($5/1000 consultas)     | Funcionalidad de búsqueda web                    |
| **SearXNG**       | Ilimitado (autoalojado)         | Metabúsqueda orientada a la privacidad (70+ motores) |
| **Groq**          | Nivel gratuito disponible       | Inferencia rápida (Llama, Mixtral)               |
| **Cerebras**      | Nivel gratuito disponible       | Inferencia rápida (Llama, Qwen, etc.)            |
| **LongCat**       | Hasta 5 M tokens/día            | Inferencia rápida (nivel gratuito)               |

---

<div align="center">
  <img src="assets/logo.jpg" alt="PicoClaw Meme" width="512">
</div>