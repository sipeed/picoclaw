# 飞书

飞书（国际版名称：Lark）是字节跳动旗下的企业协作平台。它通过事件驱动的 Webhook 同时支持中国和全球市场。

## 配置

```json
{
  "channels": {
    "feishu": {
      "enabled": true,
      "app_id": "cli_xxx",
      "app_secret": "xxx",
      "encrypt_key": "",
      "verification_token": "",
      "allow_from": []
    }
  }
}
```

| 字段               | 类型   | 必填 | 描述                             |
| ------------------ | ------ | ---- | -------------------------------- |
| enabled            | bool   | 是   | 是否启用飞书频道                 |
| app_id             | string | 是   | 飞书应用的 App ID(以cli\_开头)   |
| app_secret         | string | 是   | 飞书应用的 App Secret            |
| encrypt_key        | string | 否   | 事件回调加密密钥                 |
| verification_token | string | 否   | 用于Webhook事件验证的Token       |
| allow_from         | array  | 否   | 用户ID白名单，空表示允许所有用户 |

## 设置流程

1. 前往 [飞书开放平台](https://open.feishu.cn/)创建应用程序
2. 获取 App ID 和 App Secret
3. **权限**：在「权限管理」中勾选并发布：
   - `im:message`（接收与发送消息）
   - `im:message.p2p_msg:readonly`（**获取用户发给机器人的单聊消息**，否则私聊收不到）
   - `im:message:send_as_bot`（以应用身份发消息）
   - 群聊需 `im:message.group_at_msg:readonly`；如需「用户进入单聊」事件可勾选 `im:chat.access_event.bot_p2p_chat:read`
4. **事件订阅**：在「事件订阅」中启用「长连接」（WebSocket），并订阅 **「接收消息」**（`im.message.receive_v1`）
5. 设置加密（可选，生产环境建议启用）
6. 将 App ID、App Secret、Encrypt Key 和 Verification Token（若启用加密）填入配置文件
7. **修改权限或事件后需重新发布应用版本**，否则不生效

## 机器人不回复时排查

- 看运行日志：发一条消息后是否出现 **「Feishu message received」**。
  - **没有**：飞书未把消息推给应用。请确认已订阅「接收消息」、已开通 `im:message.p2p_msg:readonly` 并重新发布版本。
  - **有「Dropping message: sender not in allow list」**：当前用户不在白名单；将 `allow_from` 设为 `[]` 表示允许所有人，或加入对应用户 ID。
  - **有「Feishu message received」但仍无回复**：消息已进入应用，问题在后续 agent 或发回飞书环节，请查看 agent 与通道的错误日志。
