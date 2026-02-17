# PicoClaw Telegram Bot Setup Guide

Complete step-by-step guide to set up PicoClaw as a Telegram bot.

## Prerequisites

- Telegram account
- Running PicoClaw instance (deployed)
- GitHub secrets configured (for automated deployment)

## Step 1: Create a Telegram Bot with BotFather

1. Open Telegram and search for **@BotFather**
2. Start the chat and send `/start`
3. Send `/newbot` to create a new bot
4. Follow the prompts:
   - **Name**: Give your bot a name (e.g., "PicoClaw AI")
   - **Username**: Must be unique and end with `_bot` (e.g., `picoclaw_bot`)
5. **Copy the token** provided (looks like `123456789:ABCdefGHIjklMNOpqrSTUvwxYZ`)
   - ⚠️ **Keep this token secret!** Anyone with this token can control your bot.

## Step 2: Configure PicoClaw

### Option A: Using Environment Variables (Production)

Add to your `.env` file or GitHub Secrets:

```bash
PICOCLAW_CHANNELS_TELEGRAM_ENABLED=true
PICOCLAW_CHANNELS_TELEGRAM_TOKEN=YOUR_BOT_TOKEN_HERE
```

### Option B: Using config.json (Development)

Edit `config/config.json`:

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN_HERE",
      "proxy": "",
      "allow_from": []
    }
  }
}
```

### Optional: User Whitelist

To restrict access to specific users, add their Telegram user IDs:

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN_HERE",
      "allow_from": ["123456789", "987654321"]
    }
  }
}
```

To find your Telegram user ID:
1. Send any message to your bot
2. Check the logs: `cat /opt/picoclaw/logs/picoclaw.log | grep "user_id"`
3. Your user ID will be in the output

## Step 3: Deploy to Hostinger

### For GitHub Actions Deployment

1. Add the bot token to your GitHub repository secrets:
   - Go to **Settings** → **Secrets and Variables** → **Actions**
   - Click **New repository secret**
   - **Name**: `PICOCLAW_TELEGRAM_BOT_TOKEN`
   - **Value**: Your BotFather token

2. Update `.github/workflows/deploy-hostinger.yml` to use the secret:
   ```yaml
   env:
     PICOCLAW_CHANNELS_TELEGRAM_ENABLED: "true"
     PICOCLAW_CHANNELS_TELEGRAM_TOKEN: ${{ secrets.PICOCLAW_TELEGRAM_BOT_TOKEN }}
   ```

### For Manual Deployment

SSH into your server:
```bash
ssh root@YOUR_HOSTINGER_IP
nano /opt/picoclaw/config/.env
```

Add:
```
PICOCLAW_CHANNELS_TELEGRAM_ENABLED=true
PICOCLAW_CHANNELS_TELEGRAM_TOKEN=YOUR_BOT_TOKEN
```

Restart the container:
```bash
cd /opt/picoclaw && docker compose -f docker-compose.production.yml restart picoclaw
```

## Step 4: Start Using Your Bot

1. Find your bot on Telegram (search by username: @your_bot_username)
2. Send `/start` to initialize
3. Start chatting!

### Available Commands

- `/start` - Initialize the bot
- `/help` - Show available commands
- `/show` - Show current agent info
- `/list` - List available agents

### Features

✅ **Text Messages** - Ask questions, get AI responses
✅ **Voice Messages** - Send voice notes (auto-transcribed)
✅ **Images** - Send photos for analysis
✅ **Documents** - Share files for processing
✅ **Multi-agent** - Switch between different AI agents
✅ **Thinking Indicator** - See when the AI is processing

## Step 5: Verify It's Working

Check the logs to confirm the bot is running:

```bash
ssh root@YOUR_HOSTINGER_IP
tail -f /opt/picoclaw/logs/picoclaw.log | grep -i telegram
```

You should see:
```
[INFO] [telegram] Starting Telegram bot (polling mode)...
[INFO] [telegram] Telegram bot connected
```

Send a test message to your bot and check:
```bash
tail -f /opt/picoclaw/logs/picoclaw.log | grep -i "Received message"
```

## Troubleshooting

### Bot doesn't respond

1. Check if Telegram is enabled:
   ```bash
   docker exec picoclaw cat /opt/picoclaw/config/.env | grep TELEGRAM
   ```

2. Verify the token is correct (no spaces, exact copy from BotFather)

3. Check logs for errors:
   ```bash
   docker exec picoclaw tail -100 /opt/picoclaw/logs/picoclaw.log | grep -i telegram
   ```

### "Failed to create telegram bot"

- Token is invalid or expired
- Token is incomplete (missing characters)
- Try creating a new bot with BotFather

### "Message rejected by allowlist"

- Your Telegram user ID is not in the whitelist
- Check your actual ID by removing `allow_from` temporarily
- Add your ID to the whitelist

## Advanced: Using Proxy

If you're in a region with restricted access to Telegram, configure a proxy:

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN_HERE",
      "proxy": "socks5://user:pass@proxy-host:1080",
      "allow_from": []
    }
  }
}
```

## Security Notes

⚠️ **Never commit your bot token to version control**
⚠️ **Use GitHub Secrets for CI/CD deployments**
⚠️ **Consider using user whitelists for sensitive deployments**
⚠️ **Regularly rotate tokens if compromised**

## Getting Help

- Telegram Bot API Docs: https://core.telegram.org/bots
- BotFather Commands: https://core.telegram.org/bots#botfather
- PicoClaw Issues: https://github.com/sipeed/picoclaw/issues
