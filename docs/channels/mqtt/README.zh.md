# 📡 MQTT 渠道

PicoClaw 支持将任意 MQTT 客户端作为消息渠道。设备或服务向 Broker 发布请求，PicoClaw 订阅后处理并将响应发布回去。

## 🚀 快速开始

**1. 在 `~/.picoclaw/config.json` 中添加渠道：**

```json
{
  "channel_list": {
    "mqtt": {
      "enabled": true,
      "type": "mqtt",
      "settings": {
        "broker": "tcp://localhost:1883",
        "agent_id": "assistant"
      }
    }
  }
}
```

**2. 启动网关：**

```bash
picoclaw gateway
```

**3. 用任意 MQTT 客户端发送消息：**

```bash
mosquitto_pub -t "/picoclaw/assistant/device1/request" \
  -m '{"text": "查一下CPU使用率"}'
```

**4. 订阅响应：**

```bash
mosquitto_sub -t "/picoclaw/assistant/device1/response"
```

---

## 📨 Topic 结构

```
{prefix}/{agent_id}/{client_id}/request    # 客户端 → PicoClaw
{prefix}/{agent_id}/{client_id}/response   # PicoClaw → 客户端
```

| 段 | 说明 |
|----|------|
| `prefix` | Topic 前缀，由服务端配置，默认 `/picoclaw` |
| `agent_id` | PicoClaw 实例标识，对应配置中的 `agent_id` 字段 |
| `client_id` | 客户端自定义会话标识——同一设备保持相同 ID 可维持上下文连续性 |

### 消息体（JSON）

```json
{ "text": "你的消息内容" }
```

---

## ⚙️ 配置说明

### config.json

```json
{
  "channel_list": {
    "mqtt": {
      "enabled": true,
      "type": "mqtt",
      "settings": {
        "broker": "ssl://your-broker:8883",
        "agent_id": "assistant",
        "topic_prefix": "/picoclaw",
        "client_id": "",
        "keep_alive": 60,
        "qos": 0
      }
    }
  }
}
```

### .security.yml（用户名和密码）

用户名和密码存储于 `~/.picoclaw/.security.yml`，不写入 `config.json`：

```yaml
channel_list:
  mqtt:
    settings:
      username: your_username
      password: your_password
```

### 字段说明

| 字段 | 位置 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `broker` | `settings` | 是 | — | MQTT Broker 地址，如 `tcp://host:1883`、`ssl://host:8883` |
| `agent_id` | `settings` | 是 | — | Agent 标识，作为 topic 路径的一部分 |
| `topic_prefix` | `settings` | 否 | `/picoclaw` | Topic 命名空间前缀 |
| `username` | `.security.yml` | 否 | — | Broker 认证用户名 |
| `password` | `.security.yml` | 否 | — | Broker 认证密码 |
| `client_id` | `settings` | 否 | 自动生成 | 发送给 Broker 的 paho 客户端 ID。未配置时自动生成为 `picoclaw-mqtt-{agent_id}-{8位hex}`，进程生命周期内固定不变，断线重连时复用同一 ID |
| `keep_alive` | `settings` | 否 | `60` | MQTT 心跳间隔（秒） |
| `qos` | `settings` | 否 | `0` | 发布和订阅的 QoS 级别：`0`、`1` 或 `2` |

### 环境变量

所有字段均可通过环境变量配置：

| 环境变量 | 对应字段 |
|----------|----------|
| `PICOCLAW_CHANNELS_MQTT_BROKER` | `broker` |
| `PICOCLAW_CHANNELS_MQTT_AGENT_ID` | `agent_id` |
| `PICOCLAW_CHANNELS_MQTT_TOPIC_PREFIX` | `topic_prefix` |
| `PICOCLAW_CHANNELS_MQTT_USERNAME` | `username` |
| `PICOCLAW_CHANNELS_MQTT_PASSWORD` | `password` |
| `PICOCLAW_CHANNELS_MQTT_CLIENT_ID` | `client_id` |
| `PICOCLAW_CHANNELS_MQTT_KEEP_ALIVE` | `keep_alive` |
| `PICOCLAW_CHANNELS_MQTT_QOS` | `qos` |

---

## 🔄 断线重连

连接断开后 PicoClaw 会自动以 5 秒间隔重连 Broker，重连成功后自动重新订阅。断线重连时复用相同的 Broker 客户端 ID，Broker 能正确识别为同一连接。

---

## ⚠️ 注意事项

- **TLS**：支持 SSL/TLS（Broker 地址使用 `ssl://`），默认跳过证书验证。
- **流式响应**：流式输出时会向 response topic 发送多条消息，客户端按顺序拼接即为完整回复。
- **client_id 与会话 ID 的区别**：topic 路径中的 `client_id` 由客户端应用自行设置，用于区分会话；它与 PicoClaw paho 连接 Broker 时使用的客户端 ID 是两个独立的概念。
- **多实例部署**：若多个 PicoClaw 实例使用相同 `agent_id` 连接同一 Broker，需为每个实例配置不同的 `client_id` 以避免 Broker 层面的冲突。
