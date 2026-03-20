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
      "group_trigger": {
        "mention_only": false,
        "prefixes": []
      }
    }
  }
}
```

| 字段          | 类型   | 必填 | 描述                             |
| ------------- | ------ | ---- | -------------------------------- |
| enabled       | bool   | 是   | 是否启用钉钉频道                 |
| client_id     | string | 是   | 钉钉应用的 AppKey（也作为 robotCode 使用） |
| client_secret | string | 是   | 钉钉应用的 AppSecret             |
| allow_from    | array  | 否   | 用户ID白名单，空表示允许所有用户 |
| group_trigger | object | 否   | 群聊触发配置                     |

## 设置流程

1. 前往 [钉钉开放平台](https://open.dingtalk.com/)
2. 创建一个企业内部应用
3. 从应用设置中获取 AppKey（Client ID）和 AppSecret（Client Secret）
4. 配置机器人的回调模式和事件订阅
5. 将 AppKey 和 AppSecret 填入配置文件中

## 消息发送机制

钉钉频道支持两种消息发送方式：

### 1. Session Webhook 回复（优先）

当用户发送消息给机器人时，钉钉会提供一个临时的 `session_webhook`，有效期约 2 小时。系统优先使用此方式回复消息，因为它更简单且不需要额外的 API 调用。

### 2. 主动消息（Proactive Messaging）

当 `session_webhook` 不可用或已过期时，系统会自动切换到钉钉 OpenAPI 发送主动消息：

- **单聊消息**: 使用 `/v1.0/robot/oToMessages/batchSend` API
- **群聊消息**: 使用 `/v1.0/robot/groupMessages/send` API

主动消息功能使得以下场景成为可能：
- 心跳通知（Heartbeat）
- 设备告警
- 定时任务提醒
- 其他无需用户先发起对话的消息推送

### 主动消息的工作原理

1. **首次交互**: 用户发送消息后，系统会存储用户的 `staffId`（单聊）或 `openConversationId`（群聊）
2. **后续推送**: 即使 `session_webhook` 过期，系统仍可通过 OpenAPI 主动发送消息
3. **无历史交互**: 如果用户从未发送过消息，只要知道用户的 `staffId`，系统也可以主动发送单聊消息

## API 参考

| API | 用途 |
|-----|------|
| `POST /v1.0/oauth2/accessToken` | 获取访问令牌 |
| `POST /v1.0/robot/oToMessages/batchSend` | 发送单聊消息 |
| `POST /v1.0/robot/groupMessages/send` | 发送群聊消息 |

### 获取访问令牌

```http
POST https://api.dingtalk.com/v1.0/oauth2/accessToken
Content-Type: application/json

{
  "appKey": "YOUR_APP_KEY",
  "appSecret": "YOUR_APP_SECRET"
}
```

响应：
```json
{
  "accessToken": "xxx",
  "expireIn": 7200
}
```

### 发送单聊消息

```http
POST https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend
Content-Type: application/json
X-Acs-Dingtalk-Access-Token: ACCESS_TOKEN

{
  "robotCode": "YOUR_APP_KEY",
  "userIds": ["STAFF_ID"],
  "msgKey": "sampleMarkdown",
  "msgParam": "{\"title\":\"标题\",\"text\":\"内容\"}"
}
```

### 发送群聊消息

```http
POST https://api.dingtalk.com/v1.0/robot/groupMessages/send
Content-Type: application/json
X-Acs-Dingtalk-Access-Token: ACCESS_TOKEN

{
  "robotCode": "YOUR_APP_KEY",
  "openConversationId": "CONVERSATION_ID",
  "msgKey": "sampleMarkdown",
  "msgParam": "{\"title\":\"标题\",\"text\":\"内容\"}"
}
```

## 官方文档

- [钉钉机器人开发文档](https://open.dingtalk.com/document/orgapp/the-robot-sends-a-group-message)
- [获取访问令牌](https://open.dingtalk.com/document/development/obtain-the-access-token-of-an-internal-app)
