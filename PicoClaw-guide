# PicoClaw Social Media Integration Guide

Complete configuration guide for connecting PicoClaw to Telegram, Discord, WhatsApp, WeChat, and Feishu.

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Telegram Setup](#telegram-setup)
- [Discord Setup](#discord-setup)
- [WhatsApp Setup](#whatsapp-setup)
- [WeChat/Feishu Setup](#wechatfeishu-setup)
- [QQ Setup](#qq-setup)
- [DingTalk Setup](#dingtalk-setup)
- [LINE Setup](#line-setup)
- [Slack Setup](#slack-setup)
- [Configuration File Reference](#configuration-file-reference)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

Before configuring any social media channel, ensure you have:

1. **PicoClaw installed** - Download from [GitHub Releases](https://github.com/sipeed/picoclaw/releases)
2. **API Key** - Get from [OpenRouter](https://openrouter.ai/keys) or [Zhipu](https://open.bigmodel.cn)
3. **Config file** - Located at `~/.picoclaw/config.json` (Linux/Mac) or `C:\Users\YOUR_NAME\.picoclaw\config.json` (Windows)

---

## Telegram Setup

**Difficulty:** ⭐ Easy (Recommended for beginners)

### Step 1: Create a Telegram Bot

1. Open Telegram app or [Telegram Web](https://web.telegram.org)
2. Search for `@BotFather`
3. Send the command: `/newbot`
4. Follow the prompts:
   - Enter a bot name (e.g., `MyPicoClaw Bot`)
   - Enter a bot username (must end with `bot`, e.g., `MyPicoClaw_bot`)
5. **Copy the bot token** (looks like `123456789:ABCdefGHIjklMNOpqrsTUVwxyz`)

### Step 2: Get Your User ID

1. Search for `@userinfobot` on Telegram
2. Send any message
3. **Copy your User ID** (a number like `123456789`)

### Step 3: Configure PicoClaw

Edit your config file:

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "anthropic/claude-3-haiku",
      "max_tokens": 4096,
      "temperature": 0.7
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "YOUR_OPENROUTER_API_KEY"
    }
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

### Step 4: Start the Gateway

```bash
picoclaw gateway
```

### Step 5: Test

Open Telegram, find your bot, and send a message!

---

## Discord Setup

**Difficulty:** ⭐⭐ Medium

### Step 1: Create a Discord Application

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Click **New Application**
3. Enter a name (e.g., `PicoClaw Bot`)
4. Click **Create**

### Step 2: Create a Bot

1. On the left menu, click **Bot**
2. Click **Add Bot**
3. Click **Yes, do it!**
4. Under **TOKEN**, click **Reset Token**
5. **Copy the token** (save it - you won't see it again)

### Step 3: Enable Intents (Critical!)

In the Bot settings page:

1. Scroll down to **Privileged Gateway Intents**
2. Enable these options:
   - ✅ **MESSAGE CONTENT INTENT** (Required!)
   - ✅ **SERVER MEMBERS INTENT** (Optional)
   - ✅ **PRESENCE INTENT** (Optional)
3. Click **Save Changes**

### Step 4: Get Your User ID

1. Open Discord app
2. Go to **Settings** → **Advanced**
3. Enable **Developer Mode**
4. Right-click your username/profile
5. Click **Copy User ID**

### Step 5: Invite Bot to Server

1. Go back to [Discord Developer Portal](https://discord.com/developers/applications)
2. Select your application
3. Click **OAuth2** → **URL Generator**
4. Under **Scopes**, check: `bot`
5. Under **Bot Permissions**, check:
   - ✅ `Read Messages/View Channels`
   - ✅ `Send Messages`
   - ✅ `Read Message History`
   - ✅ `Add Reactions`
6. Copy the generated URL at the bottom
7. Open the URL in your browser
8. Select your server and click **Authorize**

### Step 6: Configure PicoClaw

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "anthropic/claude-3-haiku",
      "max_tokens": 4096,
      "temperature": 0.7
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "YOUR_OPENROUTER_API_KEY"
    }
  },
  "channels": {
    "discord": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

### Step 7: Start the Gateway

```bash
picoclaw gateway
```

---

## WhatsApp Setup

**Difficulty:** ⭐⭐⭐ Hard (Requires Docker)

### Prerequisites

- Docker installed on your machine
- WhatsApp installed on your phone

### Step 1: Run WhatsApp Bridge

```bash
docker run -d --name whatsapp-bridge -p 3001:3001 mileusna/whatsapp-web-bridge
```

### Step 2: Link Your WhatsApp

1. Open browser: `http://localhost:3001`
2. Scan the QR code with your WhatsApp app:
   - Open WhatsApp → Settings → Linked Devices → Link a Device
3. Wait for connection confirmation

### Step 3: Configure PicoClaw

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "anthropic/claude-3-haiku",
      "max_tokens": 4096,
      "temperature": 0.7
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "YOUR_OPENROUTER_API_KEY"
    }
  },
  "channels": {
    "whatsapp": {
      "enabled": true,
      "bridge_url": "ws://localhost:3001",
      "allow_from": []
    }
  }
}
```

### Step 4: Start the Gateway

```bash
picoclaw gateway
```

---

## WeChat/Feishu Setup

**Difficulty:** ⭐⭐⭐ Hard

**Note:** WeChat personal accounts don't support bots. Use **Feishu (Lark)** for similar functionality.

### Step 1: Create a Feishu App

1. Go to [Feishu Open Platform](https://open.feishu.cn/app)
2. Click **Create Custom App** (创建企业自建应用)
3. Fill in app name and description
4. Click **Create**

### Step 2: Get Credentials

1. In your app, go to **Credentials & Basic Info** (凭证与基础信息)
2. Copy **App ID** (App ID)
3. Copy **App Secret** (App Secret)

### Step 3: Enable Bot

1. Go to **Bot** (机器人) in the left menu
2. Enable **Add Bot Capability** (添加应用能力 → 机器人)
3. Configure bot name and avatar

### Step 4: Configure Permissions

1. Go to **Permission Management** (权限管理)
2. Request these permissions:
   - `im:message` - Get and send messages
   - `im:message:send_as_bot` - Send messages as bot
   - `im:chat` - Get chat information

### Step 5: Set Up Event Subscription

1. Go to **Event Subscriptions** (事件订阅)
2. Enable events
3. Add event: `im.message.receive_v1`
4. For webhook URL, you need a public HTTPS URL

### Step 6: Expose Local Server (Using ngrok)

```bash
# Install ngrok, then run:
ngrok http 18790

# Copy the HTTPS URL (e.g., https://abc123.ngrok.io)
```

### Step 7: Configure PicoClaw

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "anthropic/claude-3-haiku",
      "max_tokens": 4096,
      "temperature": 0.7
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "YOUR_OPENROUTER_API_KEY"
    }
  },
  "channels": {
    "feishu": {
      "enabled": true,
      "app_id": "YOUR_APP_ID",
      "app_secret": "YOUR_APP_SECRET",
      "encrypt_key": "",
      "verification_token": "",
      "allow_from": []
    }
  }
}
```

### Step 8: Start the Gateway

```bash
picoclaw gateway
```

---

## QQ Setup

**Difficulty:** ⭐⭐ Medium

### Step 1: Create QQ Bot

1. Go to [QQ Open Platform](https://q.qq.com/)
2. Click **Login** and sign in with QQ
3. Click **Create Application** (创建应用)
4. Select **QQ Robot** (QQ机器人)
5. Fill in the required information
6. Copy **App ID** and **App Secret**

### Step 2: Configure PicoClaw

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "anthropic/claude-3-haiku",
      "max_tokens": 4096,
      "temperature": 0.7
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "YOUR_OPENROUTER_API_KEY"
    }
  },
  "channels": {
    "qq": {
      "enabled": true,
      "app_id": "YOUR_APP_ID",
      "app_secret": "YOUR_APP_SECRET",
      "allow_from": []
    }
  }
}
```

### Step 3: Start the Gateway

```bash
picoclaw gateway
```

---

## DingTalk Setup

**Difficulty:** ⭐⭐ Medium

### Step 1: Create DingTalk App

1. Go to [DingTalk Open Platform](https://open.dingtalk.com/)
2. Click **Console** (控制台)
3. Create an **Internal App** (企业内部应用)
4. Go to **App Info** and copy:
   - **Client ID** (AppKey)
   - **Client Secret** (AppSecret)

### Step 2: Configure Robot

1. In your app, go to **Robot** (机器人)
2. Enable robot capability
3. Configure message receiving settings

### Step 3: Configure PicoClaw

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "anthropic/claude-3-haiku",
      "max_tokens": 4096,
      "temperature": 0.7
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "YOUR_OPENROUTER_API_KEY"
    }
  },
  "channels": {
    "dingtalk": {
      "enabled": true,
      "client_id": "YOUR_CLIENT_ID",
      "client_secret": "YOUR_CLIENT_SECRET",
      "allow_from": []
    }
  }
}
```

### Step 4: Start the Gateway

```bash
picoclaw gateway
```

---

## LINE Setup

**Difficulty:** ⭐⭐⭐ Hard (Requires HTTPS webhook)

### Step 1: Create LINE Official Account

1. Go to [LINE Developers Console](https://developers.line.biz/)
2. Click **Console**
3. Create a **Provider** (or use existing)
4. Create a **Messaging API Channel**
5. Fill in the required information

### Step 2: Get Credentials

1. In your channel, go to **Messaging API**
2. Copy:
   - **Channel Secret**
   - **Channel Access Token** (click Issue if needed)

### Step 3: Set Up Webhook

Since LINE requires HTTPS, you need a public URL:

```bash
# Using ngrok
ngrok http 18791

# Copy the HTTPS URL
```

In LINE Developers Console:
1. Go to **Messaging API** → **Webhook settings**
2. Set Webhook URL: `https://your-ngrok-url.ngrok.io/webhook/line`
3. Enable **Use webhook**

### Step 4: Configure PicoClaw

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "anthropic/claude-3-haiku",
      "max_tokens": 4096,
      "temperature": 0.7
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "YOUR_OPENROUTER_API_KEY"
    }
  },
  "channels": {
    "line": {
      "enabled": true,
      "channel_secret": "YOUR_CHANNEL_SECRET",
      "channel_access_token": "YOUR_CHANNEL_ACCESS_TOKEN",
      "webhook_host": "0.0.0.0",
      "webhook_port": 18791,
      "webhook_path": "/webhook/line",
      "allow_from": []
    }
  }
}
```

### Step 5: Start the Gateway

```bash
picoclaw gateway
```

---

## Slack Setup

**Difficulty:** ⭐⭐ Medium

### Step 1: Create Slack App

1. Go to [Slack API](https://api.slack.com/apps)
2. Click **Create New App**
3. Choose **From scratch**
4. Enter app name and select workspace
5. Click **Create App**

### Step 2: Configure Bot Permissions

1. Go to **OAuth & Permissions**
2. Under **Bot Token Scopes**, add:
   - `app_mentions:read`
   - `chat:write`
   - `im:history`
   - `im:read`
   - `channels:history`
   - `groups:history`

### Step 3: Install App

1. Go to **Install App**
2. Click **Install to Workspace**
3. Allow permissions
4. Copy **Bot User OAuth Token** (starts with `xoxb-`)

### Step 4: Enable Socket Mode

1. Go to **Socket Mode**
2. Enable Socket Mode
3. Generate an App-Level Token with `connections:write` scope
4. Copy the token (starts with `xapp-`)

### Step 5: Subscribe to Events

1. Go to **Event Subscriptions**
2. Enable events
3. Under **Subscribe to bot events**, add:
   - `app_mention`
   - `message.im`

### Step 6: Configure PicoClaw

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "anthropic/claude-3-haiku",
      "max_tokens": 4096,
      "temperature": 0.7
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "YOUR_OPENROUTER_API_KEY"
    }
  },
  "channels": {
    "slack": {
      "enabled": true,
      "bot_token": "xoxb-YOUR-BOT-TOKEN",
      "app_token": "xapp-YOUR-APP-TOKEN",
      "allow_from": []
    }
  }
}
```

### Step 7: Start the Gateway

```bash
picoclaw gateway
```

---

## Configuration File Reference

### Complete Configuration Example

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true,
      "model": "anthropic/claude-3-haiku",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-xxxxxxxx",
      "api_base": "https://openrouter.ai/api/v1"
    },
    "zhipu": {
      "api_key": "xxxxxxxx",
      "api_base": "https://open.bigmodel.cn/api/paas/v4"
    },
    "anthropic": {
      "api_key": "sk-ant-xxxxxxxx",
      "api_base": "https://api.anthropic.com"
    },
    "openai": {
      "api_key": "sk-xxxxxxxx",
      "api_base": "https://api.openai.com/v1"
    },
    "gemini": {
      "api_key": "xxxxxxxx",
      "api_base": ""
    },
    "groq": {
      "api_key": "gsk_xxxxxxxx",
      "api_base": "https://api.groq.com/openai/v1"
    }
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
      "proxy": "",
      "allow_from": [123456789]
    },
    "discord": {
      "enabled": true,
      "token": "MTK3NjE4NjQxMzg3Mzg2NzY5OA.G-ABCDE.abcdefghij",
      "allow_from": [123456789012345678]
    },
    "whatsapp": {
      "enabled": false,
      "bridge_url": "ws://localhost:3001",
      "allow_from": []
    },
    "feishu": {
      "enabled": false,
      "app_id": "cli_xxxxxxxxxxxxxxxx",
      "app_secret": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "encrypt_key": "",
      "verification_token": "",
      "allow_from": []
    },
    "qq": {
      "enabled": false,
      "app_id": "1234567890",
      "app_secret": "xxxxxxxxxxxxxxxx",
      "allow_from": []
    },
    "dingtalk": {
      "enabled": false,
      "client_id": "dingxxxxxxxxxxxxxx",
      "client_secret": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "allow_from": []
    },
    "line": {
      "enabled": false,
      "channel_secret": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "channel_access_token": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "webhook_host": "0.0.0.0",
      "webhook_port": 18791,
      "webhook_path": "/webhook/line",
      "allow_from": []
    },
    "slack": {
      "enabled": false,
      "bot_token": "xoxb-xxxxxxxxxxxx-xxxxxxxxxxxx-xxxxxxxxxxxx",
      "app_token": "xapp-xxxxxxxxxxxx-xxxxxxxxxxxx-xxxxxxxxxxxx",
      "allow_from": []
    }
  },
  "tools": {
    "web": {
      "brave": {
        "enabled": true,
        "api_key": "BSAxxxxxxxxxxxxxxxxxx",
        "max_results": 5
      },
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      }
    }
  },
  "heartbeat": {
    "enabled": true,
    "interval": 30
  },
  "gateway": {
    "host": "0.0.0.0",
    "port": 18790
  }
}
```

### Configuration Options Explained

| Option | Description |
|--------|-------------|
| `allow_from` | List of allowed user IDs. Empty `[]` allows all users. |
| `enabled` | Set to `true` to enable the channel |
| `model` | AI model to use (e.g., `anthropic/claude-3-haiku`, `glm-4.7`) |
| `max_tokens` | Maximum tokens in response |
| `temperature` | Response creativity (0.0-1.0) |

---

## Troubleshooting

### Common Issues

#### 1. "not recognized as an internal or external command"
- Make sure PicoClaw is in your PATH
- Use the full path: `C:\path\to\picoclaw.exe gateway`

#### 2. "API error: Failed to authenticate"
- Check your API key is correct
- Make sure the key is in quotes in config.json
- Verify the key has remaining credits

#### 3. Telegram: "Conflict: terminated by other getUpdates"
- Another bot instance is running
- Stop all PicoClaw instances and restart

#### 4. Discord: Bot not responding
- Make sure MESSAGE CONTENT INTENT is enabled
- Check bot has permissions in the channel
- Verify bot token is correct

#### 5. WhatsApp: QR code not showing
- Check Docker container is running: `docker ps`
- Restart container: `docker restart whatsapp-bridge`

#### 6. Feishu: Not receiving messages
- Check webhook URL is publicly accessible
- Verify event subscription is configured correctly
- Make sure bot is added to the chat/group

---

## Platform Comparison

| Platform | Difficulty | Setup Time | Mobile Support |
|----------|------------|------------|----------------|
| **Telegram** | ⭐ Easy | 5 minutes | ✅ App + Web |
| **Discord** | ⭐⭐ Medium | 10 minutes | ✅ App + Web |
| **Slack** | ⭐⭐ Medium | 10 minutes | ✅ App + Web |
| **QQ** | ⭐⭐ Medium | 10 minutes | ✅ App |
| **DingTalk** | ⭐⭐ Medium | 10 minutes | ✅ App |
| **LINE** | ⭐⭐⭐ Hard | 20 minutes | ✅ App |
| **WhatsApp** | ⭐⭐⭐ Hard | 15 minutes | ✅ App |
| **Feishu** | ⭐⭐⭐ Hard | 20 minutes | ✅ App |

---

## Resources

- **PicoClaw GitHub:** https://github.com/sipeed/picoclaw
- **PicoClaw Website:** https://picoclaw.io
- **OpenRouter API Keys:** https://openrouter.ai/keys
- **Zhipu API Keys:** https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys
- **Brave Search API:** https://brave.com/search/api

---

## License

This guide is provided under MIT License. PicoClaw is developed by [Sipeed](https://sipeed.com).

---

*Last Updated: February 2026*
