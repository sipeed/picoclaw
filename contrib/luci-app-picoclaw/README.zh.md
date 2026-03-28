<p align="center">
  <img src="Screenshots/logo.svg" alt="PicoClaw LuCI" width="120">
</p>

<h1 align="center">PicoClaw LuCI — OpenWrt 网页管理界面</h1>

<p align="center">
  <b>为 <a href="https://github.com/sipeed/picoclaw">PicoClaw</a> 量身打造的 OpenWrt LuCI 美化管理面板</b>
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

## 📸 截图预览

| 桌面端 | 移动端 |
|:---:|:---:|
| ![主界面](Screenshots/screenshot_main.png) | ![手机端](Screenshots/screenshot_mobile.png) |
| ![配置编辑](Screenshots/screenshot_config.png) | — |

## ✨ 功能特性

- 🖥️ **现代化仪表盘** — 实时显示服务状态、PID、内存占用、端口监听
- 🚀 **服务控制** — 一键启动 / 停止 / 重启 PicoClaw
- ⚡ **开机自启开关** — 在网页上直接控制开机自动启动
- 📡 **通道管理** — 查看和管理所有通道（飞书、Telegram、Discord、微信、企业微信、Slack、QQ、LINE、钉钉、WhatsApp、MaixCam）
- 🔧 **表单化配置编辑** — 直观的表单界面配置 AI 模型、API 密钥、系统设置和工具
- 📝 **JSON 配置编辑** — 支持 JSON 直接编辑、格式化和验证
- 🌍 **5 种语言** — 内置国际化：简体中文、English、日本語、Português、Deutsch
- 🔄 **在线更新** — 检查新版本并直接在界面上更新
- 📊 **系统日志** — 实时查看 PicoClaw 系统日志
- 💬 **微信接入引导** — 微信个人号接入步骤引导
- 🔑 **多 AI 供应商支持** — 配置智谱、OpenAI、ChatGPT、Claude、DeepSeek、Anthropic、Ollama、Azure OpenAI
- 🎨 **响应式设计** — 完美适配桌面和手机浏览器
- 📦 **一键安装** — Python 安装脚本，通过 SSH 自动部署到任意 OpenWrt 设备

## 📋 系统要求

| 要求 | 说明 |
|---|---|
| **OpenWrt** | 24.10 / 25.xx（需安装 LuCI） |
| **架构** | ARM64、AMD64 (x86_64)、ARMv7 |
| **LuCI** | 标准 LuCI 安装即可 |
| **SSH** | 安装脚本需要路由器开启 SSH |
| **Python** | 3.6+（在电脑上运行安装脚本） |
| **PicoClaw** | [sipeed/picoclaw](https://github.com/sipeed/picoclaw) 最新版 |

## 🚀 安装方式

### 方式一：一键 Python 安装（推荐）

在你的电脑上运行，脚本会通过 SSH 自动完成所有操作。

```bash
# 安装依赖
pip install paramiko

# 下载安装脚本
wget https://github.com/GennKann/luci-app-picoclaw/releases/latest/download/install_picoclaw_luci.py

# 运行 — 按提示输入路由器信息
python install_picoclaw_luci.py
```

安装脚本会自动完成：
1. ✅ 检测系统架构（ARM64 / AMD64 / ARMv7）
2. ✅ 下载并安装最新版 PicoClaw
3. ✅ 创建 procd init.d 开机自启服务
4. ✅ 部署 LuCI 网页管理界面
5. ✅ 初始化 PicoClaw 配置
6. ✅ 启动服务并验证

### 方式二：手动安装

SSH 登录你的 OpenWrt 路由器：

```bash
# 1. 下载并安装 PicoClaw
ARCH=$(uname -m)
if echo "$ARCH" | grep -q "x86"; then PLAT="linux_amd64"; 
elif echo "$ARCH" | grep -q "armv7"; then PLAT="linux_armv7"; 
else PLAT="linux_arm64"; fi

wget -O /tmp/picoclaw "https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_${PLAT}"
chmod +x /tmp/picoclaw && mv /tmp/picoclaw /usr/bin/picoclaw

# 2. 部署 LuCI 文件（从本仓库复制）
# controller → /usr/lib/lua/luci/controller/picoclaw.lua
# template → /usr/lib/lua/luci/view/picoclaw/main.htm

# 3. 设置 init.d 服务
cp scripts/picoclaw.init /etc/init.d/picoclaw
chmod +x /etc/init.d/picoclaw && /etc/init.d/picoclaw enable

# 4. 清理缓存并启动
rm -rf /tmp/luci-* && /etc/init.d/picoclaw start
```

### 方式三：包管理器安装

> **注意：** OpenWrt 24.10 使用 `.ipk`（opkg 包管理器），OpenWrt 25.xx 使用 `.apk`（apk-tools 包管理器），两者是**不同的**包格式。

#### OpenWrt 24.10（opkg / .ipk）

```bash
opkg install luci-compat
wget -O /tmp/luci-app-picoclaw.ipk https://github.com/GennKann/luci-app-picoclaw/releases/latest/download/luci-app-picoclaw_24.10_all.ipk
opkg install /tmp/luci-app-picoclaw.ipk
```

#### OpenWrt 25.xx（apk / .apk）

OpenWrt 25.x 已将包管理器从 `opkg` 切换为 `apk-tools`，包格式从 `.ipk` 变为 `.apk`。需要在路由器上直接构建：

```bash
# 下载构建脚本
wget -O /tmp/build-apk-25xx.sh https://raw.githubusercontent.com/GennKann/luci-app-picoclaw/main/scripts/build-apk-25xx.sh
chmod +x /tmp/build-apk-25xx.sh

# 运行构建 .apk 包
/tmp/build-apk-25xx.sh

# 安装生成的包
apk add --allow-untrusted /root/luci-app-picoclaw_1.0.0_all.apk
```

> **为什么需要在路由器上构建？** OpenWrt 25.xx 使用 APKv3（Alpine Package Keeper）格式，需要 `apk-tools` 来正确构建。在路由器上构建可确保完全兼容。

## 🎯 访问地址

安装完成后，通过以下地址访问管理界面：

```
http://<路由器IP>/cgi-bin/luci/admin/services/picoclaw
```

例如：`http://192.168.1.1/cgi-bin/luci/admin/services/picoclaw`

## ⚠️ 免责声明

本项目是社区驱动的 PicoClaw LuCI 管理界面，**不是** PicoClaw 官方项目的一部分。

- **PicoClaw** 由 [Sipeed](https://github.com/sipeed) 使用 MIT 许可证开发
- 本 LuCI 界面是独立的开源工具
- 我们尊重原始项目的所有知识产权
- "PicoClaw" 和 "Sipeed" 是其各自所有者的商标

## 📄 许可证

本项目使用 [MIT 许可证](LICENSE)。

## 🙏 鸣谢

- [PicoClaw](https://github.com/sipeed/picoclaw) by Sipeed — 本项目所管理的出色 AI 助手
- [OpenWrt](https://openwrt.org/) — 运行基础
- [LuCI](https://github.com/openwrt/luci) — Web 界面框架

## 🤝 参与贡献

欢迎贡献！请随时提交 Pull Request。

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送分支 (`git push origin feature/amazing-feature`)
5. 发起 Pull Request

---

<p align="center">
  如果这个项目对你有帮助，请给一个 ⭐ 支持一下！
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
  <b>❤️ 由 <a href="https://github.com/GennKann">GennKann</a> 用心制作</b>
</p>
