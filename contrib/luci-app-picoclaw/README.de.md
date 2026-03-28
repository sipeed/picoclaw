<p align="center">
  <img src="Screenshots/logo.svg" alt="PicoClaw LuCI" width="120">
</p>

<h1 align="center">PicoClaw LuCI — OpenWrt Webverwaltung</h1>

<p align="center">
  <b>Eine schöne LuCI-Weboberfläche zur Verwaltung von <a href="https://github.com/sipeed/picoclaw">PicoClaw</a> auf OpenWrt-Routern</b>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/OpenWrt-24.10%20%7C%2025.xx-blue?logo=openwrt" alt="OpenWrt">
  <img src="https://img.shields.io/badge/LuCI-Web%20Interface-green?logo=lua" alt="LuCI">
  <img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License">
  <img src="https://img.shields.io/badge/Arch-ARM64%20%7C%20AMD64%20%7C%20ARMv7-orange" alt="Architecture">
</p>

<p align="center">
  <a href="README.md">English</a> ·
  <a href="README.zh.md">简体中文</a> ·
  <a href="README.ja.md">日本語</a> ·
  <a href="README.de.md">Deutsch</a> ·
  <a href="README.pt.md">Português</a>
</p>

---

## 📸 Screenshots

| Desktop | Mobil |
|:---:|:---:|
| ![Dashboard](Screenshots/screenshot_main.png) | ![Mobil](Screenshots/screenshot_mobile.png) |
| ![Konfiguration](Screenshots/screenshot_config.png) | — |

## ✨ Funktionen

- 🖥️ **Modernes Dashboard** — Echtzeit-Service-Status, PID, Speichernutzung und Port-Überwachung
- 🚀 **Service-Steuerung** — PicoClaw mit einem Klick starten / stoppen / neu starten
- ⚡ **Autostart-Umschalter** — Autostart direkt in der Weboberfläche aktivieren/deaktivieren
- 📡 **Kanalverwaltung** — Alle Kanäle anzeigen und verwalten (Feishu, Telegram, Discord, WeChat, WeCom, Slack, QQ, LINE, DingTalk, WhatsApp, MaixCam)
- 🔧 **Formular-Konfiguration** — Intuitive Formular-Benutzeroberfläche für AI-Modell, Anbieter, Systemeinstellungen und Tools
- 📝 **JSON-Editor** — Direkte JSON-Bearbeitung mit Formatierung und Validierung
- 🌍 **5 Sprachen** — Eingebauter i18n: English, 简体中文, 日本語, Português, Deutsch
- 🔄 **Online-Update** — Neue Versionen prüfen und direkt aktualisieren
- 🔑 **Multi-Anbieter** — API-Schlüssel für Zhipu, OpenAI, ChatGPT, Claude, DeepSeek, Anthropic, Ollama, Azure OpenAI

## 📋 Voraussetzungen

| Anforderung | Details |
|---|---|
| **OpenWrt** | 24.10 / 25.xx (mit LuCI) |
| **Architektur** | ARM64, AMD64, ARMv7 |
| **SSH** | Für das Installationsskript erforderlich |
| **Python** | 3.6+ (auf dem PC) |
| **PicoClaw** | [sipeed/picoclaw](https://github.com/sipeed/picoclaw) neueste Version |

## 🚀 Installation

### Methode 1: Python-Installer (Empfohlen)

```bash
pip install paramiko
wget https://github.com/GennKann/luci-app-picoclaw/releases/latest/download/install_picoclaw_luci.py
python install_picoclaw_luci.py
```

### Methode 2: Paket-Installation

> **Hinweis:** OpenWrt 24.10 verwendet `.ipk` (opkg), OpenWrt 25.xx verwendet `.apk` (apk-tools). Dies sind **unterschiedliche** Paketformate.

#### OpenWrt 24.10（opkg / .ipk）

```bash
opkg install luci-compat
wget -O /tmp/luci-app-picoclaw.ipk https://github.com/GennKann/luci-app-picoclaw/releases/latest/download/luci-app-picoclaw_24.10_all.ipk
opkg install /tmp/luci-app-picoclaw.ipk
```

#### OpenWrt 25.xx（apk / .apk）

OpenWrt 25.x hat den Paketmanager von `opkg` auf `apk-tools` umgestellt. Das Format änderte sich von `.ipk` zu `.apk`. Bauen Sie das Paket direkt auf dem Router:

```bash
# Build-Skript herunterladen
wget -O /tmp/build-apk-25xx.sh https://raw.githubusercontent.com/GennKann/luci-app-picoclaw/main/scripts/build-apk-25xx.sh
chmod +x /tmp/build-apk-25xx.sh

# .apk-Paket bauen
/tmp/build-apk-25xx.sh

# Generiertes Paket installieren
apk add --allow-untrusted /root/luci-app-picoclaw_1.0.0_all.apk
```

### Methode 3: Manuelle Installation

```bash
# PicoClaw installieren
ARCH=$(uname -m)
if echo "$ARCH" | grep -q "x86"; then PLAT="linux_amd64";
elif echo "$ARCH" | grep -q "armv7"; then PLAT="linux_armv7";
else PLAT="linux_arm64"; fi
wget -O /tmp/picoclaw "https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_${PLAT}"
chmod +x /tmp/picoclaw && mv /tmp/picoclaw /usr/bin/picoclaw

# LuCI-Dateien und init.d Service einrichten
# (Dateien aus diesem Repository kopieren)
```

## 🎯 Zugriff

```
http://<ROUTER-IP>/cgi-bin/luci/admin/services/picoclaw
```

## ⚠️ Haftungsausschluss

Dieses Projekt ist eine Community-getriebene LuCI-Verwaltungsoberfläche für PicoClaw und ist **kein** Teil des offiziellen PicoClaw-Projekts.

- **PicoClaw** wird von [Sipeed](https://github.com/sipeed) unter der MIT-Lizenz entwickelt
- Diese LuCI-Oberfläche ist ein unabhängiges Open-Source-Tool
- „PicoClaw" und „Sipeed" sind Marken ihrer jeweiligen Eigentümer

## 📄 Lizenz

[MIT-Lizenz](LICENSE)

## 🙏 Danksagungen

- [PicoClaw](https://github.com/sipeed/picoclaw) by Sipeed — Der großartige KI-Assistent, den dieses Projekt verwaltet
- [OpenWrt](https://openwrt.org/) — Das Fundament, auf dem dies läuft
- [LuCI](https://github.com/openwrt/luci) — Das Web-Interface-Framework

## 🤝 Mitmachen

Beiträge sind willkommen! Bitte erstellen Sie einen Pull Request.

1. Forken Sie das Repository
2. Erstellen Sie einen Feature-Branch (`git checkout -b feature/amazing-feature`)
3. Committen Sie Ihre Änderungen (`git commit -m 'Add amazing feature'`)
4. Pushen Sie den Branch (`git push origin feature/amazing-feature`)
5. Erstellen Sie einen Pull Request

---

<p align="center">
  Wenn Sie dieses Projekt nützlich finden, geben Sie ihm bitte einen ⭐!
</p>

<p align="center">
  <a href="https://github.com/GennKann/luci-app-picoclaw">
    <img src="https://img.shields.io/github/stars/GennKann/luci-app-picoclaw?style=social" alt="Stars">
  </a>
  <a href="https://github.com/GennKann/luci-app-picoclaw/fork">
    <img src="https://img.shields.io/github/forks/GennKann/luci-app-picoclaw?style=social" alt="Forks">
  </a>
</p>

<p align="center">
  <b>Made with ❤️ by <a href="https://github.com/GennKann">GennKann</a></b>
</p>
