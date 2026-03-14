# 🚀 Deploy PicoClaw for Free

## Quick Deploy

### Railway.app (Recommended)

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/template)

1. Click "Deploy on Railway"
2. Set environment variables:
   - `TELEGRAM_BOT_TOKEN` - Your bot token from @BotFather
   - `ZHIPU_API_KEY` - Your GLM API key from Z.AI

### Render.com

[![Deploy to Render](https://render.com/images/deploy-to-render-button.svg)](https://render.com/deploy)

1. Click "Deploy to Render"
2. Connect GitHub repo
3. Set environment variables
4. Deploy

### Fly.io

```bash
# Install CLI
brew install flyctl

# Login
fly auth login

# Deploy
fly launch
fly deploy
```

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `TELEGRAM_BOT_TOKEN` | ✅ | Bot token from @BotFather |
| `ZHIPU_API_KEY` | ✅ | GLM API key from Z.AI |
| `WEB_SEARCH_ENABLED` | ❌ | Enable web search (default: true) |
| `WEB_SEARCH_PROVIDER` | ❌ | duckduckgo or brave |

## Files Created

- `railway.toml` - Railway config
- `fly.toml` - Fly.io config  
- `render.yaml` - Render blueprint
- `deploy.sh` - Quick deploy script

## Run Deploy Script

```bash
cd /Users/adisusilayasa/picoclaw
./deploy.sh
```
