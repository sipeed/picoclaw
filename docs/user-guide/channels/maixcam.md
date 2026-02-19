# MaixCam Setup

MaixCam is an AI-enabled camera device that can detect and report events to PicoClaw.

## Prerequisites

- A MaixCam device
- Network connectivity between MaixCam and PicoClaw server

## Overview

MaixCam is a hardware device with AI capabilities (person detection, object recognition, etc.). It connects to PicoClaw via TCP and sends event notifications when detections occur.

## Step 1: Set Up MaixCam Device

1. Power on your MaixCam device
2. Connect it to your network (WiFi or Ethernet)
3. Configure the device to send events to your PicoClaw server

Refer to your MaixCam documentation for specific setup instructions.

## Step 2: Configure PicoClaw

Edit `~/.picoclaw/config.json`:

```json
{
  "channels": {
    "maixcam": {
      "enabled": true,
      "host": "0.0.0.0",
      "port": 18790,
      "allow_from": []
    }
  }
}
```

| Option | Required | Description |
|--------|----------|-------------|
| `enabled` | Yes | Set to `true` to enable |
| `host` | No | Host to bind TCP server (default: `0.0.0.0`) |
| `port` | No | Port for TCP server (default: `18790`) |
| `allow_from` | No | Array of allowed device identifiers |

## Step 3: Start the Gateway

```bash
picoclaw gateway
```

You should see:
```
Starting MaixCam channel server
MaixCam server listening host=0.0.0.0 port=18790
```

## Step 4: Configure MaixCam to Connect

On your MaixCam device, configure it to connect to PicoClaw:

- Server IP: Your PicoClaw server's IP address
- Server Port: `18790` (or your configured port)

## Step 5: Test

1. Trigger a detection event on your MaixCam (e.g., walk in front of the camera)
2. The AI agent will receive the detection event and can respond accordingly

## Message Format

MaixCam sends JSON messages over TCP:

### Person Detection Event

```json
{
  "type": "person_detected",
  "timestamp": 1708346400.123,
  "data": {
    "class_name": "person",
    "class_id": 0,
    "score": 0.95,
    "x": 100,
    "y": 200,
    "w": 50,
    "h": 100
  }
}
```

### Heartbeat Event

```json
{
  "type": "heartbeat",
  "timestamp": 1708346400.123,
  "data": {}
}
```

### Status Event

```json
{
  "type": "status",
  "timestamp": 1708346400.123,
  "data": {
    "status": "online",
    "battery": 85
  }
}
```

## Features

### Person Detection

When a person is detected, PicoClaw receives:

- Class name (e.g., "person")
- Confidence score
- Position (x, y)
- Bounding box size (w, h)

### Status Updates

The device can send status updates including:
- Online/offline status
- Battery level
- System health

### Command Response

PicoClaw can send commands back to the MaixCam:

```json
{
  "type": "command",
  "timestamp": 0,
  "message": "Command from AI",
  "chat_id": "default"
}
```

## Use Cases

### Security Monitoring

The AI agent can:
- Log detection events
- Send alerts to other channels
- Trigger actions based on detection

### Smart Home Integration

Connect with other smart home devices:
- Turn on lights when person detected
- Trigger recordings
- Send notifications

### Activity Logging

Track and analyze:
- Movement patterns
- Peak activity times
- Anomaly detection

## Troubleshooting

### Device not connecting

1. Verify network connectivity
2. Check firewall allows incoming connections on configured port
3. Verify MaixCam is configured with correct server IP and port

### No events received

1. Check MaixCam logs for errors
2. Verify detection features are enabled on the device
3. Check PicoClaw gateway logs for connection messages

### Connection drops frequently

1. Check network stability
2. Verify power supply to MaixCam
3. Check for interference if using WiFi

### Slow response times

1. Check network latency
2. Verify MaixCam processing is not overloaded
3. Consider reducing detection frequency

## Advanced Configuration

### Custom Port

```json
{
  "channels": {
    "maixcam": {
      "enabled": true,
      "host": "0.0.0.0",
      "port": 9999
    }
  }
}
```

### Multiple Devices

Multiple MaixCam devices can connect to the same PicoClaw server. Each device maintains its own connection.

### With Other Channels

Combine MaixCam with other channels for notifications:

```json
{
  "channels": {
    "maixcam": {
      "enabled": true,
      "host": "0.0.0.0",
      "port": 18790
    },
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

The AI agent can then forward MaixCam alerts to Telegram.

## Example Agent Response

When MaixCam sends a person detection, you can configure your agent to respond:

```
Person detected!
Class: person
Confidence: 95.00%
Position: (100, 200)
Size: 50x100
```

The agent can then:
- Log the event to memory
- Send a notification to another channel
- Trigger additional analysis

## Technical Details

### Connection Protocol

- Transport: TCP
- Encoding: JSON (newline-delimited or continuous stream)
- Direction: Bidirectional (events from device, commands to device)

### Message Processing

1. MaixCam connects to PicoClaw TCP server
2. MaixCam sends JSON events
3. PicoClaw parses and routes events to the AI agent
4. AI agent processes and can respond with commands

## See Also

- [Channels Overview](README.md)
- [Gateway Command](../cli/gateway.md)
- [MaixCam Documentation](https://maix.sipeed.com/)
- [Troubleshooting](../../operations/troubleshooting.md)
