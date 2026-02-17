# PicoClaw on Android (Termux) Installation Guide

This guide provides two methods to install and run **PicoClaw** on your Android device.

## 🚀 Recommended: Easy One-Line Manager
We have provided an interactive script to handle installation, configuration, and management automatically.

Run this command in Termux:
```bash
pkg install -y curl && curl -sSL https://raw.githubusercontent.com/sipeed/picoclaw/main/assets/scripts/picoclaw-manager.sh -o picoclaw-manager.sh && chmod +x picoclaw-manager.sh && ./picoclaw-manager.sh
```

<img src="../assets/termux/termux-01.png" width="300" alt="PicoClaw Termux Manager">


---

## 🛠️ Manual Method
If you prefer to set up everything manually, follow these steps:

## Prerequisites
- Android device with **Termux** installed (Download from [F-Droid](https://f-droid.org/en/packages/com.termux/)).
- Working internet connection.

## 1. Environment Setup
Open Termux and install the necessary packages:
```bash
pkg update && pkg upgrade
pkg install -y curl jq tmux
```

## 2. Download Pre-compiled Binary
PicoClaw provides pre-compiled Android binaries for every release — no build tools needed.

```bash
# Fetch latest version tag
VERSION=$(curl -s https://api.github.com/repos/sipeed/picoclaw/releases/latest | jq -r '.tag_name')
echo "Installing PicoClaw $VERSION..."

# Download the Android arm64 binary (built with GOOS=android for Termux compatibility)
curl -fSL "https://github.com/sipeed/picoclaw/releases/download/${VERSION}/picoclaw_Android_arm64.tar.gz" -o /tmp/picoclaw.tar.gz
tar -xzf /tmp/picoclaw.tar.gz -C /tmp picoclaw

# Install to Termux bin directory
cp /tmp/picoclaw $PREFIX/bin/picoclaw
chmod +x $PREFIX/bin/picoclaw
rm /tmp/picoclaw.tar.gz /tmp/picoclaw
```

Now you can run `picoclaw` from anywhere!

## 3. Initial Configuration
Initialise the workspace and generate a default configuration:
```bash
picoclaw onboard
```

This creates the configuration file at `~/.picoclaw/config.json`.

## 4. Connect to LLM and Telegram
Edit your configuration to add your Gemini (or other) API key and Telegram bot token:

### Add API Key (Gemini Example)
```bash
# Replace YOUR_GEMINI_KEY with your actual key
GEMINI_KEY="YOUR_GEMINI_KEY"
jq ".providers.gemini.api_key = \"$GEMINI_KEY\" | .agents.defaults.provider = \"gemini\" | .agents.defaults.model = \"gemini-2.0-flash-lite\"" ~/.picoclaw/config.json > ~/.picoclaw/config.json.tmp && mv ~/.picoclaw/config.json.tmp ~/.picoclaw/config.json
```

### Enable Telegram Bot
```bash
# Replace YOUR_BOT_TOKEN with your Telegram token
TELEGRAM_TOKEN="YOUR_BOT_TOKEN"
jq ".channels.telegram.enabled = true | .channels.telegram.token = \"$TELEGRAM_TOKEN\" | .channels.telegram.allow_from = []" ~/.picoclaw/config.json > ~/.picoclaw/config.json.tmp && mv ~/.picoclaw/config.json.tmp ~/.picoclaw/config.json
```

## 5. Running PicoClaw
It's recommended to run PicoClaw inside a `tmux` session so it keeps running in the background:

```bash
# Start a new tmux session named 'picoclaw'
tmux new -s picoclaw

# Inside tmux, start the gateway
picoclaw gateway
```

To detach from tmux, press `Ctrl+B` then `D`.
To re-attach later: `tmux attach -t picoclaw`.

## 6. Success!
Once the gateway is running, you should see:
`[INFO] telegram: Telegram bot connected {username=...}`

You can now chat with your PicoClaw assistant directly on Telegram!

<img src="../assets/termux/termux-02.png" width="300" alt="PicoClaw Termux Status">
