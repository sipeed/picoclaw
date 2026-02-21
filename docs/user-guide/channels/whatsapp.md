# WhatsApp Setup

WhatsApp is the world's most popular messaging app. PicoClaw connects via a bridge service.

## Prerequisites

- A WhatsApp account
- A WhatsApp bridge service (e.g., mautrix-whatsapp, whatsapp-web.js)

## Overview

PicoClaw requires a bridge service to connect to WhatsApp because WhatsApp does not provide a direct bot API for personal accounts. The bridge acts as a middleware between WhatsApp and PicoClaw.

## Step 1: Set Up a WhatsApp Bridge

### Option A: mautrix-whatsapp (Recommended)

[mautrix-whatsapp](https://github.com/mautrix/whatsapp) is a Matrix-WhatsApp bridge that can be adapted for PicoClaw.

### Option B: Custom Bridge

You can create a custom WebSocket bridge using libraries like:
- [whatsapp-web.js](https://github.com/pedroslopez/whatsapp-web.js) (Node.js)
- [Baileys](https://github.com/WhiskeySockets/Baileys) (Node.js)

The bridge should:
1. Connect to WhatsApp
2. Expose a WebSocket server
3. Forward messages between WhatsApp and PicoClaw

## Step 2: Bridge Protocol

Your bridge should communicate with PicoClaw via WebSocket using this message format:

### Incoming Messages (to PicoClaw)

```json
{
  "type": "message",
  "from": "sender_phone_number",
  "chat": "chat_id",
  "content": "message text",
  "id": "message_id",
  "from_name": "Sender Name",
  "media": []
}
```

### Outgoing Messages (from PicoClaw)

```json
{
  "type": "message",
  "to": "chat_id",
  "content": "response text"
}
```

## Step 3: Configure PicoClaw

Edit `~/.picoclaw/config.json`:

```json
{
  "channels": {
    "whatsapp": {
      "enabled": true,
      "bridge_url": "ws://localhost:3001",
      "allow_from": []
    }
  }
}
```

| Option | Required | Description |
|--------|----------|-------------|
| `enabled` | Yes | Set to `true` to enable |
| `bridge_url` | Yes | WebSocket URL of your bridge service |
| `allow_from` | No | Array of allowed phone numbers |

## Step 4: Start the Gateway

```bash
picoclaw gateway
```

You should see:
```
Starting WhatsApp channel connecting to ws://localhost:3001...
WhatsApp channel connected
```

## Step 5: Test

1. Open WhatsApp on your phone
2. Send a message to the connected account
3. The bot should respond!

## Features

### Media Support

The bridge can forward media files:

```json
{
  "type": "message",
  "from": "+1234567890",
  "chat": "+1234567890",
  "content": "Check this image",
  "media": ["/path/to/image.jpg"]
}
```

### Group Chats

Group chat support depends on your bridge implementation.

### Message Metadata

Additional metadata can be included:

```json
{
  "type": "message",
  "from": "+1234567890",
  "chat": "group_id@g.us",
  "content": "Hello group!",
  "id": "msg_123",
  "from_name": "John Doe"
}
```

## Troubleshooting

### Connection refused

1. Verify your bridge service is running
2. Check the bridge_url is correct
3. Ensure the WebSocket server is listening

### Messages not received

1. Check bridge logs for errors
2. Verify WhatsApp connection is active
3. Ensure the bridge is forwarding messages correctly

### Messages not sent

1. Check PicoClaw logs for errors
2. Verify the bridge is handling outgoing messages
3. Check WhatsApp session is valid

### Bridge disconnects frequently

1. Check network stability
2. Verify WhatsApp session is persistent
3. Consider implementing reconnection logic in your bridge

## Bridge Implementation Example

Here's a simple example using Node.js with whatsapp-web.js:

```javascript
const { Client, LocalAuth } = require('whatsapp-web.js');
const WebSocket = require('ws');

// WhatsApp client
const waClient = new Client({
  authStrategy: new LocalAuth()
});

// WebSocket server for PicoClaw
const wss = new WebSocket.Server({ port: 3001 });

let picoclawConnection = null;

wss.on('connection', (ws) => {
  picoclawConnection = ws;
  console.log('PicoClaw connected');

  ws.on('message', (data) => {
    const msg = JSON.parse(data);
    if (msg.type === 'message') {
      // Send to WhatsApp
      waClient.sendMessage(msg.to, msg.content);
    }
  });
});

// Forward WhatsApp messages to PicoClaw
waClient.on('message', (msg) => {
  if (picoclawConnection) {
    picoclawConnection.send(JSON.stringify({
      type: 'message',
      from: msg.from,
      chat: msg.from,
      content: msg.body,
      id: msg.id._serialized,
      from_name: msg._data.notifyName
    }));
  }
});

waClient.initialize();
```

## Important Notes

### WhatsApp Terms of Service

Using unofficial WhatsApp clients may violate WhatsApp's Terms of Service. Use at your own risk.

### Business API

For production use, consider the official [WhatsApp Business API](https://business.whatsapp.com/).

### Session Persistence

Ensure your bridge properly saves and restores WhatsApp sessions to avoid repeated QR code scans.

## See Also

- [Channels Overview](README.md)
- [Gateway Command](../cli/gateway.md)
- [mautrix-whatsapp](https://github.com/mautrix/whatsapp)
- [whatsapp-web.js](https://github.com/pedroslopez/whatsapp-web.js)
- [Troubleshooting](../../operations/troubleshooting.md)
