# 🤖 PicoClaw Telegram Bot - Quick Start

**Get your PicoClaw AI assistant on Telegram in 3 minutes!**

## ⚡ Quick Setup

### 1️⃣ Get Your Bot Token

1. Open Telegram → Search `@BotFather`
2. Send `/newbot`
3. Follow prompts (name + username ending in `_bot`)
4. **Copy the token** 🔐

### 2️⃣ Add to GitHub Secrets

- Go to your repo: **Settings** → **Secrets and Variables** → **Actions**
- Click **New repository secret**
- Name: `PICOCLAW_TELEGRAM_BOT_TOKEN`
- Value: Your token from step 1

### 3️⃣ Deploy

Push to main branch or trigger workflow:
```bash
git push origin main
```

The bot will be live in ~2 minutes ✨

---

## 🎯 Start Using

Find your bot on Telegram (search username) and send:
```
/start
Hello!
```

### Commands
- `/start` - Begin
- `/help` - Show help
- `/show` - Agent info
- `/list` - List agents

### Features
✨ Text messages
🎤 Voice notes (auto-transcribed)
📸 Images
📄 Documents

---

## 🔧 Manual Setup (Without GitHub Actions)

### SSH Setup
```bash
ssh root@YOUR_IP
nano /opt/picoclaw/config/.env
```

Add:
```
PICOCLAW_CHANNELS_TELEGRAM_ENABLED=true
PICOCLAW_CHANNELS_TELEGRAM_TOKEN=YOUR_TOKEN_HERE
```

Restart:
```bash
docker compose -f /opt/picoclaw/docker-compose.yml restart picoclaw
```

---

## 🛡️ Security

- ⚠️ Never commit tokens to git
- 🔐 Use GitHub Secrets
- 👥 Optional: Whitelist users in `allow_from` config

---

## 📚 Full Guide

See [docs/TELEGRAM_SETUP.md](docs/TELEGRAM_SETUP.md) for advanced options.

---

**Questions?** Check logs:
```bash
ssh root@YOUR_IP
tail -f /opt/picoclaw/logs/picoclaw.log | grep -i telegram
```
