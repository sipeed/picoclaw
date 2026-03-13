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
| allow_from         | array  | 否   | 用户ID白名单，空表示所有用户   |
| random_reaction_emoji | array | 否   | 随机添加的表情列表，空则使用默认 "Pin" |

## 设置流程

1. 前往 [飞书开放平台](https://open.feishu.cn/)创建应用程序
2. 获取 App ID 和 App Secret
3. 配置事件订阅和Webhook URL
4. 设置加密(可选,生产环境建议启用)
5. 将 App ID、App Secret、Encrypt Key 和 Verification Token(如果启用加密) 填入配置文件中
6. 自定义你希望 PicoClaw react 你消息时的表情（可选, Reference URL: [Feishu Emoji List](https://open.larkoffice.com/document/server-docs/im-v1/message-reaction/emojis-introduce)）

## 已支持能力

当前 `picoclaw` 的飞书通道已基于官方 Go SDK 接入并支持：

- WebSocket 收消息与事件订阅
- 文本 / Markdown 卡片发送
- 占位消息发送与卡片编辑
- 消息表情反应
- 图片、文件、音频、视频上传发送
- 入站消息图片/文件资源下载
- 消息详情查询、消息列表查询、消息回复
- 卡片消息解析（标题、文本、图片键、按钮）
- 用户查询、用户列表、按邮箱/手机号查用户
- 群聊信息查询、群成员列表、群列表、建群、群发消息
- 飞书消息分享链接 token 解析与消息查找辅助

这些能力覆盖了仓库中 `feishu-skill`、`feishu-file` Python 示例里的主要已实现功能；大文件分片上传仍未实现。

## 推荐权限

为了启用上面的能力，建议在飞书开放平台为应用配置至少以下权限（实际命名以飞书后台为准）：

- `im:message`
- `im:message:readonly`
- `im:message.resource`
- `im:chat`
- `im:chat:readonly`
- `contact:user.base:readonly`
- `contact:user.id:readonly`
- Drive / 云文档相关文件读写权限（用于文件上传、下载、查询、删除）

如果只需要基础聊天能力，可先只开启消息和群聊只读/发送权限。

## 开发说明

Go 侧飞书增强接口位于 `pkg/channels/feishu/`，可直接复用下列能力：

- `GetMessage` / `GetMessageByID`
- `ListMessages`
- `ReplyMessage`
- `GetUserInfo` / `ListUsers`
- `GetUserIDByEmail` / `GetUserIDByMobile`
- `GetGroupInfo` / `ListGroupMembers` / `ListGroups` / `CreateGroup`
- 入站图片 / 文件 / 音频资源自动下载与存储
- `GetMessageFromShareLink`
- `GetDriveRootFolder` / `GetDriveFolder` / `GetDriveFile` / `ListDriveFiles`
- `UploadDriveFile` / `DownloadDriveFile` / `DeleteDriveFile`
- `InitiateMultipartUpload` / `UploadMultipartChunk` / `CompleteMultipartUpload`

这些方法返回的是 PicoClaw 内部统一结构，而不是直接暴露 SDK 原始类型，便于后续被工具层或技能层复用。

## 工具层说明

当前工具层新增了 `feishu_parse`，用于在本地直接解析：

- 飞书消息内容 JSON
- 飞书卡片 JSON
- 飞书分享链接中的 token

这个工具不依赖远程飞书 API，适合在 agent 推理过程中快速理解飞书 payload。

`feishu_parse` 已经接入默认工具初始化流程；只要在工具配置中启用 `feishu_parse`，agent 就能直接调用。

同时，工具层已经准备好了 `feishu_remote` 的远程查询/操作接口模型。它支持通过注入的 Feishu 客户端统一暴露：

- 消息：`get_message`、`list_messages`、`reply_message`、`get_message_from_share_link`
- 用户：`get_user`、`list_users`、`get_user_id_by_email`、`get_user_id_by_mobile`
- 群组：`create_group`、`get_group`、`list_group_members`、`list_groups`、`send_group_message`
- Drive：`get_drive_root_folder`、`get_drive_folder`、`get_drive_file`、`list_drive_files`、`download_drive_file`、`delete_drive_file`、`upload_drive_file`
- 大文件上传：`initiate_multipart_upload`、`upload_multipart_chunk`、`complete_multipart_upload`

当前实现中，当运行时存在已初始化的飞书 channel 时，`AgentLoop.SetChannelManager(...)` 会自动把该 channel 适配为远程客户端，并注册 `feishu_remote` 工具。

> 说明：工具层并不直接依赖 `pkg/channels/feishu`。当前通过适配器接口完成注入，这样可以保持 channel 层与 tools 层解耦，同时支持未来替换成别的 Feishu client 实现。

## 集成测试

仓库中提供了飞书集成测试骨架：`pkg/channels/feishu/integration_test.go`

运行前请设置：

- `FEISHU_APP_ID`
- `FEISHU_APP_SECRET`

并使用 Go integration build tag 执行。当前骨架先覆盖参数校验与连通性起点，后续可继续增加真实消息、用户、群聊、Drive 的回归测试。
