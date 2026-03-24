> 返回 [README](../../../../README.zh.md)

# 企业微信智能机器人 (AI Bot)

企业微信智能机器人（AI Bot）是企业微信官方提供的 AI 对话接入方式，支持私聊与群聊，内置流式响应协议。PicoClaw 当前同时支持两种接入模式：

- WebSocket 长连接模式：使用 `bot_id` + `secret`，优先级更高，推荐使用
- Webhook 短连接模式：使用 `token` + `encoding_aes_key`，兼容传统回调，并支持超时后通过 `response_url` 主动推送最终回复

## 与其他 WeCom 通道的对比

| 特性 | WeCom Bot | WeCom App | **WeCom AI Bot** |
|------|-----------|-----------|-----------------|
| 私聊 | ✅ | ✅ | ✅ |
| 群聊 | ✅ | ❌ | ✅ |
| 流式输出 | ❌ | ❌ | ✅ |
| 超时主动推送 | ❌ | ✅ | ✅ |
| 配置复杂度 | 低 | 高 | 中 |

## 配置

### WebSocket 长连接模式（推荐）

```json
{
  "channels": {
    "wecom_aibot": {
      "enabled": true,
      "bot_id": "YOUR_BOT_ID",
      "secret": "YOUR_SECRET",
      "allow_from": [],
      "welcome_message": "你好！有什么可以帮助你的吗？",
      "max_steps": 10
    }
  }
}
```

### Webhook 短连接模式

```json
{
  "channels": {
    "wecom_aibot": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_43_CHAR_ENCODING_AES_KEY",
      "webhook_path": "/webhook/wecom-aibot",
      "allow_from": [],
      "welcome_message": "你好！有什么可以帮助你的吗？",
      "processing_message": "⏳ Processing, please wait. The results will be sent shortly.",
      "max_steps": 10
    }
  }
}
```

### WebSocket 模式字段

| 字段   | 类型   | 必填 | 描述                                       |
|--------|--------|------|--------------------------------------------|
| bot_id | string | 是   | AI Bot 的唯一标识，在 AI Bot 管理页面配置 |
| secret | string | 是   | AI Bot 的密钥，在 AI Bot 管理页面配置     |

### Webhook 模式字段

| 字段             | 类型   | 必填 | 描述                                         |
|------------------|--------|------|----------------------------------------------|
| token            | string | 是   | 回调验证令牌，在 AI Bot 管理页面配置         |
| encoding_aes_key | string | 是   | 43 字符 AES 密钥，在 AI Bot 管理页面随机生成 |
| webhook_path     | string | 否   | Webhook 路径，默认 `/webhook/wecom-aibot`    |
| processing_message | string | 否 | 流式超时后返回给用户的提示语                 |

### 通用字段

| 字段            | 类型   | 必填 | 描述                                     |
|-----------------|--------|------|------------------------------------------|
| allow_from      | array  | 否   | 用户 ID 白名单，空数组表示允许所有用户   |
| welcome_message | string | 否   | 用户进入聊天时发送的欢迎语，留空则不发送 |
| reply_timeout   | int    | 否   | 回复超时时间（秒，默认：5）              |
| max_steps       | int    | 否   | Agent 最大执行步骤数（默认：10）         |

## 模式选择

- 当 `bot_id` 和 `secret` 同时存在时，PicoClaw 会优先使用 WebSocket 长连接模式
- 否则，当 `token` 和 `encoding_aes_key` 同时存在时，PicoClaw 会使用 Webhook 短连接模式

## 设置流程

### WebSocket 长连接模式

1. 登录 [企业微信管理后台](https://work.weixin.qq.com/wework_admin)
2. 进入"应用管理" → "智能机器人"，创建或选择一个 AI Bot
3. 在 AI Bot 配置页面，配置 Bot 的名称、头像等信息，获取 `Bot ID` 和 `Secret`
4. 在 PicoClaw 配置文件中添加上述配置，重启 PicoClaw

### Webhook 短连接模式

1. 登录 [企业微信管理后台](https://work.weixin.qq.com/wework_admin)
2. 进入"应用管理" → "智能机器人"，创建或选择一个 AI Bot
3. 在 AI Bot 配置页面，填写"消息接收"信息：
   - **URL**：`http://<your-server-ip>:18790/webhook/wecom-aibot`
   - **Token**：随机生成或自定义
   - **EncodingAESKey**：点击"随机生成"，得到 43 字符密钥
4. 将 Token 和 EncodingAESKey 填入 PicoClaw 配置文件，启动服务后回到管理后台保存

> [!TIP]
> 服务器需要能被企业微信服务器访问。如在内网或本地开发，可使用 [ngrok](https://ngrok.com) 或 frp 做内网穿透。

## Webhook 模式的流式响应协议

Webhook 模式使用"流式拉取"协议，区别于普通 Webhook 的一次性回复：

```
用户发消息
  │
  ▼
PicoClaw 立即返回 {finish: false}（Agent 开始处理）
  │
  ▼
企业微信每隔约 1 秒拉取一次 {msgtype: "stream", stream: {id: "..."}}
  │
  ├─ Agent 未完成 → 返回 {finish: false}（继续等待）
  │
  └─ Agent 完成 → 返回 {finish: true, content: "回答内容"}
```

**超时处理**（任务超过约 30 秒）：

若 Agent 处理时间超过轮询窗口，PicoClaw 会：

1. 立即关闭流，向用户显示 `processing_message` 提示语
2. Agent 继续在后台运行
3. Agent 完成后，通过消息中携带的 `response_url` 将最终回复主动推送给用户

> `response_url` 由企业微信颁发，有效期 1 小时，只可使用一次，无需加密，直接 POST markdown 消息体即可。

## 超时提示语

配置 `processing_message` 后，当 Webhook 模式的流式轮询超时并切换到 `response_url` 主动推送模式时，PicoClaw 会先返回这段提示语来结束当前流。

```json
"processing_message": "⏳ Processing, please wait. The results will be sent shortly."
```

## 欢迎语

配置 `welcome_message` 后，当用户打开与 AI Bot 的聊天窗口时（`enter_chat` 事件），PicoClaw 会自动回复该欢迎语。留空则静默忽略。

```json
"welcome_message": "你好！我是 PicoClaw AI 助手，有什么可以帮你？"
```

## 常见问题

### WebSocket 模式无法连接

- 检查 `bot_id` 和 `secret` 是否填写正确
- 查看日志中是否有 WebSocket 连接或鉴权失败信息
- 确认服务器可以访问企业微信长连接接口

### 回调 URL 验证失败


- 确认 `token` 与 `encoding_aes_key` 填写正确
- 确认服务器防火墙已开放对应端口
- 检查 PicoClaw 日志是否收到了来自企业微信的验证请求

### 消息没有回复

- 检查 `allow_from` 是否意外限制了发送者
- 查看日志中是否出现 `context canceled` 或 Agent 错误
- 确认 Agent 配置（`model_name` 等）正确

### 超长任务没有收到最终推送

- 确认消息回调中携带了 `response_url`
- 确认服务器能主动访问外网
- 查看日志关键词 `response_url mode` 和 `Sending reply via response_url`

## 参考文档

- [企业微信 AI Bot 接入文档](https://developer.work.weixin.qq.com/document/path/101463)
- [流式响应协议说明](https://developer.work.weixin.qq.com/document/path/100719)
- [response_url 主动回复](https://developer.work.weixin.qq.com/document/path/101138)
