# Mattermost

Mattermost 是一个开源的团队协作平台。PicoClaw 通过 WebSocket API v4 和 REST API v4 连接到 Mattermost 服务器，支持接收和发送消息、文件上传、线程回复和输入指示器。

## 配置

```json
{
  "channels": {
    "mattermost": {
      "enabled": true,
      "url": "https://your-mattermost-server.com",
      "token": "YOUR_BOT_TOKEN",
      "username": "picoclaw",
      "reply_in_thread": true,
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

| 字段             | 类型   | 必填 | 描述                                     |
| ---------------- | ------ | ---- | ---------------------------------------- |
| enabled          | bool   | 是   | 是否启用 Mattermost 频道                 |
| url              | string | 是   | Mattermost 服务器地址                    |
| token            | string | 是   | 机器人访问令牌                           |
| username         | string | 否   | 机器人用户名（用于去除 @提及）           |
| reply_in_thread  | bool   | 否   | 在频道中自动使用线程回复（默认：true）   |
| allow_from       | array  | 否   | 用户ID白名单，空表示允许所有用户         |
| group_trigger    | object | 否   | 群组触发设置                             |
| typing           | object | 否   | 输入指示器设置                           |
| placeholder      | object | 否   | 占位消息设置（默认启用，文本："Thinking... 💭"）|

## 设置流程

1. 前往 Mattermost 管理后台 → 集成 → 机器人帐户 → 添加机器人帐户
2. 复制机器人令牌
3. 将令牌填入配置文件中
4. 将机器人添加到需要的频道
5. 通过私信或 @提及 与机器人交互

## 功能

- **线程回复**：频道消息自动使用线程，私信保持平面结构
- **自动重连**：WebSocket 断开后自动重连（指数退避 5s-60s）
- **消息分割**：超长消息自动分割（上限 4000 字符）
- **文件上传**：支持通过 MediaSender 接口上传文件
- **输入指示器**：支持显示"正在输入"状态
- **消息编辑**：支持编辑已发送的消息
- **占位消息**：发送"思考中..."占位消息，完成后替换为实际回复
