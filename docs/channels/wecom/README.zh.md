> 返回 [README](../../../README.zh.md)

# 企业微信

PicoClaw 现在将企业微信统一为一个 `channels.wecom` 渠道，并基于企业微信官方 AI Bot WebSocket 协议实现。
这取代了旧的 `wecom`、`wecom_app`、`wecom_aibot` 三套配置模型。

## 当前渠道能力

- 支持私聊和群聊
- 支持企业微信侧流式回复
- 支持接收文本、语音、图片、文件、视频和 mixed 消息
- 支持发送文本与媒体消息（`image`、`file`、`voice`、`video`）
- 支持通过 `picoclaw auth wecom` 扫码写入配置
- 支持统一白名单与 `reasoning_channel_id`

> 这个渠道不再需要公网 webhook 回调地址。PicoClaw 会主动向企业微信发起 WebSocket 连接。

## 快速开始

### 方式 1：命令行扫码登录

运行：

```bash
picoclaw auth wecom
```

该命令会在终端打印二维码，等待你在企业微信中确认，然后把生成的 `bot_id` 和 `secret` 写入
`channels.wecom`。

如果需要更长等待时间，可以加 `--timeout`：

```bash
picoclaw auth wecom --timeout 10m
```

### 方式 2：手动配置

```json
{
  "channels": {
    "wecom": {
      "enabled": true,
      "bot_id": "YOUR_BOT_ID",
      "secret": "YOUR_SECRET",
      "websocket_url": "wss://openws.work.weixin.qq.com",
      "send_thinking_message": true,
      "allow_from": [],
      "reasoning_channel_id": ""
    }
  }
}
```

## 配置字段

| 字段 | 类型 | 必填 | 说明 |
| ---- | ---- | ---- | ---- |
| `enabled` | bool | 否 | 是否启用企业微信渠道。 |
| `bot_id` | string | 是 | 企业微信 AI Bot 标识。渠道启用时必填。 |
| `secret` | string | 是 | 企业微信 AI Bot 密钥。渠道启用时必填。 |
| `websocket_url` | string | 否 | WebSocket 地址，默认 `wss://openws.work.weixin.qq.com`。 |
| `send_thinking_message` | bool | 否 | 是否在流式最终回复前先发送一段 `Processing...` 开场消息，默认 `true`。 |
| `allow_from` | array | 否 | 发送者白名单；空数组表示允许所有发送者。 |
| `reasoning_channel_id` | string | 否 | 可选的 reasoning/thinking 输出目标。 |

## 运行时行为

- PicoClaw 会保留当前会话对应的企业微信 turn，优先继续同一个流式回复。
- 如果流式上下文已经失效，回复会自动回退到主动推送消息。
- 收到的媒体会先下载到 media store，再交给 Agent 处理。
- 发出的媒体会先按分片上传到企业微信，再作为普通媒体消息发送。

## 迁移说明

这个分支移除了旧的多通道企业微信模型。

| 旧配置 | 现在怎么做 |
| ------ | ---------- |
| `channels.wecom` webhook 机器人 | 改为使用 `bot_id` + `secret` 的 `channels.wecom`。 |
| `channels.wecom_app` | 删除，统一迁移到 `channels.wecom`。 |
| `channels.wecom_aibot` | 配置迁移到 `channels.wecom`。 |
| `token`、`encoding_aes_key`、`webhook_url`、`webhook_path` | 企业微信渠道不再使用这些字段。 |
| `corp_id`、`corp_secret`、`agent_id` | 企业微信渠道不再使用这些字段。 |
| 企业微信下的 `welcome_message`、`processing_message`、`max_steps` | 不再属于企业微信渠道配置。 |

## 常见问题

### `picoclaw auth wecom` 超时

- 用更大的 `--timeout` 重新执行。
- 确认是在企业微信里完成了确认，而不只是扫描二维码。

### WebSocket 连接失败

- 检查 `bot_id` 和 `secret` 是否正确。
- 确认运行环境可以访问 `wss://openws.work.weixin.qq.com`。

### 消息没有回到企业微信

- 检查 `allow_from` 是否拦截了发送者。
- 检查启动日志或 launcher 校验，确认 `channels.wecom.bot_id` / `channels.wecom.secret` 已填写。

