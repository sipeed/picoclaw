<div align="center">
  <img src="assets/logo.webp" alt="PicoClaw" width="512">

  <h1>PicoClaw: Asistente de IA Ultra-Eficiente en Go</h1>

  <h3>Hardware por 10$ · 10MB de RAM · Arranque en 1s · 皮皮虾，我们走！</h3>

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

🦐 PicoClaw es un Asistente de IA personal ultraligero inspirado en [nanobot](https://github.com/HKUDS/nanobot), reescrito desde cero en Go mediante un proceso de autogestión, donde el propio agente de IA dirigió toda la migración de arquitectura y la optimización del código.

⚡️ Corre en hardware de $10 con <10MB de RAM: ¡Eso es un 99% menos de memoria que OpenClaw y 98% más barato que una Mac mini!

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
> * **SIN CRIPTO:** PicoClaw **NO** tiene ningún token/moneda oficial. Cualquier afirmación en `pump.fun` u otras plataformas comerciales son **ESTAFAS**.
>
> * **DOMINIO OFICIAL:** El **ÚNICO** sitio web oficial es **[picoclaw.io](https://picoclaw.io)**, y el sitio de la empresa es **[sipeed.com](https://sipeed.com)**
> * **Advertencia:** Muchos dominios `.ai/.org/.com/.net/...` están registrados por terceros.
> * **Advertencia:** picoclaw se encuentra actualmente en desarrollo inicial y puede tener problemas de seguridad de red no resueltos. No lo implementes en entornos de producción antes de la versión v1.0.
> * **Nota:** picoclaw ha incorporado recientemente muchos PRs, lo que puede resultar en un mayor consumo de memoria (10–20MB) en las últimas versiones. Planeamos priorizar la optimización de recursos tan pronto como las funcionalidades actuales alcancen un estado estable.

## 📢 Noticias

2026-02-16 🎉 ¡PicoClaw alcanzó 12K estrellas en una semana! ¡Gracias a todos por su apoyo! PicoClaw está creciendo más rápido de lo que jamás imaginamos. Dado el alto volumen de PRs, necesitamos con urgencia mantenedores en la comunidad. Nuestros roles para voluntarios y la hoja de ruta están publicados oficialmente [aquí](ROADMAP.md) —¡Estamos ansiosos por tenerte a bordo!

2026-02-13 🎉 ¡PicoClaw alcanzó 5000 estrellas en 4 días! ¡Gracias a la comunidad! Hay un montón de PRs e issues llegando (durante las vacaciones del Año Nuevo Chino), estamos finalizando la Hoja de Ruta del proyecto y creando el Grupo de Desarrolladores para acelerar el avance de PicoClaw.  
🚀 Llamada a la Acción: Por favor envía tus sugerencias de funciones en GitHub Discussions. Las revisaremos y priorizaremos en nuestra próxima reunión semanal.

2026-02-09 🎉 ¡PicoClaw ha sido Lanzado! Creado en 1 día para llevar Agentes de IA a hardware de $10 con <10MB de RAM. 🦐 PicoClaw，Let's Go！

## ✨ Características

🪶 **Ultra-Ligero**: Huella de memoria de <10MB — 99% más ligero que Clawdbot (en funciones base).

💰 **Costo Mínimo**: Lo bastante eficiente como para ejecutarse en hardware de $10 — 98% más barato que un Mac mini.

⚡️ **Rápido como el Rayo**: Un inicio 400 veces más rápido, arranca en 1 segundo incluso en monocore de 0.6GHz.

🌍 **Verdadera Portabilidad**: Un único binario ejecutable e independiente en RISC-V, ARM, MIPS, y x86, ¡Un clic y a correr!

🤖 **IA Inicializada por sí sola**: Implementación autónoma e independiente nativa en Go — 95% del código del núcleo fue generado por el Agente con refinamiento humano guiado.

|                               | OpenClaw      | NanoBot                  | **PicoClaw**                              |
| ----------------------------- | ------------- | ------------------------ | ----------------------------------------- |
| **Lenguaje**                  | TypeScript    | Python                   | **Go**                                    |
| **RAM**                       | >1GB          | >100MB                   | **< 10MB**                                |
| **Arranque**</br>(núcleo 0.8GHz) | >500s         | >30s                     | **<1s**                                   |
| **Costo**                      | Mac Mini 599$ | Muchos Linux SBC </br>~50$ | **Cualquier Placa Linux**</br>**Desde 10$** |

<img src="assets/compare.jpg" alt="PicoClaw" width="512">

## 🦾 Demonstration

### 🛠️ Standard Assistant Workflows

<table align="center">
  <tr align="center">
    <th><p align="center">🧩 Full-Stack Engineer</p></th>
    <th><p align="center">🗂️ Logging & Planning Management</p></th>
    <th><p align="center">🔎 Web Search & Learning</p></th>
  </tr>
  <tr>
    <td align="center"><p align="center"><img src="assets/picoclaw_code.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_memory.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_search.gif" width="240" height="180"></p></td>
  </tr>
  <tr>
    <td align="center">Develop • Deploy • Scale</td>
    <td align="center">Schedule • Automate • Memory</td>
    <td align="center">Discovery • Insights • Trends</td>
  </tr>
</table>

### 📱 Run on old Android Phones

¡Dale a tu teléfono de hace una década una segunda vida! Conviértelo en un Asistente de IA inteligente con PicoClaw. Inicio rápido:

1. **Instala Termux** (Disponible en F-Droid o Google Play).
2. **Ejecuta los siguientes comandos**

```bash
# Note: Replace v0.1.1 with the latest version from the Releases page
wget https://github.com/sipeed/picoclaw/releases/download/v0.1.1/picoclaw-linux-arm64
chmod +x picoclaw-linux-arm64
pkg install proot
termux-chroot ./picoclaw-linux-arm64 onboard
```

¡Y luego sigue las instrucciones en la sección "Inicio Rápido" para terminar de configurar!
<img src="assets/termux.jpg" alt="PicoClaw" width="512">

### 🐜 Innovative Low-Footprint Deploy

¡PicoClaw puede ser instalado en casi cualquier dispositivo Linux!

- $9.9 [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) versión E(Ethernet) o W(WiFi6), para un Asistente del Hogar Mínimo
- $30~50 [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html), o $100 [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html) para Mantenimiento Automatizado de Servidores
- $50 [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) o $100 [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera) para Monitoreo Inteligente

<https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4>

🌟 ¡Más Casos de Instalación Están Por Venir!

## 📦 Install

### Instalación mediante binario precompilado

Descarga el firmware de tu respectiva plataforma accediendo a la página de [releases](https://github.com/sipeed/picoclaw/releases).

### Instalación desde el código fuente (últimas funciones, recomendado para desarrolladores)

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# Build, no need to install
make build

# Build for multiple platforms
make build-all

# Build for Raspberry Pi Zero 2 W (32-bit: make build-linux-arm; 64-bit: make build-linux-arm64)
make build-pi-zero

# Build And Install
make install
```

**Raspberry Pi Zero 2 W:** Usa el binario que corresponda a tu SO: Raspberry Pi OS 32-bit → `make build-linux-arm` (salida: `build/picoclaw-linux-arm`); 64-bit → `make build-linux-arm64` (salida: `build/picoclaw-linux-arm64`). O ejecuta `make build-pi-zero` para compilar todo a la vez.

## 🐳 Docker Compose

También puedes ejecutar PicoClaw usando Docker Compose sin instalar nada en local.

```bash
# 1. Clona este proyecto
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. En el primer inicio — auto genera el docker/data/config.json y luego se cierra
docker compose -f docker/docker-compose.yml --profile gateway up
# El contenedor imprimirá en consola un "First-run setup complete." y se detendrá a sí mismo.

# 3. Pon tus contraseñas y llaves de la API
vim docker/data/config.json   # Pon las api keys del proveedor, los tokens, etc.

# 4. Iniciar
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

> [!TIP]
> **Usuarios de Docker**: Por defecto, el Gateway escucha del enlace `127.0.0.1` el cual es inaccesible desde el host público local. Si necesitas acceder a los endpoints o los puertos, establece la variable `PICOCLAW_GATEWAY_HOST=0.0.0.0` para tu entorno local o actualiza de manera manual el archivo de `config.json`.

```bash
# 5. Comprobar registros (logs)
docker compose -f docker/docker-compose.yml logs -f picoclaw-gateway

# 6. Parar
docker compose -f docker/docker-compose.yml --profile gateway down
```

### Modo Interfaz (Consola Web)

La imagen del `launcher` incluye explusivamente los 3 binarios (`picoclaw`, `picoclaw-launcher`, `picoclaw-launcher-tui`) e inicia de manera predeterminada la consola web, la cual proporciona una UI dentro del explorador para uso común y configuraciones fáciles.

```bash
docker compose -f docker/docker-compose.yml --profile launcher up -d
```

Abre http://localhost:18800 en tu navegador. El `launcher` administrará el proceso del gateway automáticamente.

> [!WARNING]
> La consola web por el momento no cuenta con soporte de autenticación. Evite la exposición a servidores y entornos públicos de internet.

### Modo Agente (Disparo Único)

```bash
# Haz una pregunta
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent -m "¿Cuanto es 2+2?"

# Modo Interactivo
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent
```

### Actualización

```bash
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

### 🚀 Inicio Rápido

> [!TIP]
> Establece tus Llaves de API (API Key) en `~/.picoclaw/config.json`. Obtén las llaves aquí: [Volcengine (CodingPlan)](https://console.volcengine.com) (LLM) · [OpenRouter](https://openrouter.ai/keys) (LLM) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) (LLM). Las herramientas de búsqueda de internet son opcionales — obtén una gratis aquí: [Tavily API](https://tavily.com) (1000 res/mes) o [Brave Search API](https://brave.com/search/api) (2000 res/mes).

**1. Inicialización**

```bash
picoclaw onboard
```

**2. Configuración** (`~/.picoclaw/config.json`)

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

> **Nuevo**: El formato de configuración en `model_list` actualmente permite la funcionalidad de proveedores de 'zero-code addition'. Revisa la [Configuración del Modelo](#model-configuration-model_list) para más detalles.
> `request_timeout` es opcional y usa los milisegundos. Si se omite o es `<= 0`, PicoClaw usa el valor esperado de la terminal por defecto (120s).

**3. Obtener Llaves API**

* **Proveedores de LLM**: [OpenRouter](https://openrouter.ai/keys) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) · [Anthropic](https://console.anthropic.com) · [OpenAI](https://platform.openai.com) · [Gemini](https://aistudio.google.com/api-keys)
* **Búsqueda Web** (opcional):
  * [Brave Search](https://brave.com/search/api) - De subscripción ($5/1000 request, ~$5-6/mes)
  * [Perplexity](https://www.perplexity.ai) - Entrenado y enfocado a interfaces y aplicaciones de chat AI
  * [SearXNG](https://github.com/searxng/searxng) - Hosting local propio (gratis, API y llave no requerida)
  * [Tavily](https://tavily.com) - Específicamente enfocado y optimizado en Agentes (1000 request/mes)
  * DuckDuckGo - Búsqueda en general por defecto y fallback (API no requerida)

> **Nota**: Revisa el template de configuración completo aquí: `config.example.json`.

**4. Chatear**

```bash
picoclaw agent -m "¿Cuanto es 2+2?"
```

¡Eso es todo! Ya tienes a tu asistente personal en solo 2 minutos.

---

## 💬 Aplicaciones de Chat

Habla con tu picoclaw a través de Telegram, Discord, WhatsApp, Matrix, QQ, DingTalk, LINE o WeCom.

> **Nota**: Todos los canales que utilizan webhooks (LINE, WeCom, etc.) están servidos sobre un solo servidor HTTP Gateway abierto y compartido (`gateway.host`:`gateway.port`, el predeterminado es `127.0.0.1:18790`). No hay puertos pre-configurados para los canales. Nota: Feishu usa modo WebSocket/SDK y no utilizará el servidor HTTP webhook compartido de este servicio.

| Canal      | Configuración                              |
| ------------ | ---------------------------------- |
| **Telegram** | Fácil (solo con el token)                |
| **Discord**  | Fácil (token de bot + intents)         |
| **WhatsApp** | Fácil (nativo: escanear código QR; o usando puente de URL) |
| **Matrix**   | Medio (homeserver + token de acceso a un bot) |
| **QQ**       | Fácil (AppID + AppSecret)           |
| **DingTalk** | Medio (credenciales de app)           |
| **LINE**     | Medio (credenciales + URL del webhook) |
| **WeCom AI Bot** | Medio (Token + llave AES)       |

<details>
<summary><b>Telegram</b> (Recomendado)</summary>

**1. Crea tu bot**

* Abre Telegram, y busca al bot `@BotFather`
* Envíale `/newbot`, y sigue las instrucciones en la pantalla
* Copia el token de acceso

**2. Configura tu bot**

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

> Obtén tu propia ID de usuario hablando con el bot `@userinfobot` en la aplicación de Telegram.

**3. Ejecútalo**

```bash
picoclaw gateway
```

**4. Menú de comandos en Telegram (se auto-registra al inicio)**

PicoClaw ahora es capaz de mantener las definiciones y registros de los comandos en un solo sitio compartido. Al encenderlo, Telegram inmediatamente auto-registrará todos los comandos del bot que puedan ser permitidos (por ejemplo `/start`, `/help`, `/show`, `/list`) manteniendo en sintonía su menú de comandos en tiempo de ejecución.
El registro del menú de comandos de Telegram conservará su UX de descubrimiento por canales de chat locales; sin embargo, la ejecución en sí de comandos genéricos también es manejada mediante un lazo al agente que procesa a través del ejecutor propio de comandos.

Si el comando de registros falla (por ejemplo por culpa de errores en el API o en la conexión de la red), el canal todavía comenzará a correr con naturalidad y PicoClaw simplemente reintentará ejecutar la registración fallida de fondo.

</details>

<details>
<summary><b>Discord</b></summary>

**1. Crea un bot**

* Ve aquí: <https://discord.com/developers/applications>
* Crea tu aplicación → Bot → Añadir Bot (Add Bot)
* Copia el token que te dará el bot

**2. Habilita 'Intents'**

* Dentro de las opciones del Bot o 'Bot settings', marca para activar esto: **MESSAGE CONTENT INTENT**
* (Opcional) Actívalo también para el uso general de la lista a usar con miembros y datos de **SERVER MEMBERS INTENT** 

**3. Obtén de Discord tu propio User ID (ID de Usuario)**
* Configuración en Discord → Avanzado (Advanced) → activa el **Developer Mode** (Modo desarrollador)
* Da click derecho a tu avatar y selecciona → **Copy User ID**

**4. Configuración**

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

**5. Invita a tu bot**

* Ve a OAuth2 → Generador de enlace de URL
* Busca los Scopes y selecciona: `bot`
* Permisos del Bot (Bot Permissions): Selecciona `Send Messages` y `Read Message History`
* Ya deberías haber generado un enlace, abre el enlace generado de invitación de URL que has creado y añade tú solo al equipo servidor (server) en que estés.

**Opcional: Disparador en modo de grupos**

Por defecto, el bot siempre tratará de responder a todo mensaje que encuentre en su canal de los servidores. Si quieres configurar esto, añade una restricción simple con solo @ y la mención que tú elijas que lo llame:

```json
{
  "channels": {
    "discord": {
      "group_trigger": { "mention_only": true }
    }
  }
}
```

También le puedes asignar activadores de comando específicos con atajos en prefijos (como decir: `!bot`):

```json
{
  "channels": {
    "discord": {
      "group_trigger": { "prefixes": ["!bot"] }
    }
  }
}
```

**6. Ejecuta tu script**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>WhatsApp</b> (Nativo usando whatsmeow)</summary>

PicoClaw puede integrarse y conectarse con tu WhatsApp de dos formas distintas:

- **Nativo (Recomendado):** En un solo proceso usando la herramienta [whatsmeow](https://github.com/tulir/whatsmeow). Sin necesidad de ningún puente de configuración separado. Establece el flag de `"use_native": true` y simplemente deja en blanco `bridge_url`. Al arrancar el programa por primera vez, escanea tu código QR particular con WhatsApp (Mediante Dispositivos Enlazados). La sesión o tu token será directamente almacenada a través de tu entorno en (`workspace/whatsapp/`). Esto es **opcional** para asegurar mantener a tu binario lo más pequeño y ligero posible; compila el proyecto con `-tags whatsapp_native` (usa `make build-whatsapp-native` o construye así `go build -tags whatsapp_native ./cmd/...`).
- **Puente:** Conéctate con WhatsApp a través de un puente o enlace provisto mediante WebSocket externo. Establece y llena `bridge_url` (usa `ws://localhost:3001` como ejemplo)  y deja a `use_native` como falso.

**Configuración (Método nativo)**

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

Si `session_store_path` se encuentra vacío, la sesión pasará a ser almacenada en un directorio como `&lt;workspace&gt;/whatsapp/`. Para iniciar todo debes ejecutar mediante `picoclaw gateway`; en la primera ocasión, debes en la terminal escanear y visualizar el código QR mediante la misma aplicación de tu WhatsApp → Dispositivos Vinculados o Linked Devices.

</details>

<details>
<summary><b>QQ</b></summary>

**1. Crea un bot**

- Ve a [QQ Open Platform](https://q.qq.com/#)
- Genera una aplicación → Obten su respectiva **AppID** y su **AppSecret**

**2. Configuración**

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

> Pon `allow_from` como un arreglo vacío para brindar los permisos a cualquiera, o en todo caso puedes restringir accesos mediante y añadiendo varios números de QQ para dar un correcto permiso y autorización específicos.

**3. Ejecútalo**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>DingTalk</b></summary>

**1. Crea un bot**

* Dirígete a [Open Platform](https://open.dingtalk.com/)
* Construye una internal app
* Selecciona la llave Client ID y Client Secret y cópialas

**2. Configuración**

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

> Pon `allow_from` con un arreglo vacío para ceder permisos a nivel global y universal, o delimita accesos escribiendo y guardando las IDs de usuarios DingTalk para brindar un seguro acceso particular solo al personal y equipo.

**3. Ejecútalo**

```bash
picoclaw gateway
```
</details>

<details>
<summary><b>Matrix</b></summary>

**1. Configura una cuenta para un bot**

* Utiliza al homeserver que suelas preferir (e.g. `https://matrix.org` o en host local)
* Crea en él a un usuario y extrae y obtén el pertinente token clave de su perfil.

**2. Configuración**

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

**3. Ejecútalo**

```bash
picoclaw gateway
```

Para parámetros extras muy detallados (`device_id`, `join_on_invite`, `group_trigger`, `placeholder`, `reasoning_channel_id`), puedes ver a fondo visitando la [Guía de Configuración Matrix Channel](docs/channels/matrix/README.md).

</details>

<details>
<summary><b>LINE</b></summary>

**1. Crea una Cuenta Oficial de LINE**

- Dirígete a la [LINE Developers Console](https://developers.line.biz/)
- Crea un proveedor (provider) → Crea un canal Messaging API
- Copia tu respectivo **Channel Secret** y el **Channel Access Token**

**2. Configuración**

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

> Un webhook de LINE es servido y usado sobre un servidor o nodo de Gateway compartido (`gateway.host`:`gateway.port`, el predeterminado es `127.0.0.1:18790`).

**3. Establecer y asentar el Webhook URL**

LINE requiere siempre el uso de protocolo HTTPS para webhooks. Emplea proxy en verso o túnel seguro:

```bash
# Example with ngrok (gateway default port is 18790)
ngrok http 18790
```

Llegado este punto, establece la URL del webhook de LINE Developers Console hacia o apuntando a `https://your-domain/webhook/line` y activa la opción llamada **Use webhook**.

**4. Ejecútalo**

```bash
picoclaw gateway
```

> Aviso para uso en chats de canales grupales, el AI Bot solamente dará respuesta única si es @mencionado. Cuando se contesta se añade cita (quote) del mensaje de origen.

</details>

<details>
<summary><b>WeCom (企业微信)</b></summary>

PicoClaw soporta tres formatos distintos en integraciones WeCom:

**Opción 1: WeCom Bot (Bot)** - Modo de instalación súper sencillo, soportando los chats o conversaciones colectivas.
**Opción 2: WeCom App (Custom App)** - Incluye mejores opciones, mensajería de alertas activas, orientado pero sólo aplicable a los chats privados.
**Opción 3: WeCom AI Bot (AI Bot)** - Agente Oficial de bot integrado de inteligencia, soporta respuesta fluida e integrable en chats grupales o DM privado.

Lee atentamente e infórmate bajo detalle en la sección [Guía de Configuración WeCom AI Bot](docs/channels/wecom/wecom_aibot/README.zh.md) sobre detalles de cada procedimiento de tu elección.

**Inicio Rápido - WeCom Bot:**

**1. Crear a un bot**

* Dirígete e ingresa propiamente y ve a tu Consola Admin de WeCom → Chat en Grupo (Group Chat) → Añadir al Bot en Conjunto (Add Group Bot)
* Selecciona la URL de tipo formato del webhoook y pégala (eje: `https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx`)

**2. Configuración**

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

> El webhook de WeCom se encuentra sirviéndose ahora al estar inter-conectado con la aplicación por defecto y su Gateway (`gateway.host`:`gateway.port`, enlace original en `127.0.0.1:18790`).

**Montaje Rápido o Startup - WeCom App:**

**1. Crea una app**

* Dirígete y abre la Consola de Administración WeCom → Funciones de la App u Opciones(App Management) → Crea la App
* Consigue la llave encriptada de **AgentId** y el propio **Secret**
* Transfiriéndite hacia tu sección o portal "Mi empresa" (My Company page), resalta y copia todo contenido del  **CorpID**

**2. Configuraciones en torno a Mensajes Entrantes**

* Dentro mismo desde sus App detail Settings o "App details", clickea el cuadro de recibir mensaje "Receive Message" → "Set API"
* Localiza y ajusta ahora allí la URL del API por la vía y puerto de la siguiente sintaxis `http://your-server:18790/webhook/wecom-app`
* Desencadena y crea un enlace en forma de **Token** y **EncodingAESKey**

**3. Configuración**

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

**4. Ejecútalo**

```bash
picoclaw gateway
```

> **Nota**: El sistema original de respuesta llamado WeCom callback usa el puerto de salida base y de puente conocido como Gateway port (default 18790). Por razones lógicas usa tú a su vez y de preferencia reverse un protocolo de proxy en red para HTTPS.

**Montaje Rápido e instalación de Setup - WeCom AI Bot:**

**1. Crea un Bot (Específico al AI Bot)**

* Abre a través del navegador tu interfaz WeCom, loguéate del propio WeCom Admin Console → App Management → y busca opciones del AI Bot dentro.
* Directamente desde esas propias preferencias allí asignadas de esa propia IA "AI Bot settings", asienta su función en configuración para la variable de URL callback : `http://your-server:18791/webhook/wecom-aibot`
* Copia el **Token** originado y cliquea generar aleatorio de su llave ("Random Generate") referida y necesaria para el **EncodingAESKey**

**2. Configuración**

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

**3. Run**

```bash
picoclaw gateway
```

> **Note**: WeCom AI Bot uses streaming pull protocol — no reply timeout concerns. Long tasks (>30 seconds) automatically switch to `response_url` push delivery.

</details>

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> Únete a la Red Social de Agente (Agent Social Network)

Conecta Picoclaw a la red social global Agent Social Network solo mandando también un único mensaje desde la CLI principal o desde cualquier Chat App vinculada e integrada al mismo.

**Lee con atención `https://clawdchat.ai/skill.md` y guíate por la información mostrada de la página para unirte a [ClawdChat.ai](https://clawdchat.ai)**

## ⚙️ Ajustes y Configuración

Fichero index de configuración: `~/.picoclaw/config.json`

### Variables de Entorno y Variables Locales

A través de las variables de entorno puedes sobrescribir parámetros por defecto de los directorios preestablecidos. Resultando en algo altamente provechoso y aconsejable al desarrollar en puertos de instalaciones portátiles, despliegues docker encapsulados, para probar un picoclaw como un host temporal en formato de system service. Dichas dependientes controlarán direcciones de pathing (rutas estables de carpeta) por separado.

| Variable          | Descripción                                                                                                                             | Directorio Default              |
|-------------------|-----------------------------------------------------------------------------------------------------------------------------------------|---------------------------|
| `PICOCLAW_CONFIG` | Sobrescribe toda la ruta al file original `.json` de configuración inicial principal. Le permite a tu picoclaw saber por propia carga en `config.json` cuál parámetro montar y levantar primero e ignora y exime las otras dependencias del ecosistema general. | `~/.picoclaw/config.json` |
| `PICOCLAW_HOME`   | Interviene sobrescribiendo a su homólogo nativo y original de base de datos base (root) desde las particiones del picoclaw alojado ahí. Realiza también cambios alternos dentro del árbol de localización pre-estructurada por omisión del actual `workspace` entre más registros.          | `~/.picoclaw`             |

**Ejemplos:**

```bash
# Iniciar picoclaw usando un layout específico con configuración propia apartada
# La ruta general del folder y de workspace son procesadas y evaluadas luego de cargar lo estipulado para ese archivo config base
PICOCLAW_CONFIG=/etc/picoclaw/production.json picoclaw gateway

# Ejecuta el picoclaw pero reubicando donde residirán sus dependencias de todos tus datas e info a nivel local como en la base -> /opt/picoclaw
# Su índice y el Config en sí por pre-carga son reubicados con -> ~/.picoclaw/config.json
# Luego la terminal del Workspace te generará su resultado final en -> /opt/picoclaw/workspace
PICOCLAW_HOME=/opt/picoclaw picoclaw agent

# Usar ambos a la en conjunto a la par para realizar un Setup completamente estructurado  y customizado por el usuario a la medida
PICOCLAW_HOME=/srv/picoclaw PICOCLAW_CONFIG=/srv/picoclaw/main.json picoclaw gateway
```

### Esquema de Directorio y Workspace

PicoClaw mantiene los valores internos almacenados mediante y del modo configurado que hayas designado (el inicial es: `~/.picoclaw/workspace`):

```
~/.picoclaw/workspace/
├── sessions/          # Tus sesiones del diálogo individual o grupal conversado en el tiempo y su histórico
├── memory/           # Datos en variables de resguardo de a Memoria con largo término y contexto local (MEMORY.md)
├── state/            # El state actual de pre-uso interactivo guardado del usuario (del último chat, etc.)
├── cron/             # Trabajos pre-cargados a nivel de datos por su database en forma base y listado prehecho
├── skills/           # Todo el panel base por skills y facultades únicas personalizadas (skills)
├── AGENTS.md         # Fichero guía guía o parámetro a comportamientos o instrucciones dictaminadas a la IA
├── HEARTBEAT.md      # Parámetros por cronómetros para revisar (revisa cada 30 mn pre-asignado de fabrica)
├── IDENTITY.md       # Preceptos propios del AI / Base principal
├── SOUL.md           # Estructura referida al sistema abstracto al que reza del tipo soul y de cognición de rol general
└── USER.md           # Preferencias designadas e instruídas de un User particular del sistema (y al agente)
```

### Fuentes directas y Skills Sources Extras

De igual magnitud, aquí listamos dónde puedes encontrar o por origen dónde de forma de origen por omisión las "skills" principales radican (loads base/global):

1. `~/.picoclaw/workspace/skills` (workspace de base)
2. `~/.picoclaw/skills` (en un uso por red en global)
3. `<current-working-directory>/skills` (aquí encontrarás alojado a variables en código directas builtin o incorporadas nativamente pre compuestas)

Para pruebas y configuraciones más complejas (advanced testings y dev), estás en la capacidad de sobrescribir ese folder de builtin base root (el "root" o la raíz a tus skills). Intenta usando:

```bash
export PICOCLAW_BUILTIN_SKILLS=/path/to/skills
```

### Protocolo Sobre Políticas Unificadas en cuanto de Comandos al Ejecutivo

- Toda variante por los comandos más comunes (de "slash") se ejecutarán usando la base por paso sencillo (o path único) en `pkg/agent/loop.go` gracias a su enrutado manejado vía de las directrices `commands.Executor`.
- Ya ningún manejador ni adaptador intercede al querer usar dichos precomandos locales; ellos transfieren ese propio formato de ingreso texto nativo (inbound txt predefinido del user host) hacia otra de orden y guía bus/agent externa ya trazada de la propia y original vía de origen del agente. Al comienzo del "start up", Telegram continúa un auto pre-registro desde sus menús soportando estos iniciales formatos con y tras los encendidos del base a todo software.
- Los falsos, irreconocibles comandos slash o "ignorados de base a una lógica global" (te daremos ejemplos clave aquí de "uso": `/foo`) proseguirá ignorado al final pero filtrándose bajo pase de acceso con tu natural, regular forma usual normal como lo procesa LLM natural de procesado base en una plática y charla interactiva sin comando.
- Con los casos a comando (o un slash command como `/show` pero que pudiste haber intentado ejecutar o mandar en la interfaz de un WhatsApp para lo antes propuesto) sí estarán integrados, y aunque en verdad puedan pertenecer al listado que le sea lícito "usar a ti a modo universal" si estás de modo errante "prohibiéndolo usar" de un punto y a este punto de otra app paralela o distinta donde es negada total o parcialmente sí acabarán devueltos devolviéndote como lo es a su lado de origin de host una alerta o interrupción devolviéndolo explícito al usuario parando todas los tipos de procesos futuros a su cargo directos e indirectos generables o que le interceden subsecuentemente y/o posteriormente en lo pronto e inmediato paralizando todos por de su natural funcionamiento y labor a procesos por seguridad de inter interacciones u otros mandos de comandos no dados o previstos que intercede.
### 🔒 Entorno Sandbox (Caja de Arena Seguridad)

PicoClaw siempre de origen por su sistema nativo mantendrá el total estado a la ejecución bajo un estricto proceso de un tipo sandboxed virtual muy hermético, y únicamente bajo la condición segura a toda instancia. Como por normal función local o principal limitará tu agende interaccionando lográndo usarlo sólo para que a un modo estricto al acceder interviniera dentro y por exclusividad a sus comandos o funciones solo bajo y para a un workspace directo interno u asignado explícitamente y validado u autorizado explícitamente configurándose dentro por la partición base global u a todo este propio y a "TU workspace".

#### Configuración por Defecto

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

| Opciones                  | Por defecto                 | Descripción                               |
| ----------------------- | ----------------------- | ----------------------------------------- |
| `workspace`             | `~/.picoclaw/workspace` | La ruta activa general en la que correrá libre tu app desde el espacio reservado del agente interactivo al agente       |
| `restrict_to_workspace` | `true`                  | Condicional que delimita accesibilidad bajo a toda clase e instancia virtual y pre-carga referida a accesos de limitación bajo comandos, paths entre carpetas pre asignadas sólo en lo limitado restringido por defecto del propio o original *workspace* |

#### Herramientas Restringidas e Invocables Virtuales de Cautela de Privacidad del Usuario Final

Acerca e integrando a los estados referidos en el apartado o base de un y para  `restrict_to_workspace: true`, debes saber aquí acerca que un selectivo muy específico uso sobre tipos de llamadas listaremos como de estas formas operarán con "sandboxing":

| Herramientas (Tool)          | Para que sirve         | Estado de su Cautela o restricción (Restriction limits)                     |
| ------------- | ---------------- | -------------------------------------- |
| `read_file`   | Lee ficheros o cualquier archivo a modo de su contenido      | Lo restringe o acota "sólo" al de origen dentro a su red o del archivo que interviene propio y de el del propio "Workspace" asignado            |
| `write_file`  | Ejecuta y guarda en sobre y archivos escribiendo     | Intervenciones restrictivamente dentro por espacio delimitado a sus ficheros generados desde de ese  workspace.            |
| `list_dir`    | Carga listando a tu requerimiento sus locaciones directorios o sub     | Y la misma constante limitante exclusiva "Sólo para folders de subcarpeta/rutas interiores directas u explícitas creadas del ambiente al rededor virtual generalizado acoto a interioridad asignable a dicho *workspace*             |
| `edit_file`   | Puede o interviene con edición del archivo       | En ficheros o variables sólo en donde aplique a esa exclusiva caja sandbox delimitada en él "Workspace"         |
| `append_file` | Función anexante de añadir desde las terminaciones de archivos  | Funcional sólo interconectado a las ubicaciones u entornos acotando de los archivos del Workspace de orígen.         |
| `exec`        | Llamadas activadoras o ejecutora al usar y al querer usar e emitir un lineamiento por comando de tu sistema/shell | La instancia para poder lanzar todos/alguna forma en código o comandos tienen que hallarse dentro al entorno directo e interino confinado a el entorno o a la ruta y path para al propio o su "workspace" (de su origin local pre-fijado). |

#### Protecciones Adicionales sobre Ejecución

Aún y si `restrict_to_workspace: false` es configurado, la herramienta global de `exec` continuará evadiendo los siguientes códigos de riesgo comunes:

* `rm -rf`, `del /f`, `rmdir /s` — Eliminación brutal
* `format`, `mkfs`, `diskpart` — Formateados y manejo de disco
* `dd if=` — Modificaciones de imagen en disco
* Escrituras sin permiso hacia un `/dev/sd[a-z]` — Permisos directos o writes en disco root
* `shutdown`, `reboot`, `poweroff` — Opciones de Apagado de equipo y de sistemas
* Fork bomb en Bash `:(){ :|:& };:`

#### Ejemplos de Error

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (path outside working dir)}
```

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (dangerous pattern detected)}
```

#### Desactivar las Restricciones (Riesgo en Seguridad)

Si llegas a requerir que el Agente y sistema intercedan ingresando con permisos bajo paths desde fuera del área límite a sus variables configuradas para su trabajo directo por defecto.

**Método 1: Archivo Config File**

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

> ⚠️ **Aviso legal de precaución**: El anular de manera manual dicha condición a tu configuración expone al agente con libres autorizaciones de interactuar con el entorno sobre tu dispositivo entero. Utiliza precaución sobre su activación pre ponderando al ecosistema como uno con entorno altamente comprobado donde esto esté bien manejado y blindado y controlado a base.

#### Consistencia Continua y Confines de Seguridad perimetral

La configuración aplicada de la bandera en configuración `restrict_to_workspace` va ser propagada y pre aplciada de forma continua a lo largo por medio dentro a un todo los procesos de caminos y lazos del ambiente de ejecución:

| Ruta del Environment o Enrutado | Límites Seguros Internos (Boundary)          |
| ---------------- | ---------------------------- |
| Agente Base       | `restrict_to_workspace` ✅   |
| Modo Sub agente / Spawn | Estará obligado bajo heredadas dependencias similares de idéntica instancia y limitante anterior  ✅ |
| Lazo para una Tarea Heartbeat | Estará obligado bajo heredadas dependencias similares de idéntica instancia limitante  ✅ |

En todo modo u operaciones de vía y trazo de los enrutados y puestas las restricciones mantendrán una pre visualizable e inmutable aplicación a la variable límite inicial de tu configuración (workspace root). Sin manera o error bajo un camino paralelo evadida por permisos ignorados para ni las tasks referidas como temporizadas mediante heartbeat.

### Heartbeat (Tareas Periódicas y repetitivas)

PicoClaw puede realizar y configurar su ejecución natural temporal periódica o autómata general para rutinas sobre una tarea sin esfuerzo y de forma automágica. Ingresa al interior del ecosistema para tu agente interactivo desde el fichero para su entorno inicial llamado workspace y adjunta con nombre de archivo allí mismo un nuevo archivo de markdown: `HEARTBEAT.md`:

```markdown
# Tareas Periódicas Y Rutinarias Regulares

- Revisar mi bandeja de entradas (email) y filtrar las notificaciones de vital foco relevante e imperativa lectura
- Corresponde ahora para reportarme revisando el calendario pre configurado mío e inminentes reuniones planeadas proximamente (events)
- Avísame del clima
```

El asistente auto inspeccionará sus comandos por fichero en variables designados leyendo de paso siempre o verificando esto cíclicamente (30 mn configurables por omisión e inicio).

#### Variables tipo y Tareas de Asincronismo con Async usando Sub-agentes Spawn

Desde un enfoque global bajo parámetros de uso largo y duradero o que suelas programar tareas que mantengas procesos más exigencias demorados sobre en un tipo particular iterativo como las exploraciones y comandos web para "Web search", o para llamadas APIs lentas pre programadas "API calls", podrás y debes intentar hacer una delegación creando o requiriendo con invocar herramientas referidas mediante y de el `spawn` en modo para delegando a tareas para en general para **subagent**:

```markdown
# Labores Periódicas Y Regulares de Tiempo Autómata

## Funciones para uso Tareas express e rápidas

- Diciéndome que la respectiva hora del día es la actual de manera en el registro ahora

## Procederes y Labores Prolongadas tipo (Long Tasks con uso vía spawn como para async func a variables locales)

- Buscaré el modo dentro sobre el Internet local noticias exclusivas del tema actual y el mundo en tecnología del momento mediante y una y dando la IA general luego un extracto e pre resumen al terminar.
- Procede en modo rutinario e internamente de verificar la mensajería del chat y app para emails pre reportando solo aquellos clasificados imperativos al propio al user.
```

**Comportamientos a base Clave (Behaviors):**

| Funciones                 | Relativa y en Descripción                                               |
| ----------------------- | --------------------------------------------------------- |
| **spawn**               | Configura y arranca en instanciar el asíncrono agente sub-colegial interno el o a en modo por (async), garantizando que su origen por flujo o pipeline de base inicial referencial el local host no trabaje para bloquear su tarea en paralela.  |
| **Independent context** | Su hilo originado internamente cuenta de suyo solo su propio estado independiente para de la base original sus diálogos internos.          |
| **message tool**        | Desde un plano aparte puede también dicho propio sub agente mantener flujos e ingresos mediante reportes originados paralelos por herramientas `message` exclusivas con un aviso directo y simple al host. |
| **Non-blocking**        | Tan pronto luego es desplegado localmente este originado de manera delegada de tareas pasa al estado activo su proceso asincróno inicial y lo de las propias previas temporales funciones el latido original proseguiré su en local original curso al la que corresponda inmediatamente o después al comando de origen de Heartbeat.         |

#### Como La Plática del Propio Sub agente al host con el Interconectado Actúa u Opera Funcionalmente

```
Heartbeat triggers
    ↓
Agent reads HEARTBEAT.md
    ↓
For long task: spawn subagent
    ↓                           ↓
Continue to next task      Subagent works independently
    ↓                           ↓
All tasks done            Subagent uses "message" tool
    ↓                           ↓
Respond HEARTBEAT_OK      User receives result directly
```

Cualquier sub agente local tiene acceso habilitado y derecho al uso general interino de todo tipo de tools listadas y preexistentes a ese nodo (tools nativas como el uso de message o la opción de herramienta de rastreo general de internet web_search, así consecutivamente). E y comunicará los resultados independientemente mediante los diálogos, saltándose si es necesario los caminos centrales por del agente inicial u original.

**Configuración base:**

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

| Tipo de Parámetro o "Option"     | Valor Base Asignado | Que hace o su Descripción                         |
| ---------- | ------- | ---------------------------------- |
| `enabled`  | `true`  | Cede opciones lógicas como las booleanas y con ello puede Reactivar y/o Permitir e inhábilitarlo            |
| `interval` | `30`    | Variable con conteos del lapso natural pre programado con revisiones interinas del valor para "in minutes" en sí su mínimo: (`min: 5`) |

**Variable Entorno Local:**

* El uso configurado de `PICOCLAW_HEARTBEAT_ENABLED=false` logrará apagar el modo de rastreo por y de rutinas inmiscuidas "disable"
* A través a la variable constante `PICOCLAW_HEARTBEAT_INTERVAL=60` reconfiguramos los lapsos y minutos a el proceso iterativo deseado.

### Proveedores del Sistema de Modelos 

> [!NOTE]
> La organización a cargo de Groq actualmente dota y brinda provecho general gratuito a tu integración al usar transcripciones por grabados al enviar voz empleando con uso del módulo con API "Whisper". Si se habilita de la correcta manera en tu configuration, cualquier nota auditiva para cualquier canal configurado generará conversiones y logrando que sean auto-transcripciones automáticas ya leíbles del y al agente natural.

| Proveedores Principales de tu sistema y Agente                  | Y sus Propósitos (Purpose)                                 | Enlaces (Conseguir u obtener el API Key respectivo)                                                          |
| -------------------------- | --------------------------------------- | -------------------------------------------------------------------- |
| `gemini`                   | IA base / LLM (Gemini direct)                     | [aistudio.google.com](https://aistudio.google.com)                   |
| `zhipu`                    | LLM (Zhipu direct o base)                      | [bigmodel.cn](https://bigmodel.cn)                                   |
| `openrouter(Recomendado siempre e infalible, por probar en casos)` | LLM de la mejor elección (Por acceso a todos) | [openrouter.ai](https://openrouter.ai)                               |
| `anthropic(De prueba)`  | LLM de Anthropic (Claude directo)                     | [console.anthropic.com](https://console.anthropic.com)               |
| `openai(Puede ser por probar)`     | LLM (ChatGPT / GPT en directo nativo)                        | [platform.openai.com](https://platform.openai.com)                   |
| `deepseek(Prueba disponible pendiente)`   | LLM de base a (DeepSeek en un host de base propia local directo)                   | [platform.deepseek.com](https://platform.deepseek.com)               |
| `qwen`                     | LLM usado como el modelo inicial por default al pre cargar el local direct del Qwen en este repositorio local de Github (Qwen direct)                       | [dashscope.console.aliyun.com](https://dashscope.console.aliyun.com) |
| `groq`                     | LLM + **Herramienta especial de y para transcripción oral sobre la Voice** del susodicho servicio prestado e interactivo por de Whisper para grabarse vía de audios directos | [console.groq.com](https://console.groq.com)                         |
| `cerebras`                 | LLM (Enlace en versión tipo Cerebras base u nativo por host de tu directorio directo)                   | [cerebras.ai](https://cerebras.ai)                                   |
| `vivgrid`                  | LLM nativamente cargado directo (Del origin por origen natural a el Vivgrid directo localmente)                    | [vivgrid.com](https://vivgrid.com)                                   |

### Configuración Referente al Modelo Usado por Defecto: (model_list)

> **Lo nuevo implementado:** PicoClaw desde ahora se pre programará, servirá y correrá por base como un sistema con características desde su **model-centric** con gran particularidad nativa base hacia todo un acercamiento en general con un particular approach de modo generalizado y general hacia base desde un nodo centrado directo en su configuration inicial nativamente y única. Al designar o simplemente re escribiendo al invocar de la vía de `vendor/model` y tu formato referenciado con el prefijo tipo (e.g., y esto es ejemplo: el caso desde `zhipu/glm-4.7`) y así estar interactuando bajo un permiso con múltiples proveedores extra—**sin requerir editar líneas extrañadas nulas con código nuevo ni nada (todo zero code changes de antemano o required!)**

Esto posibilita crear opciones de **agentes cruzados para soporte universal multi-agente en uso preconfigurable e integrado nativo para el manejo con múltiples soportes en soporte desde varias interacciones simultáneamente multi AI** desde las facultativos variaciones base bajo de un método para selección versátil:

- **Múltiples inteligencias y diferentes combinables inteligencias u agentes, usando interactivos múltiples variables extra e distintos combinables pre modelos usados por vendors (providers)**: Se autorizará sin restricción a poder pre dotarle local e íntegramente de un servicio AI propio individual con el que le dictamines desde tu "own LLM endpoint" en cada variante AI u a tus host a elegir que tú así requieras integrador desde este modo.
- **Modelados fallback o recaída e resiliencia de la misma IA base:** Configura para mantener latentes pre cargados tus base principal de sistema artificial LLM pero también una variante resiliencia en un modelo a modo por un modelo recaída local un base LLM que se hará y le sirva desde respaldo local.
- **Ecualizador y nivelador integral de servidores:** Realiza distribuciones para tus procesos o sobre todas las tareas por vía del balance originado dividiendo request paralelamente sobre diferentes hosts interactuando múltiples nodos a nivel y servidores endpoint directos y bases conjuntas. 
- **Entorno en la misma Central:** Podrás siempre estar facultado sobre manejarlo con centralizaciones y gestionarlos mediante las "configuration profiles" sin problemas en todo a un único centro neurálgico a manera concentrada en un general sitio sin divisiones ("one place").

#### 📋 All Supported Vendors

#### 📋 Todos y Varios Exclusivos o Integrados Múltiples de Base Pre Configurados Tipos Soportes/Vendors Soportados por Completo a la de hoy día

| Soporte Principal "Vendor"              | Uso o Llamado a tu `model` Vía prefijada en (Prefix)    | Valor del Sistema API por "Default" (API Base)                                     | Lenguaje y modo o Familia/Protocolo del "Protocol"   | Adquirir su o un Enlace LLM de las Llaves para su Base (API Key)                                                          |
| ------------------- | ----------------- |-----------------------------------------------------| --------- | ---------------------------------------------------------------- |
| **OpenAI**          | `openai/`         | `https://api.openai.com/v1`                         | OpenAI    | [Consigue su llave desde aquí "Get Key"](https://platform.openai.com)                           |
| **Anthropic**       | `anthropic/`      | `https://api.anthropic.com/v1`                      | Anthropic | [Consígue Llave de Base](https://console.anthropic.com)                         |
| **智谱 AI (GLM)**   | `zhipu/`          | `https://open.bigmodel.cn/api/paas/v4`              | OpenAI    | [Adquiere La Propia Key](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) |
| **DeepSeek**        | `deepseek/`       | `https://api.deepseek.com/v1`                       | OpenAI    | [Su encriptado aquí](https://platform.deepseek.com)                         |
| **Google Gemini**   | `gemini/`         | `https://generativelanguage.googleapis.com/v1beta`  | OpenAI    | [Reciba tu propia LLM key Key](https://aistudio.google.com/api-keys)                  |
| **Groq**            | `groq/`           | `https://api.groq.com/openai/v1`                    | OpenAI    | [El portal web de keys y Tokens en "Key"](https://console.groq.com)                              |
| **Moonshot**        | `moonshot/`       | `https://api.moonshot.cn/v1`                        | OpenAI    | [Conseguir Aquí la "Key"](https://platform.moonshot.cn)                          |
| **通义千问 (Qwen)** | `qwen/`           | `https://dashscope.aliyuncs.com/compatible-mode/v1` | OpenAI    | [Abre el link Get Key](https://dashscope.console.aliyun.com)                  |
| **NVIDIA**          | `nvidia/`         | `https://integrate.api.nvidia.com/v1`               | OpenAI    | [Pide Y Genere Get Key](https://build.nvidia.com)                              |
| **Ollama**          | `ollama/`         | `http://localhost:11434/v1`                         | OpenAI    | No tienes por qué hacerlo Localmente ni pide llave (no key needed o none requerido)                                            |
| **OpenRouter**      | `openrouter/`     | `https://openrouter.ai/api/v1`                      | OpenAI    | [Entra Al Generador](https://openrouter.ai/keys)                            |
| **LiteLLM Proxy**   | `litellm/`        | `http://localhost:4000/v1`                          | OpenAI    | Para Esto Vas Necesitar tu Propia proxy interna "proxy key de LiteLLM"                                            |
| **VLLM**            | `vllm/`           | `http://localhost:8000/v1`                          | OpenAI    | Local local y ya sin keys y demás (Local)                                                            |
| **Cerebras**        | `cerebras/`       | `https://api.cerebras.ai/v1`                        | OpenAI    | [Usa Cerebras keys URL site web de base origen](https://cerebras.ai)                                   |
| **VolcEngine (Doubao)** | `volcengine/`     | `https://ark.cn-beijing.volces.com/api/v3`          | OpenAI    | [Pide el permiso Token API y API origin del KEY o token key "Key" ](https://console.volcengine.com)                        |
| **神算云**          | `shengsuanyun/`   | `https://router.shengsuanyun.com/api/v1`            | OpenAI    | -                                                                |
| **BytePlus**        | `byteplus/`       | `https://ark.ap-southeast.bytepluses.com/api/v3`    | OpenAI    | [Copia la Api-key personal desde la origin web oficial site URL en -> Base Keys en "Get Key"](https://console.volcengine.com)                        |
| **Vivgrid**         | `vivgrid/`        | `https://api.vivgrid.com/v1`                        | OpenAI    | [El Portal A Su App Console en Origin: API Key base En "API Key link Get"](https://vivgrid.com)                                   |
| **LongCat**         | `longcat/`        | `https://api.longcat.chat/openai`                   | OpenAI    | [Copia Su Respective API KEY Get en la Official URL link "Key"](https://longcat.chat/platform)                         |
| **Antigravity**     | `antigravity/`    | Para Y Del Origin o de Cloud Platform del Entorno Original Base Y Su Origin Host Cloud o de Google Cloud                                        | Uno Personalizado O Vía "Custom" y customizado y "Personal Custom Protocol" e "Internamente Nativo"   | Este Y Única e Exclusiva Usa Por Única Exclusividad la variante "OAuth only"                                                       |
| **GitHub Copilot**  | `github-copilot/` | `localhost:4321`                                    | El Protocolo Vía Y Base O En "gRPC"      | -                                                                |

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

#### Ejemplos Específicos según tu Vendor

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

**Anthropic (Usando clave de API key)**

```json
{
  "model_name": "claude-sonnet-4.6",
  "model": "anthropic/claude-sonnet-4.6",
  "api_key": "sk-ant-your-key"
}
```

> Ejecuta en ventana de comandos tu `picoclaw auth login --provider anthropic` que hará un paste seguro de tu propio API token en él.

**Ollama (En modo local)**

```json
{
  "model_name": "llama3",
  "model": "ollama/llama3"
}
```

**Formato de Custom Proxy/API Personalizada Proxy**

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

PicoClaw en modo de pre-fijo solo se libra y purga así mismo limpiando desde el sufijo exterior nativo base en sí `litellm/` de la línea justo previo al ser enviado tu base tipo request base original, haciendo posible que tu proxy trabaje y enviando desde `litellm/lite-gpt4`  pueda y lance un  `lite-gpt4` en sí mismo solo para ser de igual modo su equivalente y par para un tipo en `litellm/openai/gpt-4o` enviando puro origin a solo `openai/gpt-4o`.

#### Nivelador Y Distinguidor Con Load Balancing

Pre configure los puntos extremos o enlaces múltiples (endpoints) dedicados para el mismo modelo e idéntico model name—PicoClaw pre asignará a la app automáticamente poder ir interactuando internamente de un lado y entre éstos de round-robin:

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

#### Migración del Legacy pre origin config para `providers`

Esta antigua variante `providers` configuración a la iteración general  ha sido ya **depreciada (deprecated)** nativamente mas sin embargo esta permanece siendo apta bajo mantenimientos por una mera funcionalidad general a la compatibilidad con apps retrocesivas  y backward compatibility.

**Configuración Antigua Obsoleta (deprecated):**

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

**Configuración Nueva Actual (recomendada):**

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

Para mayor guianza y guías actualizadas ver en detalle la guía listada (migration guide), entra a este link [docs/migration/model-list-migration.md](docs/migration/model-list-migration.md).

### Arquitectura Del Proveedor Y Sus Conexiones

PicoClaw traza (routes) enlaces de enrutado vía un formato preestablecido categorizando proveedores agrupados o (protocol family):

- Compatible con la variante o protocolo OpenAI (OpenAI-compatible protocol): OpenRouter, los endpoints para variantes tipo compatibles pasarelas o OpenAI-compatible gateways, tales como Groq, Zhipu, e los puertos u endpoints con estilo e iniciales nativas y vLLM-style.
- Protocolos de base de Anthropic (Anthropic protocol): El propio comportamiento e instancias base de Claude.
- Codex Auth u autorización mediante su ruta y base Token (OAuth path): Formatos del Token OpenAI para accesibilidad libre, o la ruta principal del API en validaciones e autentificaciones (authentication route).

Esta opción retiene tu base de red (runtime) súper ágil y liviana pero al propio rato genera unas condiciones óptimas pre adaptadas base listas para ser compatibles con integraciones de cualquier nodo actual e incorporaciones de OpenAI (OpenAI-compatible) ya instalados y actualizados mediante tan solo en general en sí una configuración sola a base u "config operation" pre asignando sus propios (`api_base` + `api_key`).

<details>
<summary><b>Zhipu</b></summary>

**1. Generar la Llave de Claves de la Original API y su URL (Get API key and base URL)**

* Genera la base [Key URL o clave origen del API base de su servicio desde (API key)](https://bigmodel.cn/usercenter/proj-mgmt/apikeys)

**2. Configuración**

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
      "api_key": "Your API Key",
      "api_base": "https://open.bigmodel.cn/api/paas/v4"
    }
  }
}
```

**3. Ejecútalo**

```bash
picoclaw agent -m "Hola"
```

</details>

<details>
<summary><b>Configuración con Template y Ejemplo Completo (Full config example)</b></summary>

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

## Referencias de Uso sobre CLI (CLI Reference)

| Comandos Ejecutables          | Descripción Base                   |
| ------------------------- | ----------------------------- |
| `picoclaw onboard`        | Iniciador para preparar tu config & tu entorno propio workspace |
| `picoclaw agent -m "..."` | Usa a modo de Chat de diálogo a tu agente           |
| `picoclaw agent`          | Interactivo Modo en terminal "Interactive chat mode"         |
| `picoclaw gateway`        | Prende en modo Inicial y empieza a servir por base "gateway"             |
| `picoclaw status`         | Muestra estado operativo del sistema y su agente (Show status)                   |
| `picoclaw cron list`      | Lista todas aquellas y en conjunto tareas previas de tu listado (scheduled jobs)       |
| `picoclaw cron add ...`   | Anexa uno de manera nueva entre una labor al agent programada           |

### Rutinas, Alertas, Records, Tareas Programadas y Temporizadas (Scheduled Tasks / Reminders)

PicoClaw de frente de origen trae incorporado su soporte particular y en natural y rutinas al recordatorio agendadas desde la instancia por función o herramienta del tipo `cron`:

* **Un simple y único recordatorio y ya (One-time)**: "Dime en 10 minutos..." → se activa su función como un gatillo pre pasados esos referidos al momento en cuestión (10min o lo elegido)
* **Tareas a bases recurrente en repetición sucesiva (Recurring)**: "Recuérdame cada 2 horas..." → Esto se acciona como un disparador natural y latente que cada dichas 2 horas te lo enviará
* **Expresiones del tipo Cron de alto grado paramétrico (Cron Exp)**: "Alértame a las 9 am a todos mis amaneceres" → y lo enlista desde o tomando tus valores propios con ayuda del motor cron para uso a expresiones cron origin base.

Las mismas programadas acciones serán en listado o fichero guardadas dentro de su y en tu original propio `~/.picoclaw/workspace/cron/` del base folder para iterar a ser evaluadas posteriormente de auto format en manera autónoma.

## 🤝 Contribuciones del Open Source & Enfoque sobre la Hoja de Ruta (Roadmap y Contribute)

Tus pull requests (PRs) se animan y ¡son súper y más que bienvenidos y agradecidos! La composición base a esta lógica estructural del tipo de su repositorio (codebase) está predispuesta natural en diseño premeditada de en origen para hacerse ser uno pequeño intencional de lo más limpio que la normalidad para que ser entendible con lectura y sumamente amigable e interpretable comprensible y simple de la forma a toda manera posible general. 🤗

Mira con nosotros y nuestra lista completa aquí trazada bajo las metas y miras de en [La comunidad y Lo que su Hoja de Ruta o su Roadmap en la misma mira](https://github.com/sipeed/picoclaw/blob/main/ROADMAP.md).

El preámbulo a pertenecer ya del grupo base de Devs o sea Developer group en base a conformarse por los creadores: Únase ya por sí te adhieres logrando hacer de por forma exitoso primer Merge base subiendo la versión final el paso su propio primer (merged PR)!

Grupo Origin Chat para o a los de comunidad de tipo Users de usuarios (Users Groups):

En base y para Discord lo hay aquí: <https://discord.gg/V4sAZ9XWpN>

<img src="assets/wechat.png" alt="PicoClaw" width="512">

## 🐛 Problemas de Bug O de Falla General y de cómo se Resuelven (Troubleshooting)

### La función usada al querer pre revisar del buscar de la "Web" "web search error", dice referir sobre "Configuración en origin por API Key issue conflict o falta en sí para buscar de origen en base al configuration error base por no API (API key configuration issue)"

Señala un algo simple natural sin cuidado porque resulta muy normal a ti siempre en lo general (si lo ves), si esto no o es en sí la base en configuraciones hechas o simplemente porque nunca o ni porque siquiera haya pre cargado antes uno configurando una origin api del engine o base para Web. PicoClaw igual de forma útil por pre format sin intercesión nativa si le dejas, proporcionará en links enlaces de utilidad directa pre armada en vez en caso manual proporcionarle links "manual searching".

#### Niveles, Preceptos sobre Proveedores (Prioridad) Sobre Las Diferentes Búsquedas

Por defecto pre selecciona y acciona sin o con una sobre elección un sistema o listado con sus prioridades eligidas base este PicoClaw o prioriza o asienta una origin pre base buscando por su propia ruta (best available search provider algorithm routing format in this order origin):
1. **Perplexity** (Siempre a condición natural de ser con un uso general de 'enabled' o habilitadas si lo de pre cargaste el "API key pre en base configurado" y originado - de base provisto e Inteligente IA que da los rastros indexados "citations citations base")
2. **Brave Search** (Siempre y a la a condición igual estar lo 'enabled' de base al API pre configurado base original "key configured") - Uno respetando del usuario origin a origin privacidad original y pago un tanto base ("Privacy-focused paid API en $5/1000 origin pre base queries")
3. **SearXNG** (Estando también 'enabled' su habilitación y origin o URL en `base_url` estando también o si "configured") - El servicio general alojado hosteado mediante una forma original a tu o y/o en todo o local "metasearch aggregating 70+ base en pre uso u pre engine list" (es gratis).
4. **DuckDuckGo** (Habilitación base e "enabled, de forma nativa por en caso y fallback total por defecto default fall fallback route pre origin default a origen al fallo general) - Por si se lo hace u se usa no o necesita requerir ser provisto al base API Key a ninguna key origin o token de nada ni (No origin al API origin key ni base requerimiento al required list" gratis también).

#### Algunas de Diversa Variedad por Múltiples Configuraciones Opciones sobre Tus "Buscadores u Exploración a nivel en y del uso general por web (Web Search base en options config Configuration o Settings en configuraciones u base configuration Options en web Options Configuration Option en Configuration) "

**Opción Inicial Variante 1 de opción prioritaria 1 (Base Result General Mejorada Pre opción "Best Results" general en el top general y origin top)**: IA basada por el potente Perplexity AI Search motor o Engine
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

**Variante segunda U la Opción origin en base 2 Variante (Su de base Paid Base a Paid API uso origin a API "Paid Origin" o Paid Origin a la en Paid uso API de Paid pre origin API Origin para Paid de Opciones a Paid Para Paid API Usage a API en API Paid en base origen o modelo Pago API Origin API Base Paid API base API origin de la Opciones Paid u Opciones a modo Paid Use de Pago o "Paid u Paid pre origin API origin u uso o uso origen de u Base uso de Paid base Paid a Uso Opciones Usage Opciones Usage en Uso "Paid base Use de Opciones API o Pago Paid Usage de API a API en Paid Usage Opciones de Paid base Paid API base a Uso Opciones a Pago Paid")**: Genera u obtén de forma desde u en tu Origin key en forma Origin (en [https://brave.com/search/api](https://brave.com/search/api)) un Key o de llave en de base (origin token key a API key token) de origin ("Get an API a Base a Key API en Origin: $5 en / 1000 de Base a queries pre en "queries $5 a / 1000 base de las queries/mes $5 base $6 mes ~5$ dólares a o hasta 6 / mensuales" al /~ $5-6 de o /$ base a origin de "month base mensuales" de month / $5-mes a los mes /mes $ mensuales month a / mensuales / y de mes -/ de month/ mensuales month")
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

**Alternativa Para Variación Para la Base u y Opción o en su Variante Tres en base o 3 a la 3 o Variante Opciones de Base o Uso Variante y 3 u (Pre Instalada u Base o Y En Tu Entorno Origin Self base para su Opción Self en Opciones y de u Host de la misma Instalado "Self-Hosted base de Hosted Instalado u Self o Host "Opciones Self en y origin uso a Uso origin o Opciones Hosted Instalada Origin o Hosted Instalada Opciones a Hosted Opciones Opciones u de la Host origin u en self alojado mismo host" u Opciones Opciones Hosted ")**: Puedes Desplegar tú también tú solo tú a base o pre base un pre host de Opciones base o de propio (su origin Opciones Origin) su uso propio uso el instanciador ("instance Opciones de [SearXNG](https://github.com/searxng/searxng)" Opciones)
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

Beneficios de usar SearXNG:
- **Cero costo**: Sin tarifas de API ni límites de tasa de peticiones.
- **Centrado en la privacidad**: Alojado por ti mismo, sin rastreos.
- **Varios resultados**: Consulta en más de 70 motores de búsqueda simultáneamente.
- **Perfecto para máquinas virtuales (VM) en la nube**: Soluciona problemas de bloqueo de IP en centros de datos (Oracle Cloud, GCP, AWS, Azure).
- **No requiere API key**: Solamente despliega y configura la URL base.

**Opción 4 (No Requiere Configuración)**: DuckDuckGo está habilitado por defecto como opción de respaldo (no necesita API key).

Agrega la clave a `~/.picoclaw/config.json` si usas Brave:

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

### Obtengo errores de filtrado de contenido

Algunos proveedores (como Zhipu) tienen filtros de contenido. Intenta reformular tu consulta o usar un modelo diferente.

### El bot de Telegram dice "Conflict: terminated by other getUpdates"

Esto ocurre cuando hay otra instancia del bot en ejecución. Asegúrate de tener un solo `picoclaw gateway` corriendo a la vez.

---

## 📝 Comparación de Apis (API Key)

| Servicio         | Nivel Gratuito           | Caso de Uso                           |
| ---------------- | ------------------------ | ------------------------------------- |
| **OpenRouter**   | 200K tokens/mes          | Múltiples modelos (Claude, GPT-4, etc.) |
| **Volcengine CodingPlan** | ¥9.9/primer mes | Ideal para usuarios chinos, múltiples modelos SOTA (Doubao, DeepSeek, etc.) |
| **Zhipu**        | 200K tokens/mes          | Adecuado para usuarios chinos         |
| **Brave Search** | De Pago ($5/1000 búsquedas) | Funcionalidad de búsqueda en la red   |
| **SearXNG**      | Ilimitado (Autoalojado)  | Metabuscador centrado en la privacidad (70+ motores) |
| **Groq**         | Nivel gratuito disponible| Inferencia súper rápida (Llama, Mixtral) |
| **Cerebras**     | Nivel gratuito disponible| Inferencia súper rápida (Llama, Qwen, etc.) |
| **LongCat**      | Hasta 5M tokens/día      | Inferencia rápida (nivel gratuito)    |

---

<div align="center">
  <img src="assets/logo.jpg" alt="PicoClaw Meme" width="512">
</div>
