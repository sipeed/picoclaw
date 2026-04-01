> 返回 [README](../../../README.zh.md)

# 企业微信（WeCom）

PicoClaw 将企业微信整合为单一的 `channels.wecom` 渠道，基于腾讯官方企业微信 AI Bot WebSocket API 实现。
原有的 `wecom`、`wecom_app`、`wecom_aibot` 三个独立渠道已合并为统一配置模型。

> 本渠道无需公网 Webhook 回调地址。PicoClaw 主动向企业微信建立出站 WebSocket 连接。

## 支持的功能

- 单聊和群聊消息收发
- 基于企业微信 AI Bot 协议的流式回复
- 接收文本、语音、图片、文件、视频及混合消息
- 发送文本及媒体消息（`image`、`file`、`voice`、`video`）
- 通过 Web UI 或 CLI 扫码绑定
- 发送者白名单和 `reasoning_channel_id` 路由

---

## 快速开始

### 方式一：Web UI 扫码绑定（推荐）

打开 Web UI，进入 **Channels → WeCom**，点击扫码绑定按钮。用企业微信扫码并在 App 内确认，凭据自动保存。

<p align="center">
<img src="../../../assets/wecom-qr-binding.jpg" alt="Web UI 企业微信扫码绑定" width="600">
</p>

### 方式二：CLI 扫码登录

运行：

```bash
picoclaw auth wecom
```

命令执行流程：
1. 向企业微信请求二维码并在终端打印
2. 同时打印一个**二维码链接**，终端二维码不清晰时可在浏览器中打开
3. 轮询确认状态——扫码后还需要在**企业微信 App 内点击确认**
4. 成功后将 `bot_id` 和 `secret` 写入 `channels.wecom` 并保存配置

默认超时为 **5 分钟**，可通过 `--timeout` 延长：

```bash
picoclaw auth wecom --timeout 10m
```

> ⚠️ 仅扫描二维码还不够——必须在企业微信 App 内点击**确认**，否则命令会超时。

### 方式三：手动配置

如果已有企业微信 AI Bot 的 `bot_id` 和 `secret`，可直接配置：

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

---

## 配置项说明

| 字段 | 类型 | 默认值 | 说明 |
| ---- | ---- | ------ | ---- |
| `enabled` | bool | `false` | 启用企业微信渠道。 |
| `bot_id` | string | — | 企业微信 AI Bot 标识符。启用时必填。 |
| `secret` | string | — | 企业微信 AI Bot 密钥。加密存储于 `.security.yml`。启用时必填。 |
| `websocket_url` | string | `wss://openws.work.weixin.qq.com` | 企业微信 WebSocket 端点。 |
| `send_thinking_message` | bool | `true` | 在流式回复开始前发送"处理中..."提示消息。 |
| `allow_from` | array | `[]` | 发送者白名单。为空时允许所有人。 |
| `reasoning_channel_id` | string | `""` | 可选，将推理/思考内容路由到指定会话 ID。 |

### 环境变量

所有字段均可通过 `PICOCLAW_CHANNELS_WECOM_` 前缀的环境变量覆盖：

| 环境变量 | 对应字段 |
| -------- | -------- |
| `PICOCLAW_CHANNELS_WECOM_ENABLED` | `enabled` |
| `PICOCLAW_CHANNELS_WECOM_BOT_ID` | `bot_id` |
| `PICOCLAW_CHANNELS_WECOM_SECRET` | `secret` |
| `PICOCLAW_CHANNELS_WECOM_WEBSOCKET_URL` | `websocket_url` |
| `PICOCLAW_CHANNELS_WECOM_SEND_THINKING_MESSAGE` | `send_thinking_message` |
| `PICOCLAW_CHANNELS_WECOM_ALLOW_FROM` | `allow_from` |
| `PICOCLAW_CHANNELS_WECOM_REASONING_CHANNEL_ID` | `reasoning_channel_id` |

---

## 运行时行为

- PicoClaw 维护活跃的企业微信 Turn，流式回复尽可能在同一流上继续。
- 流式回复最大持续时长为 **5.5 分钟**，最小发送间隔为 **500ms**。
- 流式不可用时，回复降级为主动推送。
- 会话路由关联在 **30 分钟**无活动后过期。
- 接收到的媒体文件先下载到本地媒体存储，再传递给 Agent。
- 发送媒体时先上传为企业微信临时文件，再作为媒体消息发送。
- 自动检测并过滤重复消息（环形缓冲区，最多记录 1000 条消息 ID）。

---

## 从旧版企业微信配置迁移

| 旧配置 | 迁移方式 |
| ------ | -------- |
| `channels.wecom`（Webhook 机器人） | 改用 `channels.wecom`，填写 `bot_id` + `secret`。 |
| `channels.wecom_app` | 删除，改用 `channels.wecom`。 |
| `channels.wecom_aibot` | 将 `bot_id` 和 `secret` 移至 `channels.wecom`。 |
| `token`、`encoding_aes_key`、`webhook_url`、`webhook_path` | 已废弃，从配置中删除。 |
| `corp_id`、`corp_secret`、`agent_id` | 已废弃，从配置中删除。 |
| `welcome_message`、`processing_message`、`max_steps` | 已不属于企业微信渠道配置，删除即可。 |

---

## 常见问题

### 扫码绑定超时

- 扫码后必须在**企业微信 App 内点击确认**，仅扫码不够。
- 使用更长的超时重试：`picoclaw auth wecom --timeout 10m`
- 终端二维码不清晰时，使用命令打印的**二维码链接**在浏览器中打开。

### 二维码已过期

- 二维码有效期有限，重新运行 `picoclaw auth wecom` 获取新二维码。

### WebSocket 连接失败

- 检查 `bot_id` 和 `secret` 是否正确。
- 确认设备可以访问 `wss://openws.work.weixin.qq.com`（出站 WebSocket，无需开放入站端口）。

### 收不到回复

- 检查 `allow_from` 是否屏蔽了发送者。
- 确认 `channels.wecom.bot_id` 和 `channels.wecom.secret` 已填写且非空。
