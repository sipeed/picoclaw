# MQTT Channel Configuration Guide

## 1. Example Configuration

Add this to `config.json`:

```json
{
  "channels": {
    "mqtt": {
      "enabled": true,
      "broker": "tcp://localhost:1883",
      "client_id": "picoclaw-bot",
      "username": "",
      "password": "",
      "qos": 1,
      "retain": false,
      "tls": false,
      "subscribe_topics": ["picoclaw/chat"],
      "reply_topic": "picoclaw/reply",
      "allow_from": [],
      "group_trigger": {
        "mention_only": true
      },
      "reasoning_channel_id": "",
      "instruction": ""
    }
  }
}
```

## 2. Field Reference

| Field                | Type     | Required | Description |
|----------------------|----------|----------|-------------|
| enabled              | bool     | Yes      | Enable or disable the MQTT channel |
| broker               | string   | Yes      | MQTT broker URL (e.g., `tcp://localhost:1883`, `ssl://broker.example.com:8883`) |
| client_id            | string   | Yes      | Unique client identifier for the MQTT connection |
| username             | string   | No       | MQTT username for authentication |
| password             | string   | No       | MQTT password for authentication |
| qos                  | int      | No       | Quality of Service level (0, 1, or 2). Default: 1 |
| retain               | bool     | No       | Whether to retain messages. Default: false |
| tls                  | bool     | No       | Enable TLS/SSL connection. Default: false |
| subscribe_topics     | []string | Yes      | List of MQTT topics to subscribe to for incoming messages |
| reply_topic          | string   | No       | Topic to publish replies to. Supports placeholders: `{client_id}`, `{topic}` |
| allow_from           | []string | No       | Client ID whitelist (empty allows all) |
| group_trigger        | object   | No       | Group trigger strategy (`mention_only` / `prefixes`) |
| reasoning_channel_id | string   | No       | Target channel for reasoning output |
| instruction          | string   | No       | Optional instruction prefix added to all incoming messages |

## 3. Currently Supported

- **Message Format**: Supports both JSON and plain text messages
  - JSON format: `{"status": "your message"}`
  - Plain text: Direct text content
  - Automatic JSON parsing with fallback to plain text for malformed JSON
- **Authentication**: Username/password authentication support
- **TLS/SSL**: Secure connections with TLS configuration
- **Quality of Service**: Configurable QoS levels (0, 1, 2)
- **Topic Management**: Multiple subscribe topics and configurable reply topics
- **Message Retention**: Optional message retention on broker
- **Auto-reconnection**: Automatic reconnection with exponential backoff
- **Group Triggers**: Support for mention-only and prefix-based triggers
- **Placeholder Replacement**: Dynamic topic names using `{client_id}` and `{topic}` placeholders

## 4. Features

- **Robust Message Handling**: Intelligent parsing that handles malformed JSON gracefully
- **Flexible Topic Configuration**: Support for multiple input topics and dynamic reply topics
- **Connection Resilience**: Automatic reconnection with configurable retry intervals
- **Security**: TLS support and authentication for secure communication
- **Message Routing**: Support for reasoning channel routing and group trigger rules

## 5. Usage Notes

- The channel automatically handles malformed JSON by attempting to clean and parse it
- Reply topics can use placeholders to dynamically route responses
- Client IDs are used as sender identifiers in the messaging system
- Topics are treated as channels for message routing purposes
- The instruction field allows adding context or commands to all incoming messages