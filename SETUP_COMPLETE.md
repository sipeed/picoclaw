# 🚀 PicoClaw Complete Setup Guide

**Secure Telegram Bot on Hostinger VPS with Tailscale**

---

## 📋 Setup Overview

This guide walks you through:
1. **🔐 Tailscale** - Secure private network access
2. **🤖 Telegram** - Bot integration with @BotFather
3. **✅ Verification** - Test everything works

---

## 🛠️ Prerequisites

- Hostinger VPS running (Ubuntu 20.04+)
- GitHub account and repository
- Telegram account
- SSH access to your VPS
- `make` and `bash` installed locally
- `gh` CLI (optional, for GitHub Secrets automation)

---

## 🎯 Quick Start (3 Steps)

### Step 1: Secure with Tailscale

```bash
make setup-tailscale
```

**What it does:**
- ✅ Installs Tailscale on your VPS
- ✅ Authenticates your VPS to your Tailnet
- ✅ Blocks port 18790 from public internet
- ✅ Creates secure tunnel (only accessible via your Tailscale network)

**Time:** ~5 minutes

---

### Step 2: Setup Telegram Bot

```bash
make setup-telegram
```

**What it does:**
- ✅ Guides you to create bot with @BotFather
- ✅ Validates bot token with Telegram API
- ✅ Configures GitHub Secrets automatically (if gh CLI available)
- ✅ Deploys to your VPS
- ✅ Verifies installation

**Time:** ~10 minutes

---

### Step 3: Test Everything

```bash
# Check Tailscale status
ssh root@YOUR_IP 'tailscale status'

# Check Telegram logs
ssh root@YOUR_IP 'docker exec picoclaw tail -50 /opt/picoclaw/logs/picoclaw.log | grep -i telegram'

# Test Telegram bot
# Open Telegram and find your bot, send /start
```

---

## 📝 Step-by-Step Details

### Phase 1: Initial Server Setup (One-time)

If this is a fresh VPS:

```bash
# SSH into your server
ssh root@YOUR_HOSTINGER_IP

# Or use GitHub Actions to deploy (easier)
git push origin main
# Watch deployment at: https://github.com/YOUR_USER/YOUR_REPO/actions
```

---

### Phase 2: Tailscale Configuration

**Option A: Automated (Recommended)**

```bash
# From your local machine
make setup-tailscale
```

Follow the interactive prompts. The script will:
1. Ask for SSH details
2. Install Tailscale
3. Open authentication link (click in browser)
4. Activate Tailscale serve
5. Verify connectivity

**Option B: Manual (SSH)**

```bash
ssh root@YOUR_IP

# Install Tailscale
curl -fsSL https://tailscale.com/install.sh | sh

# Authenticate
tailscale up --hostname=picoclaw --ssh
# (Copy the URL and open in browser)

# Activate serve
tailscale serve --bg http://localhost:18790

# Verify
tailscale ip -4
```

---

### Phase 3: Telegram Bot Setup

**Option A: Automated (Recommended)**

```bash
# From your local machine
make setup-telegram
```

Follow the interactive prompts. The script will:
1. Guide you to @BotFather
2. Validate your bot token
3. Save to GitHub Secrets
4. Deploy automatically
5. Verify installation

**Option B: Manual (GitHub Secrets)**

1. **Create bot with @BotFather**
   - Open Telegram → Search `@BotFather`
   - Send `/newbot`
   - Give it a name and username
   - Copy the token

2. **Add to GitHub Secrets**
   ```bash
   gh secret set PICOCLAW_TELEGRAM_BOT_TOKEN -b YOUR_TOKEN -R your-user/picoclaw
   ```

3. **Deploy**
   ```bash
   git push origin main
   ```

4. **Verify**
   - Open Telegram, find your bot
   - Send `/start`
   - Check logs: `ssh root@YOUR_IP tail -f /opt/picoclaw/logs/picoclaw.log | grep telegram`

**Option C: Manual (Direct SSH)**

```bash
ssh root@YOUR_IP

# Edit .env
nano /opt/picoclaw/config/.env

# Add these lines:
PICOCLAW_CHANNELS_TELEGRAM_ENABLED=true
PICOCLAW_CHANNELS_TELEGRAM_TOKEN=YOUR_TOKEN_HERE

# Save and exit (Ctrl+X, Y, Enter)

# Restart Docker
docker compose -f /opt/picoclaw/docker-compose.yml restart picoclaw

# Verify
docker compose logs picoclaw | grep -i telegram
```

---

## ✅ Verification Checklist

After setup is complete, verify everything works:

### 1. Tailscale

```bash
# Check Tailscale is running
ssh root@YOUR_IP 'tailscale status'

# Check serve is active
ssh root@YOUR_IP 'ps aux | grep tailscale'

# Get your Tailnet IP
ssh root@YOUR_IP 'tailscale ip -4'
```

### 2. PicoClaw Gateway

```bash
# Check container is running
ssh root@YOUR_IP 'docker compose ps'

# Check health endpoint (via Tailscale IP)
TAILNET_IP=$(ssh root@YOUR_IP 'tailscale ip -4')
curl http://$TAILNET_IP:18790/health

# Check gateway is listening on localhost only
ssh root@YOUR_IP 'netstat -tuln | grep 18790'
# Should show: 127.0.0.1:18790 (NOT 0.0.0.0)
```

### 3. Telegram Bot

```bash
# Check logs for Telegram initialization
ssh root@YOUR_IP 'docker compose logs picoclaw | grep -i telegram'

# Expected output should include:
# "Starting Telegram bot (polling mode)..."
# "Telegram bot connected"
```

### 4. Test Telegram Bot

1. Open Telegram
2. Search for your bot (username from @BotFather)
3. Click **Start**
4. Send a message

Expected response:
```
Thinking... 💭
[Claude's response]
```

---

## 🔐 Security Summary

| Component | Status | Access Method |
|-----------|--------|----------------|
| SSH | 🔒 Protected | Tailscale tunnel |
| PicoClaw Gateway (18790) | 🔒 Protected | Tailscale tunnel only |
| Telegram Bot | 🌐 Public | via Telegram API |
| Config/Secrets | 🔐 Encrypted | GitHub Secrets |

**What's protected:**
- ✅ Port 18790 is NOT exposed to the internet
- ✅ Accessible only via your Tailscale network
- ✅ UFW firewall blocks public access
- ✅ Bot token stored in GitHub Secrets (never in code)

---

## 🐛 Troubleshooting

### Tailscale not authenticating

```bash
# Check if already authenticated
ssh root@YOUR_IP 'tailscale status'

# If not, try again
ssh root@YOUR_IP 'tailscale up --hostname=picoclaw'
```

### Telegram bot not responding

```bash
# Check if Telegram is enabled
ssh root@YOUR_IP 'grep TELEGRAM /opt/picoclaw/config/.env'

# Check logs
ssh root@YOUR_IP 'docker compose logs picoclaw | grep -i telegram'

# Verify token is correct (should start with numbers:)
ssh root@YOUR_IP 'grep TELEGRAM_TOKEN /opt/picoclaw/config/.env'
```

### Can't reach PicoClaw via Tailscale

```bash
# Check Tailscale IP
ssh root@YOUR_IP 'tailscale ip -4'

# Check if port is listening
ssh root@YOUR_IP 'netstat -tuln | grep 18790'

# Check if Docker container is running
ssh root@YOUR_IP 'docker compose ps'

# Check logs
ssh root@YOUR_IP 'docker compose logs picoclaw | tail -50'
```

---

## 📚 Additional Resources

- **Tailscale Docs**: https://tailscale.com/kb/
- **Telegram Bot API**: https://core.telegram.org/bots
- **PicoClaw Telegram Setup**: [docs/TELEGRAM_SETUP.md](docs/TELEGRAM_SETUP.md)
- **PicoClaw Quickstart**: [TELEGRAM_QUICKSTART.md](TELEGRAM_QUICKSTART.md)

---

## 🎬 Next Steps

1. **Run `make setup-tailscale`** - Secure your VPS
2. **Run `make setup-telegram`** - Add Telegram bot
3. **Test your bot** - Send a message on Telegram
4. **Configure whitelist** (optional) - Restrict to specific users
5. **Set up monitoring** (optional) - Get alerts on failures

---

## ❓ FAQ

**Q: Why Tailscale?**
A: It creates a private network between your devices and VPS. Port 18790 stays hidden from the internet while remaining accessible to you.

**Q: Can others use my bot without Tailscale?**
A: Yes! The Telegram bot is public (everyone can talk to it), but the PicoClaw gateway behind it is private (only you can manage it via Tailscale).

**Q: How much does Tailscale cost?**
A: Free for personal use (up to 100 devices). Perfect for this setup.

**Q: What if I lose my device?**
A: Remove it from your Tailnet at https://login.tailscale.com. It will lose access immediately.

**Q: Can I use a different VPN?**
A: Sure, but you'll need to configure a different security tunnel yourself. Tailscale is recommended for simplicity.

---

**Happy deploying! 🚀**
