<p align="center">
  <img src="Screenshots/logo.svg" alt="PicoClaw LuCI" width="120">
</p>

<h1 align="center">PicoClaw LuCI — Gerenciador Web OpenWrt</h1>

<p align="center">
  <b>Uma interface web LuCI bonita para gerenciar o <a href="https://github.com/sipeed/picoclaw">PicoClaw</a> em roteadores OpenWrt</b>
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

## 📸 Capturas de Tela

| Desktop | Mobile |
|:---:|:---:|
| ![Painel](Screenshots/screenshot_main.png) | ![Mobile](Screenshots/screenshot_mobile.png) |
| ![Configuração](Screenshots/screenshot_config.png) | — |

## ✨ Funcionalidades

- 🖥️ **Painel Moderno** — Status do serviço em tempo real, PID, uso de memória e monitoramento de porta
- 🚀 **Controle de Serviço** — Iniciar / Parar / Reiniciar o PicoClaw com um clique
- ⚡ **Alternar Auto-início** — Ativar/desativar inicialização automática pela interface web
- 📡 **Gerenciamento de Canais** — Visualizar e gerenciar todos os canais (Feishu, Telegram, Discord, WeChat, WeCom, Slack, QQ, LINE, DingTalk, WhatsApp, MaixCam)
- 🔧 **Editor de Configuração** — Interface de formulário intuitiva para modelo AI, provedores, configurações do sistema e ferramentas
- 📝 **Editor JSON** — Edição direta JSON com formatação e validação
- 🌍 **5 Idiomas** — i18n integrado: English, 简体中文, 日本語, Português, Deutsch
- 🔄 **Atualização Online** — Verificar novas versões e atualizar diretamente pela interface
- 🔑 **Multi-provedor** — Chaves API para Zhipu, OpenAI, ChatGPT, Claude, DeepSeek, Anthropic, Ollama, Azure OpenAI

## 📋 Requisitos

| Requisito | Detalhes |
|---|---|
| **OpenWrt** | 24.10 / 25.xx (com LuCI) |
| **Arquitetura** | ARM64, AMD64, ARMv7 |
| **SSH** | Necessário para o script de instalação |
| **Python** | 3.6+ (no PC) |
| **PicoClaw** | [sipeed/picoclaw](https://github.com/sipeed/picoclaw) versão mais recente |

## 🚀 Instalação

### Método 1: Instalador Python (Recomendado)

```bash
pip install paramiko
wget https://github.com/GennKann/luci-app-picoclaw/releases/latest/download/install_picoclaw_luci.py
python install_picoclaw_luci.py
```

### Método 2: Instalação por Pacote

> **Nota:** OpenWrt 24.10 usa `.ipk` (opkg), OpenWrt 25.xx usa `.apk` (apk-tools). São **formatos diferentes** de pacotes.

#### OpenWrt 24.10（opkg / .ipk）

```bash
opkg install luci-compat
wget -O /tmp/luci-app-picoclaw.ipk https://github.com/GennKann/luci-app-picoclaw/releases/latest/download/luci-app-picoclaw_24.10_all.ipk
opkg install /tmp/luci-app-picoclaw.ipk
```

#### OpenWrt 25.xx（apk / .apk）

OpenWrt 25.x mudou o gerenciador de pacotes de `opkg` para `apk-tools`, e o formato de `.ipk` para `.apk`. Construa o pacote diretamente no roteador:

```bash
# Baixar script de compilação
wget -O /tmp/build-apk-25xx.sh https://raw.githubusercontent.com/GennKann/luci-app-picoclaw/main/scripts/build-apk-25xx.sh
chmod +x /tmp/build-apk-25xx.sh

# Compilar pacote .apk
/tmp/build-apk-25xx.sh

# Instalar pacote gerado
apk add --allow-untrusted /root/luci-app-picoclaw_1.0.0_all.apk
```

### Método 3: Instalação Manual

```bash
# Instalar PicoClaw
ARCH=$(uname -m)
if echo "$ARCH" | grep -q "x86"; then PLAT="linux_amd64";
elif echo "$ARCH" | grep -q "armv7"; then PLAT="linux_armv7";
else PLAT="linux_arm64"; fi
wget -O /tmp/picoclaw "https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_${PLAT}"
chmod +x /tmp/picoclaw && mv /tmp/picoclaw /usr/bin/picoclaw

# Configurar arquivos LuCI e serviço init.d
# (copiar arquivos deste repositório)
```

## 🎯 Acesso

```
http://<IP_DO_ROTEADOR>/cgi-bin/luci/admin/services/picoclaw
```

## ⚠️ Aviso Legal

Este projeto é uma interface LuCI de gerenciamento do PicoClaw desenvolvida pela comunidade e **NÃO** faz parte do projeto oficial PicoClaw.

- **PicoClaw** é desenvolvido pela [Sipeed](https://github.com/sipeed) sob a licença MIT
- Esta interface LuCI é uma ferramenta open-source independente
- "PicoClaw" e "Sipeed" são marcas registradas de seus respectivos proprietários

## 📄 Licença

[Licença MIT](LICENSE)

## 🙏 Créditos

- [PicoClaw](https://github.com/sipeed/picoclaw) by Sipeed — O excelente assistente de IA que este projeto gerencia
- [OpenWrt](https://openwrt.org/) — A base sobre a qual isso é executado
- [LuCI](https://github.com/openwrt/luci) — O framework de interface web

## 🤝 Contribuição

Contribuições são bem-vindas! Sinta-se à vontade para enviar um Pull Request.

1. Faça um Fork do repositório
2. Crie um branch de funcionalidade (`git checkout -b feature/amazing-feature`)
3. Faça commit das alterações (`git commit -m 'Add amazing feature'`)
4. Faça push do branch (`git push origin feature/amazing-feature`)
5. Abra um Pull Request

---

<p align="center">
  Se você achar este projeto útil, por favor, dê uma ⭐!
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
