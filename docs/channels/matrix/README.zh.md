> 返回 [README](../../../README.zh.md)

# Matrix 通道配置指南

## 1. 配置示例

在 `config.json` 中添加：

```json
{
  "channel_list": {
    "matrix": {
      "enabled": true,
      "type": "matrix",
      "homeserver": "https://matrix.org",
      "user_id": "@your-bot:matrix.org",
      "access_token": "YOUR_MATRIX_ACCESS_TOKEN",
      "device_id": "",
      "join_on_invite": true,
      "allow_from": [],
      "group_trigger": {
        "mention_only": true
      },
      "placeholder": {
        "enabled": true,
        "text": ["Thinking...", "Processing...", "Typing..."]
      },
      "reasoning_channel_id": "",
      "message_format": "richtext",
      "crypto_database_path": "",
      "crypto_passphrase": "YOUR_MATRIX_CRYPTO_PICKLE_KEY"
    }
  }
}
```

## 2. 参数说明

| 字段                 | 类型     | 必填 | 说明 |
|----------------------|----------|------|------|
| enabled              | bool     | 是   | 是否启用 Matrix 通道 |
| homeserver           | string   | 是   | Matrix 服务器地址（例如 `https://matrix.org`） |
| user_id              | string   | 是   | 机器人 Matrix 用户 ID（例如 `@bot:matrix.org`） |
| access_token         | string   | 是   | 机器人 access token |
| device_id            | string   | 否   | 设备 ID（可选） |
| join_on_invite       | bool     | 否   | 是否自动加入邀请房间 |
| allow_from           | []string | 否   | 白名单用户（Matrix 用户 ID） |
| group_trigger        | object   | 否   | 群聊触发策略（支持 `mention_only` / `prefixes`） |
| placeholder          | object   | 否   | 占位消息配置 |
| reasoning_channel_id | string   | 否   | 思维链输出目标通道 |
| message_format       | string   | 否   | 消息格式：`richtext`（富文本）或 `plain`（纯文本） |
| crypto_database_path | string   | 否   | 加密数据库存储路径（为空时使用工作空间路径 `~/.picoclaw/workspace`） |
| crypto_passphrase    | string   | 否   | 加密数据库中 session key 的序列化密钥；设置后不能更改 |

### 占位消息配置 (Placeholder)

| 字段    | 类型            | 必填 | 说明 |
|---------|-----------------|------|------|
| enabled | bool            | 否   | 是否启用占位消息（默认：false） |
| text    | string/[]string | 否   | 占位文本。可以是单个字符串或字符串数组。如果提供多个文本，运行时会随机选择一个。默认："Thinking..." |

## 3. 当前支持

- 文本消息收发
- 图片/音频/视频/文件消息入站下载（写入 MediaStore / 本地路径回退）
- 音频消息按统一标记进入现有转写流程（`[audio: ...]`）
- 图片/音频/视频/文件消息出站发送（上传到 Matrix 媒体库后发送）
- 群聊触发规则（支持仅 @ 提及时响应）
- Typing 状态（`m.typing`）
- 占位消息（`Thinking... 💭`）+ 最终回复替换
- 自动加入邀请房间（可关闭）
- 端对端加密（E2EE）消息支持

## 4. TODO

- 富媒体细节增强（如 image/video 的尺寸、缩略图等 metadata）
