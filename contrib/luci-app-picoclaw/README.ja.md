<p align="center">
  <img src="Screenshots/logo.svg" alt="PicoClaw LuCI" width="120">
</p>

<h1 align="center">PicoClaw LuCI — OpenWrt Web管理インターフェース</h1>

<p align="center">
  <b><a href="https://github.com/sipeed/picoclaw">PicoClaw</a> 専用 OpenWrt LuCI 美しい管理パネル</b>
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

## 📸 スクリーンショット

| デスクトップ | モバイル |
|:---:|:---:|
| ![ダッシュボード](Screenshots/screenshot_main.png) | ![モバイル](Screenshots/screenshot_mobile.png) |
| ![設定エディタ](Screenshots/screenshot_config.png) | — |

## ✨ 機能

- 🖥️ **モダンダッシュボード** — サービス状態、PID、メモリ使用量、ポート監視をリアルタイム表示
- 🚀 **サービス制御** — PicoClawの開始/停止/再起動をワンクリック
- ⚡ **自動起動切替** — Web UIから起動時自動開始の有効/無効を切り替え
- 📡 **チャンネル管理** — 全チャンネルの表示・管理（Feishu、Telegram、Discord、WeChat、WeCom、Slack、QQ、LINE、DingTalk、WhatsApp、MaixCam）
- 🔧 **フォーム設定エディタ** — 直感的なフォームUIでAIモデル、プロバイダー、システム設定、ツールを設定
- 📝 **JSON設定エディタ** — JSON直接編集、フォーマット、バリデーション対応
- 🌍 **5言語対応** — 組み込みi18n：English、简体中文、日本語、Português、Deutsch
- 🔄 **オンライン更新** — 新バージョンの確認とUIからの直接更新
- 📊 **システムログ** — PicoClawシステムログをリアルタイム表示
- 🔑 **マルチプロバイダー対応** — Zhipu、OpenAI、ChatGPT、Claude、DeepSeek、Anthropic、Ollama、Azure OpenAIのAPIキー設定
- 🎨 **レスポンシブデザイン** — デスクトップとモバイルブラウザの両方に対応

## 📋 動作要件

| 要件 | 詳細 |
|---|---|
| **OpenWrt** | 24.10 / 25.xx（LuCI導入済み） |
| **アーキテクチャ** | ARM64、AMD64 (x86_64)、ARMv7 |
| **LuCI** | 標準LuCIインストール |
| **SSH** | インストーラースクリプトに必要 |
| **Python** | 3.6+（PC環境） |
| **PicoClaw** | [sipeed/picoclaw](https://github.com/sipeed/picoclaw) 最新版 |

## 🚀 インストール

### 方法1：ワンクリックPythonインストーラー（推奨）

PC上で実行 — SSH経由でルーターに自動インストールします。

```bash
pip install paramiko
wget https://github.com/GennKann/luci-app-picoclaw/releases/latest/download/install_picoclaw_luci.py
python install_picoclaw_luci.py
```

### 方法2：パッケージインストール

> **注意：** OpenWrt 24.10は`.ipk`（opkg）、OpenWrt 25.xxは`.apk`（apk-tools）を使用します。**別の**パッケージ形式です。

#### OpenWrt 24.10（opkg / .ipk）

```bash
opkg install luci-compat
wget -O /tmp/luci-app-picoclaw.ipk https://github.com/GennKann/luci-app-picoclaw/releases/latest/download/luci-app-picoclaw_24.10_all.ipk
opkg install /tmp/luci-app-picoclaw.ipk
```

#### OpenWrt 25.xx（apk / .apk）

OpenWrt 25.xではパッケージマネージャーが`opkg`から`apk-tools`に変更され、形式も`.ipk`から`.apk`になりました。ルーター上で直接ビルドします：

```bash
# ビルドスクリプトをダウンロード
wget -O /tmp/build-apk-25xx.sh https://raw.githubusercontent.com/GennKann/luci-app-picoclaw/main/scripts/build-apk-25xx.sh
chmod +x /tmp/build-apk-25xx.sh

# .apkパッケージをビルド
/tmp/build-apk-25xx.sh

# 生成されたパッケージをインストール
apk add --allow-untrusted /root/luci-app-picoclaw_1.0.0_all.apk
```

### 方法3：手動インストール

```bash
# PicoClawインストール
ARCH=$(uname -m)
if echo "$ARCH" | grep -q "x86"; then PLAT="linux_amd64"; 
elif echo "$ARCH" | grep -q "armv7"; then PLAT="linux_armv7"; 
else PLAT="linux_arm64"; fi
wget -O /tmp/picoclaw "https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_${PLAT}"
chmod +x /tmp/picoclaw && mv /tmp/picoclaw /usr/bin/picoclaw

# LuCIファイル配置
# controller → /usr/lib/lua/luci/controller/picoclaw.lua
# template → /usr/lib/lua/luci/view/picoclaw/main.htm

# init.dサービス設定
cp scripts/picoclaw.init /etc/init.d/picoclaw
chmod +x /etc/init.d/picoclaw && /etc/init.d/picoclaw enable
rm -rf /tmp/luci-* && /etc/init.d/picoclaw start
```

## 🎯 アクセス

```
http://<ルーターIP>/cgi-bin/luci/admin/services/picoclaw
```

## ⚠️ 免責事項

このプロジェクトはコミュニティ駆動のPicoClaw管理インターフェースであり、PicoClaw公式プロジェクトの一部ではありません。

- **PicoClaw**は[Sipeed](https://github.com/sipeed)によってMITライセンスで開発されています
- 本LuCIインターフェースは独立したオープンソースツールです
- "PicoClaw"および"Sipeed"は各所有者の商標です

## 📄 ライセンス

[MIT License](LICENSE)

## 🙏 クレジット

- [PicoClaw](https://github.com/sipeed/picoclaw) by Sipeed — このプロジェクトが管理する素晴らしいAIアシスタント
- [OpenWrt](https://openwrt.org/) — 実行環境
- [LuCI](https://github.com/openwrt/luci) — Webインターフェースフレームワーク

## 🤝 コントリビューション

コントリビューションをお待ちしています！お気軽にPull Requestをどうぞ。

1. リポジトリをフォーク
2. フィーチャーブランチを作成 (`git checkout -b feature/amazing-feature`)
3. 変更をコミット (`git commit -m 'Add amazing feature'`)
4. ブランチをプッシュ (`git push origin feature/amazing-feature`)
5. Pull Requestを作成

---

<p align="center">
  このプロジェクトが役立つと思ったら、⭐ をお願いします！
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
