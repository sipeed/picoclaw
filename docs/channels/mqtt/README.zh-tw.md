# 📡 MQTT 頻道

PicoClaw 支援將任何 MQTT 使用者端作為聊天頻道。裝置或服務會將要求發布至 Broker；PicoClaw 訂閱並處理要求，再發布回應。

## 🚀 快速開始

**1. 將頻道加入 `~/.picoclaw/config.json`：**

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

**2. 啟動閘道服務：**

```bash
picoclaw gateway
```

**3. 從任一 MQTT 使用者端傳送訊息：**

```bash
mosquitto_pub -t "/picoclaw/assistant/device1/request" \
  -m '{"text": "What is the CPU usage?"}'
```

**4. 訂閱以接收回應：**

```bash
mosquitto_sub -t "/picoclaw/assistant/device1/response"
```

---

## 📨 Topic 結構

```
{prefix}/{agent_id}/{client_id}/request    # Client → PicoClaw
{prefix}/{agent_id}/{client_id}/response   # PicoClaw → Client
```

| 區段 | 說明 |
|------|------|
| `prefix` | 在伺服器端設定的 Topic 前置字串。預設值：`/picoclaw` |
| `agent_id` | PicoClaw 執行個體識別碼，設定於 `agent_id` 欄位 |
| `client_id` | 使用者端定義的工作階段識別碼；每部裝置請使用固定 ID，以保留對話上下文 |

### 訊息內容（JSON）

```json
{ "text": "your message here" }
```

---

## ⚙️ 設定

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

### .security.yml（認證資訊）

使用者名稱與密碼會儲存在 `~/.picoclaw/.security.yml`，而不是 `config.json`：

```yaml
channel_list:
  mqtt:
    settings:
      username: your_username
      password: your_password
```

### 設定欄位

| 欄位 | 位置 | 必填 | 預設值 | 說明 |
|------|------|------|--------|------|
| `broker` | `settings` | 是 | — | MQTT Broker URL，例如 `tcp://host:1883`、`ssl://host:8883` |
| `agent_id` | `settings` | 是 | — | 代理人識別碼，用於 Topic 路徑的一部分 |
| `topic_prefix` | `settings` | 否 | `/picoclaw` | Topic 命名空間的前置字串 |
| `username` | `.security.yml` | 否 | — | Broker 驗證使用者名稱 |
| `password` | `.security.yml` | 否 | — | Broker 驗證密碼 |
| `client_id` | `settings` | 否 | 自動產生 | 傳送至 Broker 的 Paho Client ID。未設定時會自動產生 `picoclaw-mqtt-{agent_id}-{8-char hex}`；此 ID 在處理程序存續期間維持不變，重新連線時會沿用相同 ID |
| `keep_alive` | `settings` | 否 | `60` | MQTT Keepalive 間隔秒數 |
| `qos` | `settings` | 否 | `0` | 發布及訂閱使用的 QoS 等級：`0`、`1` 或 `2` |

### 環境變數

所有欄位都能使用環境變數設定：

| 變數 | 欄位 |
|------|------|
| `PICOCLAW_CHANNELS_MQTT_BROKER` | `broker` |
| `PICOCLAW_CHANNELS_MQTT_AGENT_ID` | `agent_id` |
| `PICOCLAW_CHANNELS_MQTT_TOPIC_PREFIX` | `topic_prefix` |
| `PICOCLAW_CHANNELS_MQTT_USERNAME` | `username` |
| `PICOCLAW_CHANNELS_MQTT_PASSWORD` | `password` |
| `PICOCLAW_CHANNELS_MQTT_CLIENT_ID` | `client_id` |
| `PICOCLAW_CHANNELS_MQTT_KEEP_ALIVE` | `keep_alive` |
| `PICOCLAW_CHANNELS_MQTT_QOS` | `qos` |

---

## 🔄 重新連線

連線中斷時，PicoClaw 會每隔 5 秒自動嘗試重新連線至 Broker。重新連線後，PicoClaw 也會自動重新訂閱。每次重新連線都會沿用相同的 Broker 端 Client ID，讓 Broker 能正確識別為同一個工作階段。

---

## ⚠️ 注意事項

- **TLS**：支援 SSL／TLS（`ssl://` Broker URL），預設會略過憑證驗證。
- **串流**：串流回覆會向 Response Topic 傳送多則訊息；請依序串接這些訊息。
- **client_id 與工作階段 ID**：Topic 路徑中的 `client_id` 由使用者端應用程式設定，用來識別對話工作階段。此 ID 與 PicoClaw 的 Paho 連線所使用的 Broker 層級 Client ID 不同。
- **多個執行個體**：如果多個 PicoClaw 執行個體使用相同的 `agent_id` 連線至同一個 Broker，請分別設定不同的 `client_id`，避免 Broker 層級發生衝突。
