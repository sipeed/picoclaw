# Mattermost

Mattermost 是常见的团队协作平台。PicoClaw 使用 Mattermost 的 WebSocket + REST API 实现实时消息收发、输入中状态、占位消息与附件处理。

## 配置

```json
{
  "channels": {
    "mattermost": {
      "enabled": true,
      "url": "https://your-mattermost.example.com",
      "bot_token": "YOUR_MATTERMOST_BOT_TOKEN",
      "allow_from": [],
      "group_trigger": {
        "mention_only": true
      },
      "typing": {
        "enabled": true
      },
      "placeholder": {
        "enabled": false,
        "text": "Thinking..."
      },
      "reasoning_channel_id": ""
    }
  }
}
```

| 字段                 | 类型   | 必填 | 描述 |
| -------------------- | ------ | ---- | ---- |
| enabled              | bool   | 是   | 是否启用 Mattermost 渠道 |
| url                  | string | 是   | Mattermost 服务地址（例如 `https://chat.example.com`） |
| bot_token            | string | 是   | Bot Access Token |
| allow_from           | array  | 否   | 用户白名单（Mattermost 用户 ID），空表示允许所有用户 |
| group_trigger        | object | 否   | 群组触发策略（`mention_only` / `prefixes`） |
| typing               | object | 否   | 输入中状态配置 |
| placeholder          | object | 否   | 占位消息配置（先发占位，后编辑为最终回复） |
| reasoning_channel_id | string | 否   | 思考/推理输出目标频道 ID |

## 设置流程

1. 在 Mattermost 系统控制台中启用 Bot Accounts 功能（如果尚未启用）
2. 创建 Bot 账号并复制访问 Token
3. 把 Bot 添加到需要响应的频道/群组
4. 在 `config.json` 中填写 `url` 与 `bot_token`
5. 启动 `picoclaw gateway`

## 行为说明

- 私聊（DM）默认会响应
- 群组/频道消息默认可响应；可通过 `group_trigger.mention_only=true` 改为仅 @ 触发
- 连接异常时会自动重连，重连成功后继续收发消息

## 常见问题

1. 启动时报 `no channels enabled`

- 确认 `channels.mattermost.enabled=true`
- 确认 `url` 与 `bot_token` 非空
- 确认实际加载的是目标配置文件（`PICOCLAW_CONFIG` / `PICOCLAW_HOME` 是否覆盖）

2. 频道内不响应

- 检查 `allow_from` 是否限制了用户
- 如果启用了 `mention_only`，请确认消息中包含对 bot 的 @ 提及
- 确认 bot 已加入对应频道并具备发言权限
