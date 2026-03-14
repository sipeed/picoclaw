# 企业微信智能机器人 (AI Bot)

企业微信智能机器人（AI Bot）是企业微信官方提供的 AI 对话接入方式，支持私聊与群聊，内置流式响应协议。

## 与其他 WeCom 通道的对比

| 特性 | WeCom Bot | WeCom App | **WeCom AI Bot** |
|------|-----------|-----------|-----------------|
| 私聊 | ✅ | ✅ | ✅ |
| 群聊 | ✅ | ❌ | ✅ |
| 流式输出 | ❌ | ❌ | ✅ |
| 超时主动推送 | ❌ | ✅ | ✅ |
| 配置复杂度 | 低 | 高 | 中 |

## 配置

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

| 字段             | 类型   | 必填 | 描述                                               |
| ---------------- | ------ | ---- | -------------------------------------------------- |
| bot_id           | string | 是   | AI Bot 的唯一标识，在 AI Bot 管理页面配置         |
| secret           | string | 是   | AI Bot 的密钥，在 AI Bot 管理页面配置             |
| allow_from       | array  | 否   | 用户 ID 白名单，空数组表示允许所有用户             |
| welcome_message  | string | 否   | 用户进入聊天时发送的欢迎语，留空则不发送           |
| reply_timeout    | int    | 否   | 回复超时时间（秒，默认：5）                        |
| max_steps        | int    | 否   | Agent 最大执行步骤数（默认：10）                   |

## 设置流程

1. 登录 [企业微信管理后台](https://work.weixin.qq.com/wework_admin)
2. 进入"应用管理" → "智能机器人"，创建或选择一个 AI Bot
3. 在 AI Bot 配置页面，配置Bot的名称、头像等信息，获取 `Bot ID` 和 `Secret`
4. 在 PicoClaw 配置文件中添加上述配置，重启 PicoClaw

## 欢迎语

配置 `welcome_message` 后，当用户打开与 AI Bot 的聊天窗口时（`enter_chat` 事件），PicoClaw 会自动回复该欢迎语。留空则静默忽略。

```json
"welcome_message": "你好！我是 PicoClaw AI 助手，有什么可以帮你？"
```

## 常见问题

### 消息没有回复

- 检查 `allow_from` 是否意外限制了发送者
- 查看日志中是否出现 `context canceled` 或 Agent 错误
- 确认 Agent 配置（`model_name` 等）正确

## 参考文档

- [企业微信 AI Bot 接入文档](https://developer.work.weixin.qq.com/document/path/101463)
