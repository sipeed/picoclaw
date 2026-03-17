# 钉钉

钉钉是阿里巴巴的企业通讯平台，在中国职场中广受欢迎。它采用流式 SDK 来维持持久连接。

## 配置

```json
{
  "channels": {
    "dingtalk": {
      "enabled": true,
      "client_id": "YOUR_CLIENT_ID",
      "client_secret": "YOUR_CLIENT_SECRET",
      "allow_from": [],
      "proactive_send": false,
      "group_trigger": {
        "mention_only": false,
        "prefixes": []
      },
      "reasoning_channel_id": ""
    }
  }
}
```

| 字段                | 类型   | 必填 | 描述                                         |
| ------------------- | ------ | ---- | -------------------------------------------- |
| enabled             | bool   | 是   | 是否启用钉钉频道                             |
| client_id           | string | 是   | 钉钉应用的 Client ID                         |
| client_secret       | string | 是   | 钉钉应用的 Client Secret                     |
| allow_from          | array  | 否   | 用户ID白名单，空表示允许所有用户             |
| proactive_send      | bool   | 否   | 是否启用主动消息发送（通过机器人API）        |
| group_trigger       | object | 否   | 群聊触发配置                                 |
| reasoning_channel_id| string | 否   | 推理消息发送的目标频道ID                     |

### 群聊触发配置

| 字段        | 类型     | 描述                                       |
| ----------- | -------- | ------------------------------------------ |
| mention_only| bool     | 是否仅在@提及时响应                        |
| prefixes    | []string | 触发前缀列表，如 `["/ai", "机器人"]`       |

## 功能特性

### 连接稳定性

钉钉频道依赖 SDK 内置的自动重连机制来保持连接稳定：

- **自动重连**：SDK 默认启用 `AutoReconnect`，连接断开时自动重连
- **心跳检测**：SDK 每 120 秒发送心跳包检测连接状态
- **Pong 超时**：心跳发送后 5 秒内未收到响应则触发重连

### 主动消息发送

启用 `proactive_send` 后，机器人可以主动向用户或群组发送消息：

- **智能路由**：优先使用会话 webhook 发送，失效时自动切换到机器人 API
- **Token 自动刷新**：每 5 分钟自动刷新访问令牌，确保 API 可用性
- **支持场景**：
  - 单聊消息：通过 `oToMessages/batchSend` API
  - 群聊消息：通过 `groupMessages/send` API

### 结构化日志

所有钉钉相关的日志均采用结构化格式输出，便于调试和监控：

```log
INFO  dingtalk: DingTalk channel started {"proactive_send": true}
DEBUG dingtalk: Received message {"sender_nick": "张三", "sender_id": "user123", "preview": "你好"}
```

## 设置流程

1. 前往 [钉钉开放平台](https://open.dingtalk.com/)
2. 创建一个企业内部应用
3. 从应用设置中获取 Client ID 和 Client Secret
4. 配置 OAuth 和事件订阅（如需要）
5. 将 Client ID 和 Client Secret 填入配置文件中
6. 如需主动消息功能，设置 `proactive_send: true`

## 环境变量

所有配置项均可通过环境变量设置：

| 环境变量                                      | 对应配置项              |
| --------------------------------------------- | ----------------------- |
| PICOCLAW_CHANNELS_DINGTALK_ENABLED            | enabled                 |
| PICOCLAW_CHANNELS_DINGTALK_CLIENT_ID          | client_id               |
| PICOCLAW_CHANNELS_DINGTALK_CLIENT_SECRET      | client_secret           |
| PICOCLAW_CHANNELS_DINGTALK_ALLOW_FROM         | allow_from              |
| PICOCLAW_CHANNELS_DINGTALK_PROACTIVE_SEND     | proactive_send          |
| PICOCLAW_CHANNELS_DINGTALK_REASONING_CHANNEL_ID| reasoning_channel_id   |
