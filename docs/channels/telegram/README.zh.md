> 返回 [README](../../project/README.zh.md)

# Telegram

Telegram Channel 通过 Telegram 机器人 API 使用长轮询实现基于机器人的通信。它支持文本消息、媒体附件（照片、语音、音频、文档）、语音转录（配置见[提供商与模型配置](../../guides/providers.zh.md#语音转录)），以及内置命令处理器。

## 配置

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "allow_from": ["123456789"],
      "settings": {
        "proxy": "",
        "use_markdown_v2": false,
        "business_mode": false,
        "business_owner": "123456789",
        "business_commands_enable": false,
        "guest_mode": false
      }
    }
  }
}
```

| 字段             | 类型   | 必填 | 描述                                                      |
| ---------------- | ------ | ---- | --------------------------------------------------------- |
| enabled          | bool   | 是   | 是否启用 Telegram 频道                                    |
| allow_from       | array  | 否   | 用户ID白名单，空表示允许所有用户                          |
| settings.proxy   | string | 否   | 连接 Telegram API 的代理 URL (例如 http://127.0.0.1:7890) |
| settings.use_markdown_v2 | bool | 否 | 启用 Telegram MarkdownV2 格式化                           |
| settings.business_mode | bool | 否 | 启用 Telegram Business 消息处理                           |
| settings.business_owner | string | 否 | 需要忽略的 Business 账号所有者 Telegram 用户 ID           |
| settings.business_commands_enable | bool | 否 | 允许在 Telegram Business 聊天中处理机器人命令             |
| settings.guest_mode | bool | 否 | 启用 Telegram Guest 消息处理和回复                        |

## 设置流程

1. 在 Telegram 中搜索 `@BotFather`
2. 发送 `/newbot` 命令并按照提示创建新机器人
3. 获取 HTTP API Token
4. 将 Token 填入配置文件中
5. (可选) 配置 `allow_from` 以限制允许互动的用户 ID (可通过 `@userinfobot` 获取 ID)

## Telegram Business Mode

设置 `settings.business_mode: true` 后，PicoClaw 会接收并回复已连接商业账号的 Telegram Business 消息。回复会使用传入的 `business_connection_id`，当机器人具备 `can_read_messages` 权限时，传入的 Business 消息会被标记为已读。

使用 `settings.business_owner` 保存商业账号所有者的 Telegram 用户 ID。来自该用户的 Business 消息会被忽略，避免机器人回复你从已连接商业账号手动发送的消息。

默认情况下，Business 聊天中的机器人命令会被忽略。设置 `settings.business_commands_enable: true` 后，可处理 `/new`、`/help`、`/show`、`/list` 和 `/use`。

示例：

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "allow_from": ["123456789"],
      "settings": {
        "business_mode": true,
        "business_owner": "123456789",
        "business_commands_enable": true
      }
    }
  }
}
```

## Telegram Guest Mode

设置 `settings.guest_mode: true` 后，PicoClaw 会接收 `guest_message` 更新，并使用 Telegram 的 `answerGuestQuery` 方法回复。Guest 消息来自机器人未加入的聊天，因此 PicoClaw 会使用传入的 `guest_query_id` 建立独立会话。

当 `settings.guest_mode` 为 `false` 时，PicoClaw 不会请求 Guest 更新，并会忽略任何已解码的 Guest 消息。Guest 回复的占位符和打字指示器会被跳过，因为 Telegram 需要单一的 `answerGuestQuery` 响应。

示例：

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "allow_from": ["123456789"],
      "settings": {
        "guest_mode": true
      }
    }
  }
}
```

## 内置命令

Telegram 会在启动时自动注册 PicoClaw 的顶级 Bot 命令，包括 `/start`、`/help`、`/show`、`/list` 和 `/use`。

与技能相关的命令：

- `/list skills`：列出当前 Agent 可见的已安装技能。
- `/use <skill> <message>`：只在本次请求中强制使用指定技能。
- `/use <skill>`：为同一聊天中的下一条消息预先启用该技能。
- `/use clear`：清除待应用的技能覆盖。

示例：

```text
/list skills
/use git explain how to squash the last 3 commits
/use git
explain how to squash the last 3 commits
```

## 高级格式化

您可以设置 `use_markdown_v2: true` 来启用增强的格式化选项。这允许机器人使用 Telegram MarkdownV2 的全部功能，包括嵌套样式、剧透和自定义等宽代码块。

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "allow_from": ["YOUR_USER_ID"],
      "settings": {
        "use_markdown_v2": true
      }
    }
  }
}
```
