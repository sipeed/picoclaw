# QQ C2C Channel · 使用与迁移指南

> 配套：[设计文档](./qq-c2c-channel.md) ｜ 通道类型：`qq_c2c` ｜ 实现：`pkg/channels/qqc2c/`
> 面向使用者的部分：配置项、能力边界、从旧 `qq` 通道迁移、常见问题。

---

## 1. 最小配置

```jsonc
"channel_list": {
  "qq_c2c": {
    "enabled": true,
    "type": "qq_c2c",
    "allow_from": ["<你的 user_openid>"],   // 建议填；留空会告警「Channel allows EVERYONE」
    "settings": {
      "app_id": "102xxxxxx",
      "app_secret": "***"
    }
  }
}
```

凭据可以放在 `.security.yml`（与其它通道一致）：

```yaml
channel_list:
  qq_c2c:
    settings:
      app_secret: <secret>
```

## 2. 配置项

| 键 | 类型 | 默认 | 说明 |
|---|---|---|---|
| `app_id` | string | — | QQ 开放平台的 AppID（必填） |
| `app_secret` | secret | — | AppSecret（必填，建议走 `.security.yml`/env） |
| `max_message_length` | int | `0` | 出站分片阈值（0=不限） |
| `markdown` | bool | `false` | 用 `msg_type=2` 发 Markdown（单聊**支持发**） |
| `streaming` | bool | `false` | 启用流式输出（`/stream_messages`） |
| `stream_interval_ms` | int | `0` | 流式增量合并节流（建议 `300`；平台限 50 QPS） |
| `typing` | bool | `false` | 发 `msg_type=6` 输入中状态 |
| `command_panel` | bool | `false` | 允许渲染 Markdown 指令面板标签 |
| `reply_window_minutes` | int | `60` | 平台限制：被动回复窗口 |
| `max_replies_per_message` | int | `4` | 平台限制：每条入站消息最多回几次 |
| `api_base` | string | 官方 | 覆盖 OpenAPI 域名（调试用） |
| `intents` | int | `(1<<25)\|(1<<26)` | 覆盖 WS 订阅位（默认：单聊事件 + 互动） |

环境变量形式：`PICOCLAW_CHANNELS_QQ_C2C_APP_ID` / `..._APP_SECRET` / `..._MARKDOWN` / `..._STREAMING` 等。

## 3. 两种发消息方式（同一个接口）

| 方式 | 条件 | 平台限制 |
|---|---|---|
| **被动回复** | 携带 `msg_id`（入站事件 `d.id`） | 60 分钟内、每条入站消息最多 **4** 次 |
| **主动消息** | **不带** `msg_id`/`event_id` | 认证机器人 10/qps、单关系 20/qpm、**每日 1000 条/用户**；用户可在 QQ 客户端关闭「允许主动发送」开关（关闭后报 `40054013 用户拒收消息`；机器人无权限报 `40034105`，超频报 `40034100`） |
| **互动召回** | `is_wakeup=true` | 用户互动后 30 天内每周期 1 条（当天 / 1-3 天 / 3-7 天 / 7-30 天） |

代码里对应：
- `Send()` —— 有 `msg_id` 走被动、没有就走**主动消息**（自动判断，无需调用方关心）
- `SendActive(ctx, openid, text)` —— 显式主动推送
- `SendWakeup(ctx, openid, text)` —— 互动召回
- 通道会跟踪用户的「允许主动消息」开关（`C2C_MSG_RECEIVE` / `C2C_MSG_REJECT` 事件）

## 4. 能力矩阵（单聊）

| 内容类型 | 单聊 |
|---|---|
| 文本 | 收发 ✅ |
| Markdown | **发** ✅ / 收 ❌ |
| 图片 · 视频 · 语音 · 文件 | 收发 ✅（需先上传拿 `file_info`） |
| 结构化卡片 | 发 ❌ / 收 ✅ |
| 引用消息 | 收 ✅ |
| 消息按钮（keyboard） | 发 ✅（单聊已开放自定义按钮）；点击回调 `INTERACTION_CREATE` |
| 指令面板（`<qqbot-cmd-enter>` / `<qqbot-cmd-input>`） | ✅（仅 Markdown 消息内） |

## 5. 从旧 `qq` 通道迁移

新通道**与旧通道并存**，可以逐个切：

1. **配好 `qq_c2c`**（上面的最小配置），先别关旧的
2. 验证：给机器人发一条消息，应看到日志 `received C2C message` → `message sent`
3. **把依赖旧通道名的地方改掉**（这是最容易漏的一步）：
   - cron 任务的投递目标：`payload.channel` 从 `"qq"` 改成 `"qq_c2c"`
     （否则任务会报 `Unknown channel for outbound message` 并被丢弃）
   - 任何 `reasoning_channel_id` / 通知脚本里的通道名
4. 关掉旧通道：`"qq": { "enabled": false }`
5. 旧通道的 `qq.settings.chat_kinds` 不再需要 —— `qq_c2c` 的 chat 类型是显式配置的

### 为什么建议淘汰旧 `qq` 通道

- 协议压在 `tencent-connect/botgo` 里，SDK 出问题只能等（例如 `Authorization: QQBot <token>` scheme 丢失导致 401）
- 没有单聊**主动消息**、**流式**、**按钮**、**指令面板**
- 未知 chat id 一律按「群」处理，出站专用目标（cron/通知）会走错接口

## 6. 常见问题

**Q: 主动消息报 `40034105 无权限`？**
A: 机器人本身没有主动消息权限/配额，去 QQ 开放平台核对。

**Q: 报 `40054013 用户拒收消息`？**
A: 用户在 QQ 客户端关掉了「允许主动发送」开关。通道会自动记住这个状态并提前拒绝，日志有 `user disabled proactive messages`。

**Q: 日志里看不到发送细节？**
A: 成功路径是 `info` 级；把 `gateway.log_level` 调成 `info` 才能看到（默认 `warn` 只留错误）。

**Q: 用户群里/频道里发的消息收不到？**
A: 本通道只订阅 **单聊**（`C2C_MESSAGE_CREATE`）与 **互动**（`INTERACTION_CREATE`），不处理群/QQ 频道事件。

**Q: 不填 `allow_from` 会怎样？**
A: 通道会打 `SECURITY: Channel allows EVERYONE` 告警，任何人都能私聊机器人 —— 生产环境建议填上自己的 openid。

---

## 7. Launcher 配置界面

`picoclaw-launcher` 面板已规划对应的「QQ C2C Channel」配置卡片（AppID / AppSecret / Markdown / 流式 / 指令面板 / 白名单 / 节流毫秒）。
后端（本 PR）先落地配置结构，前端界面随后跟进。
