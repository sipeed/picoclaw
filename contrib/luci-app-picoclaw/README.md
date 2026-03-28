# LuCI Management Interface for PicoClaw on OpenWrt

A beautiful LuCI web interface to manage PicoClaw on OpenWrt routers.

## Features

- **Modern Dashboard** — Real-time service status, PID, memory usage, and port monitoring
- **Service Control** — Start / Stop / Restart PicoClaw with one click
- **Auto-Start Toggle** — Enable or disable boot-on-start from the web UI
- **Channel Management** — View and manage all connected channels
- **Form-based Config Editor** — Intuitive form UI for AI model, providers, system settings, and tools configuration
- **JSON Config Editor** — Direct JSON editing with format and validation
- **5 Languages** — Built-in i18n: English, 简体中文, 日本語, Português, Deutsch
- **Online Update** — Check for new versions and update directly from the UI
- **System Logs** — View PicoClaw system logs in real-time
- **Responsive Design** — Works on both desktop and mobile browsers

## Installation

### One-Click Python Installer (Recommended)

```bash
pip install paramiko
python contrib/luci-app-picoclaw/install.py
```

### Manual Installation

```bash
# Copy LuCI controller
cp contrib/luci-app-picoclaw/luci/controller/picoclaw.lua /usr/lib/lua/luci/controller/

# Copy LuCI template
mkdir -p /usr/lib/lua/luci/view/picoclaw
cp contrib/luci-app-picoclaw/luci/view/picoclaw/main.htm /usr/lib/lua/luci/view/picoclaw/

# Set up init.d service
cp contrib/luci-app-picoclaw/scripts/picoclaw.init /etc/init.d/picoclaw
chmod +x /etc/init.d/picoclaw
/etc/init.d/picoclaw enable
```

## Requirements

- OpenWrt 24.10 / 25.xx with LuCI installed
- ARM64 / AMD64 / ARMv7
- PicoClaw latest version

## Screenshots

| Desktop | Mobile |
|:---:|:---:|
| ![Dashboard](Screenshots/screenshot_main.png) | ![Mobile](Screenshots/screenshot_mobile.png) |
| ![Config](Screenshots/screenshot_config.png) | — |

## Access

```
http://<ROUTER_IP>/cgi-bin/luci/admin/services/picoclaw
```

## License

MIT License — This is an independently created LuCI management interface.
It communicates with PicoClaw via its standard CLI/API interface only.

## Links

- Standalone repo: [GennKann/luci-app-picoclaw](https://github.com/GennKann/luci-app-picoclaw)
